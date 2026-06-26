package system

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// TypeController - Dictionary Type Controller
// Corresponds to PHP: apps/admin/controller/TypeController.php
type TypeController struct {
	common.BaseController
}

// Index - Type list
func (tp *TypeController) Index(c *gin.Context) {
	var types []model.DictType
	model.DB.Order("sorting ASC").Find(&types)
	common.Render(c, "system/type.html", gin.H{"types": types})
}

// Add - Add new type
func (tp *TypeController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		model.DB.Create(&model.DictType{
			Code:    c.PostForm("code"),
			Name:    c.PostForm("name"),
			Sorting: sorting,
			Status:  1,
		})
		tp.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	common.Render(c, "system/type.html", gin.H{"action": "add"})
}

// Mod - Modify type
func (tp *TypeController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		model.DB.Model(&model.DictType{}).Where("id = ?", id).Updates(map[string]interface{}{
			"code":    c.PostForm("code"),
			"name":    c.PostForm("name"),
			"sorting": sorting,
		})
		tp.JSONOKMsg(c, common.NoticeModify)
		return
	}

	var dtype model.DictType
	model.DB.First(&dtype, id)
	common.Render(c, "system/type.html", gin.H{"type": dtype, "action": "mod"})
}

// Del - Delete type
func (tp *TypeController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.DictType{}, idStr)
	tp.JSONOKMsg(c, common.NoticeDelete)
}
