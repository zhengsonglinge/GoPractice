package main

import (
	"fmt"
	"grpc"
	"log"
	"net"
	"sync"
	"time"
)

// 开启一个服务
func startServer(addr chan string) {
	// 选一个随机端口监听
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	grpc.Accept(l)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)

	// 创建 client 的时候就已经建立连接了，用于接收请求
	client, _ := grpc.Dial("tcp", <-addr)
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)

	// 发送请求并接收请求
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("grpc req %d", i)
			var reply string
			// 调用同步接口发送请求
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}

/*
start rpc server on [::]:41397
&{Foo.Sum 5 } grpc req 0
&{Foo.Sum 2 } grpc req 4
&{Foo.Sum 1 } grpc req 1
&{Foo.Sum 3 } grpc req 2
&{Foo.Sum 4 } grpc req 3
reply: grpc resp: 4
reply: grpc resp: 2
reply: grpc resp: 5
reply: grpc resp: 1
reply: grpc resp: 3
*/
