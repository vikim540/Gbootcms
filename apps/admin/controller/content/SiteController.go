package content

import (
	"pbootcms-go/apps/admin/model"
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
	common.Render(c, "content/site.html", gin.H{"sites": site})
}

// Mod - Modify site information
func (si *SiteController) Mod(c *gin.Context) {
	var site model.Site
	model.DB.FirstOrCreate(&site, model.Site{ID: 1})
	result := model.DB.Model(&site).Updates(map[string]interface{}{
		"title":       c.PostForm("title"),
		"subtitle":    c.PostForm("subtitle"),
		"domain":      c.PostForm("domain"),
		"keywords":    c.PostForm("keywords"),
		"description": c.PostForm("description"),
		"logo":        c.PostForm("logo"),
		"icp":         c.PostForm("icp"),
		"copyright":   c.PostForm("copyright"),
		"statistical": c.PostForm("statistical"),
		"theme":       c.PostForm("theme"),
	})
	if result.Error != nil {
		si.JSONFail(c, "修改失败: "+result.Error.Error())
		return
	}
	si.JSONOKMsg(c, "Modified successfully")
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
