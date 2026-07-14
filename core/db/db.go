// Package db provides the shared GORM database instance used by all model sub-packages.
// Extracted from apps/admin/model to break circular import chains.
package db

import (
	"gbootcms/config"
	"gbootcms/core/acodeplugin"
	"gbootcms/core/mediaplugin"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// OnDataChange 是數據變更回調，由 main.go 在啟動時設定為 middleware.ClearHTMLCache
// 使用回調變數而非直接 import middleware，避免循環依賴
var OnDataChange func()

// DB is the shared GORM database instance.
var DB *gorm.DB

// InitDB initialises the database connection based on the given config.
func InitDB(cfg *config.Config) error {
	var err error
	dsn := cfg.Database.DBName

	// 確保數據庫目錄存在
	if dir := filepath.Dir(dsn); dir != "" && dir != "." {
		os.MkdirAll(dir, 0755)
	}

	// glebarez/sqlite 驅動的 DSN 不支援 _pragma 參數，改用 Exec 設定
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		// 命名策略：保留 PbootCMS 原版表前綴 ay_，單數化表名（user 而非 users）
		// 配合字段名審計確保與 DB 表結構 1:1 對應。
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "ay_",
			SingularTable: true,
		},
	})
	if err != nil {
		return err
	}

	sqlDB, _ := DB.DB()
	// SQLite WAL 模式支援並發讀：多個連接可同時讀取，寫入由 WAL 內部序列化
	// MaxOpenConns=1 會導致所有查詢排隊，450 並發時平均回應 7.8 秒
	// 提高到 20 允許並發讀取，寫入仍由 SQLite 單線程保證
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	// 連接最長生命週期 5 分鐘，避免長時間持有導致連接過期。
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// 啟用 WAL 模式 + 效能 PRAGMA（glebarez 驅動不支援 DSN 參數，用 Exec 設定）
	// WAL：允許並發讀 + 單寫，讀寫互不阻塞（效能關鍵）
	// busy_timeout=5000：鎖衝突時等待 5 秒而非立即報 SQLITE_BUSY
	// synchronous=NORMAL：WAL 模式下 NORMAL 已足夠安全，FULL 會大幅降速
	// cache_size=-64000：使用 64MB 記憶體緩存（負值=KB），減少磁碟 I/O
	DB.Exec("PRAGMA journal_mode=WAL")
	DB.Exec("PRAGMA busy_timeout=5000")
	DB.Exec("PRAGMA synchronous=NORMAL")
	DB.Exec("PRAGMA cache_size=-64000")

	// 驗證 WAL 模式是否生效
	var journalMode string
	DB.Raw("PRAGMA journal_mode").Scan(&journalMode)
	slog.Info("SQLite 初始化完成", "journal_mode", journalMode, "max_conns", 20)

	// 註冊媒體緩存失效插件：
	// 對所有「會引用媒體文件」的表（slide、content、content_sort、link、company、site）
	// 的 Create / Update / Delete 操作，自動標記媒體庫緩存為臟。
	// Controller 中無需再手動調用 MarkMediaCacheDirty()。
	if err := DB.Use(&mediaplugin.MediaDirtyPlugin{}); err != nil {
		return err
	}

	// 註冊區域隔離插件：
	// 對所有含 acode 欄位的表（content、content_sort、link、slide、tags、message、
	// company、site、comment 等），自動注入 WHERE acode = ? 和填充 acode 值。
	// 通過 context 傳遞當前區域，控制器只需 .WithContext(c.Request.Context())。
	// 跨區查詢用 acodeplugin.SkipAcode(ctx)。
	if err := DB.Use(&acodeplugin.AcodePlugin{}); err != nil {
		return err
	}

	// 註冊 HTML 緩存自動清除回調：
	// 任何表的 Create/Update/Delete 操作都會清除前台 HTML 記憶體緩存
	// 確保後台發布/編輯/刪除內容後，前台立即看到最新內容
	clearHTMLCache := func(db *gorm.DB) {
		if db.Error == nil && db.RowsAffected > 0 && OnDataChange != nil {
			OnDataChange()
		}
	}
	DB.Callback().Create().After("gorm:create").Register("clear_html_cache", clearHTMLCache)
	DB.Callback().Update().After("gorm:update").Register("clear_html_cache", clearHTMLCache)
	DB.Callback().Delete().After("gorm:delete").Register("clear_html_cache", clearHTMLCache)

	return nil
}

// CloseDB closes the database connection.
func CloseDB() {
	if DB != nil {
		sqlDB, _ := DB.DB()
		sqlDB.Close()
	}
}
