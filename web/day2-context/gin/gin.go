package gin

import (
	"net/http"
)

// 使用 Conext 来替代原来的 http.RersonseWriter 和 http.Request
type HandlerFunc func(c *Context)

// 实现  http.Handler 接口
type Engine struct {
	// 用 router 结构体做路由表
	router *router
}

// Engine 的构造器
func New() *Engine {
	return &Engine{
		router: newRouter(),
	}
}

// 实现 http.Handler 接口的 ServeHTTP 方法
// 调用 router 结构体的方法处理路由
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := NewContext(w, req)
	engine.router.handle(c)
}

// 往 engine.router 中添加路由
func (engine *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
	engine.router.addRouter(method, pattern, handler)
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
