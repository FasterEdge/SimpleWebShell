// ─────────────────────────────────────────────────────────────
// FasterEdge 开源项目
// Github: https://github.com/FasterEdge
// Gitee:  https://gitee.com/FasterEdge
// ─────────────────────────────────────────────────────────────
//go:build windows

package session

import "golang.org/x/sys/windows"

// detectDiskMB 尝试获取路径所在文件系统的容量（MB）。
// Windows 实现：windows.GetDiskFreeSpaceEx。
func detectDiskMB(path string) int64 {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}
	var total uint64
	if err := windows.GetDiskFreeSpaceEx(p, nil, &total, nil); err == nil {
		return int64(total / 1024 / 1024)
	}
	return 0
}
