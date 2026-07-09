package system

import (
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
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
	page, pageSize, offset := rc.Paginate(c)
	var total int64
	model.DB.Model(&model.Role{}).Count(&total)
	var roles []model.Role
	model.DB.Order("id ASC").Offset(offset).Limit(pageSize).Find(&roles)
	var menus []model.Menu
	model.DB.Where("status = 1").Order("sorting ASC").Find(&menus)
	baseURL := "/admin/system/role/index"
	common.Render(c, "system/role.html", gin.H{
		"list":     true,
		"roles":    roles,
		"menus":    menus,
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
	})
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
		rc.LogAction(c, "新增角色成功")
		rc.JSONOKMsg(c, common.NoticeAdd)
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
		rc.LogAction(c, "修改角色成功")
		rc.JSONOKMsg(c, common.NoticeModify)
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
	rc.LogAction(c, "刪除角色成功")
	rc.JSONOKMsg(c, common.NoticeDelete)
}
