package gin

import (
	"log"
	"net/http"
	"strings"
)

type HandlerFunc func(c *Context)

// 所有和路由相关的函数，都可以交给 RouteGroup 操作
type RouteGroup struct {
	prefix      string
	middlewares []HandlerFunc // 支持中间件
	engine      *Engine       // 全局创建一个 Engine，保留一个指针，所有的 RouteGroup 指向同一个 Engine
}

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

// 将中间件放到路由组的 middlewares 数组中
func (group *RouteGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

// 每个请求调用一次 ServeHTTP 处理中间件和用户方法
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	middlewares := make([]HandlerFunc, 0)
	// 遍历所有的路由组
	for _, group := range engine.groups {
		// 用请求路径的前缀来判断适用于哪些路由组，将路由组中的中间件存放到 middlewares 中间件中
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := NewContext(w, req)
	// 将 middlewares 中间件赋值给 c.handlers，中间件也是一种 handler，与用户定义的 handler 相同，存放在同一个数组中
	c.handlers = middlewares
	// 在 router.handle 方法中将当前请求用户定义的 handler 放到 c.handlers 数组中
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
