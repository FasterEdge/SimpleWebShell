// FasterEdge 开源项目 · https://github.com/FasterEdge · https://gitee.com/FasterEdge
package main

// https://github.com/FasterEdge/SimpleWebShell
import (
	"SimpleWebShell/route"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

var (
	password  *string
	shellPath *string
	port      *string
	useTLS    *bool
	certFile  *string
	keyFile   *string
)

func init() {
	password = flag.String("key", "", "密码 (必需)")
	shellPath = flag.String("shell", "/bin/bash", "Shell路径 (Windows默认: cmd, Linux默认: /bin/bash)")
	port = flag.String("port", "8878", "监听端口 (默认: 8878)")
	useTLS = flag.Bool("tls", false, "启用HTTPS (需通过 -cert/-keyfile 指定证书与私钥路径)")
	certFile = flag.String("cert", "", "TLS证书文件路径 (PEM格式，-tls=true时必需)")
	keyFile = flag.String("keyfile", "", "TLS私钥文件路径 (PEM格式，-tls=true时必需)")
}

func main() {
	version := "1.0.20260901" // 当前系统的版本
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

	// 校验TLS参数
	if *useTLS {
		if *certFile == "" || *keyFile == "" {
			log.Fatal("错误: 启用HTTPS(-tls=true)时必须同时指定 -cert 证书路径与 -keyfile 私钥路径")
		}
		if err := validateCertFiles(*certFile, *keyFile); err != nil {
			log.Fatalf("错误: TLS证书/私钥无效: %v", err)
		}
		fmt.Printf("已启用HTTPS: 证书=%s 私钥=%s\n", *certFile, *keyFile)
	}

	// 设置Gin为发布模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin路由器。
	// 注意：不使用 gin.Default()，因为其默认日志会打印完整 URL（含 ?key=密码），
	// 导致密码泄露到服务日志。改用 gin.New() + 自定义日志（仅打印路径，不含查询串）。
	r := gin.New()
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 仅打印路径 path，不含 RawQuery，避免 key=密码 泄露到日志
		// （gin v1.12 的 param.Path 会拼上查询串，故用 Request.URL.Path）
		path := param.Request.URL.Path
		return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			path,
		)
	}))
	r.Use(gin.Recovery())

	// 设置路由
	route.SetupRoutes(r, *password, *shellPath, *port)

	// 启动服务器
	if *useTLS {
		fmt.Printf("HTTPS服务器启动在端口 %s\n", *port)
		log.Fatal(r.RunTLS(":"+*port, *certFile, *keyFile))
	}
	fmt.Printf("服务器启动在端口 %s\n", *port)
	log.Fatal(r.Run(":" + *port))
}

// validateCertFiles 校验TLS证书/私钥路径存在且能够配对解析（PEM格式）。
// 使用 crypto/tls 实际加载证书对，可提前发现证书与私钥不匹配、文件损坏等暗病。
func validateCertFiles(certPath, keyPath string) error {
	for _, path := range []string{certPath, keyPath} {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("%s 不可访问: %v", path, err)
		}
		if info.IsDir() {
			return fmt.Errorf("%s 是目录而非文件", path)
		}
	}
	// 尝试完整解析证书+私钥对，确保两者匹配且为有效 PEM
	if _, err := tls.LoadX509KeyPair(certPath, keyPath); err != nil {
		return fmt.Errorf("证书与私钥解析失败(请确认为PEM格式且配对一致): %v", err)
	}
	return nil
}
