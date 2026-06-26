package member

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"

	"github.com/gin-gonic/gin"
)

// MemberGroupController - Member Group Controller
// Corresponds to PHP: apps/admin/controller/MemberGroupController.php
type MemberGroupController struct {
	common.BaseController
}

// Index - Member group list
func (mg *MemberGroupController) Index(c *gin.Context) {
	var groups []model.MemberGroup
	model.DB.Order("id ASC").Find(&groups)
	common.Render(c, "member/group.html", gin.H{"groups": groups})
}

// Add - Add new member group
func (mg *MemberGroupController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		model.DB.Create(&model.MemberGroup{
			Code:   c.PostForm("code"),
			Name:   c.PostForm("name"),
			Status: 1,
		})
		mg.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	common.Render(c, "member/group.html", gin.H{"action": "add"})
}

// Mod - Modify member group
func (mg *MemberGroupController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := parseInt(idStr)

	if c.Request.Method == "POST" {
		model.DB.Model(&model.MemberGroup{}).Where("id = ?", id).Updates(map[string]interface{}{
			"code": c.PostForm("code"),
			"name": c.PostForm("name"),
		})
		mg.JSONOKMsg(c, common.NoticeModify)
		return
	}

	var group model.MemberGroup
	model.DB.First(&group, id)
	common.Render(c, "member/group.html", gin.H{"group": group, "action": "mod"})
}

// Del - Delete member group
func (mg *MemberGroupController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.MemberGroup{}, idStr)
	mg.JSONOKMsg(c, common.NoticeDelete)
}

// parseInt - String to integer
func parseInt(s string) (int, error) {
	var n int
	_, err := scanInt(s, &n)
	return n, err
}

func scanInt(s string, n *int) (int, error) {
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		*n = *n*10 + int(c-'0')
	}
	return *n, nil
}
