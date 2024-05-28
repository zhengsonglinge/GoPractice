package main

import (
	"encoding/json"
	"fmt"
	"grpc"
	"grpc/codec"
	"log"
	"net"
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
	addr := make(chan string)
	go startServer(addr)

	// 使用 addr 管道的目的是等到服务端监听之后再发送客户端请求
	// 下面的代码就是简单的 rpc 客户端了
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)

	// 发送 options
	json.NewEncoder(conn).Encode(grpc.DefaultOption)
	// 通过连接创建编码格式
	cc := codec.NewGobCodec(conn)

	// 发送请求和接收响应
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		cc.Write(h, fmt.Sprintf("grpc req %d", h.Seq))
		cc.ReadHeader(h)
		var reply string
		cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}

/*
go run main.go
2024/05/29 00:33:03 start rpc server on [::]:35289
2024/05/29 00:33:04 &{Foo.Sum 0 } grpc req 0
2024/05/29 00:33:04 reply: grpc resp: 0
2024/05/29 00:33:04 &{Foo.Sum 1 } grpc req 1
2024/05/29 00:33:04 reply: grpc resp: 1
2024/05/29 00:33:04 &{Foo.Sum 2 } grpc req 2
2024/05/29 00:33:04 reply: grpc resp: 2
2024/05/29 00:33:04 &{Foo.Sum 3 } grpc req 3
2024/05/29 00:33:04 reply: grpc resp: 3
2024/05/29 00:33:04 &{Foo.Sum 4 } grpc req 4
2024/05/29 00:33:04 reply: grpc resp: 4
*/
