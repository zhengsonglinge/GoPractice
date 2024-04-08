package gin

import (
	"log"
	"time"
)

// 自定义的中间件
func Logger() HandlerFunc {
	return func(c *Context) {
		// 计算程序运行时间
		t := time.Now()
		c.Next()
		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}
