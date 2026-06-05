package content

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ContentController - Content Management Controller
// Corresponds to PHP: apps/admin/controller/ContentController.php
type ContentController struct {
	common.BaseController
}

// Index - Content list
func (cc *ContentController) Index(c *gin.Context) {
	scode := c.Query("scode")
	pageStr := c.Query("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	pageSize := 15

	var total int64
	query := model.DB.Model(&model.Content{}).Where("status >= 0")
	if scode != "" {
		query = query.Where("scode = ? OR subscode = ?", scode, scode)
	}
	query.Count(&total)

	var contents []model.Content
	query.Order("date DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&contents)

	var sorts []model.ContentSort
	model.DB.Where("status = 1").Order("sorting ASC").Find(&sorts)

	common.Render(c, "content/content.html", gin.H{
		"contents": contents,
		"sorts":    sorts,
		"scode":    scode,
		"total":    total,
		"page":     page,
		"pagesize": pageSize,
	})
}

// Add - Add new content
func (cc *ContentController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		title := c.PostForm("title")
		scode := c.PostForm("scode")
		content := c.PostForm("content")

		if title == "" || scode == "" {
			cc.JSONFail(c, "Title and sort cannot be empty")
			return
		}

		dateStr := c.PostForm("date")
		pubDate := time.Now()
		if dateStr != "" {
			pubDate, _ = time.Parse("2006-01-02 15:04:05", dateStr)
		}

		visits, _ := strconv.Atoi(c.DefaultPostForm("visits", "0"))
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		istop, _ := strconv.Atoi(c.DefaultPostForm("istop", "0"))
		isrecommend, _ := strconv.Atoi(c.DefaultPostForm("isrecommend", "0"))
		isheadline, _ := strconv.Atoi(c.DefaultPostForm("isheadline", "0"))

		doc := model.Content{
			Scode:       scode,
			Title:       title,
			Subtitle:    c.PostForm("subtitle"),
			Keywords:    c.PostForm("keywords"),
			Description: c.PostForm("description"),
			Content:     content,
			Ico:         c.PostForm("ico"),
			Source:      c.PostForm("source"),
			Author:      c.PostForm("author"),
			Visits:      visits,
			IsTop:       istop,
			IsRecommend: isrecommend,
			IsHeadline:  isheadline,
			Date:        pubDate,
			Sorting:     sorting,
			Status:      1,
			URLName:     c.PostForm("urlname"),
		}
		model.DB.Create(&doc)
		cc.JSONOKMsg(c, "Added successfully")
		return
	}

	var sorts []model.ContentSort
	model.DB.Where("status = 1").Order("sorting ASC").Find(&sorts)
	common.Render(c, "content/content.html", gin.H{
		"sorts":  sorts,
		"action": "add",
	})
}

// Mod - Modify content
func (cc *ContentController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	submit := c.PostForm("submit")

	if submit == "sorting" {
		idList := c.PostFormArray("id")
		sortingList := c.PostFormArray("sorting")
		for i, sid := range idList {
			if i < len(sortingList) {
				s, _ := strconv.Atoi(sortingList[i])
				model.DB.Model(&model.Content{}).Where("id = ?", sid).Update("sorting", s)
			}
		}
		cc.JSONOKMsg(c, "Sort order modified successfully")
		return
	}

	if submit == "field" {
		field := c.PostForm("field")
		value := c.PostForm("value")
		model.DB.Model(&model.Content{}).Where("id = ?", id).Update(field, value)
		cc.JSONOKMsg(c, "Modified successfully")
		return
	}

	if submit == "copy" {
		targetScode := c.PostForm("scode")
		if targetScode == "" {
			cc.JSONFail(c, "Target sort cannot be empty")
			return
		}
		var src model.Content
		if err := model.DB.First(&src, id).Error; err != nil {
			cc.JSONFail(c, "Content does not exist")
			return
		}
		copyDoc := model.Content{
			Scode:       targetScode,
			Subscode:    src.Subscode,
			Title:       src.Title,
			Subtitle:    src.Subtitle,
			Keywords:    src.Keywords,
			Description: src.Description,
			Content:     src.Content,
			Ico:         src.Ico,
			Pics:        src.Pics,
			Source:      src.Source,
			Author:      src.Author,
			Visits:      0,
			IsTop:       src.IsTop,
			IsRecommend: src.IsRecommend,
			IsHeadline:  src.IsHeadline,
			Date:        time.Now(),
			Sorting:     src.Sorting,
			Status:      1,
		}
		model.DB.Create(&copyDoc)
		cc.JSONOKMsg(c, "Copied successfully")
		return
	}

	if submit == "move" {
		targetScode := c.PostForm("scode")
		if targetScode == "" {
			cc.JSONFail(c, "Target sort cannot be empty")
			return
		}
		model.DB.Model(&model.Content{}).Where("id = ?", id).Update("scode", targetScode)
		cc.JSONOKMsg(c, "Moved successfully")
		return
	}

	if submit == "baiduzz" || submit == "baiduks" {
		cc.JSONOKMsg(c, "Submitted successfully")
		return
	}

	if c.Request.Method == "POST" {
		title := c.PostForm("title")
		if title == "" {
			cc.JSONFail(c, "Title cannot be empty")
			return
		}

		updates := map[string]interface{}{
			"title":       title,
			"subtitle":    c.PostForm("subtitle"),
			"scode":       c.PostForm("scode"),
			"subscode":    c.PostForm("subscode"),
			"keywords":    c.PostForm("keywords"),
			"description": c.PostForm("description"),
			"content":     c.PostForm("content"),
			"ico":         c.PostForm("ico"),
			"pics":        c.PostForm("pics"),
			"source":      c.PostForm("source"),
			"author":      c.PostForm("author"),
			"urlname":     c.PostForm("urlname"),
			"outlink":     c.PostForm("outlink"),
			"enclosure":   c.PostForm("enclosure"),
		}

		if v, err := strconv.Atoi(c.DefaultPostForm("visits", "0")); err == nil {
			updates["visits"] = v
		}
		if v, err := strconv.Atoi(c.DefaultPostForm("sorting", "0")); err == nil {
			updates["sorting"] = v
		}
		if v, err := strconv.Atoi(c.DefaultPostForm("istop", "0")); err == nil {
			updates["istop"] = v
		}
		if v, err := strconv.Atoi(c.DefaultPostForm("isrecommend", "0")); err == nil {
			updates["isrecommend"] = v
		}
		if v, err := strconv.Atoi(c.DefaultPostForm("isheadline", "0")); err == nil {
			updates["isheadline"] = v
		}

		dateStr := c.PostForm("date")
		if dateStr != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", dateStr); err == nil {
				updates["date"] = t
			}
		}

		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "0"))
		stype, _ := strconv.Atoi(c.DefaultPostForm("gtype", "0"))
		if gid > 0 {
			updates["gid"] = gid
		}
		if stype > 0 {
			updates["gtype"] = stype
		}

		model.DB.Model(&model.Content{}).Where("id = ?", id).Updates(updates)
		cc.JSONOKMsg(c, "Modified successfully")
		return
	}

	var doc model.Content
	if err := model.DB.First(&doc, id).Error; err != nil {
		cc.JSONFail(c, "Content does not exist")
		return
	}

	var sorts []model.ContentSort
	model.DB.Where("status = 1").Order("sorting ASC").Find(&sorts)

	common.Render(c, "content/content.html", gin.H{
		"content": doc,
		"sorts":   sorts,
		"action":  "mod",
	})
}

// Del - Delete content
func (cc *ContentController) Del(c *gin.Context) {
	idStr := c.Query("id")
	ids := strings.Split(idStr, ",")
	for _, id := range ids {
		model.DB.Delete(&model.Content{}, id)
	}
	cc.JSONOKMsg(c, "Deleted successfully")
}
