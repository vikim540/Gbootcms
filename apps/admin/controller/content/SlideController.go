package content

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SlideController - Slide Controller
// Corresponds to PHP: apps/admin/controller/SlideController.php
type SlideController struct {
	common.BaseController
}

// Index - Slide list
func (sl *SlideController) Index(c *gin.Context) {
	var slides []model.Slide
	model.DB.Order("sorting ASC").Find(&slides)
	common.Render(c, "content/slide.html", gin.H{"slides": slides})
}

// Add - Add new slide
func (sl *SlideController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "1"))
		model.DB.Create(&model.Slide{
			GID:     gid,
			Pic:     c.PostForm("pic"),
			Link:    c.PostForm("link"),
			Title:   c.PostForm("title"),
			Sorting: sorting,
		})
		sl.JSONOKMsg(c, "Added successfully")
		return
	}
	common.Render(c, "content/slide.html", gin.H{"action": "add"})
}

// Mod - Modify slide
func (sl *SlideController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "1"))
		model.DB.Model(&model.Slide{}).Where("id = ?", id).Updates(map[string]interface{}{
			"gid":     gid,
			"pic":     c.PostForm("pic"),
			"link":    c.PostForm("link"),
			"title":   c.PostForm("title"),
			"sorting": sorting,
		})
		sl.JSONOKMsg(c, "Modified successfully")
		return
	}

	var slide model.Slide
	model.DB.First(&slide, id)
	common.Render(c, "content/slide.html", gin.H{"slide": slide, "action": "mod"})
}

// Del - Delete slide
func (sl *SlideController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.Slide{}, idStr)
	sl.JSONOKMsg(c, "Deleted successfully")
}
