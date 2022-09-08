package gee

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
)

//	打印用于调试的堆栈跟踪
func trace(message string) string {
	var pcs [32]uintptr             //	uintptr 足够大的 容纳任何指针
	n := runtime.Callers(3, pcs[:]) //	跳过前三个 caller
	//	第0个caller是callers本身，第1个是上一层的trace，第2个是上一层的defer func() 为了简洁，跳过前三个

	var str strings.Builder
	str.WriteString(message + "\nTraceback:")
	for _, pc := range pcs[:n] {
		//	获取对应的函数
		fn := runtime.FuncForPC(pc)
		//	获取文件名和行号
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return str.String()
}

func Recovery() HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				log.Printf("%s\n\n", trace(message))
				//	500
				c.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()

		c.Next()
	}
}
