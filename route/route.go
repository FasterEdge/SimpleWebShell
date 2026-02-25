package route

import (
	"github.com/gin-gonic/gin"
)

var (
	password  string
	shellPath string
)

func SetupRoutes(r *gin.Engine, passwordPtr string, shellPathPtr string, portPtr string) {
	// 保存传入的配置到包作用域变量，供handler使用
	password = passwordPtr
	shellPath = shellPathPtr

	// 设置路由
	r.GET("/", handleRoot)                    // 欢迎界面
	r.GET("/get", handleGet)                  // 处理GET请求执行shell命令
	r.POST("/post", handlePost)               // 处理POST请求执行shell命令
	r.POST("/file_send", handleFileSend)      // 处理文件上传
	r.GET("/file_receive", handleFileReceive) // 处理文件下载
}
