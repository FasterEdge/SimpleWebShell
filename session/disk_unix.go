// ─────────────────────────────────────────────────────────────
// FasterEdge 开源项目
// Github: https://github.com/FasterEdge
// Gitee:  https://gitee.com/FasterEdge
// ─────────────────────────────────────────────────────────────
//go:build !windows

package session

import "syscall"

// detectDiskMB 尝试获取路径所在文件系统的容量（MB）。
// Unix 实现：syscall.Statfs。
func detectDiskMB(path string) int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err == nil {
		total := uint64(st.Blocks) * uint64(st.Bsize)
		return int64(total / 1024 / 1024)
	}
	return 0
}