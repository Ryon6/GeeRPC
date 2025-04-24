package Test

import (
	"context"
	"fmt"
	"geerpc/geerpc"
	"geerpc/xclient"
	"log"
	"sync"
	"time"
)

func Demo() {
	const registryAddr = "http://localhost:8001/geerpc/demo_registry"
	var wg sync.WaitGroup
	var server_addr = make(chan string, 10)
	wg.Add(1)
	go startRegistry(&wg, registryAddr)
	wg.Wait()
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go startXServer("tcp", &wg, registryAddr, server_addr)
	}
	wg.Add(1)
	go startXServer("http", &wg, registryAddr, server_addr)
	wg.Wait()

	size := 200000
	service_method := "CalcService.Add"

	addr := <-server_addr
	client, err := geerpc.XDial(addr, nil)
	if err != nil {
		log.Fatal("创建客户端失败:", err)
	}

	// 调用封装好的异步测试函数
	results, err := asyncCallDemo(client, service_method, size)
	if err != nil {
		log.Println("异步调用测试失败:", err)
	} else {
		log.Println("异步调用测试成功, 结果:", len(results))
	}

	// 新增xclient异步调用演示
	// demoXClientAsync(registryAddr)
}

func demoXClientAsync(registryAddr string) {
	fmt.Println("\n=== xclient异步调用演示 ===")

	// 创建XClient
	discovery := xclient.NewGeeRegistryDiscovery(registryAddr, 0)
	xc := xclient.NewXClient(discovery, xclient.RandomSelect, nil)
	defer xc.Close()

	// 示例1: 使用Go方法
	fmt.Println("--- Go方法示例 ---")
	call := xc.Go("CalcService.Add", &CArgs{A: 10, B: 20}, new(int), nil)
	<-call.Done
	if call.Error != nil {
		log.Printf("调用失败: %v", call.Error)
	} else {
		log.Printf("10 + 20 = %d", *call.Reply.(*int))
	}

	// 示例2: 使用AsyncCall方法
	fmt.Println("\n--- AsyncCall方法示例 ---")
	done := xc.AsyncCall("CalcService.Add", &CArgs{A: 30, B: 40}, new(int))
	call = <-done
	if call.Error != nil {
		log.Printf("调用失败: %v", call.Error)
	} else {
		log.Printf("30 + 40 = %d", *call.Reply.(*int))
	}

	// 示例3: 使用AsyncBroadcast方法
	fmt.Println("\n--- AsyncBroadcast方法示例 ---")
	results := xc.AsyncBroadcast(context.Background(), "CalcService.Add", &CArgs{A: 50, B: 60}, new(int))

	// 缺失结束逻辑，会一直阻塞等待results
	for call := range results {
		if call.Error != nil {
			log.Printf("广播调用失败: %v", call.Error)
		} else {
			log.Printf("50 + 60 = %d (来自 %s)", *call.Reply.(*int), call.ServiceMethod)
		}
	}
}

// asyncCallDemo 封装高性能客户端异步调用测试
func asyncCallDemo(client *geerpc.Client, method string, size int) ([]int, error) {
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	// 使用并发安全的map存储结果
	resultMap := &sync.Map{}
	errChan := make(chan error, size)

	// 创建工作池
	workerPool := make(chan struct{}, 10) // 限制并发数为10
	for i := 0; i < size; i++ {
		wg.Add(1)
		workerPool <- struct{}{} // 获取worker

		go func(idx int) {
			defer func() {
				<-workerPool // 释放worker
				wg.Done()
			}()

			args := &CArgs{A: 100 * idx, B: 100 * idx}
			var reply int
			call := client.Go(method, args, &reply, nil)

			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
			case <-call.Done:
				if call.Error != nil {
					errChan <- fmt.Errorf("调用%d失败: %v", idx, call.Error)
				} else {
					resultMap.Store(idx, reply)
				}
			}
		}(i)
	}

	// 等待所有调用完成
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 收集错误
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	// 如果有错误则返回第一个错误
	if len(errs) > 0 {
		return nil, errs[0]
	}

	// 按顺序收集结果
	results := make([]int, size)
	resultMap.Range(func(key, value interface{}) bool {
		results[key.(int)] = value.(int)
		return true
	})

	return results, nil
}
