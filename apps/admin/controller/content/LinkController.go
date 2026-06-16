package content

import (
	"pbootcms-go/apps/admin/helper"
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

// getGids returns available link group IDs for the template dropdown.
func (lk *LinkController) getGids() []int {
	var gids []int
	model.DB.Model(&model.Link{}).Distinct("gid").Pluck("gid", &gids)
	found := false
	for _, g := range gids {
		if g == 1 {
			found = true
			break
		}
	}
	if !found {
		gids = append([]int{1}, gids...)
	}
	return gids
}

// Index - Friend link list
func (lk *LinkController) Index(c *gin.Context) {
	var links []model.Link
	model.DB.Order("sorting ASC, id ASC").Find(&links)

	data := gin.H{
		"links": links,
		"list":  true,
		"gids":  lk.getGids(),
	}
	common.Render(c, "content/link.html", data)
}

// Add - Add new link
func (lk *LinkController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "1"))
		if gid == 0 {
			var maxGID int
			model.DB.Model(&model.Link{}).Select("COALESCE(MAX(gid),0)").Scan(&maxGID)
			gid = maxGID + 1
		}
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
	common.Render(c, "content/link.html", gin.H{"list": true, "gids": lk.getGids()})
}

// Mod - Modify link
func (lk *LinkController) Mod(c *gin.Context) {
	action := c.Param("action")
	params := helper.ParseWildcardAction(action)

	idStr := params["id"]
	if idStr == "" {
		idStr = c.Param("id")
	}
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	// Handle status toggle: /mod/id/123/field/status/value/0
	if field, ok := params["field"]; ok && field == "status" {
		value := params["value"]
		model.DB.Model(&model.Link{}).Where("id = ?", id).Update("status", value)
		lk.JSONOKMsg(c, "OK")
		return
	}

	// Handle batch sorting (POST with submit=sorting)
	if lk.IsBatchSort(c) {
		lk.BatchSort(c, &model.Link{}, "sorting", 255)
		return
	}

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

	// GET: show edit form
	var link model.Link
	model.DB.First(&link, id)

	data := gin.H{
		"link":   link,
		"mod":    true,
		"gids":   lk.getGids(),
		"get_id": idStr,
	}
	common.Render(c, "content/link.html", data)
}

// Del - Delete link
func (lk *LinkController) Del(c *gin.Context) {
	action := c.Param("action")
	params := helper.ParseWildcardAction(action)
	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		if id > 0 {
			model.DB.Delete(&model.Link{}, id)
		}
	}
	lk.JSONOKMsg(c, "Deleted successfully")
}
