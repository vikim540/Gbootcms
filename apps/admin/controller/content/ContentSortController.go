package content

import (
	"fmt"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ContentSortController - Content Sort Management Controller
// Corresponds to PHP: apps/admin/controller/content/ContentSortController.php
type ContentSortController struct {
	common.BaseController
}

// Index - Sort list
func (csc *ContentSortController) Index(c *gin.Context) {
	var sorts []model.ContentSort
	model.DB.Order("sorting ASC, id ASC").Find(&sorts)
	common.Render(c, "content/contentsort.html", gin.H{"sorts": sorts})
}

// Add - Add new sort
func (csc *ContentSortController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		multiplename := c.PostForm("multiplename")
		if multiplename != "" {
			names := strings.Split(multiplename, ",")
			var lastSort model.ContentSort
			model.DB.Order("id DESC").First(&lastSort)
			lastCodeNum := 0
			fmt.Sscanf(lastSort.Scode, "%d", &lastCodeNum)

			for _, name := range names {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				lastCodeNum++
				newScode := fmt.Sprintf("%d", lastCodeNum)
				model.DB.Create(&model.ContentSort{
					Scode:  newScode,
					Pcode:  c.PostForm("pcode"),
					Name:   name,
					Type:   1,
					Sort:   lastCodeNum,
					Status: 1,
				})
			}
			csc.JSONOKMsg(c, "Batch added successfully")
			return
		}

		name := c.PostForm("name")
		scode := c.PostForm("scode")
		if name == "" || scode == "" {
			csc.JSONFail(c, "Name and code cannot be empty")
			return
		}
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		stype, _ := strconv.Atoi(c.DefaultPostForm("type", "1"))

		sort := model.ContentSort{
			Scode:        scode,
			Pcode:        c.PostForm("pcode"),
			Name:         name,
			Subname:      c.PostForm("subname"),
			Type:         stype,
			ListTpl:      c.PostForm("listtpl"),
			ContentTpl:   c.PostForm("contenttpl"),
			Ico:          c.PostForm("ico"),
			Pic:          c.PostForm("pic"),
			Keywords:     c.PostForm("keywords"),
			Description:  c.PostForm("description"),
			Sort:         sorting,
			URLName:      c.PostForm("urlname"),
			Outlink:      c.PostForm("outlink"),
			Status:       1,
		}
		model.DB.Create(&sort)

		if stype == 1 && c.PostForm("outlink") == "" {
			model.DB.Create(&model.Content{
				Scode:  scode,
				Title:  name,
				Status: 1,
				Date:   time.Now(),
			})
		}
		csc.JSONOKMsg(c, "Added successfully")
		return
	}

	var sorts []model.ContentSort
	model.DB.Order("sorting ASC").Find(&sorts)
	common.Render(c, "content/contentsort.html", gin.H{
		"sorts":  sorts,
		"action": "add",
	})
}

// Mod - Modify sort
func (csc *ContentSortController) Mod(c *gin.Context) {
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
				model.DB.Model(&model.ContentSort{}).Where("id = ?", sid).Update("sorting", s)
			}
		}
		csc.JSONOKMsg(c, "Sort order modified successfully")
		return
	}

	field := c.Query("field")
	value := c.Query("value")
	if field != "" {
		model.DB.Model(&model.ContentSort{}).Where("id = ?", id).Update(field, value)
		csc.JSONOKMsg(c, "Modified successfully")
		return
	}

	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		stype, _ := strconv.Atoi(c.DefaultPostForm("type", "1"))

		model.DB.Model(&model.ContentSort{}).Where("id = ?", id).Updates(map[string]interface{}{
			"scode":       c.PostForm("scode"),
			"pcode":       c.PostForm("pcode"),
			"name":        c.PostForm("name"),
			"subname":     c.PostForm("subname"),
			"type":        stype,
			"listtpl":     c.PostForm("listtpl"),
			"contenttpl":  c.PostForm("contenttpl"),
			"ico":         c.PostForm("ico"),
			"pic":         c.PostForm("pic"),
			"keywords":    c.PostForm("keywords"),
			"description": c.PostForm("description"),
			"sorting":     sorting,
			"urlname":     c.PostForm("urlname"),
			"outlink":     c.PostForm("outlink"),
		})

		if stype == 1 {
			var existing model.Content
			result := model.DB.Where("scode = ?", c.PostForm("scode")).First(&existing)
			if result.Error != nil && c.PostForm("outlink") == "" {
				model.DB.Create(&model.Content{
					Scode:  c.PostForm("scode"),
					Title:  c.PostForm("name"),
					Status: 1,
					Date:   time.Now(),
				})
			}
		}

		csc.JSONOKMsg(c, "Modified successfully")
		return
	}

	var sort model.ContentSort
	model.DB.First(&sort, id)
	var sorts []model.ContentSort
	model.DB.Order("sorting ASC").Find(&sorts)
	common.Render(c, "content/contentsort.html", gin.H{
		"sort":   sort,
		"sorts":  sorts,
		"action": "mod",
	})
}

// Del - Delete sort
func (csc *ContentSortController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.ContentSort{}, idStr)
	csc.JSONOKMsg(c, "Deleted successfully")
}
