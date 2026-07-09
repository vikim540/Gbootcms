package system

import (
	"crypto/md5"
	"fmt"
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserController - User Management Controller
// Corresponds to PHP: apps/admin/controller/UserController.php
type UserController struct {
	common.BaseController
}

// Index - User list
func (uc *UserController) Index(c *gin.Context) {
	page, pageSize, offset := uc.Paginate(c)
	var total int64
	model.DB.Model(&model.AdminUser{}).Count(&total)
	var users []model.AdminUser
	model.DB.Order("id ASC").Offset(offset).Limit(pageSize).Find(&users)
	var roles []model.Role
	model.DB.Where("status = 1").Find(&roles)
	baseURL := "/admin/system/user/index"
	common.Render(c, "system/user.html", gin.H{
		"list":     true,
		"users":    users,
		"roles":    roles,
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
	})
}

// Add - Add new user
func (uc *UserController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		password := c.PostForm("password")
		encPwd := fmt.Sprintf("%x", md5.Sum([]byte(password)))
		model.DB.Create(&model.AdminUser{
			Username: c.PostForm("username"),
			Password: encPwd,
			Realname: c.PostForm("realname"),
			Rcodes:   c.PostForm("rcodes"),
			Status:   1,
		})
		uc.LogAction(c, "新增用戶成功")
		uc.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	var roles []model.Role
	model.DB.Where("status = 1").Find(&roles)
	common.Render(c, "system/user.html", gin.H{"roles": roles, "action": "add"})
}

// Mod - Modify user
func (uc *UserController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		updates := map[string]interface{}{
			"realname": c.PostForm("realname"),
			"rcodes":   c.PostForm("rcodes"),
		}
		password := c.PostForm("password")
		if password != "" {
			updates["password"] = fmt.Sprintf("%x", md5.Sum([]byte(password)))
		}
		model.DB.Model(&model.AdminUser{}).Where("id = ?", id).Updates(updates)
		uc.LogAction(c, "修改用戶成功")
		uc.JSONOKMsg(c, common.NoticeModify)
		return
	}

	var user model.AdminUser
	model.DB.First(&user, id)
	var roles []model.Role
	model.DB.Where("status = 1").Find(&roles)
	common.Render(c, "system/user.html", gin.H{"user": user, "roles": roles, "action": "mod"})
}

// Del - Delete user
func (uc *UserController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.AdminUser{}, idStr)
	uc.LogAction(c, "刪除用戶成功")
	uc.JSONOKMsg(c, common.NoticeDelete)
}
