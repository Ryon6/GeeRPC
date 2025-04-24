package xclient

import (
	"context"
	. "geerpc/geerpc"
	"io"
	"reflect"
	"sync"
)

type XClient struct {
	d       Discovery
	mode    SelectMode
	opt     *Option
	mu      sync.Mutex // protect following
	clients map[string]*Client
}

var _ io.Closer = (*XClient)(nil)

func NewXClient(d Discovery, mode SelectMode, opt *Option) *XClient {
	return &XClient{d: d, mode: mode, opt: opt, clients: make(map[string]*Client)}
}

func (xc *XClient) Close() error {
	xc.mu.Lock()
	defer xc.mu.Unlock()
	for key, client := range xc.clients {
		// ignore how to deal with the error
		_ = client.Close()
		delete(xc.clients, key)
	}
	return nil
}

// 复用已经创建好的 Socket 连接
// 使用 clients 保存创建成功的 Client 实例
func (xc *XClient) dial(rpcAddr string) (*Client, error) {
	xc.mu.Lock()
	defer xc.mu.Unlock()
	client, ok := xc.clients[rpcAddr]
	if ok && !client.IsAvailable() {
		_ = client.Close()
		delete(xc.clients, rpcAddr)
		client = nil
	}

	if client == nil {
		var err error
		client, err = XDial(rpcAddr, xc.opt)
		if err != nil {
			return nil, err
		}
		xc.clients[rpcAddr] = client
	}

	return client, nil
}

func (xc *XClient) call(rpcAddr string, ctx context.Context, serviceMethod string, args, reply interface{}) error {
	client, err := xc.dial(rpcAddr)
	if err != nil {
		return err
	}
	return client.Call(ctx, serviceMethod, args, reply)
}

func (xc *XClient) goCall(rpcAddr string, serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	client, err := xc.dial(rpcAddr)
	if err != nil {
		call := &Call{
			ServiceMethod: serviceMethod,
			Error:         err,
		}
		if done != nil {
			done <- call
		}
		return call
	}
	return client.Go(serviceMethod, args, reply, done)
}

// Call invokes the named function, waits for it to complete,
// and returns its error status.
// xc will choose a proper server.
func (xc *XClient) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	rpcAddr, err := xc.d.Get(xc.mode)
	if err != nil {
		return err
	}
	return xc.call(rpcAddr, ctx, serviceMethod, args, reply)
}

// Go invokes the function asynchronously. It returns the Call structure representing
// the invocation. The done channel will signal when the call is complete by returning
// the same Call object. If done is nil, the channel will be allocated automatically.
// If non-nil, done must be buffered or Go will deliberately crash.
func (xc *XClient) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	rpcAddr, err := xc.d.Get(xc.mode)
	if err != nil {
		call := &Call{
			ServiceMethod: serviceMethod,
			Error:         err,
		}
		if done != nil {
			done <- call
		}
		return call
	}
	return xc.goCall(rpcAddr, serviceMethod, args, reply, done)
}

// AsyncCall invokes the function asynchronously and returns a channel that will
// receive the result when the call completes.
func (xc *XClient) AsyncCall(serviceMethod string, args, reply interface{}) <-chan *Call {
	done := make(chan *Call, 10)
	xc.Go(serviceMethod, args, reply, done)
	return done
}

func (xc *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	servers, err := xc.d.GetAll()
	if err != nil {
		return err
	}
	var mu sync.Mutex // protect e and replyDone
	var wg sync.WaitGroup
	var e error
	replyDone := reply == nil // if reply is nil, don't need to set value
	ctx, cancel := context.WithCancel(ctx)
	for _, rpcAddr := range servers {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var clonedReply interface{}
			if reply != nil {
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			// 某调用失败后，cancel根据ctx通知停止正在执行xc.call的协程
			err := xc.call(rpcAddr, ctx, serviceMethod, args, clonedReply)
			mu.Lock()
			if err != nil && e == nil {
				e = err
				cancel()
			}
			if err == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mu.Unlock()
		}(rpcAddr)
	}
	wg.Wait()
	return e
}

// AsyncBroadcast invokes the function on all servers asynchronously and returns
// a channel that will receive all results as they complete.
func (xc *XClient) AsyncBroadcast(ctx context.Context, serviceMethod string, args, reply interface{}) <-chan *Call {
	servers, err := xc.d.GetAll()
	if err != nil {
		done := make(chan *Call, 1)
		done <- &Call{
			ServiceMethod: serviceMethod,
			Error:         err,
		}
		return done
	}

	done := make(chan *Call, len(servers))
	var clonedReply interface{}
	if reply != nil {
		clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
	}

	for _, rpcAddr := range servers {
		go func(rpcAddr string) {
			err := xc.call(rpcAddr, ctx, serviceMethod, args, clonedReply)
			call := &Call{
				ServiceMethod: serviceMethod,
				Error:         err,
			}
			if err == nil && reply != nil {
				call.Reply = clonedReply
			}
			done <- call
		}(rpcAddr)
	}

	return done
}
