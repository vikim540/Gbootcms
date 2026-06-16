package system

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MenuController - Menu Management Controller
// Corresponds to PHP: apps/admin/controller/MenuController.php
type MenuController struct {
	common.BaseController
}

// Index - Menu list
func (mc *MenuController) Index(c *gin.Context) {
	var menus []model.Menu
	model.DB.Order("sorting ASC, id ASC").Find(&menus)
	common.Render(c, "system/menu.html", gin.H{"list": true, "menus": menus})
}

// Add - Add new menu
func (mc *MenuController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		model.DB.Create(&model.Menu{
			Mcode:    c.PostForm("mcode"),
			Pcode:    c.PostForm("pcode"),
			Name:     c.PostForm("name"),
			URL:      c.PostForm("url"),
			Ico:      c.PostForm("ico"),
			Sorting:  sorting,
			Status:   1,
			Shortcut: 0,
		})
		mc.JSONOKMsg(c, "Added successfully")
		return
	}
	var menus []model.Menu
	model.DB.Order("sorting ASC").Find(&menus)
	common.Render(c, "system/menu.html", gin.H{"menus": menus, "action": "add"})
}

// Mod - Modify menu
func (mc *MenuController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	// 批量排序：POST submit=sorting
	if mc.IsBatchSort(c) {
		mc.BatchSort(c, &model.Menu{}, "sorting", 255)
		return
	}

	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))
		model.DB.Model(&model.Menu{}).Where("id = ?", id).Updates(map[string]interface{}{
			"mcode":   c.PostForm("mcode"),
			"pcode":   c.PostForm("pcode"),
			"name":    c.PostForm("name"),
			"url":     c.PostForm("url"),
			"ico":     c.PostForm("ico"),
			"sorting": sorting,
			"status":  status,
		})
		mc.JSONOKMsg(c, "Modified successfully")
		return
	}

	var menu model.Menu
	model.DB.First(&menu, id)
	var menus []model.Menu
	model.DB.Order("sorting ASC").Find(&menus)
	common.Render(c, "system/menu.html", gin.H{"menu": menu, "menus": menus, "action": "mod"})
}

// Del - Delete menu
func (mc *MenuController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.Menu{}, idStr)
	mc.JSONOKMsg(c, "Deleted successfully")
}
