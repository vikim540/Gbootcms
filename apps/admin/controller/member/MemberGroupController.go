package member

import (
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MemberGroupController - 會員等級控制器
// 對應 PHP: apps/admin/controller/MemberGroupController.php
type MemberGroupController struct {
	common.BaseController
}

// Index - 會員等級列表（含新增Tab）
func (mg *MemberGroupController) Index(c *gin.Context) {
	// 分頁處理
	page, pageSize, offset := mg.Paginate(c)
	baseURL := "/admin/member/group/index"

	// 統計總記錄數
	var total int64
	model.DB.WithContext(c.Request.Context()).Model(&model.MemberGroup{}).Count(&total)

	var groups []model.MemberGroup
	model.DB.WithContext(c.Request.Context()).Order("gcode ASC, id ASC").Offset(offset).Limit(pageSize).Find(&groups)
	common.Render(c, "member/group.html", gin.H{
		"list":     true,
		"groups":   groups,
		"C":        "member/group",
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
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
		model.DB.WithContext(c.Request.Context()).Model(&model.MemberGroup{}).Where("gcode = ?", gcode).Count(&count)
		if count > 0 {
			mg.JSONFail(c, "等級編號不能重複")
			return
		}

		lscore, _ := strconv.Atoi(c.DefaultPostForm("lscore", "0"))
		uscore, _ := strconv.Atoi(c.DefaultPostForm("uscore", "9999999999"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		now := time.Now().Format("2006-01-02 15:04:05")
		username := mg.GetAdminUsername(c)
		if err := model.DB.WithContext(c.Request.Context()).Create(&model.MemberGroup{
			Gcode:       gcode,
			Gname:       gname,
			Description: c.PostForm("description"),
			Lscore:      lscore,
			Uscore:      uscore,
			Status:      status,
			CreateUser:  username,
			UpdateUser:  username,
			CreateTime:  now,
			UpdateTime:  now,
		}).Error; err != nil {
			mg.JSONFail(c, "新增失敗："+err.Error())
			return
		}
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
		if err := model.DB.WithContext(c.Request.Context()).Model(&model.MemberGroup{}).Where("id = ?", id).Update(field, value).Error; err != nil {
			mg.JSONFail(c, "修改失敗："+err.Error())
			return
		}
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
		model.DB.WithContext(c.Request.Context()).Model(&model.MemberGroup{}).Where("gcode = ? AND id <> ?", gcode, id).Count(&count)
		if count > 0 {
			mg.JSONFail(c, "等級編號不能重複")
			return
		}

		lscore, _ := strconv.Atoi(c.DefaultPostForm("lscore", "0"))
		uscore, _ := strconv.Atoi(c.DefaultPostForm("uscore", "9999999999"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		now := time.Now().Format("2006-01-02 15:04:05")
		if err := model.DB.WithContext(c.Request.Context()).Model(&model.MemberGroup{}).Where("id = ?", id).Updates(map[string]interface{}{
			"gcode":       gcode,
			"gname":       gname,
			"description": c.PostForm("description"),
			"lscore":      lscore,
			"uscore":      uscore,
			"status":      status,
			"update_user": mg.GetAdminUsername(c),
			"update_time": now,
		}).Error; err != nil {
			mg.JSONFail(c, "修改失敗："+err.Error())
			return
		}
		mg.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// GET 載入修改頁面
	var group model.MemberGroup
	model.DB.WithContext(c.Request.Context()).First(&group, id)
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
	model.DB.WithContext(c.Request.Context()).Model(&model.Member{}).Where("gid = ?", strconv.Itoa(id)).Count(&memberCount)
	if memberCount > 0 {
		mg.JSONFail(c, "會員等級下存在用戶，無法直接刪除")
		return
	}

	if err := model.DB.WithContext(c.Request.Context()).Delete(&model.MemberGroup{}, id).Error; err != nil {
		mg.JSONFail(c, "刪除失敗："+err.Error())
		return
	}
	mg.JSONOKMsg(c, common.NoticeDelete)
}
