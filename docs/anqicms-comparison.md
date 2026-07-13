# Gbootcms vs AnQiCMS 功能對比與升級建議

> 分析日期：2026-07-13 | AnQiCMS v3.5.9 | Gbootcms 基於 PbootCMS 3.2.12 移植

## 核心結論

Gbootcms 與 AnQiCMS 同為 Go 語言 CMS，但定位不同。Gbootcms 是 **PbootCMS 的忠實移植版**，主打 100% 數據庫兼容和無縫遷移；AnQiCMS 是 **原生設計的企業級 CMS**，主打 SEO + AI + 多站點。

## 技術棧對比

| 層級 | Gbootcms | AnQiCMS | 差距 |
|------|----------|---------|------|
| 語言 | Go 1.25 | Go 1.25 | 持平 |
| Web 框架 | Gin v1.12 | Iris v12 | 相當 |
| ORM | GORM v1.31 | GORM v1.31 | 持平 |
| 數據庫 | SQLite（純 Go 驅動） | MySQL / SQLite | AnQiCMS 支援億級數據 |
| 後台 UI | Layui + jQuery | React.js + Ant Design | AnQiCMS 現代化 |
| 認證 | 記憶體 Session + Cookie | JWT | JWT 更適合 API |
| 全文搜索 | 無（LIKE 查詢） | 悟空/ES/MeiliSearch/ZincSearch | 顯著差距 |
| AI 整合 | 無 | OpenAI/DeepSeek/星火 + MCP | 顯著差距 |
| 壓縮 | Brotli + Gzip | 未明確 | Gbootcms 優勢 |
| 物件存儲 | 本地 | 10 種（OSS/COS/七牛/R2/S3 等） | 顯著差距 |
| 部署 | 單二進制 + bat | Docker + 寶塔 + 二進制 + LNMP | 差距 |

## 升級優化建議（按優先級排序）

### P0 高優先級

1. **Docker 容器化部署** — 多階段構建，鏡像 < 30MB
2. **全文搜索功能** — 整合 MeiliSearch（輕量、純 Go 客戶端、支援中文分詞）
3. **301 重定向管理** — 新增 ay_301_redirect 表或利用 ay_config 存儲規則
4. **文檔回收站** — 利用 GORM DeletedAt 實現軟刪除
5. **API 接口體系** — 構建 RESTful API，為前後端分離做準備

### P1 中優先級

6. JSON-LD 結構化數據 — 純模板層實現
7. 外鏈 nofollow 自動處理 — 中間件/渲染層
8. 批量內容導入 — Excel/CSV
9. 定時發布增強 — 目前已靠查詢 `date <= now` 過濾實現（與 PbootCMS 一致）
10. 雲存儲支援 — 抽象存儲層接口
11. 敏感詞/防採集干擾碼
12. LLMs.txt 生成

### P2 低優先級

13. AI 寫作整合
14. 多站點管理
15. 前後端分離後台
16. GraphQL API
17. 整頁翻譯
18. 商城/訂單系統

## Gbootcms 的獨特優勢（不應放棄）

- **PbootCMS 100% 兼容** — 數據庫結構、URL 路由、模板語法完全兼容
- **零 CGO 依賴** — 純 Go SQLite 驅動，解壓即用
- **Brotli 壓縮** — 比 Gzip 壓縮率高 15-25%
- **模板熱重載** — fsnotify 監聽，開發時即改即看

## 關鍵原則

所有升級建議均以 **不改動原 PbootCMS 數據庫表結構** 為硬約束。新增功能通過：新增獨立表、利用 ay_config 配置項、純中間件/渲染層實現、可選組件等方式完成。
