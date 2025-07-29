package Test

import (
	"geerpc/geerpc"
	"geerpc/registry"
	"net"
	"net/http"
	"sync"
)

func StartRegistry(wg *sync.WaitGroup, path string) {
	r := registry.NewGeeRegistry(0)
	r.HandleHTTP(path)
	l, _ := net.Listen("tcp", ":8001")
	wg.Done()
	_ = http.Serve(l, r)
}

func StartXServer(serverType string, wg *sync.WaitGroup, regAddr string, serverAddr chan string) {
	switch serverType {
	case "http":
		startHTTPServer(wg, regAddr, serverAddr)
	case "tcp":
		startTCPServer(wg, regAddr, serverAddr)
	default:
		panic("未知服务器类型: " + serverType)
	}
}

func startHTTPServer(wg *sync.WaitGroup, reg_addr string, serverAddr chan string) {
	// 创建http server，注册服务service，发送心跳，监听端口
	server := geerpc.NewServer()
	var cs CalcService
	server.Register(&cs)
	server.HandleHTTP()

	l, _ := net.Listen("tcp", ":0") // 创建监听器
	addr := "http@" + l.Addr().String()
	registry.Heartbeat(reg_addr, addr, 0)
	serverAddr <- addr

	wg.Done()
	http.Serve(l, server)
}

func startTCPServer(wg *sync.WaitGroup, reg_addr string, serverAddr chan string) {
	// 创建tcp server，注册服务service，发送心跳，监听端口
	server := geerpc.NewServer()
	var cs CalcService
	server.Register(&cs)

	l, _ := net.Listen("tcp", ":0") // 创建监听器
	addr := "tcp@" + l.Addr().String()
	registry.Heartbeat(reg_addr, "tcp@"+l.Addr().String(), 0)
	serverAddr <- addr

	wg.Done()
	server.Accept(l)
}
