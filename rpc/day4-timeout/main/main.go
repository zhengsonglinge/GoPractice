package main

import (
	"context"
	"grpc"
	"log"
	"net"
	"sync"
	"time"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

// 开启一个服务
func startServer(addr chan string) {
	// 注册服务
	var foo Foo
	if err := grpc.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}

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
			// 构造参数
			args := &Args{Num1: i, Num2: i * i}
			var reply int
			// 调用同步接口发送请求
			// 添加超时 context
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			if err := client.Call(ctx, "Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}

/*
rpc server: register Foo.Sum
start rpc server on [::]:34305
reply: 12
reply: 0
reply: 2
reply: 6
reply: 20
*/
