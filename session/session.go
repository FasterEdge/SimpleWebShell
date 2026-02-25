package session

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Info 保存每个 session 的工作目录
// 可以扩展更多字段（如创建时间、上次访问时间）。
type Info struct {
	Dir       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	mu   sync.RWMutex
	data = make(map[string]*Info)
)

// New 创建一个新的 session，初始工作目录为服务器当前工作目录。
func New() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)
	cwd, _ := os.Getwd()
	mu.Lock()
	data[id] = &Info{Dir: cwd, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	mu.Unlock()
	return id
}

// List 返回所有 session 的快照（拷贝）。
func List() map[string]Info {
	mu.RLock()
	defer mu.RUnlock()
	res := make(map[string]Info, len(data))
	for k, v := range data {
		res[k] = *v
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

// GetDir 获取 session 的当前工作目录。
func GetDir(id string) (string, bool) {
	mu.RLock()
	defer mu.RUnlock()
	info, ok := data[id]
	if !ok {
		return "", false
	}
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
	}
	mu.Unlock()
}
