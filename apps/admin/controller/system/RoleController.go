package system

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RoleController - Role Management Controller
// Corresponds to PHP: apps/admin/controller/RoleController.php
type RoleController struct {
	common.BaseController
}

// Index - Role list
func (rc *RoleController) Index(c *gin.Context) {
	var roles []model.Role
	model.DB.Order("id ASC").Find(&roles)
	var menus []model.Menu
	model.DB.Where("status = 1").Order("sorting ASC").Find(&menus)
	common.Render(c, "system/role.html", gin.H{"roles": roles, "menus": menus})
}

// Add - Add new role
func (rc *RoleController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		levels := c.PostForm("levels")
		model.DB.Create(&model.Role{
			Code:   c.PostForm("code"),
			Name:   c.PostForm("name"),
			Levels: levels,
			Status: 1,
		})
		rc.JSONOKMsg(c, "新增成功")
		return
	}
	var menus []model.Menu
	model.DB.Where("status = 1").Order("sorting ASC").Find(&menus)
	common.Render(c, "system/role.html", gin.H{"menus": menus, "action": "add"})
}

// Mod - Modify role
func (rc *RoleController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		model.DB.Model(&model.Role{}).Where("id = ?", id).Updates(map[string]interface{}{
			"code":   c.PostForm("code"),
			"name":   c.PostForm("name"),
			"levels": c.PostForm("levels"),
		})
		rc.JSONOKMsg(c, "修改成功")
		return
	}

	var role model.Role
	model.DB.First(&role, id)
	var menus []model.Menu
	model.DB.Where("status = 1").Order("sorting ASC").Find(&menus)
	common.Render(c, "system/role.html", gin.H{"role": role, "menus": menus, "action": "mod"})
}

// Del - Delete role
func (rc *RoleController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.Role{}, idStr)
	rc.JSONOKMsg(c, "刪除成功")
}
