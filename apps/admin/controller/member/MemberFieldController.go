package member

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MemberFieldController - 會員欄位控制器
// 對應 PHP: apps/admin/controller/MemberFieldController.php
type MemberFieldController struct {
	common.BaseController
}

// Index - 欄位列表
func (mf *MemberFieldController) Index(c *gin.Context) {
	var fields []model.MemberField
	model.DB.Order("sorting ASC").Find(&fields)
	common.Render(c, "member/field.html", gin.H{"fields": fields})
}

// Add - 新增欄位
func (mf *MemberFieldController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		length, _ := strconv.Atoi(c.DefaultPostForm("length", "0"))
		model.DB.Create(&model.MemberField{
			Name:        c.PostForm("name"),
			Length:      length,
			Required:    required,
			Description: c.PostForm("description"),
			Sorting:     sorting,
			Status:      1,
		})
		mf.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	common.Render(c, "member/field.html", gin.H{"action": "add"})
}

// Mod - 修改欄位
func (mf *MemberFieldController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		length, _ := strconv.Atoi(c.DefaultPostForm("length", "0"))
		model.DB.Model(&model.MemberField{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name":        c.PostForm("name"),
			"length":      length,
			"required":    required,
			"description": c.PostForm("description"),
			"sorting":     sorting,
		})
		mf.JSONOKMsg(c, common.NoticeModify)
		return
	}

	var field model.MemberField
	model.DB.First(&field, id)
	common.Render(c, "member/field.html", gin.H{"field": field, "action": "mod"})
}

// Del - 刪除欄位
func (mf *MemberFieldController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.MemberField{}, idStr)
	mf.JSONOKMsg(c, common.NoticeDelete)
}
