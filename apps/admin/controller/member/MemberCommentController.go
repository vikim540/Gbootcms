package member

import (
	"fmt"
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"gbootcms/apps/common/middleware"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MemberCommentController - 會員評論控制器
// 對應 PHP: apps/admin/controller/member/MemberCommentController.php
type MemberCommentController struct {
	common.BaseController
}

// allowedSearchFields 允許搜索的欄位白名單（防止SQL注入）
var allowedSearchFields = map[string]bool{
	"b.title":      true,
	"a.comment":    true,
	"c.username":   true,
	"c.nickname":   true,
}

// invalidateCommentsCache 查詢評論涉及的 contentid 並精準失效對應文章快取
// member_comment 在 skipTables 中，GORM 回調不觸發，後台操作需手動失效
func invalidateCommentsCache(ids []string) {
	if len(ids) == 0 {
		return
	}
	var contentIDs []uint
	model.DB.Model(&model.MemberComment{}).Where("id IN ?", ids).
		Distinct("contentid").Pluck("contentid", &contentIDs)
	for _, cid := range contentIDs {
		if cid > 0 {
			middleware.InvalidateTag(fmt.Sprintf("content:%d", cid))
		}
	}
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
		if !comment.CreateTime.IsZero() {
			comment.CreateTimeStr = comment.CreateTime.Format("2006-01-02 15:04:05")
		}
		common.Render(c, "member/comment.html", gin.H{"more": true, "comment": comment, "C": "member/comment"})
		return
	}

	// 列表模式：支持 field+keyword 搜索
	field := c.Query("field")
	keyword := c.Query("keyword")

	// 分頁處理
	page, pageSize, offset := mc.Paginate(c)
	baseURL := "/admin/member/comment/index"
	if field != "" && keyword != "" && allowedSearchFields[field] {
		baseURL += "?field=" + field + "&keyword=" + keyword
	}

	query := model.DB.Table("ay_member_comment a").
		Select("a.*, b.title, c.username, c.nickname, c.headpic").
		Joins("LEFT JOIN ay_content b ON a.contentid=b.id").
		Joins("LEFT JOIN ay_member c ON a.uid=c.id").
		Order("a.id DESC")

	// 白名單驗證 field（防止SQL注入）
	if field != "" && keyword != "" && allowedSearchFields[field] {
		query = query.Where(field+" LIKE ?", "%"+keyword+"%")
	}

	// 統計總記錄數（獨立查詢，避免與含 Select 的查詢重用 Statement 造成污染）
	var total int64
	countQuery := model.DB.Table("ay_member_comment a").
		Joins("LEFT JOIN ay_content b ON a.contentid=b.id").
		Joins("LEFT JOIN ay_member c ON a.uid=c.id")
	if field != "" && keyword != "" && allowedSearchFields[field] {
		countQuery = countQuery.Where(field+" LIKE ?", "%"+keyword+"%")
	}
	countQuery.Count(&total)

	var comments []model.CommentView
	query.Offset(offset).Limit(pageSize).Find(&comments)
	// 格式化時間
	for i := range comments {
		if !comments[i].CreateTime.IsZero() {
			comments[i].CreateTimeStr = comments[i].CreateTime.Format("2006-01-02 15:04:05")
		}
	}
	common.Render(c, "member/comment.html", gin.H{
		"list":     true,
		"comments": comments,
		"C":        "member/comment",
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
	})
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
				invalidateCommentsCache(list)
			}
			mc.JSONOKMsg(c, common.NoticeModify)
			return
		case "verify0":
			if len(list) > 0 {
				model.DB.Model(&model.MemberComment{}).Where("id IN ?", list).Update("status", 0)
				invalidateCommentsCache(list)
			}
			mc.JSONOKMsg(c, common.NoticeModify)
			return
		}
	}

	// 解析 *action 通配符
	params := helper.ParseWildcardAction(c.Param("action"))
	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	field := params["field"]
	if field == "" {
		field = c.Query("field")
	}
	value := params["value"]
	if value == "" {
		value = c.Query("value")
	}

	// GET：單字段修改（狀態切換）
	if field != "" && value != "" {
		id, _ := strconv.Atoi(idStr)
		model.DB.Model(&model.MemberComment{}).Where("id = ?", id).Update(field, value)
		invalidateCommentsCache([]string{idStr})
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
	for i := range comments {
		if !comments[i].CreateTime.IsZero() {
			comments[i].CreateTimeStr = comments[i].CreateTime.Format("2006-01-02 15:04:05")
		}
	}
	common.Render(c, "member/comment.html", gin.H{"list": true, "comments": comments, "C": "member/comment"})
}

// Del - 刪除評論（單條/批量）
func (mc *MemberCommentController) Del(c *gin.Context) {
	// POST：批量刪除
	if c.Request.Method == "POST" {
		list := c.PostFormArray("list[]")
		if len(list) > 0 {
			// 刪除前先查詢涉及的 contentid（刪除後無法再查詢）
			invalidateCommentsCache(list)
			model.DB.Where("id IN ?", list).Delete(&model.MemberComment{})
		}
		mc.JSONOKMsg(c, common.NoticeDelete)
		return
	}

	// GET：單條刪除 — 支援 *action 通配符路徑: /del/id/123
	params := helper.ParseWildcardAction(c.Param("action"))
	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	if idStr == "" {
		mc.JSONFail(c, "缺少刪除目標ID")
		return
	}
	// 刪除前先查詢涉及的 contentid（刪除後無法再查詢）
	invalidateCommentsCache([]string{idStr})
	model.DB.Delete(&model.MemberComment{}, idStr)
	mc.JSONOKMsg(c, common.NoticeDelete)
}
