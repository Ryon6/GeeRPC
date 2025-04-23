package Test

import (
	"context"
	. "geerpc/geerpc"
	"geerpc/registry"
	"geerpc/xclient"
	"log"
	"net"
	"net/http"
	"sync"
)

// 一个url，服务发现会通过向该url发送GET请求，获取注册到注册中心的方法
const registryPath = "http://localhost:8001/geercp/demo_registry"

func startServer(wg *sync.WaitGroup, registryAddr string) {
	var cs CalcService
	// 监听所有接口：
	// 绑定到 [::]:12345 或 0.0.0.0:12345 表示监听所有网络接口上的该端口。
	// 无论客户端通过 IPv4 或 IPv6 地址访问，只要目标端口是 12345，监听器都会接收连接请求。
	l, _ := net.Listen("tcp", ":8002")
	log.Printf("所有到%s的请求都将被server处理", l.Addr().String())
	server := NewServer()
	server.Register(&cs) //将cs提供的方法都注册到server.serviceMap中
	registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	wg.Done()
	server.Accept(l) //
}

// 启动注册中心
func startRegistry(wg *sync.WaitGroup, registryPath string) {
	// 创建注册中心实例
	r := registry.NewGeeRegistry(0)
	r.HandleHTTP("/geercp/demo_registry")
	l, _ := net.Listen("tcp", ":8001")
	log.Printf("所有到%s的请求都会被传递给注册中心的r.serveHTTP()", l.Addr().String())
	wg.Done()
	http.Serve(l, r)
}

func DemoCalc() {
	var wg sync.WaitGroup
	wg.Add(1)
	go startServer(&wg, registryPath)
	wg.Add(1)
	go startRegistry(&wg, registryPath)
	wg.Wait()

	d := xclient.NewGeeRegistryDiscovery(registryPath, 0) // 服务发现registryPath的注册中心发送HTTP报文发现服务

	xc := xclient.NewXClient(d, xclient.RandomSelect, nil) //
	args := &CArgs{A: 100, B: 200}
	var reply int
	err := xc.Call(context.Background(), "CalcService.Add", args, &reply)
	log.Println(reply, err)
}
