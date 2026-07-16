package content

import (
	"fmt"
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
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
func (sl *SlideController) getGids(c *gin.Context) []int {
	var gids []int
	model.DB.WithContext(c.Request.Context()).Model(&model.Slide{}).Distinct("gid").Pluck("gid", &gids)
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

// getGroups 返回所有輪播圖分組，包含名稱和輪播圖數量
// 對於有 slide 記錄但無分組名稱的 gid，自動生成 "分組N" 臨時名稱
func (sl *SlideController) getGroups(c *gin.Context) []model.SlideGroup {
	// 1. 從 DB 載入已命名的分組
	var groups []model.SlideGroup
	model.DB.WithContext(c.Request.Context()).Order("sorting ASC, gid ASC").Find(&groups)

	// 2. 取得所有 slide 中的 distinct gid
	var slideGIDs []int
	model.DB.WithContext(c.Request.Context()).Model(&model.Slide{}).Distinct("gid").Pluck("gid", &slideGIDs)

	// 3. 為沒有分組名稱的 gid 生成臨時名稱
	groupGIDSet := make(map[int]bool)
	for _, g := range groups {
		groupGIDSet[g.GID] = true
	}
	for _, gid := range slideGIDs {
		if !groupGIDSet[gid] {
			groups = append(groups, model.SlideGroup{
				GID:  gid,
				Name: fmt.Sprintf("分組%d", gid),
			})
		}
	}

	// 4. 統計每個 gid 下的輪播圖數量（用 Find 而非 Scan，確保 AcodePlugin 正確注入 acode 過濾）
	type gidCount struct {
		GID   int   `gorm:"column:gid"`
		Count int64 `gorm:"column:count"`
	}
	var counts []gidCount
	model.DB.WithContext(c.Request.Context()).Model(&model.Slide{}).Select("gid, COUNT(*) as count").Group("gid").Find(&counts)
	countMap := make(map[int]int64)
	for _, gc := range counts {
		countMap[gc.GID] = gc.Count
	}

	// 5. 填充 Count 到每個分組
	for i := range groups {
		groups[i].Count = countMap[groups[i].GID]
	}

	return groups
}

// Index - Slide list (shows list + add form via tabs)
func (sl *SlideController) Index(c *gin.Context) {
	page, pageSize, offset := sl.Paginate(c)

	// 支援 ?gid=X 篩選
	filterGID := c.Query("gid")

	var slides []model.Slide
	query := model.DB.WithContext(c.Request.Context()).Model(&model.Slide{})
	if filterGID != "" {
		if gid, err := strconv.Atoi(filterGID); err == nil && gid > 0 {
			query = query.Where("gid = ?", gid)
		}
	}
	var total int64
	query.Count(&total)

	// 分頁 baseURL 帶篩選參數
	baseURL := "/admin/content/slide/index"
	if filterGID != "" {
		baseURL += "?gid=" + filterGID
	}

	query.Order("gid ASC, sorting ASC, id ASC").Offset(offset).Limit(pageSize).Find(&slides)

	// 載入分組列表（含名稱和數量）
	groups := sl.getGroups(c)

	// 建立 gid → name 映射，填充到每個 slide
	groupMap := make(map[int]string)
	for _, g := range groups {
		groupMap[g.GID] = g.Name
	}
	for i := range slides {
		if name, ok := groupMap[slides[i].GID]; ok {
			slides[i].GroupName = name
		} else {
			slides[i].GroupName = fmt.Sprintf("分組%d", slides[i].GID)
		}
	}

	data := gin.H{
		"slides":    slides,
		"list":      true,
		"C":         "content/slide",
		"groups":    groups,
		"filterGID": filterGID,
		"gids":      sl.getGids(c),
		"pagebar":   helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize":  pageSize,
	}
	common.Render(c, "content/slide.html", data)
}

// GroupManage 輪播圖分組管理 AJAX 端點
// action: list(GET/POST) | add(POST) | edit(POST) | del(POST)
func (sl *SlideController) GroupManage(c *gin.Context) {
	action := c.DefaultPostForm("action", "list")
	if action == "list" {
		// GET 也視為 list
		if c.Request.Method == "GET" {
			action = "list"
		}
	}

	switch action {
	case "list":
		groups := sl.getGroups(c)
		sl.JSONOK(c, groups)

	case "add":
		name := c.PostForm("name")
		if name == "" {
			sl.JSONFail(c, "分組名稱不能為空")
			return
		}
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))

		// 自動遞增 gid：取當前最大 gid + 1
		var maxGID int
		model.DB.WithContext(c.Request.Context()).Model(&model.Slide{}).Select("COALESCE(MAX(gid),0)").Scan(&maxGID)
		gid := maxGID + 1

		now := time.Now().Format("2006-01-02 15:04:05")
		if err := model.DB.WithContext(c.Request.Context()).Create(&model.SlideGroup{
			GID:        gid,
			Name:       name,
			Sorting:    sorting,
			CreateTime: now,
			UpdateTime: now,
		}).Error; err != nil {
			sl.JSONFail(c, "新增分組失敗："+err.Error())
			return
		}
		sl.LogAction(c, "新增輪播圖分組："+name)
		sl.JSONOKMsg(c, "分組新增成功")

	case "edit":
		id, _ := strconv.Atoi(c.PostForm("id"))
		name := c.PostForm("name")
		if name == "" {
			sl.JSONFail(c, "分組名稱不能為空")
			return
		}
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		now := time.Now().Format("2006-01-02 15:04:05")

		if id == 0 {
			// 自動生成的臨時分組（id=0），需用 gid 創建新記錄
			gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "0"))
			if gid == 0 {
				sl.JSONFail(c, "缺少分組GID")
				return
			}
			// 檢查是否已存在同 gid 的分組記錄（防止併發重複創建）
			var existing model.SlideGroup
			result := model.DB.WithContext(c.Request.Context()).Where("gid = ?", gid).First(&existing)
			if result.Error == nil {
				// 已存在記錄，改為更新
				if err := model.DB.WithContext(c.Request.Context()).Model(&model.SlideGroup{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
					"name":        name,
					"sorting":     sorting,
					"update_time": now,
				}).Error; err != nil {
					sl.JSONFail(c, "修改分組失敗："+err.Error())
					return
				}
			} else {
				// 不存在，創建新記錄
				if err := model.DB.WithContext(c.Request.Context()).Create(&model.SlideGroup{
					GID:        gid,
					Name:       name,
					Sorting:    sorting,
					CreateTime: now,
					UpdateTime: now,
				}).Error; err != nil {
					sl.JSONFail(c, "新增分組失敗："+err.Error())
					return
				}
			}
			sl.LogAction(c, "修改輪播圖分組："+name)
			sl.JSONOKMsg(c, "分組修改成功")
			return
		}

		// 正常更新已有記錄
		if err := model.DB.WithContext(c.Request.Context()).Model(&model.SlideGroup{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name":        name,
			"sorting":     sorting,
			"update_time": now,
		}).Error; err != nil {
			sl.JSONFail(c, "修改分組失敗："+err.Error())
			return
		}
		sl.LogAction(c, "修改輪播圖分組："+name)
		sl.JSONOKMsg(c, "分組修改成功")

	case "del":
		id, _ := strconv.Atoi(c.PostForm("id"))
		if id == 0 {
			sl.JSONFail(c, "缺少分組ID")
			return
		}
		if err := model.DB.WithContext(c.Request.Context()).Delete(&model.SlideGroup{}, id).Error; err != nil {
			sl.JSONFail(c, "刪除分組失敗："+err.Error())
			return
		}
		sl.LogAction(c, "刪除輪播圖分組")
		sl.JSONOKMsg(c, "分組刪除成功")

	default:
		sl.JSONFail(c, "未知操作")
	}
}

// Add - Add new slide (POST only; GET is handled by Index tabs)
func (sl *SlideController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "1"))
		if gid == 0 {
			// Auto-increment gid: find max gid and add 1
			var maxGID int
			model.DB.WithContext(c.Request.Context()).Model(&model.Slide{}).Select("COALESCE(MAX(gid),0)").Scan(&maxGID)
			gid = maxGID + 1
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		username := sl.GetAdminUsername(c)
		if err := model.DB.WithContext(c.Request.Context()).Create(&model.Slide{
			GID:        gid,
			Pic:        c.PostForm("pic"),
			PicMobile:  c.PostForm("pic_mobile"),
			Link:       c.PostForm("link"),
			Title:      c.PostForm("title"),
			Subtitle:   c.PostForm("subtitle"),
			ButtonText: c.PostForm("button_text"),
			Sorting:    sorting,
			CreateUser: username,
			UpdateUser: username,
			CreateTime: now,
			UpdateTime: now,
		}).Error; err != nil {
			sl.JSONFail(c, "新增失敗："+err.Error())
			return
		}
		sl.LogAction(c, "新增輪播圖成功")
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
		sl.LogAction(c, "修改輪播圖成功")
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
		username := sl.GetAdminUsername(c)
		if err := model.DB.WithContext(c.Request.Context()).Model(&model.Slide{}).Where("id = ?", id).Updates(map[string]interface{}{
			"gid":         gid,
			"pic":         c.PostForm("pic"),
			"pic_mobile":  c.PostForm("pic_mobile"),
			"link":        c.PostForm("link"),
			"title":       c.PostForm("title"),
			"subtitle":    c.PostForm("subtitle"),
			"button_text": c.PostForm("button_text"),
			"sorting":     sorting,
			"update_user": username,
			"update_time": now,
		}).Error; err != nil {
			sl.JSONFail(c, "修改失敗："+err.Error())
			return
		}
		sl.LogAction(c, "修改輪播圖成功")
		sl.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// GET: show edit form
	var slide model.Slide
	model.DB.WithContext(c.Request.Context()).First(&slide, id)

	// 填充分組名稱
	groups := sl.getGroups(c)
	groupMap := make(map[int]string)
	for _, g := range groups {
		groupMap[g.GID] = g.Name
	}
	if name, ok := groupMap[slide.GID]; ok {
		slide.GroupName = name
	}

	data := gin.H{
		"slide":  slide,
		"mod":    true,
		"C":      "content/slide",
		"groups": groups,
		"gids":   sl.getGids(c),
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
			if err := model.DB.WithContext(c.Request.Context()).Delete(&model.Slide{}, id).Error; err != nil {
				sl.JSONFail(c, "刪除失敗："+err.Error())
				return
			}
		}
	}
	sl.LogAction(c, "刪除輪播圖成功")
	sl.JSONOKMsg(c, common.NoticeDelete)
}
