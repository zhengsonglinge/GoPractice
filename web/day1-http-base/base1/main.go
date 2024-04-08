package main

import (
	"fmt"
	"log"
	"net/http"
)

/*
ListendAndServer 函数的第二个参数是 http.Handler
任何结构体只要实现了 ServeHTTP 方法即可传入第二个参数，用来代理所有的 HTTP 请求

	package http
	type Handler interface {
	    ServeHTTP(w ResponseWriter, r *Request)
	}
	func ListenAndServe(address string, h Handler) error
*/
func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/hello", helloHandler)
	// 第一个参数是监听地址和端口
	// 第二个参数是处理所有 HTTP 请求的实例，使用 nil 表示使用标准库中的实例
	log.Fatal(http.ListenAndServe(":9999", nil))
}

func indexHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
}

func helloHandler(w http.ResponseWriter, req *http.Request) {
	for k, v := range req.Header {
		fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
	}
}
