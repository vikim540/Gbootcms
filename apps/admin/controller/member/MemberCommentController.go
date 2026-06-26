package member

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MemberCommentController - Member Comment Controller
// Corresponds to PHP: apps/admin/controller/CommentController.php
type MemberCommentController struct {
	common.BaseController
}

// Index - Comment list
func (mc *MemberCommentController) Index(c *gin.Context) {
	var comments []model.Comment
	model.DB.Order("create_time DESC").Find(&comments)
	common.Render(c, "member/comment.html", gin.H{"comments": comments})
}

// Mod - Modify comment
func (mc *MemberCommentController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		ischeck, _ := strconv.Atoi(c.DefaultPostForm("ischeck", "1"))
		model.DB.Model(&model.Comment{}).Where("id = ?", id).Update("ischeck", ischeck)
		mc.JSONOKMsg(c, common.NoticeOperation)
		return
	}

	var comment model.Comment
	model.DB.First(&comment, id)
	common.Render(c, "member/comment.html", gin.H{"comment": comment, "action": "mod"})
}

// Del - Delete comment
func (mc *MemberCommentController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.Comment{}, idStr)
	mc.JSONOKMsg(c, common.NoticeDelete)
}
