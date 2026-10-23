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
	"sync/atomic"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		os.MkdirAll(dir, 0777)
	}

	// 智能糾偏與主庫自愈：
	// 如果配置的 dsn 不存在或小於 300KB，但 data/ 目錄下存在完整的主數據庫 0f13e78cfb83ebff93064bc3b9b7157d.db，自動切換至主庫
	dbDir := filepath.Dir(dsn)
	primaryDB := filepath.Join(dbDir, "0f13e78cfb83ebff93064bc3b9b7157d.db")
	curFi, curErr := os.Stat(dsn)
	if (os.IsNotExist(curErr) || (curErr == nil && curFi.Size() < 300000)) && dsn != primaryDB {
		if pfi, perr := os.Stat(primaryDB); perr == nil && pfi.Size() > 500000 {
			slog.Info("檢測到目錄下存在完整的主數據庫，自動糾偏並對齊切換", "from", dsn, "to", primaryDB)
			dsn = primaryDB
			cfg.Database.DBName = primaryDB
		}
	}

	// 智能初始化：如果業務資料庫不存在或為 0 字節空文件，自動從出廠完整範本（data.default.db）拷貝初始化
	if fi, err := os.Stat(dsn); os.IsNotExist(err) || (err == nil && fi.Size() == 0) {
		defaultDB := filepath.Join(filepath.Dir(dsn), "data.default.db")
		if _, defErr := os.Stat(defaultDB); defErr == nil {
			slog.Info("未檢測到生產資料庫，自動從出廠完整範本初始化", "src", defaultDB, "dst", dsn)
			data, readErr := os.ReadFile(defaultDB)
			if readErr == nil {
				_ = os.WriteFile(dsn, data, 0666)
			}
		}
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
	// visits 等統計欄位更新不清除快取（否則每次瀏覽都清空快取）
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
			"session":        true, // 會話持久化（不影響前台頁面）
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
		// visits 等統計欄位更新不清除快取
		// 雙重檢查：Statement.Selects（Select().Update() 鏈式）+ Statement.Dest（UpdateColumn/Update/Updates）
		// 注意：GORM 的 UpdateColumn("visits", ...) 只設 Dest 不設 Selects，
		// 因此必須同時檢查 Dest 才能攔截所有路徑
		skipCols := map[string]bool{
			"visits": true,
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
// 支援四種場景：
//  1. Create/Save(&Model{ID: 37}) → 從 model 反射取 ID
//  2. Delete(&Model{}, 37) → GORM 將 inline condition 設為主鍵
//  3. Model(&Model{}).Where("id = ?", 37).Update(...) → 從 WHERE 條件提取
//  4. Model(&Model{}).Where("id IN ?", []int{37}).Update(...) → 從 IN 條件提取
func getPrimaryKeyID(stmt *gorm.Statement) int {
	if stmt == nil || stmt.Model == nil || stmt.Schema == nil {
		return 0
	}
	pkField := stmt.Schema.PrioritizedPrimaryField
	if pkField != nil {
		rv := reflect.ValueOf(stmt.Model)
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Struct {
			fieldVal := rv.FieldByName(pkField.Name)
			if fieldVal.IsValid() {
				switch fieldVal.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if id := int(fieldVal.Int()); id > 0 {
						return id
					}
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					if id := int(fieldVal.Uint()); id > 0 {
						return id
					}
				}
			}
		}
	}
	// 方法 2：從 WHERE 條件提取 ID（Update/Delete 場景）
	return getIDFromWhereClauses(stmt)
}

// getIDFromWhereClauses 從 GORM Statement 的 WHERE 子句中提取 id 值
// 用於 .Model(&Model{}).Where("id = ?", 37).Updates(...) 場景
func getIDFromWhereClauses(stmt *gorm.Statement) int {
	if stmt == nil {
		return 0
	}
	// 方法 A：從 clause.Where 表達式提取
	if stmt.Clauses != nil {
		if whereClause, ok := stmt.Clauses["WHERE"]; ok && whereClause.Expression != nil {
			if where, ok := whereClause.Expression.(clause.Where); ok {
				for _, expr := range where.Exprs {
					if id := extractIDFromExpr(expr); id > 0 {
						return id
					}
					if andConds, ok := expr.(clause.AndConditions); ok {
						for _, e := range andConds.Exprs {
							if id := extractIDFromExpr(e); id > 0 {
								return id
							}
						}
					}
				}
			}
		}
	}
	// 方法 B：從已執行的 SQL + Vars 中提取（fallback）
	return getIDFromSQL(stmt)
}

// getIDFromSQL 從已執行的 SQL 字串和綁定參數中提取 id 值
// 適用於 UPDATE ... WHERE `id` = ? AND `acode` = ? 場景
func getIDFromSQL(stmt *gorm.Statement) int {
	if stmt == nil {
		return 0
	}
	sqlStr := stmt.SQL.String()
	if sqlStr == "" {
		return 0
	}
	lowerSQL := strings.ToLower(sqlStr)
	// 尋找 "id = ?" 或 "`id` = ?" 或 "id=?" 的位置
	// 支援 GORM 生成的 SQL 格式：`id` = ? 或 id = ?
	patterns := []string{"`id` = ?", "`id`=?", "id = ?", "id=?"}
	for _, pattern := range patterns {
		idx := strings.Index(lowerSQL, pattern)
		if idx < 0 {
			continue
		}
		// 計算此 ? 是 SQL 中第幾個佔位符
		questionMarkEnd := idx + len(pattern) - 1 // 指向 ?
		paramIndex := strings.Count(lowerSQL[:questionMarkEnd], "?")
		if paramIndex < len(stmt.Vars) {
			if id := toIntID(stmt.Vars[paramIndex]); id > 0 {
				return id
			}
		}
	}
	return 0
}

// extractIDFromExpr 從單個表達式中提取 id 值
func extractIDFromExpr(expr clause.Expression) int {
	// 處理 NamedExpr（原始 SQL 如 "id = ?"）
	if named, ok := expr.(clause.NamedExpr); ok {
		sqlLower := strings.ToLower(named.SQL)
		if strings.Contains(sqlLower, "id") && (strings.Contains(sqlLower, "=") || strings.Contains(sqlLower, "in")) {
			for _, v := range named.Vars {
				if id := toIntID(v); id > 0 {
					return id
				}
			}
		}
	}
	// 處理 Eq{"id": 37}
	if eq, ok := expr.(clause.Eq); ok {
		if col, ok := eq.Column.(string); ok && col == "id" {
			return toIntID(eq.Value)
		}
		if col, ok := eq.Column.(clause.Column); ok && col.Name == "id" {
			return toIntID(eq.Value)
		}
	}
	// 處理 IN{"id": [37, 38]}
	if in, ok := expr.(clause.IN); ok {
		if col, ok := in.Column.(string); ok && col == "id" {
			if len(in.Values) > 0 {
				return toIntID(in.Values[0])
			}
		}
		if col, ok := in.Column.(clause.Column); ok && col.Name == "id" {
			if len(in.Values) > 0 {
				return toIntID(in.Values[0])
			}
		}
	}
	return 0
}

// toIntID 尷各種類型轉為正整數 ID
func toIntID(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int8:
		return int(val)
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return int(val)
	case uint:
		return int(val)
	case uint8:
		return int(val)
	case uint16:
		return int(val)
	case uint32:
		return int(val)
	case uint64:
		return int(val)
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


// ──────────────────────────────────────────────
// Config cache — 無鎖讀取（atomic.Pointer，對標 Swoole 6 原子操作）
// 每個頁面渲染呼叫 GetConfigValue 8+ 次，RWMutex 在高並發下產生鎖競爭
// atomic.Pointer.Load 是單條 CPU 指令，完全無鎖
// ──────────────────────────────────────────────

var configCachePtr atomic.Pointer[map[string]string]

type sysConfigItem struct {
	Name  string `gorm:"column:name"`
	Value string `gorm:"column:value"`
}

// preloadConfigCache 一次性載入所有配置到記憶體
func preloadConfigCache() {
	if DB == nil {
		m := make(map[string]string)
		configCachePtr.Store(&m)
		return
	}
	var configs []sysConfigItem
	DB.Table("ay_config").Find(&configs)
	m := make(map[string]string, len(configs))
	for _, c := range configs {
		if c.Value != "" {
			m[c.Name] = c.Value
		}
	}
	configCachePtr.Store(&m)
}

// ClearConfigCache 清除配置快取（由 GORM 回調觸發）
func ClearConfigCache() {
	configCachePtr.Store(nil)
}

// GetConfigValue reads a config value by name, returning defaultVal if not found or empty.
// 使用 atomic.Pointer 無鎖讀取，避免高並發下的 RWMutex 鎖競爭
func GetConfigValue(name, defaultVal string) string {
	p := configCachePtr.Load()
	if p == nil {
		preloadConfigCache()
		p = configCachePtr.Load()
	}
	if p != nil {
		if v, ok := (*p)[name]; ok {
			return v
		}
	}
	return defaultVal
}

// GetDBName returns the current database file name from config.
func GetDBName() string {
	cfg := config.Get()
	if cfg != nil {
		return cfg.Database.DBName
	}
	return ""
}

// TableName adds the ay_ prefix to a table name.
func TableName(name string) string {
	return "ay_" + name
}
