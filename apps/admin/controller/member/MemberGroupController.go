package member

import (
	"pbootcms-go/apps/admin/helper"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MemberGroupController - 會員等級控制器
// 對應 PHP: apps/admin/controller/MemberGroupController.php
type MemberGroupController struct {
	common.BaseController
}

// Index - 會員等級列表（含新增Tab）
func (mg *MemberGroupController) Index(c *gin.Context) {
	var groups []model.MemberGroup
	model.DB.Order("gcode ASC, id ASC").Find(&groups)
	common.Render(c, "member/group.html", gin.H{
		"list":   true,
		"groups": groups,
		"C":      "member/group",
	})
}

// Add - 新增會員等級
func (mg *MemberGroupController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		gcode := c.PostForm("gcode")
		gname := c.PostForm("gname")

		if gcode == "" {
			mg.JSONFail(c, "等級編號不能為空")
			return
		}
		if gname == "" {
			mg.JSONFail(c, "等級名稱不能為空")
			return
		}

		// 檢查編號是否重複
		var count int64
		model.DB.Model(&model.MemberGroup{}).Where("gcode = ?", gcode).Count(&count)
		if count > 0 {
			mg.JSONFail(c, "等級編號不能重複")
			return
		}

		lscore, _ := strconv.Atoi(c.DefaultPostForm("lscore", "0"))
		uscore, _ := strconv.Atoi(c.DefaultPostForm("uscore", "9999999999"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		model.DB.Create(&model.MemberGroup{
			Gcode:       gcode,
			Gname:       gname,
			Description: c.PostForm("description"),
			Lscore:      lscore,
			Uscore:      uscore,
			Status:      status,
		})
		mg.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	// GET 請求重定向到列表頁（新增表單是列表頁的Tab）
	c.Redirect(302, "/admin/member/group/index")
}

// Mod - 修改會員等級（支援狀態切換 + 完整修改）
// 路由: /admin/member/group/mod/*action
// action 格式: /id/123 或 /id/123/field/status/value/0
func (mg *MemberGroupController) Mod(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))

	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	// 單欄位切換（狀態開關）
	field := params["field"]
	if field == "" {
		field = c.Query("field")
	}
	value := params["value"]
	if value == "" {
		value = c.Query("value")
	}

	if field != "" && value != "" {
		// 執行單欄位更新
		model.DB.Model(&model.MemberGroup{}).Where("id = ?", id).Update(field, value)
		c.Redirect(302, "/admin/member/group/index")
		return
	}

	if c.Request.Method == "POST" {
		gcode := c.PostForm("gcode")
		gname := c.PostForm("gname")

		if gcode == "" {
			mg.JSONFail(c, "等級編號不能為空")
			return
		}
		if gname == "" {
			mg.JSONFail(c, "等級名稱不能為空")
			return
		}

		// 檢查編號是否重複（排除自身）
		var count int64
		model.DB.Model(&model.MemberGroup{}).Where("gcode = ? AND id <> ?", gcode, id).Count(&count)
		if count > 0 {
			mg.JSONFail(c, "等級編號不能重複")
			return
		}

		lscore, _ := strconv.Atoi(c.DefaultPostForm("lscore", "0"))
		uscore, _ := strconv.Atoi(c.DefaultPostForm("uscore", "9999999999"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		model.DB.Model(&model.MemberGroup{}).Where("id = ?", id).Updates(map[string]interface{}{
			"gcode":       gcode,
			"gname":       gname,
			"description": c.PostForm("description"),
			"lscore":      lscore,
			"uscore":      uscore,
			"status":      status,
		})
		mg.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// GET 載入修改頁面
	var group model.MemberGroup
	model.DB.First(&group, id)
	common.Render(c, "member/group.html", gin.H{
		"mod":   true,
		"group": group,
		"C":     "member/group",
	})
}

// Del - 刪除會員等級
func (mg *MemberGroupController) Del(c *gin.Context) {
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
		mg.JSONFail(c, "缺少刪除目標ID")
		return
	}
	id, _ := strconv.Atoi(idStr)

	// 檢查等級下是否有會員
	var memberCount int64
	model.DB.Model(&model.Member{}).Where("gid = ?", strconv.Itoa(id)).Count(&memberCount)
	if memberCount > 0 {
		mg.JSONFail(c, "會員等級下存在用戶，無法直接刪除")
		return
	}

	model.DB.Delete(&model.MemberGroup{}, id)
	mg.JSONOKMsg(c, common.NoticeDelete)
}

