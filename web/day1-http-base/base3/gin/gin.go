package gin

import (
	"fmt"
	"net/http"
)

// 将 func 重命名，简化代码
type HandlerFunc func(http.ResponseWriter, *http.Request)

// 实现  http.Handler 接口
type Engine struct {
	// 用  map 存储路由，即作为一个静态路由表
	router map[string]HandlerFunc
}

// Engine 的构造器
func New() *Engine {
	return &Engine{
		router: make(map[string]HandlerFunc),
	}
}

// 实现 http.Handler 接口的 ServeHTTP 方法
// 如果 engine 的 router map 中存储了处理路由的方法，直接用方法处理路由，否则返回 404
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := req.Method + "-" + req.URL.Path
	if handler, ok := engine.router[key]; ok {
		handler(w, req)
	} else {
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}

// 往 engine.router 中添加路由
func (engine *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
	key := method + "-" + pattern
	engine.router[key] = handler
}

func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.addRoute("GET", pattern, handler)
}

func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.addRoute("POST", pattern, handler)
}

// 使用 Engine 代理所有 HTTP 请求
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}
