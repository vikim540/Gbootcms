# Gbootcms 開發者與維運手冊 (Wiki)

歡迎查閱 **Gbootcms** 官方文檔庫。本項目是基於 PbootCMS 3.2.12 的現代化 Go 語言原生移植版，專為高並發、極致輕量與生產級穩定性而設計。

本手冊嚴格遵循**奧卡姆剃刀原則（如無必要，勿增實體）**，文字言簡意賅，並為核心功能提供立即可用的程式碼案例。

---

## 核心章節導覽

| 專題模組 | 內容摘要 | 關鍵代碼範例 |
| :--- | :--- | :--- |
| [快速上手 (Quick Start)](Quick-Start) | 開發環境、本地編譯、發行版打包與啟動 | `build-run.bat`, `build-linux.bat` |
| [系統架構 (Architecture)](Architecture) | 單一 DB 連線池、唯一 Migrate 入口、遙測單一資料源 | `core/db/db.go`, `telemetry.go` |
| [前台標籤語法 (TagParser)](TagParser-Guide) | `{gboot:xxx}` 標籤語法、欄目與內容解析 | `{gboot:nav}`, `{gboot:list}` |
| [原生圖片壓縮 (Image Compression)](Image-Compression) | 純 Go 伺服器端 WebP 轉碼與雙三次縮放管道 | `imgcompress.CompressToWebP()` |
| [單模板多語言 (Multilingual & i18n)](Multilingual-and-i18n) | 單套 HTML 字典化國際化、杜絕 72 個克隆模板 | `{i18n:Search}`, `en.json` |
| [SEO 與 GEO 配套 (SEO & GEO)](SEO-and-GEO) | 動態 Host 感知 Robots、多語系 Sitemap、JSON-LD | `robots.txt`, `llms.txt`, Schema |
| [生產部署 (Deployment Guide)](Deployment-Guide) | UPX 極限壓縮二進制、1Panel API 自動化熱部署 | `deploy-1panel.ps1`, Docker |

---

## 快速代碼範例：30 秒啟動一個自定義控制器

```go
package hello

import (
	"net/http"
	"gbootcms/apps/common"
	"gbootcms/core/db"
	"github.com/gin-gonic/gin"
)

type HelloController struct {
	common.BaseController
}

func (ctrl *HelloController) Ping(c *gin.Context) {
	// 透過唯一 DB 連線執行查詢
	var count int64
	db.DB.Table("ay_content").Count(&count)

	// 使用統一 JSON 格式響應
	ctrl.JSONOK(c, gin.H{
		"message": "pong",
		"total_articles": count,
	})
}
```
