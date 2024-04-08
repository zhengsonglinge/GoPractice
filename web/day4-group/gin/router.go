package gin

import (
	"net/http"
	"strings"
)

// 路由结构体
// 使用 map 存储前缀树和路径对应的处理方法
// roots key eg, roots['GET'] roots['POST']
// handlers key eg, handlers['GET-/p/:lang/doc'], handlers['POST-/p/book']
type router struct {
	roots    map[string]*node
	handlers map[string]HandlerFunc
}

// 路由构造器
func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// 解析注册的路径
// 只允许有一个 *
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")
	parts := make([]string, 0)
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

// 添加路由
func (r *router) addRouter(method string, pattern string, handler HandlerFunc) {
	parts := parsePattern(pattern)
	key := method + "-" + pattern

	// 前缀树是以方法作为根节点的，不存在则创建
	if _, ok := r.roots[method]; !ok {
		r.roots[method] = &node{}
	}

	// 在前缀树中递归插入路由
	r.roots[method].insert(pattern, parts, 0)
	r.handlers[key] = handler
}

// 查找路由，返回值是前缀树叶子节点和动态路由的参数
// 动态路由的参数放到 map 里面
func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	params := make(map[string]string)
	root, ok := r.roots[method]

	if !ok {
		return nil, nil
	}

	// 递归查找前缀树叶子节点
	n := root.search(searchParts, 0)

	if n != nil {
		// 获取前缀树注册过的路径的数组
		parts := parsePattern(n.pattern)
		// 获取模糊匹配的参数，例如 /p/:lang 匹配 /p/go 则，将 lang = go
		// /static/*filepath，可以匹配 /static/js/jQuery.js，将 filepath = js/jQuery.js
		for index, part := range parts {
			if part[0] == ':' {
				params[part[1:]] = searchParts[index]
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}
	return nil, nil
}

// 查找路由，使用路由方法处理参数
func (r *router) handle(c *Context) {
	// 在调用匹配到的 handler 前，将解析出来的路由参数赋值给了 c.Params。
	// 这样就能够在 handler 中，通过 Context 对象访问到具体的值了
	node, params := r.getRoute(c.Method, c.Path)
	if node != nil {
		c.Params = params
		key := c.Method + "-" + node.pattern
		r.handlers[key](c)
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	}
}
