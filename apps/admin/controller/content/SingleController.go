package content

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"

	"github.com/gin-gonic/gin"
)

// SingleController - Single Page Controller
// Corresponds to PHP: apps/admin/controller/SingleController.php
type SingleController struct {
	common.BaseController
}

// Index - Single page list
func (sg *SingleController) Index(c *gin.Context) {
	var sorts []model.ContentSort
	model.DB.Where("type = 2 AND status = 1").Order("sorting ASC").Find(&sorts)

	var contents []model.Content
	model.DB.Where("scode IN (SELECT scode FROM ay_content_sort WHERE type = 2)").Order("date DESC").Find(&contents)

	common.Render(c, "content/single.html", gin.H{
		"sorts":    sorts,
		"contents": contents,
	})
}

// Mod - Modify single page
func (sg *SingleController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := parseInt(idStr)

	if c.Request.Method == "POST" {
		model.DB.Model(&model.Content{}).Where("id = ?", id).Updates(map[string]interface{}{
			"title":       c.PostForm("title"),
			"content":     c.PostForm("content"),
			"keywords":    c.PostForm("keywords"),
			"description": c.PostForm("description"),
		})
		sg.JSONOKMsg(c, "Modified successfully")
		return
	}

	var doc model.Content
	model.DB.First(&doc, id)
	common.Render(c, "content/single.html", gin.H{
		"content": doc,
		"action":  "mod",
	})
}

// Del - Delete single page
func (sg *SingleController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.Content{}, idStr)
	sg.JSONOKMsg(c, "Deleted successfully")
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
