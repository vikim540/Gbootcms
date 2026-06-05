package content

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// LinkController - Friend Link Controller
// Corresponds to PHP: apps/admin/controller/LinkController.php
type LinkController struct {
	common.BaseController
}

// Index - Friend link list
func (lk *LinkController) Index(c *gin.Context) {
	var links []model.Link
	model.DB.Order("sorting ASC").Find(&links)
	common.Render(c, "content/link.html", gin.H{"links": links})
}

// Add - Add new link
func (lk *LinkController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "1"))
		model.DB.Create(&model.Link{
			GID:     gid,
			Logo:    c.PostForm("logo"),
			Link:    c.PostForm("link"),
			Title:   c.PostForm("title"),
			Sorting: sorting,
		})
		lk.JSONOKMsg(c, "Added successfully")
		return
	}
	common.Render(c, "content/link.html", gin.H{"action": "add"})
}

// Mod - Modify link
func (lk *LinkController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "1"))
		model.DB.Model(&model.Link{}).Where("id = ?", id).Updates(map[string]interface{}{
			"gid":     gid,
			"logo":    c.PostForm("logo"),
			"link":    c.PostForm("link"),
			"title":   c.PostForm("title"),
			"sorting": sorting,
		})
		lk.JSONOKMsg(c, "Modified successfully")
		return
	}

	var link model.Link
	model.DB.First(&link, id)
	common.Render(c, "content/link.html", gin.H{"link": link, "action": "mod"})
}

// Del - Delete link
func (lk *LinkController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.Link{}, idStr)
	lk.JSONOKMsg(c, "Deleted successfully")
}
