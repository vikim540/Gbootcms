package member

import (
	"pbootcms-go/apps/admin/helper"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MemberFieldController - 會員欄位控制器
// 對應 PHP: apps/admin/controller/member/MemberFieldController.php
type MemberFieldController struct {
	common.BaseController
}

// Index - 會員欄位列表（含新增Tab）
func (mf *MemberFieldController) Index(c *gin.Context) {
	var fields []model.MemberField
	model.DB.Order("sorting ASC, id ASC").Find(&fields)
	common.Render(c, "member/field.html", gin.H{
		"list":   true,
		"fields": fields,
		"C":      "member/field",
	})
}

// Add - 新增會員欄位
func (mf *MemberFieldController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		name := c.PostForm("name")
		description := c.PostForm("description")

		if name == "" {
			mf.JSONFail(c, "欄位名稱不能為空")
			return
		}
		if description == "" {
			mf.JSONFail(c, "欄位描述不能為空")
			return
		}

		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		length, _ := strconv.Atoi(c.DefaultPostForm("length", "20"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		now := time.Now()
		username := mf.GetAdminUsername(c)
		model.DB.Create(&model.MemberField{
			Name:        name,
			Length:      length,
			Required:    required,
			Description: description,
			Sorting:     sorting,
			Status:      status,
			CreateUser:  username,
			UpdateUser:  username,
			CreateTime:  now,
			UpdateTime:  now,
		})
		mf.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	// GET 請求重定向到列表頁
	c.Redirect(302, "/admin/member/field/index")
}

// Mod - 修改會員欄位（支援狀態切換 + 完整修改）
// 路由: /admin/member/field/mod/*action
func (mf *MemberFieldController) Mod(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))

	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	// 單欄位切換（狀態/必填開關）
	field := params["field"]
	if field == "" {
		field = c.Query("field")
	}
	value := params["value"]
	if value == "" {
		value = c.Query("value")
	}

	if field != "" && value != "" {
		model.DB.Model(&model.MemberField{}).Where("id = ?", id).Update(field, value)
		c.Redirect(302, "/admin/member/field/index")
		return
	}

	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		model.DB.Model(&model.MemberField{}).Where("id = ?", id).Updates(map[string]interface{}{
			"description": c.PostForm("description"),
			"required":    required,
			"sorting":     sorting,
			"status":      status,
		})
		mf.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// GET 載入修改頁面
	var field1 model.MemberField
	model.DB.First(&field1, id)
	common.Render(c, "member/field.html", gin.H{
		"mod":   true,
		"field": field1,
		"C":     "member/field",
	})
}

// Del - 刪除會員欄位
func (mf *MemberFieldController) Del(c *gin.Context) {
	// 支援 *action 通配符路徑: /del/id/123
	params := helper.ParseWildcardAction(c.Param("action"))
	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	if idStr == "" {
		idStr = c.PostForm("id")
	}
	if idStr == "" {
		mf.JSONFail(c, "缺少刪除目標ID")
		return
	}
	model.DB.Delete(&model.MemberField{}, idStr)
	mf.JSONOKMsg(c, common.NoticeDelete)
}
