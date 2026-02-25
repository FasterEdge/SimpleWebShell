package session

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
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

// New 创建一个新的 session，初始工作目录为服务器当前工作目录。
func New() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)
	cwd, _ := os.Getwd()
	now := time.Now()
	mu.Lock()
	data[id] = &Info{
		Dir:         cwd,
		CreatedAt:   now,
		UpdatedAt:   now,
		LastAccess:  now,
		AccessCount: 0,
		Env:         map[string]string{},
		History:     []string{},
		Metadata:    map[string]string{},
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
