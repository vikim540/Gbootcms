package content

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// TagsController - Tags Management Controller
// Corresponds to PHP: apps/admin/controller/TagsController.php
type TagsController struct {
	common.BaseController
}

// Index - Tags list
func (tg *TagsController) Index(c *gin.Context) {
	idStr := c.Query("id")
	if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		var tag model.Tags
		if err := model.DB.First(&tag, id).Error; err == nil {
			common.Render(c, "content/tags.html", gin.H{"more": true, "tags": tag})
			return
		}
	}

	field := c.Query("field")
	keyword := c.Query("keyword")

	var tags []model.Tags
	query := model.DB.Model(&model.Tags{})
	if field != "" && keyword != "" {
		query = query.Where(field+" LIKE ?", "%"+keyword+"%")
	}
	query.Order("sorting ASC, id ASC").Find(&tags)

	common.Render(c, "content/tags.html", gin.H{"list": true, "tags": tags})
}

// Add - Add new tag
func (tg *TagsController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		name := c.PostForm("name")
		link := c.PostForm("link")

		if name == "" {
			tg.JSONFail(c, "Name cannot be empty")
			return
		}

		if link == "" {
			tg.JSONFail(c, "Link cannot be empty")
			return
		}

		model.DB.Create(&model.Tags{
			Name: name,
			Link: link,
		})
		tg.JSONOKMsg(c, "新增成功")
		return
	}
	common.Render(c, "content/tags.html", gin.H{"action": "add"})
}

// Del - Delete tag
func (tg *TagsController) Del(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		tg.JSONFail(c, "Invalid parameter")
		return
	}
	model.DB.Delete(&model.Tags{}, idStr)
	tg.JSONOKMsg(c, "刪除成功")
}

// Mod - Modify tag
func (tg *TagsController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	if idStr == "" {
		tg.JSONFail(c, "Invalid parameter")
		return
	}
	id, _ := strconv.Atoi(idStr)

	field := c.Query("field")
	value := c.Query("value")
	if field != "" {
		model.DB.Model(&model.Tags{}).Where("id = ?", id).Update(field, value)
		tg.JSONOKMsg(c, "修改成功")
		return
	}

	if c.Request.Method == "POST" {
		name := c.PostForm("name")
		link := c.PostForm("link")

		if name == "" {
			tg.JSONFail(c, "Name cannot be empty")
			return
		}

		if link == "" {
			tg.JSONFail(c, "Link cannot be empty")
			return
		}

		model.DB.Model(&model.Tags{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name": name,
			"link": link,
		})
		tg.JSONOKMsg(c, "修改成功")
		return
	}

	var tag model.Tags
	if err := model.DB.First(&tag, id).Error; err != nil {
		tg.JSONFail(c, "Content does not exist")
		return
	}
	common.Render(c, "content/tags.html", gin.H{"mod": true, "tags": tag})
}
