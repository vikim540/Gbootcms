# 系統架構 (Architecture)

Gbootcms 嚴格依循**奧卡姆剃刀原則**構建，杜絕層層封裝與偽分層。

---

## 1. 架構全景圖

```
                    +---------------------------+
                    |    HTTP 請求 (Client)     |
                    +-------------+-------------+
                                  |
                                  v
                    +---------------------------+
                    |    Gin 路由器 (route.go)  |
                    +-------------+-------------+
                                  |
         +------------------------+------------------------+
         |                                                 |
         v                                                 v
+------------------+                              +------------------+
| 前台 TagParser   |                              | 後台 Pongo2      |
| ({gboot:xxx})    |                              | (Layui+UnoCSS)   |
| 單模板 + i18n 字典|                              | 遙測監控 / 媒體庫 |
+--------+---------+                              +--------+---------+
         |                                                 |
         +------------------------+------------------------+
                                  |
                                  v
                    +---------------------------+
                    |  唯一 DB 引擎 (core/db)   |
                    |  SQLite WAL 讀寫池 (20:1) |
                    +---------------------------+
```

---

## 2. 唯一資料庫入口 (`core/db/db.go`)

全系統只有一個資料庫實例 `core/db.DB`，以 WAL 模式運行，徹底消除 SQL 排隊瓶頸。

```go
// core/db/db.go - WAL 連線池配置
func InitDB(cfg *config.Config) error {
    db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logMode),
    })
    if err != nil {
        return err
    }
    
    sqlDB, err := db.DB()
    sqlDB.SetMaxOpenConns(20)              // 允許 20 個並發讀操作
    sqlDB.SetMaxIdleConns(5)
    sqlDB.SetConnMaxLifetime(time.Hour)

    // PRAGMA 性能調優
    db.Exec("PRAGMA journal_mode = WAL;")        // WAL 模式：並發讀與單寫互不阻塞
    db.Exec("PRAGMA synchronous = NORMAL;")     // 平衡性能與安全性
    db.Exec("PRAGMA busy_timeout = 5000;")       // 鎖定等待 5 秒
    db.Exec("PRAGMA cache_size = -64000;")      // 64MB 記憶體頁快取

    DB = db
    return nil
}
```

---

## 3. 唯一資料庫遷移入口 (`apps/admin/model/migrate.go`)

取代過去碎片化的 migrate 檔案，收斂為單一全域函數：

```go
// apps/admin/model/migrate.go
func MigrateAll() error {
    // 依序執行系統、內容、會員三模組的 GORM AutoMigrate
    if err := system.AutoMigrate(DB); err != nil {
        return err
    }
    if err := content.AutoMigrate(DB); err != nil {
        return err
    }
    if err := member.AutoMigrate(DB); err != nil {
        return err
    }
    return nil
}
```

---

## 4. 統一遙測引擎 (`core/version/telemetry.go`)

無論在後台首頁快捷卡片，還是詳細伺服器狀態頁，所有指標均由同一函數產出，保證數值 100% 同步：

```go
// 調用範例
telemetry := version.GetTelemetry()
fmt.Printf("進程運行時間: %s\n", telemetry.Uptime)
fmt.Printf("已分配記憶體: %.2f MB\n", telemetry.AllocMB)
fmt.Printf("活躍 Goroutines: %d\n", telemetry.Goroutines)
```
