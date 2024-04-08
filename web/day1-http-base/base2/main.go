package main

import (
	"fmt"
	"log"
	"net/http"
)

// 结构体 Engine 实现了 http.Handler 接口
type Engine struct{}

// 实现 http.Handler 接口中的方法 serveHTTP
// 第一个参数用来构造响应，第二个参数中保存请求的所有信息
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/":
		fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
	case "/hello":
		for k, v := range req.Header {
			fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
		}
	default:
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}

func main() {
	// 因为第二参数是接口，是指针，这里需要 new 方法返回的指针
	engine := new(Engine)
	// 使用 engine 拦截所有 HTTP 请求自己处理
	log.Fatal(http.ListenAndServe(":9999", engine))
}
