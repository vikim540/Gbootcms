package member

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MemberCommentController - 會員評論控制器
// 對應 PHP: apps/admin/controller/member/MemberCommentController.php
type MemberCommentController struct {
	common.BaseController
}

// Index - 評論列表/搜索/詳情
func (mc *MemberCommentController) Index(c *gin.Context) {
	// 詳情模式：帶 id 參數時顯示單條評論詳情
	if idStr := c.Query("id"); idStr != "" {
		id, _ := strconv.Atoi(idStr)
		var comment model.CommentView
		model.DB.Table("ay_member_comment a").
			Select("a.*, b.title, c.username, c.nickname, c.headpic, d.username as pusername, d.nickname as pnickname").
			Joins("LEFT JOIN ay_content b ON a.contentid=b.id").
			Joins("LEFT JOIN ay_member c ON a.uid=c.id").
			Joins("LEFT JOIN ay_member d ON a.puid=d.id").
			Where("a.id = ?", id).
			First(&comment)
		common.Render(c, "member/comment.html", gin.H{"more": true, "comment": comment})
		return
	}

	// 列表模式：支持 field+keyword 搜索
	field := c.Query("field")
	keyword := c.Query("keyword")

	query := model.DB.Table("ay_member_comment a").
		Select("a.*, b.title, c.username, c.nickname, c.headpic").
		Joins("LEFT JOIN ay_content b ON a.contentid=b.id").
		Joins("LEFT JOIN ay_member c ON a.uid=c.id").
		Order("a.id DESC")

	if field != "" && keyword != "" {
		query = query.Where(field+" LIKE ?", "%"+keyword+"%")
	}

	var comments []model.CommentView
	query.Find(&comments)
	common.Render(c, "member/comment.html", gin.H{"list": true, "comments": comments})
}

// Mod - 修改評論（單字段切換/批量審核/批量禁用）
func (mc *MemberCommentController) Mod(c *gin.Context) {
	// POST：批量操作
	if c.Request.Method == "POST" {
		submit := c.PostForm("submit")
		list := c.PostFormArray("list[]")

		switch submit {
		case "verify1":
			if len(list) > 0 {
				model.DB.Model(&model.MemberComment{}).Where("id IN ?", list).Update("status", 1)
			}
			mc.JSONOKMsg(c, common.NoticeModify)
			return
		case "verify0":
			if len(list) > 0 {
				model.DB.Model(&model.MemberComment{}).Where("id IN ?", list).Update("status", 0)
			}
			mc.JSONOKMsg(c, common.NoticeModify)
			return
		}
	}

	// GET：單字段修改（狀態切換）
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	field := c.Query("field")
	value := c.Query("value")

	if field != "" && value != "" {
		id, _ := strconv.Atoi(idStr)
		model.DB.Model(&model.MemberComment{}).Where("id = ?", id).Update(field, value)
		mc.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// 無參數時返回列表
	var comments []model.CommentView
	model.DB.Table("ay_member_comment a").
		Select("a.*, b.title, c.username, c.nickname, c.headpic").
		Joins("LEFT JOIN ay_content b ON a.contentid=b.id").
		Joins("LEFT JOIN ay_member c ON a.uid=c.id").
		Order("a.id DESC").
		Find(&comments)
	common.Render(c, "member/comment.html", gin.H{"list": true, "comments": comments})
}

// Del - 刪除評論（單條/批量）
func (mc *MemberCommentController) Del(c *gin.Context) {
	// POST：批量刪除
	if c.Request.Method == "POST" {
		list := c.PostFormArray("list[]")
		if len(list) > 0 {
			model.DB.Where("id IN ?", list).Delete(&model.MemberComment{})
		}
		mc.JSONOKMsg(c, common.NoticeDelete)
		return
	}

	// GET：單條刪除
	idStr := c.Query("id")
	model.DB.Delete(&model.MemberComment{}, idStr)
	mc.JSONOKMsg(c, common.NoticeDelete)
}
