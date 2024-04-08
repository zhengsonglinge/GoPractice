package gin

import (
	"log"
	"net/http"
	"path"
	"strings"

	"html/template"
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

	// Go语言内置了text/template和html/template2个模板标准库，
	// 其中html/template为 HTML 提供了较为完整的支持。包括普通变量渲染、列表渲染、对象渲染等。
	htmlTemplates *template.Template // 将所有模板加载到内存
	funcMap       template.FuncMap   // 自定义模板渲染函数
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
	c.engine = engine
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

// 将所有的静态文件放在/usr/web目录下，那么filepath的值即是该目录下文件的相对地址。
// 映射到真实的文件后，将文件返回，静态服务器就实现了。
// http.FileSystem 实现了文件返回功能
// 之前的动态路由通配符 * 已经匹配了 filePath，filePath就是文件的相对地址，将相对路径映射到绝对路径后将文件返回
func (group *RouteGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(group.prefix, relativePath)
	// 将 url 中的路径代理到文件描述符服务器路径
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filePath")
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

// 用户将磁盘文件夹 root 映射到路由 relativePath 相对路径
// r.Static("/assets", "/usr/web/blog/static")
// 或相对路径 r.Static("/assets", "./static")
func (group *RouteGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	group.GET(urlPattern, handler)
}

/*
静态文件服务器是如何用 http 包实现的：
StripPrefix 是剥离前缀的意思
	fs := http.FileServer(http.Dir("/home/go/src/js"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
http.StripPrefix 方法会将请求路径中的 /static/ ，代理到文件系统的 fs 处理。
*/

// 自定义渲染函数 funcMap
func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

// 加载模板
func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(
		template.New("").
			Funcs(engine.funcMap).
			ParseGlob(pattern),
	)
}
