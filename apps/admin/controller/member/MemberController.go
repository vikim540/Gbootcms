package member

import (
	"crypto/md5"
	"fmt"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MemberController - Member Management Controller
// Corresponds to PHP: apps/admin/controller/MemberController.php
type MemberController struct {
	common.BaseController
}

// Index - Member list
func (mb *MemberController) Index(c *gin.Context) {
	var members []model.Member
	model.DB.Order("register_time DESC").Find(&members)
	var groups []model.MemberGroup
	model.DB.Where("status = 1").Find(&groups)
	common.Render(c, "member/member.html", gin.H{"members": members, "groups": groups})
}

// Add - Add new member
func (mb *MemberController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		password := c.PostForm("password")
		encPwd := fmt.Sprintf("%x", md5.Sum([]byte(password)))
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "0"))
		model.DB.Create(&model.Member{
			Username: c.PostForm("username"),
			Nickname: c.PostForm("nickname"),
			Password: encPwd,
			Email:    c.PostForm("email"),
			GID:      gid,
			Status:   1,
		})
		mb.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	var groups []model.MemberGroup
	model.DB.Where("status = 1").Find(&groups)
	common.Render(c, "member/member.html", gin.H{"groups": groups, "action": "add"})
}

// Mod - Modify member
func (mb *MemberController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "0"))
		updates := map[string]interface{}{
			"nickname": c.PostForm("nickname"),
			"email":    c.PostForm("email"),
			"gid":      gid,
		}
		password := c.PostForm("password")
		if password != "" {
			updates["password"] = fmt.Sprintf("%x", md5.Sum([]byte(password)))
		}
		model.DB.Model(&model.Member{}).Where("id = ?", id).Updates(updates)
		mb.JSONOKMsg(c, common.NoticeModify)
		return
	}

	var member model.Member
	model.DB.First(&member, id)
	var groups []model.MemberGroup
	model.DB.Where("status = 1").Find(&groups)
	common.Render(c, "member/member.html", gin.H{"member": member, "groups": groups, "action": "mod"})
}

// Del - Delete member
func (mb *MemberController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.Member{}, idStr)
	mb.JSONOKMsg(c, common.NoticeDelete)
}
