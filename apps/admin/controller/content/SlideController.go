package content

import (
	"pbootcms-go/apps/admin/helper"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// SlideController - Slide Controller
// Corresponds to PHP: apps/admin/controller/SlideController.php
type SlideController struct {
	common.BaseController
}

// getGids returns available slide group IDs for the template dropdown.
func (sl *SlideController) getGids() []int {
	// Query distinct gids from existing slides
	var gids []int
	model.DB.Model(&model.Slide{}).Distinct("gid").Pluck("gid", &gids)
	// Always include gid=1 as default
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

// Index - Slide list (shows list + add form via tabs)
func (sl *SlideController) Index(c *gin.Context) {
	var slides []model.Slide
	model.DB.Order("gid ASC, sorting ASC, id ASC").Find(&slides)

	data := gin.H{
		"slides": slides,
		"list":   true,
		"gids":   sl.getGids(),
	}
	common.Render(c, "content/slide.html", data)
}

// Add - Add new slide (POST only; GET is handled by Index tabs)
func (sl *SlideController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "1"))
		if gid == 0 {
			// Auto-increment gid: find max gid and add 1
			var maxGID int
			model.DB.Model(&model.Slide{}).Select("COALESCE(MAX(gid),0)").Scan(&maxGID)
			gid = maxGID + 1
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		model.DB.Create(&model.Slide{
			GID:        gid,
			Pic:        c.PostForm("pic"),
			PicMobile:  c.PostForm("pic_mobile"),
			Link:       c.PostForm("link"),
			Title:      c.PostForm("title"),
			Subtitle:   c.PostForm("subtitle"),
			Sorting:    sorting,
			CreateUser: "admin",
			UpdateUser: "admin",
			CreateTime: now,
			UpdateTime: now,
		})
		sl.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	// GET: redirect to index (the add form is in the Index tabs)
	c.Redirect(302, "/admin/slide/index")
}

// Mod - Modify slide (supports both status toggle and edit form)
func (sl *SlideController) Mod(c *gin.Context) {
	action := c.Param("action")
	params := helper.ParseWildcardAction(action)

	// Also support :id style (legacy route)
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
		// Slides don't have a status field in current model; ignore gracefully
		sl.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// Handle sorting batch update (POST with submit=sorting)
	if sl.IsBatchSort(c) {
		sl.BatchSort(c, &model.Slide{}, "sorting", 255)
		return
	}

	if c.Request.Method == "POST" {
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "1"))
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		now := time.Now().Format("2006-01-02 15:04:05")
		model.DB.Model(&model.Slide{}).Where("id = ?", id).Updates(map[string]interface{}{
			"gid":         gid,
			"pic":         c.PostForm("pic"),
			"pic_mobile":  c.PostForm("pic_mobile"),
			"link":        c.PostForm("link"),
			"title":       c.PostForm("title"),
			"subtitle":    c.PostForm("subtitle"),
			"sorting":     sorting,
			"update_user": "admin",
			"update_time": now,
		})
		sl.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// GET: show edit form
	var slide model.Slide
	model.DB.First(&slide, id)

	data := gin.H{
		"slide":  slide,
		"mod":    true,
		"gids":   sl.getGids(),
		"get_id": idStr,
	}
	common.Render(c, "content/slide.html", data)
}

// Del - Delete slide
func (sl *SlideController) Del(c *gin.Context) {
	action := c.Param("action")
	params := helper.ParseWildcardAction(action)
	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		if id > 0 {
			model.DB.Delete(&model.Slide{}, id)
		}
	}
	sl.JSONOKMsg(c, common.NoticeDelete)
}
