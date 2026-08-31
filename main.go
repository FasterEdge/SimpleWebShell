package main

// https://github.com/FasterEdge/SimpleWebShell
import (
	"SimpleWebShell/route"
	"flag"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

var (
	password  *string
	shellPath *string
	port      *string
)

func init() {
	password = flag.String("key", "", "密码 (必需)")
	shellPath = flag.String("shell", "/bin/bash", "Shell路径 (Windows默认: cmd, Linux默认: /bin/bash)")
	port = flag.String("port", "8878", "监听端口 (默认: 8878)")
}

func main() {
	version := "1.0.20260831" // 当前系统的版本
	flag.Parse()

	// 检查必需参数
	if *password == "" {
		log.Fatal("错误: 必须指定密码参数 -key")
	}

	// 输出参数
	fmt.Printf("SimpleWebShell %s By FasterEdge\n", version)
	fmt.Printf("开源官网：https://github.com/FasterEdge\n")
	fmt.Printf("SimpleWebShell仅用于边缘计算场景中远程操作和热更新，禁止用于非法用途！\n")
	fmt.Printf("密码已设置为：%s\n", *password)
	fmt.Printf("Shell路径: %s\n", *shellPath)
	fmt.Printf("监听端口: %s\n", *port)

	// 设置Gin为发布模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin路由器
	r := gin.Default()

	// 设置路由
	route.SetupRoutes(r, *password, *shellPath, *port)

	// 启动服务器
	fmt.Printf("服务器启动在端口 %s\n", *port)
	log.Fatal(r.Run(":" + *port))
}
