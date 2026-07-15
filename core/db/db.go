// Package db provides the shared GORM database instance used by all model sub-packages.
// Extracted from apps/admin/model to break circular import chains.
package db

import (
	"fmt"
	"gbootcms/config"
	"gbootcms/core/acodeplugin"
	"gbootcms/core/mediaplugin"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// OnDataChange 是數據變更回調，由 main.go 在啟動時設定
// 參數：tableName（去 ay_ 前綴的表名），id（變更記錄的主鍵 ID，0 表示無法確定）
// 使用回調變數而非直接 import middleware，避免循環依賴
var OnDataChange func(tableName string, id int)

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
	// mmap_size：記憶體映射 I/O，大型 SQLite 讀取效能提升顯著（256MB）
	DB.Exec("PRAGMA mmap_size=268435456")
	// temp_store=MEMORY：臨時表和排序用記憶體而非磁碟，提升 ORDER BY / GROUP BY 效能
	DB.Exec("PRAGMA temp_store=MEMORY")

	// 建立高頻查詢索引（冪等操作，不修改表結構，符合硬約束 #1）
	// 這些索引覆蓋前台路由匹配、列表查詢、欄目遞迴等全表掃描熱點
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_content_filename ON ay_content(filename)",
		"CREATE INDEX IF NOT EXISTS idx_content_urlname ON ay_content(urlname)",
		"CREATE INDEX IF NOT EXISTS idx_content_scode_status ON ay_content(scode, status, date)",
		"CREATE INDEX IF NOT EXISTS idx_content_acode ON ay_content(acode)",
		"CREATE INDEX IF NOT EXISTS idx_sort_filename ON ay_content_sort(filename)",
		"CREATE INDEX IF NOT EXISTS idx_sort_urlname ON ay_content_sort(urlname)",
		"CREATE INDEX IF NOT EXISTS idx_sort_scode ON ay_content_sort(scode)",
		"CREATE INDEX IF NOT EXISTS idx_sort_pcode ON ay_content_sort(pcode)",
		"CREATE INDEX IF NOT EXISTS idx_comment_contentid ON ay_member_comment(contentid, pid, status)",
	}
	for _, idx := range indexes {
		if err := DB.Exec(idx).Error; err != nil {
			slog.Warn("建立索引失敗", "sql", idx, "error", err)
		}
	}

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
	// 基於 Cache Tag 精準失效：僅失效受影響的頁面，而非清空全部快取
	// visits/likes/oppose 等統計欄位更新不清除快取（否則每次瀏覽都清空快取）
	clearHTMLCache := func(db *gorm.DB) {
		if db.Error != nil || db.RowsAffected == 0 || OnDataChange == nil {
			return
		}
		// 取得表名（去除 ay_ 前綴，與 skipTables 列表一致比對）
		tableName := ""
		if db.Statement != nil && db.Statement.Table != "" {
			tableName = strings.TrimPrefix(db.Statement.Table, "ay_")
		}

		// 這些表的變更不影響前台 HTML 頁面（純統計/日誌/權限/系統用途）
		skipTables := map[string]bool{
			"syslog":         true, // 系統日誌（蜘蛛訪問、管理員操作）
			"member":         true, // 會員資料（不影響前台頁面）
			"member_comment": true, // 評論（由 comment controller 精準失效 content:{id} tag）
			"member_log":     true, // 會員活動日誌
			"member_field":   true, // 會員字段定義
			"member_group":   true, // 會員等級
			"message":        true, // 留言（不影響已渲染頁面）
			"user":           true, // 管理員用戶
			"user_role":      true, // 管理員角色關聯
			"role":           true, // 角色
			"role_area":      true, // 角色區域
			"role_level":     true, // 角色權限
			"menu_action":    true, // 菜單動作
			"database":       true, // 數據庫備份記錄
			"media_mark":     true, // 媒體標記
			"301_redirect":   true, // 301 重定向規則
			"dict_type":      true, // 字典類型
		}
		if skipTables[tableName] {
			return
		}
		// visits/likes/oppose 等統計欄位更新不清除快取
		// 雙重檢查：Statement.Selects（Select().Update() 鏈式）+ Statement.Dest（UpdateColumn/Update/Updates）
		// 注意：GORM 的 UpdateColumn("visits", ...) 只設 Dest 不設 Selects，
		// 因此必須同時檢查 Dest 才能攔截所有路徑
		skipCols := map[string]bool{
			"visits": true, "likes": true, "oppose": true,
			"login_count": true, "last_login_ip": true, "last_login_time": true, "score": true,
		}
		if db.Statement != nil {
			// 路徑 1：Select("visits").Update(...) 設置 Selects
			if len(db.Statement.Selects) > 0 {
				for _, col := range db.Statement.Selects {
					if skipCols[col] {
						return
					}
				}
			}
			// 路徑 2：UpdateColumn("visits", ...) / Update("visits", ...) 設置 Dest 為 map
			if dest, ok := db.Statement.Dest.(map[string]interface{}); ok && len(dest) > 0 {
				allSkipped := true
				for col := range dest {
					if !skipCols[col] {
						allSkipped = false
						break
					}
				}
				if allSkipped {
					return
				}
			}
		}
		// 提取主鍵 ID（用於精準 tag 失效，如 content:37）
		id := getPrimaryKeyID(db.Statement)
		slog.Info("GORM callback → OnDataChange", "table", tableName, "id", id, "selects", db.Statement.Selects, "dest_type", fmt.Sprintf("%T", db.Statement.Dest))
		OnDataChange(tableName, id)
	}
	DB.Callback().Create().After("gorm:create").Register("clear_html_cache", clearHTMLCache)
	DB.Callback().Update().After("gorm:update").Register("clear_html_cache", clearHTMLCache)
	DB.Callback().Delete().After("gorm:delete").Register("clear_html_cache", clearHTMLCache)

	return nil
}

// getPrimaryKeyID 從 GORM Statement 中提取主鍵 ID
// 用於 Cache Tag 精準失效（如 content:37）
// 支援三種場景：
//  1. Create/Save(&Model{ID: 37}) → 從 model 反射取 ID
//  2. Delete(&Model{}, 37) → GORM 將 inline condition 設為主鍵
//  3. Model(&Model{}).Where("id = ?", 37).Update(...) → 無法取得，返回 0
func getPrimaryKeyID(stmt *gorm.Statement) int {
	if stmt == nil || stmt.Model == nil || stmt.Schema == nil {
		return 0
	}
	pkField := stmt.Schema.PrioritizedPrimaryField
	if pkField == nil {
		return 0
	}
	rv := reflect.ValueOf(stmt.Model)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return 0
	}
	fieldVal := rv.FieldByName(pkField.Name)
	if !fieldVal.IsValid() {
		return 0
	}
	switch fieldVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(fieldVal.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(fieldVal.Uint())
	}
	return 0
}

// CloseDB closes the database connection.
func CloseDB() {
	if DB != nil {
		sqlDB, _ := DB.DB()
		sqlDB.Close()
	}
}
