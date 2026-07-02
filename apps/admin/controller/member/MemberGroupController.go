package member

import (
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

// Index - 會員等級列表
func (mg *MemberGroupController) Index(c *gin.Context) {
	var groups []model.MemberGroup
	model.DB.Order("id ASC").Find(&groups)
	common.Render(c, "member/group.html", gin.H{"groups": groups})
}

// Add - 新增會員等級
func (mg *MemberGroupController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		lscore, _ := strconv.Atoi(c.DefaultPostForm("lscore", "0"))
		uscore, _ := strconv.Atoi(c.DefaultPostForm("uscore", "0"))
		model.DB.Create(&model.MemberGroup{
			Gcode:       c.PostForm("gcode"),
			Gname:       c.PostForm("gname"),
			Description: c.PostForm("description"),
			Lscore:      lscore,
			Uscore:      uscore,
			Status:      1,
		})
		mg.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	common.Render(c, "member/group.html", gin.H{"action": "add"})
}

// Mod - 修改會員等級
func (mg *MemberGroupController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		lscore, _ := strconv.Atoi(c.DefaultPostForm("lscore", "0"))
		uscore, _ := strconv.Atoi(c.DefaultPostForm("uscore", "0"))
		model.DB.Model(&model.MemberGroup{}).Where("id = ?", id).Updates(map[string]interface{}{
			"gcode":       c.PostForm("gcode"),
			"gname":       c.PostForm("gname"),
			"description": c.PostForm("description"),
			"lscore":      lscore,
			"uscore":      uscore,
		})
		mg.JSONOKMsg(c, common.NoticeModify)
		return
	}

	var group model.MemberGroup
	model.DB.First(&group, id)
	common.Render(c, "member/group.html", gin.H{"group": group, "action": "mod"})
}

// Del - 刪除會員等級
func (mg *MemberGroupController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.MemberGroup{}, idStr)
	mg.JSONOKMsg(c, common.NoticeDelete)
}
