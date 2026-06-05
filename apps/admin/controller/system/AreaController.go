package system

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AreaController - Area Management Controller
// Corresponds to PHP: apps/admin/controller/AreaController.php
type AreaController struct {
	common.BaseController
}

// Index - Area list
func (ar *AreaController) Index(c *gin.Context) {
	var areas []model.Area
	model.DB.Order("sorting ASC").Find(&areas)
	common.Render(c, "system/area.html", gin.H{"areas": areas})
}

// Add - Add new area
func (ar *AreaController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		model.DB.Create(&model.Area{
			Code:    c.PostForm("code"),
			Name:    c.PostForm("name"),
			Sorting: sorting,
			Status:  1,
		})
		ar.JSONOKMsg(c, "Added successfully")
		return
	}
	common.Render(c, "system/area.html", gin.H{"action": "add"})
}

// Mod - Modify area
func (ar *AreaController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		model.DB.Model(&model.Area{}).Where("id = ?", id).Updates(map[string]interface{}{
			"code":    c.PostForm("code"),
			"name":    c.PostForm("name"),
			"sorting": sorting,
		})
		ar.JSONOKMsg(c, "Modified successfully")
		return
	}

	var area model.Area
	model.DB.First(&area, id)
	common.Render(c, "system/area.html", gin.H{"area": area, "action": "mod"})
}

// Del - Delete area
func (ar *AreaController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.Area{}, idStr)
	ar.JSONOKMsg(c, "Deleted successfully")
}
