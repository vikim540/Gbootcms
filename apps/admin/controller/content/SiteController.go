package content

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/common"

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
	model.DB.FirstOrCreate(&site, model.Site{ID: 1})
	common.Render(c, "content/site.html", gin.H{
		"sites":  site,
		"labels": content.GetAllLabels(),
	})
}

// Mod - Modify site information
func (si *SiteController) Mod(c *gin.Context) {
	var site model.Site
	model.DB.FirstOrCreate(&site, model.Site{ID: 1})

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

	result := model.DB.Model(&site).Updates(map[string]interface{}{
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

// Server - Server information page
func (si *SiteController) Server(c *gin.Context) {
	common.Render(c, "system/server.html", gin.H{
		"server_os":   "Windows",
		"server_soft": "Go/Gin",
		"php_version": "Go 1.22",
		"db_type":     "SQLite/MySQL",
		"file_upload": "Allowed",
		"max_upload":  "50M",
	})
}
