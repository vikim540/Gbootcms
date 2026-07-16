# Gbootcms 深度優化計劃

> 靈感來源：PHP 8.5（`#[NoDiscard]`、管道運算子、URI 擴展）+ Swoole 6（多線程模式、io_uring、連線池、原子操作）
> 目標：從「查漏補缺」升級為「專業級系統工程」

---

## 審計概況

| 優先級 | 數量 | 核心問題 |
|--------|------|----------|
| Critical | 3 | 無優雅關閉、開放重定向、SpiderLog goroutine 洩漏 |
| High | 8 | Session 無持久化、WithContext 大面積缺失、.Error 未檢查、XSS 注入點 |
| Medium | 7 | goroutine 超時洩漏、動態 SQL 拼接、壓縮錯誤忽略 |
| Low | 7 | 死代碼、日誌不一致、Config 鎖競爭 |

---

## Phase 1: 架構穩定性（C1 + C3 + H1）

### 1.1 優雅關閉（C1）

**問題**：`r.Run(addr)` 直接阻塞，收到 SIGTERM/SIGINT 時：
- 進行中的請求被強制中斷
- `defer model.CloseDB()` 不執行，SQLite WAL 未 checkpoint
- 快取預熱 goroutine 無法清理

**方案**：
- 改用 `http.Server` + `signal.Notify` + `srv.Shutdown(ctx)`
- 30 秒超時 context 確保連接關閉
- DB 連接正確關閉，WAL checkpoint 完成

**對標**：Swoole 6 的 `Server->shutdown()` 優雅關閉機制

### 1.2 SpiderLog Worker Pool（C3）

**問題**：每次蜘蛛訪問建立一個 goroutine 直接寫 DB：
- goroutine 數量無上限
- 所有 goroutine 競爭 SQLite 寫鎖
- 無批量寫入優化

**方案**：
- 帶緩衝 channel（容量 1000）+ 3 個固定 worker goroutine
- Worker 批量積累 50 條或 5 秒超時後 batch insert
- 使用 `context.WithCancel` 在關閉時等待 worker 排空

**對標**：Swoole 6 的連線池 + 批量操作理念

### 1.3 SQLite Session 持久化（H1）

**問題**：
- 重啟後所有 session 丟失（所有用戶被踢出）
- 記憶體 map 無大小上限
- 預設 session key 硬編碼不安全

**方案**：
- 新增 `ay_session` 表（sid PK + data JSON + created_at + last_activity）
- 寫入策略：SetSession 時同步寫 DB（SQLite WAL 下的單行寫入 < 1ms）
- 讀取策略：先讀記憶體快取（LRU），miss 時回 DB
- 啟動時載入活躍 session 到記憶體
- 清理策略：保留現有定時清理 + DB 過期行刪除
- Cookie Secure 標誌根據請求 scheme 動態設定

**對標**：Swoole 的常駐記憶體 + DB 備份模式

---

## Phase 2: 安全加固（C2 + H6 + H7 + H8）

### 2.1 開放重定向修復（C2）
- 重定向前驗證 target URL：只允許相對路徑或同域名
- 拒絕 `//`、`http://`、`https://` 開頭的外部 URL

### 2.2 XSS 注入封堵（H6 + H7）
- `site_status.go`：`closeSiteNote` 經 HTML 轉義後再插入
- `template_helpers.go`：所有動態插入 HTML 的變數經 `html.EscapeString`

### 2.3 Cookie Secure 標誌（H8）
- 根據 `c.Request.TLS != nil` 動態設定 Secure 標誌
- HTTPS 環境下 Secure=true，HTTP 下 false

---

## Phase 3: 代碼品質（H2 + H3-H5 + L1-L7）

### 3.1 WithContext 審計（H2）— 違反硬約束 #53
~50+ 處 acode 表查詢缺少 `.WithContext(c.Request.Context())`，全面補齊。

### 3.2 .Error 全檢（H3-H5）— 對標 PHP 8.5 `#[NoDiscard]`
- `front.go` visits/likes/oppose
- `seed.go` 11 處 Create
- `ModelModel.go`、`LabelModel.go`、`ExtFieldModel.go` 多處 Raw/Exec
- `ContentSortService.go`、`AreaModel.go` Update 結果

### 3.3 代碼清理（L1-L7）
- 刪除死代碼 `encodeCookie`/`decodeCookie`
- `fmt.Printf` → `slog.Error`
- `identifySpider` 改用 map 結構
- 縮排/重複路由整理

---

## Phase 4: 效能瓶頸（L7 + M1 + M2-M7）

### 4.1 Config 無鎖讀取（L7）
- `atomic.Pointer[map[string]string]` 替代 `sync.RWMutex`
- 寫入時構建新 map 後 atomic swap，讀取完全無鎖

### 4.2 MediaController goroutine 取消（M1）
- `context.WithCancel` 讓超時後 goroutine 可退出

### 4.3 壓縮錯誤處理（M2）
- 檢查所有 `writer.Write` / `writer.Close` 返回值

### 4.4 動態 SQL 白名單化（M3-M5, M7）
- `fmt.Sprintf` SQL 改為參數化查詢或嚴格白名單

---

## 驗證策略

每個 Phase 完成後：
1. `go build -o bin/gbootcms.exe .` 編譯通過
2. 啟動伺服器，測試前台首頁、後台登入、API 搜尋
3. 測試對應 Phase 的修復點
4. 提交並推送到 GitHub
5. 等待用戶確認後進入下一 Phase
