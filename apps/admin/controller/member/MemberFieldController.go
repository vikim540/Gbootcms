package member

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MemberFieldController - Member Field Controller
// Corresponds to PHP: apps/admin/controller/MemberFieldController.php
type MemberFieldController struct {
	common.BaseController
}

// Index - Field list
func (mf *MemberFieldController) Index(c *gin.Context) {
	var fields []model.MemberField
	model.DB.Order("sorting ASC").Find(&fields)
	common.Render(c, "member/field.html", gin.H{"fields": fields})
}

// Add - Add new field
func (mf *MemberFieldController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		model.DB.Create(&model.MemberField{
			Name:     c.PostForm("name"),
			Field:    c.PostForm("field"),
			Type:     c.PostForm("type"),
			Required: required,
			Sorting:  sorting,
			Status:   1,
		})
		mf.JSONOKMsg(c, "新增成功")
		return
	}
	common.Render(c, "member/field.html", gin.H{"action": "add"})
}

// Mod - Modify field
func (mf *MemberFieldController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		model.DB.Model(&model.MemberField{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name":     c.PostForm("name"),
			"field":    c.PostForm("field"),
			"type":     c.PostForm("type"),
			"required": required,
			"sorting":  sorting,
		})
		mf.JSONOKMsg(c, "修改成功")
		return
	}

	var field model.MemberField
	model.DB.First(&field, id)
	common.Render(c, "member/field.html", gin.H{"field": field, "action": "mod"})
}

// Del - Delete field
func (mf *MemberFieldController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.MemberField{}, idStr)
	mf.JSONOKMsg(c, "刪除成功")
}
