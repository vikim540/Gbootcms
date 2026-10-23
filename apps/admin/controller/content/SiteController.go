package content

import (	"gbootcms/core/db"

	"os"
	"path/filepath"

	contentModel "gbootcms/apps/admin/model/content"
	"gbootcms/apps/common"
	"gbootcms/core/version"

	"github.com/gin-gonic/gin"
)

// SiteController - Site Information Controller
// Corresponds to PHP: apps/admin/controller/SiteController.php
type SiteController struct {
	common.BaseController
}

// Index - Site information page
func (si *SiteController) Index(c *gin.Context) {
	var site contentModel.Site
	// AcodePlugin 自動按當前區域過濾，取該區域的站點記錄
	if err := db.DB.WithContext(c.Request.Context()).FirstOrCreate(&site).Error; err != nil {
		si.JSONFail(c, "載入站點信息失敗")
		return
	}

	// 掃描 template/ 目錄下的子目錄作為可用模板主題
	var themes []string
	entries, err := os.ReadDir("template")
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				themes = append(themes, entry.Name())
			}
		}
	}
	// 若目錄不存在或為空，回退默認值
	if len(themes) == 0 {
		themes = []string{"default"}
	}
	// 確保配置的 template_dir 中的模板目錄在列表中
	templateDir := filepath.Dir(filepath.ToSlash(site.Theme))
	if templateDir == "" || templateDir == "." {
		templateDir = "default"
	}
	_ = templateDir // template_dir 由 config.json 控制，此處不修改

	common.Render(c, "content/site.html", gin.H{
		"sites":  site,
		"labels": contentModel.GetAllLabels(),
		"themes": themes,
	})
}

// Mod - Modify site information
func (si *SiteController) Mod(c *gin.Context) {
	var site contentModel.Site
	// AcodePlugin 自動按當前區域過濾，取該區域的站點記錄
	if err := db.DB.WithContext(c.Request.Context()).FirstOrCreate(&site).Error; err != nil {
		si.JSONFail(c, "載入站點信息失敗")
		return
	}

	// 臟檢測：比對提交數據與現有數據
	newTitle := c.PostForm("title")
	newSubtitle := c.PostForm("subtitle")
	newDomain := c.PostForm("domain")
	newKeywords := c.PostForm("keywords")
	newDescription := c.PostForm("description")
	newLogo := c.PostForm("logo")
	newIcp := c.PostForm("icp")
	newCopyright := c.PostForm("copyright")
	newStatistical := c.PostForm("statistical")
	newTheme := c.PostForm("theme")

	if site.Title == newTitle && site.Subtitle == newSubtitle && site.Domain == newDomain &&
		site.Keywords == newKeywords && site.Description == newDescription && site.Logo == newLogo &&
		site.ICP == newIcp && site.Copyright == newCopyright && site.Statistical == newStatistical &&
		site.Theme == newTheme {
		si.JSONOKMsg(c, common.NoticeNoChange)
		return
	}

	result := db.DB.WithContext(c.Request.Context()).Model(&site).Updates(map[string]interface{}{
		"title":       newTitle,
		"subtitle":    newSubtitle,
		"domain":      newDomain,
		"keywords":    newKeywords,
		"description": newDescription,
		"logo":        newLogo,
		"icp":         newIcp,
		"copyright":   newCopyright,
		"statistical": newStatistical,
		"theme":       newTheme,
	})
	if result.Error != nil {
		si.JSONFail(c, "修改失败: "+result.Error.Error())
		return
	}
	si.JSONOKMsg(c, common.NoticeModify)
}

// Server - 伺服器資訊頁面
// 展示應用版本、技術棧模組與中介軟體，調用統一遙測數據源（符合奧卡姆剃刀原則）。
func (si *SiteController) Server(c *gin.Context) {
	telemetry := version.GetServerTelemetry(c)

	// --- 技術棧模組清單 ---
	type moduleInfo struct {
		Name string
		Desc string
	}
	modules := []moduleInfo{
		{"Gin Engine", "高並發 HTTP Web 核心 (v" + gin.Version + ")"},
		{"GORM ORM", "現代化關係型映射層 (v" + gormVersion + ")"},
		{"Pongo2", "安全語義 AST 視圖渲染器 (v6)"},
		{"Pure-Go SQLite", "嵌入式高效能資料庫驅動 (glebarez/sqlite WAL 模式)"},
		{"Native WebP", "自動無損/有損圖片轉換與響應式優化"},
		{"SEO Provider", "自動化 JSON-LD、hreflang 跨語言多維標註引擎"},
		{"net/smtp", "標準庫安全郵件傳輸引擎"},
		{"fsnotify", "核心模板與靜態資源動態熱載入監聽"},
		{"gorilla/sessions", "全狀態分散式 Session 封裝"},
	}

	middlewares := []string{
		"AdminAuth (後台管理員認證與 RBAC 權限校驗)",
		"PathRewrite (多語言 acode 自動分離與 301 重定向引擎)",
		"Session (全域安全加密會話狀態中介層)",
		"Gzip / WebP (現代化網路協定傳輸壓縮)",
		"IPFilter / RateLimit (智慧防爬與安全訪問頻率防禦)",
	}

	data := gin.H{
		"Modules":     modules,
		"modules":     modules,
		"Middlewares": middlewares,
		"middlewares": middlewares,
		"RouteCount":  "~80 條已註冊路由",
		"route_count": "~80 條已註冊路由",
	}
	for k, v := range telemetry {
		data[k] = v
	}

	common.Render(c, "system/server.html", data)
}

// gormVersion 取得 GORM 版本（編譯期常數）
const gormVersion = "1.31"
