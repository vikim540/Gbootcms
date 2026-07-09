package content

import (
	"gbootcms/apps/admin/model"
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/common"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gin-gonic/gin"
)

// SiteController - Site Information Controller
// Corresponds to PHP: apps/admin/controller/SiteController.php
type SiteController struct {
	common.BaseController
}

// Index - Site information page
func (si *SiteController) Index(c *gin.Context) {
	var site model.Site
	// AcodePlugin 自動按當前區域過濾，取該區域的站點記錄
	model.DB.WithContext(c.Request.Context()).FirstOrCreate(&site)

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
		"labels": content.GetAllLabels(),
		"themes": themes,
	})
}

// Mod - Modify site information
func (si *SiteController) Mod(c *gin.Context) {
	var site model.Site
	// AcodePlugin 自動按當前區域過濾，取該區域的站點記錄
	model.DB.WithContext(c.Request.Context()).FirstOrCreate(&site)

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

	result := model.DB.WithContext(c.Request.Context()).Model(&site).Updates(map[string]interface{}{
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
// 展示應用版本、技術棧模組與中介軟體，讓管理者一目了然專案技術全貌。
func (si *SiteController) Server(c *gin.Context) {
	// --- 技術棧模組清單 ---
	type moduleInfo struct {
		Name string
		Desc string
	}
	modules := []moduleInfo{
		{"Gin", "HTTP Web 框架 (v" + gin.Version + ")"},
		{"GORM", "ORM 資料庫框架 (v" + gormVersion + ")"},
		{"pongo2", "Django 語法模板引擎 (v6)"},
		{"SQLite", "嵌入式資料庫引擎 (pure Go, glebarez/sqlite)"},
		{"net/smtp", "標準庫 SMTP 郵件發送"},
		{"crypto/tls", "標準庫 TLS 加密傳輸"},
		{"image/*", "標準庫圖片解碼 (JPEG/PNG/GIF/BMP/WebP)"},
		{"fsnotify", "檔案系統事件監聽 (模板熱更新)"},
		{"gorilla/sessions", "Session 會話管理"},
		{"net/http", "標準庫 HTTP 伺服器"},
	}

	// --- 已載入的中介軟體 ---
	middlewares := []string{
		"AdminAuth (後台登入驗證)",
		"PathRewrite (路徑重寫)",
		"Session (會話管理)",
	}

	common.Render(c, "system/server.html", gin.H{
		"go_version":   runtime.Version(),
		"goos":         runtime.GOOS,
		"goarch":       runtime.GOARCH,
		"web_software": "Go " + runtime.Version() + " / Gin v" + gin.Version,
		"db_driver":    "SQLite (GORM + glebarez/sqlite)",
		"modules":      modules,
		"middlewares":  middlewares,
		"route_count":  "~80 條已註冊路由",
	})
}

// gormVersion 取得 GORM 版本（編譯期常數）
const gormVersion = "1.31"
