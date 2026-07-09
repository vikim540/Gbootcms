// Package db provides the shared GORM database instance used by all model sub-packages.
// Extracted from apps/admin/model to break circular import chains.
package db

import (
	"gbootcms/config"
	"gbootcms/core/acodeplugin"
	"gbootcms/core/mediaplugin"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// DB is the shared GORM database instance.
var DB *gorm.DB

// InitDB initialises the database connection based on the given config.
func InitDB(cfg *config.Config) error {
	var err error
	dsn := cfg.Database.DBName
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
	// 紅線約束：SQLite 必須單線程寫入，防止 SQLITE_BUSY 鎖死。
	// 參考 .trae/rules/PbootCMS (PHP) to Go 嚴格重構與修復開發規範.md § 1.3
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	// 連接最長生命週期 5 分鐘，避免長時間持有導致連接過期。
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

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

	return nil
}

// CloseDB closes the database connection.
func CloseDB() {
	if DB != nil {
		sqlDB, _ := DB.DB()
		sqlDB.Close()
	}
}
