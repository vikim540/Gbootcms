package content

import (
	"pbootcms-go/apps/admin/helper"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"

	"github.com/gin-gonic/gin"
)

// SingleController - Single Page Controller
type SingleController struct {
	common.BaseController
}

// Index - Single page list
func (sg *SingleController) Index(c *gin.Context) {
	mcode := c.Query("mcode")
	if mcode == "" {
		mcode = c.Param("mcode")
	}

	var sorts []model.ContentSort
	model.DB.Where("type = 2 AND status = 1").Order("sorting ASC").Find(&sorts)

	var contents []model.Content
	model.DB.Where("scode IN (SELECT scode FROM ay_content_sort WHERE type = 2)").Order("date DESC").Find(&contents)

	common.Render(c, "content/single.html", gin.H{
		"sorts":      sorts,
		"contents":   helper.AddSortName(contents, sorts),
		"list":       true,
		"model_name": "单页",
	})
}

// Mod - Modify single page
func (sg *SingleController) Mod(c *gin.Context) {
	// Parse wildcard action param: /id/123 or /123/field/status/value/0
	params := helper.ParseWildcardAction(c.Param("action"))

	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	id := helper.ParseInt(idStr)

	mcode := params["mcode"]
	if mcode == "" {
		mcode = c.Query("mcode")
	}

	// Handle single field update via URL path or query params
	field := params["field"]
	if field == "" {
		field = c.Query("field")
	}
	value := params["value"]
	if value == "" {
		value = c.Query("value")
	}
	if field != "" && id > 0 {
		allowedFields := map[string]bool{"status": true, "istop": true, "isrecommend": true}
		if !allowedFields[field] {
			sg.JSONFail(c, "field not allowed: "+field)
			return
		}
		model.DB.Model(&model.Content{}).Where("id = ?", id).Update(field, value)
		sg.JSONOKMsg(c, "Modified successfully")
		return
	}

	if c.Request.Method == "POST" {
		updates := map[string]interface{}{
			"title":       c.PostForm("title"),
			"content":     c.PostForm("content"),
			"keywords":    c.PostForm("keywords"),
			"description": c.PostForm("description"),
			"subtitle":    c.PostForm("subtitle"),
			"author":      c.PostForm("author"),
			"source":      c.PostForm("source"),
			"ico":         c.PostForm("ico"),
			"pics":        c.PostForm("pics"),
			"tags":        c.PostForm("tags"),
			"titlecolor":  c.PostForm("titlecolor"),
			"enclosure":   c.PostForm("enclosure"),
		}
		if v := c.PostForm("date"); v != "" {
			updates["date"] = v
		}
		if v := c.PostForm("status"); v != "" {
			updates["status"] = helper.ParseInt(v)
		}
		model.DB.Model(&model.Content{}).Where("id = ?", id).Updates(updates)
		sg.JSONOKMsg(c, "Modified successfully")
		return
	}

	var doc model.Content
	model.DB.First(&doc, id)

	// Get mcode from content's sort if not in query
	if mcode == "" {
		var sort model.ContentSort
		if model.DB.Where("scode = ?", doc.Scode).First(&sort).Error == nil {
			mcode = sort.Mcode
		}
	}

	common.Render(c, "content/single.html", gin.H{
		"content":    doc,
		"mod":        true,
		"model_name": "单页",
		"extfield":   helper.GetExtFieldsByMcode(mcode),
	})
}

// Del - Delete single page
func (sg *SingleController) Del(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		sg.JSONFail(c, "ID required")
		return
	}
	model.DB.Delete(&model.Content{}, idStr)
	sg.JSONOKMsg(c, "Deleted successfully")
}
