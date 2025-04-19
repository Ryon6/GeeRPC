package Test

import (
	"encoding/json"
	"fmt"
	"gee-rpc/codec"
	geerpc "gee-rpc/geerpc"
	"log"
	"net"
	"time"
)

func StartServer(addr chan string) {
	lis, err := net.Listen("tcp", ":8001")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", lis.Addr())
	addr <- lis.Addr().String()
	geerpc.Accept(lis)
}

func DemoMain01() {
	addr := make(chan string)
	go StartServer(addr)

	// in fact, following code is like a simple geerpc client
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)

	// send option
	_ = json.NewEncoder(conn).Encode(geerpc.DefaultOption)
	cc := codec.NewGobCodec(conn)
	// send request & receive response
	for i := 0; i < 5; i++ {
		h := codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		_ = cc.Write(&h, fmt.Sprintf("geerpc req %d", h.Seq))
		_ = cc.ReadHeader(&h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}
