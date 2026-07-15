package system

import (
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"strconv"
	"time"

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
	model.DB.WithContext(c.Request.Context()).Order("sorting ASC, id ASC").Find(&menus)
	common.Render(c, "system/menu.html", gin.H{"list": true, "menus": menus})
}

// Add - Add new menu
func (mc *MenuController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		now := time.Now()
		username := mc.GetAdminUsername(c)
		if err := model.DB.WithContext(c.Request.Context()).Create(&model.Menu{
			Mcode:       c.PostForm("mcode"),
			Pcode:       c.PostForm("pcode"),
			Name:        c.PostForm("name"),
			URL:         c.PostForm("url"),
			Ico:         c.PostForm("ico"),
			Sorting:     sorting,
			Status:      1,
			Shortcut:    0,
			CreateUser:  username,
			UpdateUser:  username,
			CreateTime:  now,
			UpdateTime:  now,
		}).Error; err != nil {
			mc.JSONFail(c, "新增失敗："+err.Error())
			return
		}
		mc.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	var menus []model.Menu
	model.DB.WithContext(c.Request.Context()).Order("sorting ASC").Find(&menus)
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
		now := time.Now()
		if err := model.DB.WithContext(c.Request.Context()).Model(&model.Menu{}).Where("id = ?", id).Updates(map[string]interface{}{
			"mcode":       c.PostForm("mcode"),
			"pcode":       c.PostForm("pcode"),
			"name":        c.PostForm("name"),
			"url":         c.PostForm("url"),
			"ico":         c.PostForm("ico"),
			"sorting":     sorting,
			"status":      status,
			"update_user": mc.GetAdminUsername(c),
			"update_time": now,
		}).Error; err != nil {
			mc.JSONFail(c, "修改失敗："+err.Error())
			return
		}
		mc.JSONOKMsg(c, common.NoticeModify)
		return
	}

	var menu model.Menu
	model.DB.WithContext(c.Request.Context()).First(&menu, id)
	var menus []model.Menu
	model.DB.WithContext(c.Request.Context()).Order("sorting ASC").Find(&menus)
	common.Render(c, "system/menu.html", gin.H{"menu": menu, "menus": menus, "action": "mod"})
}

// Del - Delete menu
func (mc *MenuController) Del(c *gin.Context) {
	idStr := c.Query("id")
	if err := model.DB.WithContext(c.Request.Context()).Delete(&model.Menu{}, idStr).Error; err != nil {
		mc.JSONFail(c, "刪除失敗："+err.Error())
		return
	}
	mc.JSONOKMsg(c, common.NoticeDelete)
}
