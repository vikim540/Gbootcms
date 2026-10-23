package version

import (
	"fmt"
	"time"
)

var (
	// Version 應用程式版本
	Version = "v1.0.0"
	// GitCommit 編譯時由 -ldflags 注入的 Git Commit 雜湊
	GitCommit = "HEAD"
	// BuildTime 編譯時由 -ldflags 注入的構建時間
	BuildTime = "2026-10-23"
	// AppTitle 應用程式標題
	AppTitle = "GbootCMS 高性能企業級內容管理系統"
	// StartTime 進程啟動時間，用於計算 Uptime
	StartTime = time.Now()
)

// Uptime 返回進程運行時間的友好格式
func Uptime() string {
	d := time.Since(StartTime)
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	if hours >= 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%d天 %d小時 %d分", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d小時 %d分 %d秒", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d分 %d秒", minutes, seconds)
}
