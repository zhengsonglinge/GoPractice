package gin

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
)

// 获取触发 panic 的堆栈信息
func trace(message string) string {
	var pcs [32]uintptr
	// Callers 用来返回调用栈的程序计数器,
	// 第 0 个 Caller 是 Callers 本身，第 1 个是上一层 trace，第 2 个是再上一层的 defer func。
	// 为了日志简洁，跳过了前 3 个 Caller。
	n := runtime.Callers(3, pcs[:])

	var str strings.Builder
	str.WriteString(message + "\nTraceback:")
	for _, pc := range pcs[:n] {
		// 获取对应函数
		fn := runtime.FuncForPC(pc)
		// 获取对应函数名和行号
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return str.String()
}

// 之前实现了中间件的作用，错误处理可以作为一个中间件加强框架能力
func Recovery() HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				log.Printf("%s\n\n", trace(message))
				c.Fail(http.StatusInternalServerError, "InternalServerError")
			}
		}()
		c.Next()
	}
}
