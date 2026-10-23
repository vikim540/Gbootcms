# Gbootcms 迭代與後續開發規劃 (Roadmap & Iterations)

> 本文檔與 [Current iteration · dev](https://github.com/users/vikim540/projects/2) 項目面板保持嚴格同步。
> 專案遵循**奧卡姆剃刀原則**與敏捷漸進演進，以極簡、高效、生產級穩定性為核心考量。

---

## 迭代全景一覽 (Sprint Overview)

| 迭代週期 | 專題方向 | 對應 Issue | 當前狀態 | 關鍵技術指標 |
| :--- | :--- | :--- | :--- | :--- |
| **Iteration 1** | 媒體庫路徑健全度與純 Go WebP 轉碼驗收 | [#1](https://github.com/vikim540/Gbootcms/issues/1) | **進行中 (OPEN)** | 雙向路徑標準化、純 Go WebP 轉碼（體積減 85%）、零 CGO |
| **Iteration 2** | 邊緣 CDN Cache-Tag 快取聯動與分級快取 | [#2](https://github.com/vikim540/Gbootcms/issues/2) | **規劃中 (Backlog)** | Cloudflare Purge API、非阻塞廣播、Redis/記憶體雙驅動 |
| **Iteration 3** | 全站動態 Markdown 快照鏡像與 AI 檢索代理加速 | [#3](https://github.com/vikim540/Gbootcms/issues/3) | **規劃中 (Backlog)** | `/content/{id}.md` 純文字鏡像、`/llms.txt` 自動更新、GSC Indexing API |
| **Iteration 4** | 管理員雙因子認證 (2FA / TOTP) 與安全防暴破 | [#4](https://github.com/vikim540/Gbootcms/issues/4) | **規劃中 (Backlog)** | RFC 6238 TOTP、智慧 IP 動態封鎖（連續 5 次錯誤拉黑 30m） |
| **Iteration 5** | SQLite 在線熱快照備份與 Cloudflare R2 雲端容災 | [#5](https://github.com/vikim540/Gbootcms/issues/5) | **規劃中 (Backlog)** | `VACUUM INTO` 不鎖表熱快照、S3/R2 物件存儲同步、7 天滾動歷史 |

---

## 各迭代詳細技術規劃

### Iteration 1：媒體庫健全度與 WebP 原生管道（當前閉環）
- **Issue 追蹤**：[Issue #1](https://github.com/vikim540/Gbootcms/issues/1)
- **核心成果**：
  - 徹底解決數據庫路徑（`upload/...`）與物理路徑（`static/upload/...`）不一致引起的「未使用」誤判與清理誤刪。
  - 導入純 Go `gowebp`，全上傳入口自動壓縮轉碼為 `.webp`，消除前端 Canvas 偽壓縮。
  - 修復 `media.html` 腳本時序中斷問題，在線伺服器實測 10 份預設資源全部正確解析與標記。

---

### Iteration 2：邊緣 CDN 快取聯動與分級快取架構
- **Issue 追蹤**：[Issue #2](https://github.com/vikim540/Gbootcms/issues/2)
- **技術設計**：
  1. 響應標頭自動帶入 `Cache-Tag: content-{id}, sort-{scode}, site-{acode}`。
  2. GORM 變更回呼中，非阻塞 Goroutine 調用 Cloudflare 邊緣 Purge API。
  3. 抽象 `CacheDriver` 介面，預設純 Go 記憶體快取，選配支援 Redis 集群快取。

---

### Iteration 3：全站動態 Markdown 鏡像與 AI 檢索管線
- **Issue 追蹤**：[Issue #3](https://github.com/vikim540/Gbootcms/issues/3)
- **技術設計**：
  1. 支援所有詳情頁以 `/news/15.md` 或 `Accept: text/markdown` 直接獲取純淨 Markdown 內容。
  2. 動態生成與自動滾動維護 `/llms.txt` 和 `/llms-full.txt`。
  3. 整合 Google Search Console Indexing API，在發布或修改文章時毫秒級向 Google 推送抓取請求。

---

### Iteration 4：後台管理安全防禦加固與 2FA
- **Issue 追蹤**：[Issue #4](https://github.com/vikim540/Gbootcms/issues/4)
- **技術設計**：
  1. 實現純 Go TOTP 動態碼校驗（支援 Google Authenticator 與 1Password）。
  2. 強化 `ip_filter.go`：密碼連續 5 次錯誤自動封鎖客戶端 IP 30 分鐘，防暴力字典攻擊。
  3. 安全審計日誌（`ay_syslog`）寫入關鍵變更。

---

### Iteration 5：SQLite 線上快照與雲端異地容災
- **Issue 追蹤**：[Issue #5](https://github.com/vikim540/Gbootcms/issues/5)
- **技術設計**：
  1. 呼叫 SQLite 3 原生 `VACUUM INTO` 進行無鎖原子一致性快照。
  2. 純 Go 輕量級 S3 協定驅動，自動將壓縮快照推送至 Cloudflare R2 儲存桶。
  3. 後台提供「備份管理」介面，支援查看快照清單、即時下載與災難一鍵復原。
