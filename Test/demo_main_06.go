// 服务发现与负载均衡

package Test

import (
	"context"
	"geerpc/geerpc"
	"geerpc/xclient"
	"log"
	"net"
	"sync"
	"time"
)

func startServer_06(addrCh chan string) {
	var foo Foo
	l, _ := net.Listen("tcp", ":0")
	server := geerpc.NewServer()
	_ = server.Register(&foo)
	addrCh <- l.Addr().String()
	server.Accept(l)
}

func call_06(addr1, addr2 string) {
	d := xclient.NewMultiServerDiscovery([]string{"tcp@" + addr1, "tcp@" + addr2})
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "call", "Foo.Sum", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func broadcast_06(addr1, addr2 string) {
	d := xclient.NewMultiServerDiscovery([]string{"tcp@" + addr1, "tcp@" + addr2})
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			// expect 2 - 5 timeout
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func DemoMain06() {
	log.SetFlags(0)
	ch1 := make(chan string)
	ch2 := make(chan string)
	// start two servers
	go startServer_06(ch1)
	go startServer_06(ch2)

	addr1 := <-ch1
	addr2 := <-ch2

	time.Sleep(time.Second)
	call_06(addr1, addr2)
	broadcast_06(addr1, addr2)
}
