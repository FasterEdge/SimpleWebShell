// ─────────────────────────────────────────────────────────────
// FasterEdge 开源项目
// Github: https://github.com/FasterEdge
// Gitee:  https://gitee.com/FasterEdge
// ─────────────────────────────────────────────────────────────
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ResourceLimits 表示可选的资源限制（示例）。
type ResourceLimits struct {
	CPUCores float64
	MemoryMB int64
	DiskMB   int64
}

// Permissions 表示简单的权限信息（示例）。
type Permissions struct {
	Owner  string
	Groups []string
	Read   bool
	Write  bool
	Exec   bool
}

// Info 保存每个 session 的更多状态
// 可根据需要继续扩展。注意：包含 slice/map 字段，因此要在返回给调用方时做深拷贝。
type Info struct {
	Dir         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastAccess  time.Time
	AccessCount int64

	Owner       string
	Tags        []string
	Env         map[string]string
	Shell       string
	History     []string
	GitBranch   string
	Mounts      []string
	Metadata    map[string]string
	ExpiresAt   time.Time
	ReadOnly    bool
	Limits      ResourceLimits
	Permissions Permissions
}

var (
	mu   sync.RWMutex
	data = make(map[string]*Info)
)

const defaultHistoryLimit = 200

// 会话清理参数：防止长期运行下会话无限累积占用内存。
const (
	// sessionIdleTTL 会话超过该空闲时长（无任何访问）即被后台清理
	sessionIdleTTL = 24 * time.Hour
	// sessionMaxCount 会话数量上限，超出后清理最久未访问的会话
	sessionMaxCount = 1000
	// cleanupInterval 后台清理协程的扫描间隔
	cleanupInterval = 30 * time.Minute
)

// StartCleaner 启动后台会话清理协程（幂等，可多次调用）。
// 仅清理空闲超时或超过数量上限的会话，不影响活跃会话。
func StartCleaner() {
	cleanerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(cleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				prune()
			}
		}()
	})
}

var cleanerOnce sync.Once

// prune 清理空闲超时或超出上限的会话。
func prune() {
	now := time.Now()
	mu.Lock()
	defer mu.Unlock()

	for id, info := range data {
		if now.Sub(info.LastAccess) > sessionIdleTTL {
			delete(data, id)
		}
	}

	// 仍超过上限则按最近访问时间清理最久未用的会话
	if len(data) > sessionMaxCount {
		type kv struct {
			id   string
			last time.Time
		}
		oldest := make([]kv, 0, len(data))
		for id, info := range data {
			oldest = append(oldest, kv{id: id, last: info.LastAccess})
		}
		sort.Slice(oldest, func(i, j int) bool { return oldest[i].last.Before(oldest[j].last) })
		for i := 0; i < len(oldest)-sessionMaxCount; i++ {
			delete(data, oldest[i].id)
		}
	}
}

// New 创建一个新的 session，初始工作目录为服务器当前工作目录。
func New() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)
	cwd, _ := os.Getwd()
	now := time.Now()

	// 确保后台清理协程在首次创建会话时启动（幂等）
	StartCleaner()

	owner, groups := currentUserInfo()
	shell := os.Getenv("SHELL")
	env := envMap()
	gitBranch := currentGitBranch(cwd)
	limits := detectLimits(cwd)
	meta := defaultMetadata(cwd, shell, gitBranch, owner)

	mu.Lock()
	data[id] = &Info{
		Dir:         cwd,
		CreatedAt:   now,
		UpdatedAt:   now,
		LastAccess:  now,
		AccessCount: 0,
		Owner:       owner,
		Tags:        []string{},
		Env:         env,
		Shell:       shell,
		History:     []string{},
		GitBranch:   gitBranch,
		Mounts:      []string{},
		Metadata:    meta,
		ReadOnly:    false,
		Limits:      limits,
		Permissions: Permissions{Owner: owner, Groups: groups, Read: true, Write: true, Exec: true},
	}
	mu.Unlock()
	return id
}

// Clone 返回 Info 的深拷贝，以便安全返回给调用方。
func (i *Info) Clone() Info {
	clone := Info{
		Dir:         i.Dir,
		CreatedAt:   i.CreatedAt,
		UpdatedAt:   i.UpdatedAt,
		LastAccess:  i.LastAccess,
		AccessCount: i.AccessCount,

		Owner:       i.Owner,
		Shell:       i.Shell,
		GitBranch:   i.GitBranch,
		ExpiresAt:   i.ExpiresAt,
		ReadOnly:    i.ReadOnly,
		Limits:      i.Limits,
		Permissions: i.Permissions,
	}

	if len(i.Tags) > 0 {
		clone.Tags = make([]string, len(i.Tags))
		copy(clone.Tags, i.Tags)
	}
	if len(i.Mounts) > 0 {
		clone.Mounts = make([]string, len(i.Mounts))
		copy(clone.Mounts, i.Mounts)
	}
	if len(i.History) > 0 {
		clone.History = make([]string, len(i.History))
		copy(clone.History, i.History)
	}
	if len(i.Env) > 0 {
		clone.Env = make(map[string]string, len(i.Env))
		for k, v := range i.Env {
			clone.Env[k] = v
		}
	}
	if len(i.Metadata) > 0 {
		clone.Metadata = make(map[string]string, len(i.Metadata))
		for k, v := range i.Metadata {
			clone.Metadata[k] = v
		}
	}
	if len(i.Permissions.Groups) > 0 {
		clone.Permissions.Groups = make([]string, len(i.Permissions.Groups))
		copy(clone.Permissions.Groups, i.Permissions.Groups)
	}

	return clone
}

// List 返回所有 session 的快照（深拷贝）。
func List() map[string]Info {
	mu.RLock()
	defer mu.RUnlock()
	res := make(map[string]Info, len(data))
	for k, v := range data {
		res[k] = v.Clone()
	}
	return res
}

// Exists 判断 session 是否存在。
// nolint:unused // used by route handlers
func Exists(id string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := data[id]
	return ok
}

// useExists is a helper to avoid unused warning during static checks.
var _ = Exists

// Delete 删除指定 session。
func Delete(id string) bool {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := data[id]; ok {
		delete(data, id)
		return true
	}
	return false
}

// GetDir 获取 session 的当前工作目录，同时更新访问信息。
func GetDir(id string) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	info, ok := data[id]
	if !ok {
		return "", false
	}
	info.LastAccess = time.Now()
	info.AccessCount++
	info.UpdatedAt = time.Now()
	return info.Dir, true
}

// MustGetDir 获取目录（调用者应确保存在）。
func MustGetDir(id string) string {
	d, _ := GetDir(id)
	return d
}

// SetDir 设置 session 的当前工作目录。
func SetDir(id, dir string) {
	mu.Lock()
	if info, ok := data[id]; ok {
		info.Dir = filepath.Clean(dir)
		info.UpdatedAt = time.Now()
		info.LastAccess = time.Now()
		info.AccessCount++
	}
	mu.Unlock()
}

// AppendHistory 将一条命令加入会话历史并保证长度上限。
func AppendHistory(id, cmd string) {
	if cmd == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if info, ok := data[id]; ok {
		info.History = append(info.History, cmd)
		if len(info.History) > defaultHistoryLimit {
			// 保持最近的条目
			start := len(info.History) - defaultHistoryLimit
			info.History = info.History[start:]
		}
		info.LastAccess = time.Now()
		info.AccessCount++
	}
}

// 引用函数以避免未使用警告（静态检查）。
var _ = AppendHistory

// detectLimits 尝试获取资源上限，失败则返回0。
func detectLimits(cwd string) ResourceLimits {
	return ResourceLimits{
		CPUCores: float64(runtime.NumCPU()),
		MemoryMB: detectMemoryMB(),
		DiskMB:   detectDiskMB(cwd),
	}
}

// detectMemoryMB 尝试获取物理内存（MB）。
func detectMemoryMB() int64 {
	// 1) /proc/meminfo (Linux)
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						return kb / 1024
					}
				}
			}
		}
	}
	// 2) sysctl hw.memsize (macOS/FreeBSD)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sysctl", "-n", "hw.memsize")
	if out, err := cmd.Output(); err == nil {
		if bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
			return bytes / 1024 / 1024
		}
	}
	return 0
}

// detectDiskMB 尝试获取路径所在文件系统的容量（MB）。
// 平台相关实现见 disk_unix.go / disk_windows.go。

// defaultMetadata 填充通用元数据，失败忽略。
func defaultMetadata(cwd, shell, gitBranch, owner string) map[string]string {
	meta := make(map[string]string)
	if h, err := os.Hostname(); err == nil {
		meta["hostname"] = h
	}
	meta["os"] = runtime.GOOS
	meta["arch"] = runtime.GOARCH
	if shell != "" {
		meta["shell"] = shell
	}
	if gitBranch != "" {
		meta["git_branch"] = gitBranch
	}
	if owner != "" {
		meta["owner"] = owner
	}
	meta["cwd"] = cwd

	// uname -a (best-effort)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "uname", "-a")
	if out, err := cmd.Output(); err == nil {
		meta["uname"] = strings.TrimSpace(string(out))
	}
	return meta
}

// currentUserInfo 获取当前用户和组信息，失败则返回空字符串/空切片。
func currentUserInfo() (string, []string) {
	owner := ""
	groups := []string{}
	if u, err := user.Current(); err == nil {
		if u.Username != "" {
			owner = u.Username
		} else if u.Name != "" {
			owner = u.Name
		}
		if gids, err := u.GroupIds(); err == nil {
			for _, gid := range gids {
				groups = append(groups, gid)
			}
		}
	}
	// 补充使用 id -Gn，失败无所谓
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "id", "-Gn")
	if out, err := cmd.Output(); err == nil {
		fields := strings.Fields(string(out))
		if len(fields) > 0 {
			groups = append(groups, fields...)
		}
	}
	return owner, groups
}

// envMap 读取环境变量为 map，失败返回空 map。
func envMap() map[string]string {
	res := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			res[parts[0]] = parts[1]
		}
	}
	return res
}

// currentGitBranch 尝试获取当前工作目录的 git 分支，失败返回空字符串。
func currentGitBranch(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
