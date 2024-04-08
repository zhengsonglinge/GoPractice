package gin

import (
	"log"
	"net/http"
)

type HandlerFunc func(c *Context)

// 所有和路由相关的函数，都可以交给 RouteGroup 操作
type RouteGroup struct {
	prefix      string
	middlewares []HandlerFunc // 支持中间件
	engine      *Engine       // 全局创建一个 Engine，保留一个指针，所有的 RouteGroup 指向同一个 Engine
}

// 类似于继承，Engine 继承自 RouteGroup，拥有 RouteGroup 所有属性和方法
// (go 没有继承，只有委托)
// Engine 负责所有请求代理，不仅有分组路由，还有其他功能
type Engine struct {
	router      *router
	*RouteGroup               // 内嵌 RouteGroup
	groups      []*RouteGroup // 存储所有的路由组
}

// Engine 的构造器
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouteGroup = &RouteGroup{engine: engine}
	engine.groups = []*RouteGroup{engine.RouteGroup}
	return engine
}

// 实现 http.Handler 接口的 ServeHTTP 方法
// 调用 router 结构体的方法处理路由
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := NewContext(w, req)
	engine.router.handle(c)
}

// 使用 Engine 代理所有 HTTP 请求
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

// 创建新的 Group 路由分组并返回这个新的 RouteGroup
func (group *RouteGroup) Group(prefix string) *RouteGroup {
	engine := group.engine
	newGroup := &RouteGroup{
		prefix: group.prefix + prefix,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

// 所有和路由相关的函数，都可以交给 RouteGroup 操作
func (group *RouteGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRouter(method, pattern, handler)
}

func (group *RouteGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

func (group *RouteGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}
