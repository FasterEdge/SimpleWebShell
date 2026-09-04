// ─────────────────────────────────────────────────────────────
// FasterEdge 开源项目
// Github: https://github.com/FasterEdge
// Gitee:  https://gitee.com/FasterEdge
// ─────────────────────────────────────────────────────────────
package route

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"github.com/FasterEdge/SimpleWebShell/pages"
	"github.com/FasterEdge/SimpleWebShell/session"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"os/exec"

	"github.com/gin-gonic/gin"
)

// secureCompare 常量时间比较两个字符串，避免时序攻击泄露密码信息。
func secureCompare(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}

// 处理根路径
func handleRoot(c *gin.Context) {
	// 检查密钥
	key := c.Query("key")
	if !secureCompare(key, password) {
		// 密钥不正确，只显示基本信息
		c.String(http.StatusOK, "SimpleWebshell 1.0.20260901 By FasterEdge")
		return
	}

	// 密钥正确，显示WebShell界面
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, pages.GetWebShellHTML())
}

// session 创建
func handleSessionCreate(c *gin.Context) {
	key := c.Query("key")
	if !secureCompare(key, password) {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}
	id := session.New()
	c.String(http.StatusOK, id)
}

// session 列表
func handleSessionList(c *gin.Context) {
	key := c.Query("key")
	if !secureCompare(key, password) {
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
	if !secureCompare(key, password) {
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
	if !secureCompare(key, password) {
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
	if !secureCompare(key, password) {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessID := c.Query("session")
	if sessID != "" && !session.Exists(sessID) {
		c.String(http.StatusBadRequest, "session 不存在")
		return
	}

	// 获取命令（Gin 的 c.Query 已做 URL 解码，勿再二次解码，
	// 否则含字面 % 的命令会被破坏或报"解码错误"）
	cmd := c.Query("cmd")
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
	if !secureCompare(key, password) {
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
	// 仅当命令以 "cd" 作为独立词开头时才走会话内目录切换，
	// 避免把 cdecl、cdrecord 等以 cd 开头的命令误判为目录切换
	if trim == "cd" || strings.HasPrefix(trim, "cd ") || strings.HasPrefix(trim, "cd\t") {
		parts := strings.SplitN(cmdStr, "&&", 2)
		first := strings.TrimSpace(parts[0])
		target := strings.TrimSpace(strings.TrimPrefix(first, "cd"))
		if target == "" {
			target = homeDir()
		}
		if strings.HasPrefix(target, "~") {
			target = filepath.Join(homeDir(), strings.TrimPrefix(target, "~"))
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

// homeDir 跨平台获取当前用户主目录，避免 Windows 无 HOME 环境变量时返回空。
func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return h
	}
	return os.Getenv("HOME")
}

// runShellInDir 在指定目录执行 shell 命令。
// 安全边界：命令执行带超时(默认60s)，输出上限 1MiB，
// 防止挂起命令永久占用 handler、无界输出耗尽内存。
const (
	shellCommandTimeout = 60 * time.Second
	shellOutputLimit    = 1 << 20 // 1MiB
)

// limitedBuffer 只保留前 limit 字节, 后续写入丢弃并计数 (防止巨量输出耗尽内存)
type limitedBuffer struct {
	buf      []byte
	limit    int
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	room := b.limit - len(b.buf)
	if room > 0 {
		if len(p) <= room {
			b.buf = append(b.buf, p...)
		} else {
			b.buf = append(b.buf, p[:room]...)
			b.exceeded = true
		}
	} else if len(p) > 0 {
		b.exceeded = true
	}
	return len(p), nil // 返回全量计数, 保持 io.Writer 语义
}

func (b *limitedBuffer) String() string { return string(b.buf) }

func runShellInDir(cmdStr, dir string) (string, error) {
	// 根据shell类型选择参数形式
	shellLower := strings.ToLower(shellPath)

	// 超时控制：到时强杀命令，避免 handler 永久阻塞
	ctx, cancel := context.WithTimeout(context.Background(), shellCommandTimeout)
	defer cancel()
	var cmd *exec.Cmd
	if strings.Contains(shellLower, "cmd") || strings.Contains(shellLower, "cmd.exe") {
		cmd = exec.CommandContext(ctx, shellPath, "/c", cmdStr)
	} else if strings.Contains(shellLower, "powershell") || strings.Contains(shellLower, "pwsh") {
		cmd = exec.CommandContext(ctx, shellPath, "-Command", cmdStr)
	} else {
		cmd = exec.CommandContext(ctx, shellPath, "-c", cmdStr)
	}
	cmd.Dir = dir

	// 受限输出: 最多保留 1MiB, 其余丢弃 (CombinedOutput 会先收集全部输出再截断,
	// 遇到 yes/cat /dev/zero 等无限输出时内存先被撑爆, 截断形同虚设)
	buf := &limitedBuffer{limit: shellOutputLimit}
	cmd.Stdout = buf
	cmd.Stderr = buf

	err := cmd.Run()
	var out string
	if ctx.Err() == context.DeadlineExceeded {
		out = buf.String() + "\n[SimpleWebShell] 命令超时(60s)已终止"
	} else {
		out = buf.String()
		if buf.exceeded {
			out += "\n[SimpleWebShell] 输出超过 1MiB 已截断"
		}
	}
	return out, err
}

// 处理文件上传
// 上传使用 multipart/form-data，支持 path 字段（目标全路径或目录）。
// 实现与字段顺序无关：先流式缓冲所有文件到临时文件，解析完所有表单字段（含 path）后再落盘，
// 避免前端将 path 字段排在 file 之后时目标路径失效的暗病；同时支持一次上传多个文件。
func handleFileSend(c *gin.Context) {
	// 优先从 query 获取 key
	key := c.Query("key")
	if !secureCompare(key, password) {
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
	var destPath string // 来自表单的目标路径（可以是目录或完整路径）
	type pendingFile struct {
		tmpPath  string
		filename string
	}
	var pending []pendingFile
	var savedFiles []string
	// 单次上传总大小上限: 默认 2GiB, 可用环境变量 MAX_UPLOAD_SIZE(字节) 覆盖,
	// 防止异常/恶意客户端无限上传耗尽磁盘 (边缘设备磁盘通常有限)
	maxUpload := int64(2 << 30)
	if v := os.Getenv("MAX_UPLOAD_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxUpload = n
		}
	}
	var totalWritten int64
	// cleanup 清理临时文件与已写入的目标文件
	cleanup := func() {
		for _, pf := range pending {
			_ = os.Remove(pf.tmpPath)
		}
		for _, f := range savedFiles {
			_ = os.Remove(f)
		}
	}
	defer func() {
		// 在处理完毕若发生 panic/错误，清理临时文件
		if r := recover(); r != nil {
			cleanup()
			c.String(http.StatusInternalServerError, fmt.Sprintf("上传处理发生异常: %v", r))
		}
	}()

	// 处理 multipart 各部分：表单字段即时记录，文件字段缓冲到临时文件
	for {
		part, perr := mr.NextPart()
		if perr == io.EOF {
			break
		}
		if perr != nil {
			cleanup()
			c.String(http.StatusInternalServerError, fmt.Sprintf("读取 multipart 部分失败: %v", perr))
			return
		}

		if part.FileName() == "" {
			// 普通表单字段, 限制大小: 多读 1 字节探测是否超限, 超限明确拒绝而非静默截断
			const maxFieldBytes = 64 << 10
			name := part.FormName()
			data, _ := io.ReadAll(io.LimitReader(part, maxFieldBytes+1))
			_ = part.Close()
			if len(data) > maxFieldBytes {
				cleanup()
				c.String(http.StatusRequestEntityTooLarge, fmt.Sprintf("表单字段 %s 超过大小上限 %d 字节", name, maxFieldBytes))
				return
			}
			val := strings.TrimSpace(string(data))
			if name == "path" {
				destPath = val
			}
			continue
		}

		// 文件字段：先写入临时文件，避免依赖 path 字段的到达顺序
		filename := filepath.Base(part.FileName())
		tmp, terr := os.CreateTemp("", "sws-upload-*")
		if terr != nil {
			_ = part.Close()
			cleanup()
			c.String(http.StatusInternalServerError, fmt.Sprintf("创建临时文件失败: %v", terr))
			return
		}
		buf := make([]byte, 32*1024)
		for {
			// 检查客户端是否已经取消
			select {
			case <-ctx.Done():
				_ = tmp.Close()
				_ = os.Remove(tmp.Name())
				cleanup()
				c.String(http.StatusRequestTimeout, "上传已取消")
				return
			default:
			}

			n, rerr := part.Read(buf)
			if n > 0 {
				totalWritten += int64(n)
				if totalWritten > maxUpload {
					_ = tmp.Close()
					_ = os.Remove(tmp.Name())
					cleanup()
					c.String(http.StatusRequestEntityTooLarge,
						fmt.Sprintf("上传超过大小上限 %d 字节", maxUpload))
					return
				}
				if _, werr := tmp.Write(buf[:n]); werr != nil {
					_ = tmp.Close()
					_ = os.Remove(tmp.Name())
					cleanup()
					c.String(http.StatusInternalServerError, fmt.Sprintf("写入临时文件失败: %v", werr))
					return
				}
			}
			if rerr != nil {
				if rerr == io.EOF {
					break
				}
				_ = tmp.Close()
				_ = os.Remove(tmp.Name())
				cleanup()
				c.String(http.StatusInternalServerError, fmt.Sprintf("读取上传数据失败: %v", rerr))
				return
			}
		}
		_ = tmp.Close()
		pending = append(pending, pendingFile{tmpPath: tmp.Name(), filename: filename})
	}

	// 所有字段已解析完毕，此时 destPath 已确定，将临时文件移动到最终位置
	for _, pf := range pending {
		// 计算输出路径：如果表单指定了 destPath
		var outPath string
		if destPath == "" {
			outPath = pf.filename
		} else {
			// 如果 destPath 指向一个已存在的目录或以路径分隔符结尾，则把文件名追加到目录
			if fi, err := os.Stat(destPath); err == nil && fi.IsDir() {
				outPath = filepath.Join(destPath, pf.filename)
			} else if strings.HasSuffix(destPath, string(os.PathSeparator)) {
				outPath = filepath.Join(destPath, pf.filename)
			} else {
				// 否则把 destPath 当作完整文件路径（覆盖或创建）
				outPath = destPath
			}
		}

		// 确保目标目录存在
		if dir := filepath.Dir(outPath); dir != "" {
			_ = os.MkdirAll(dir, 0755)
		}

		// 仅当目标是"本次新建"(覆盖前不存在)时才登记进 savedFiles:
		// cleanup 只应删除本次上传产生的文件; 若覆盖了已有文件而后续某文件失败,
		// 删除它会导致"旧内容已被覆盖, 新内容又被删"的双重数据丢失
		_, statErr := os.Stat(outPath)
		existedBefore := statErr == nil

		if err := moveFile(pf.tmpPath, outPath); err != nil {
			cleanup()
			c.String(http.StatusInternalServerError, fmt.Sprintf("保存文件失败: %v", err))
			return
		}
		if !existedBefore {
			savedFiles = append(savedFiles, outPath)
		}
	}

	// 上传完成
	c.Header("Content-Type", "application/json; charset=utf-8")
	respBody, _ := json.Marshal(map[string]any{
		"status":  "ok",
		"files":   savedFiles,
		"message": "上传完成",
	})
	c.String(http.StatusOK, string(respBody))
}

// moveFile 将 src 移动到 dst。跨文件系统（如临时目录与目标目录不同挂载点）时回退为拷贝+删除。
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// 回退：拷贝后删除源文件，兼容跨设备/Windows 行为
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}
	_ = os.Remove(src)
	return nil
}

// 处理文件下载
// 请求示例: /file_receive?key=xxx&path=/full/path/to/file
func handleFileReceive(c *gin.Context) {
	// 验证密钥
	key := c.Query("key")
	if !secureCompare(key, password) {
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
	// 目录不可下载: Linux 下 open 目录成功但 Read 返回 EISDIR, 会以 200+空 body 结束,
	// 前端无法区分空文件与失败; 显式拒绝
	if fi.IsDir() {
		c.String(http.StatusBadRequest, fmt.Sprintf("路径是目录: %s", filePath))
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
	if !secureCompare(key, password) {
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
