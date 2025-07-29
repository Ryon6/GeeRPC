package main

import (
	"context"
	"flag"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"geerpc/codec"
	"geerpc/geerpc"
	Test "geerpc/test"
	"geerpc/xclient"
)

var (
	serverAddr = flag.String("server", "", "server address")
	clientNum  = flag.Int("clients", 2000, "number of clients")
	duration   = flag.Duration("duration", 10*time.Second, "test duration")
)

func main() {
	const registryAddr = "http://localhost:8001/geerpc/demo_registry"
	service_method := "CalcService.Add"

	flag.Parse()

	var (
		success uint64
		failed  uint64
		wg      sync.WaitGroup
	)

	serverAddr := make(chan string, 1)
	wg.Add(1)
	go Test.StartRegistry(&wg, registryAddr)
	wg.Wait()
	wg.Add(1)
	go Test.StartXServer("tcp", &wg, registryAddr, serverAddr)
	wg.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	start := time.Now()

	// Start clients
	for i := 0; i < *clientNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var opt *geerpc.Option = &geerpc.Option{
				CodecType:      codec.JsonType,
				MagicNumber:    geerpc.DefaultOption.MagicNumber,
				ConnectTimeout: geerpc.DefaultOption.ConnectTimeout,
				HandleTimeout:  geerpc.DefaultOption.HandleTimeout,
			}

			d := xclient.NewGeeRegistryDiscovery(registryAddr, 0)
			client := xclient.NewXClient(d, xclient.RandomSelect, opt)
			defer client.Close()

			for ctx.Err() == nil {
				var reply int
				err := client.Call(ctx, service_method, &Test.CArgs{A: 1, B: 2}, &reply)
				if err != nil {
					atomic.AddUint64(&failed, 1)
					log.Printf("call failed: %v", err)
					continue
				}
				atomic.AddUint64(&success, 1)
			}
		}()
	}

	// Wait for test to complete
	<-ctx.Done()
	wg.Wait()

	elapsed := time.Since(start)
	total := success + failed
	qps := float64(success) / elapsed.Seconds()

	log.Printf("Clients: %d, Duration: %v", *clientNum, elapsed)
	log.Printf("Success: %d, Failed: %d, Total: %d", success, failed, total)
	log.Printf("QPS: %.2f", qps)
}

type Args struct {
	Num1, Num2 int
}
