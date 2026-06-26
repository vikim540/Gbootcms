package system

import (
	"crypto/md5"
	"fmt"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
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
	var users []model.AdminUser
	model.DB.Order("id ASC").Find(&users)
	var roles []model.Role
	model.DB.Where("status = 1").Find(&roles)
	common.Render(c, "system/user.html", gin.H{"users": users, "roles": roles})
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
	uc.JSONOKMsg(c, common.NoticeDelete)
}
