package content

import (
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// LinkController - 友情鏈接控制器
// 對應 PHP: apps/admin/controller/LinkController.php
type LinkController struct {
	common.BaseController
}

// getGids 返回可用的分組 ID 列表供模板下拉選擇
func (lk *LinkController) getGids(c *gin.Context) []int {
	var gids []int
	model.DB.WithContext(c.Request.Context()).Model(&model.Link{}).Distinct("gid").Pluck("gid", &gids)
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

// Index - 友情鏈接列表
func (lk *LinkController) Index(c *gin.Context) {
	page, pageSize, offset := lk.Paginate(c)

	var links []model.Link
	query := model.DB.WithContext(c.Request.Context()).Model(&model.Link{})
	var total int64
	query.Count(&total)
	query.Order("sorting ASC, id ASC").Offset(offset).Limit(pageSize).Find(&links)

	baseURL := "/admin/content/link/index"
	data := gin.H{
		"links":    links,
		"list":     true,
		"gids":     lk.getGids(c),
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
	}
	common.Render(c, "content/link.html", data)
}

// Add - 新增友情鏈接
func (lk *LinkController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "1"))
		if gid == 0 {
			var maxGID int
			model.DB.WithContext(c.Request.Context()).Model(&model.Link{}).Select("COALESCE(MAX(gid),0)").Scan(&maxGID)
			gid = maxGID + 1
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		// acode 由 AcodePlugin 自動填充，無需手動設置
		if err := model.DB.WithContext(c.Request.Context()).Create(&model.Link{
			GID:        gid,
			Name:       c.PostForm("name"),
			Link:       c.PostForm("link"),
			Logo:       c.PostForm("logo"),
			Sorting:    sorting,
			CreateUser: lk.GetAdminUsername(c),
			UpdateUser: lk.GetAdminUsername(c),
			CreateTime: now,
			UpdateTime: now,
		}).Error; err != nil {
			lk.JSONFail(c, "新增失敗："+err.Error())
			return
		}
		lk.LogAction(c, "新增友情鏈接成功")
		lk.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	common.Render(c, "content/link.html", gin.H{"list": true, "gids": lk.getGids(c)})
}

// Mod - 修改友情鏈接（支援批量排序和單條編輯）
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

	// 批量排序
	if lk.IsBatchSort(c) {
		lk.BatchSort(c, &model.Link{}, "sorting", 255)
		return
	}

	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		gid, _ := strconv.Atoi(c.DefaultPostForm("gid", "1"))
		now := time.Now().Format("2006-01-02 15:04:05")
		if err := model.DB.WithContext(c.Request.Context()).Model(&model.Link{}).Where("id = ?", id).Updates(map[string]interface{}{
			"gid":         gid,
			"name":        c.PostForm("name"),
			"link":        c.PostForm("link"),
			"logo":        c.PostForm("logo"),
			"sorting":     sorting,
			"update_user": lk.GetAdminUsername(c),
			"update_time": now,
		}).Error; err != nil {
			lk.JSONFail(c, "修改失敗："+err.Error())
			return
		}
		lk.LogAction(c, "修改友情鏈接成功")
		lk.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// GET: 顯示編輯表單
	var link model.Link
	model.DB.WithContext(c.Request.Context()).First(&link, id)

	data := gin.H{
		"link":   link,
		"mod":    true,
		"gids":   lk.getGids(c),
		"get_id": idStr,
	}
	common.Render(c, "content/link.html", data)
}

// Del - 刪除友情鏈接
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
			if err := model.DB.WithContext(c.Request.Context()).Delete(&model.Link{}, id).Error; err != nil {
				lk.JSONFail(c, "刪除失敗："+err.Error())
				return
			}
		}
	}
	lk.LogAction(c, "刪除友情鏈接成功")
	lk.JSONOKMsg(c, common.NoticeDelete)
}
