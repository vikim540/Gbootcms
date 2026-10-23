package version

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gbootcms/apps/admin/model"

	"github.com/gin-gonic/gin"
)

// GetServerTelemetry 統一收集系統運行時遙測數據（單一數據源，符合奧卡姆剃刀原則）
// 供 /admin/index/home 與 /admin/site/server 同時調用，杜絕重複計算與數據歧異。
func GetServerTelemetry(c *gin.Context) gin.H {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024)
	totalAllocMB := fmt.Sprintf("%.2f MB", float64(m.TotalAlloc)/1024/1024)
	sysMB := fmt.Sprintf("%.2f MB", float64(m.Sys)/1024/1024)
	numGC := fmt.Sprintf("%d 次", m.NumGC)
	goroutines := runtime.NumGoroutine()
	uptime := Uptime()

	// 資料庫檔案與 WAL 狀態
	dbPath := model.GetDBName()
	dbSizeStr := "未知"
	walSizeStr := "無 / 未開啟"
	if fi, err := os.Stat(dbPath); err == nil {
		dbSizeStr = fmt.Sprintf("%.2f MB", float64(fi.Size())/1024/1024)
	}
	walPath := dbPath + "-wal"
	if fi, err := os.Stat(walPath); err == nil {
		walSizeStr = fmt.Sprintf("%.2f KB (活躍)", float64(fi.Size())/1024)
	}

	// 上傳資源體積（static/upload/）
	var uploadSize int64
	_ = filepath.Walk("static/upload", func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			uploadSize += info.Size()
		}
		return nil
	})
	uploadSizeStr := fmt.Sprintf("%.2f MB", float64(uploadSize)/1024/1024)

	dbType := model.GetConfigValue("database.type", "sqlite")
	serverSoftware := "Go " + runtime.Version() + " / Gin"
	webSoftware := "Go " + runtime.Version() + " / Gin v" + gin.Version

	host := ""
	port := ""
	if c != nil && c.Request != nil {
		host = c.Request.Host
		port = c.Request.URL.Port()
	}

	return gin.H{
		// 運行時性能
		"Uptime":            uptime,
		"uptime":            uptime,
		"Goroutines":        goroutines,
		"goroutines":        goroutines,
		"AllocMB":           allocMB,
		"alloc_mb":          allocMB,
		"TotalAllocMB":      totalAllocMB,
		"total_alloc_mb":    totalAllocMB,
		"SysMB":             sysMB,
		"sys_mb":            sysMB,
		"NumGC":             numGC,
		"num_gc":            numGC,
		"DBSize":            dbSizeStr,
		"db_size":           dbSizeStr,
		"WalSize":           walSizeStr,
		"wal_size":          walSizeStr,
		"UploadSize":        uploadSizeStr,
		"upload_size":       uploadSizeStr,

		// 基礎環境
		"PhpOs":             runtime.GOOS,
		"Goos":              runtime.GOOS,
		"goos":              runtime.GOOS,
		"Goarch":            runtime.GOARCH,
		"goarch":            runtime.GOARCH,
		"PhpVersion":        "Go " + runtime.Version(),
		"GoVersion":         runtime.Version(),
		"go_version":        runtime.Version(),
		"ServerSoftware":    serverSoftware,
		"WebSoftware":       webSoftware,
		"web_software":      webSoftware,
		"DbDriver":          strings.ToUpper(dbType),
		"DBDriver":          "SQLite 3 (GORM + glebarez/sqlite WAL)",
		"db_driver":         "SQLite 3 (GORM + glebarez/sqlite WAL)",
		"UploadMaxFilesize": model.GetConfigValue("upload_max_size", "50") + "M",
		"PostMaxSize":       model.GetConfigValue("upload_max_size", "50") + "M",
		"HttpHost":          host,
		"ServerName":        host,
		"ServerPort":        port,
		"ServerAddr":        host,
	}
}
