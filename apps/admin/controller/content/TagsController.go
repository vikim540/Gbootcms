package content

import (
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// TagsController - 文章內鏈管理控制器
// 對應 PHP: apps/admin/controller/TagsController.php
type TagsController struct {
	common.BaseController
}

// tagsSearchFields 搜索白名單，防止 SQL 注入
var tagsSearchFields = map[string]bool{
	"name": true,
	"link": true,
}

// Index - 內鏈列表
func (tg *TagsController) Index(c *gin.Context) {
	page, pageSize, offset := tg.Paginate(c)

	idStr := c.Query("id")
	if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		var tag model.Tags
		if err := model.DB.WithContext(c.Request.Context()).First(&tag, id).Error; err == nil {
			common.Render(c, "content/tags.html", gin.H{"more": true, "tags": tag, "C": "tags"})
			return
		}
	}

	field := c.Query("field")
	keyword := c.Query("keyword")

	var tags []model.Tags
	query := model.DB.WithContext(c.Request.Context()).Model(&model.Tags{})
	if field != "" && keyword != "" && tagsSearchFields[field] {
		query = query.Where(field+" LIKE ?", "%"+keyword+"%")
	}
	var total int64
	query.Count(&total)
	query.Order("sorting ASC, id ASC").Offset(offset).Limit(pageSize).Find(&tags)

	baseURL := "/admin/content/tags/index"
	if field != "" && keyword != "" && tagsSearchFields[field] {
		baseURL += "?field=" + field + "&keyword=" + keyword
	}
	common.Render(c, "content/tags.html", gin.H{
		"list":     true,
		"tags":     tags,
		"C":        "tags",
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
	})
}

// Add - 新增內鏈
func (tg *TagsController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		name := c.PostForm("name")
		link := c.PostForm("link")

		if name == "" {
			tg.LogAction(c, "新增文章內鏈失敗")
			tg.JSONFail(c, "名稱不能為空")
			return
		}

		if link == "" {
			tg.LogAction(c, "新增文章內鏈失敗")
			tg.JSONFail(c, "連結不能為空")
			return
		}

		now := time.Now().Format("2006-01-02 15:04:05")
		username := tg.GetAdminUsername(c)
		if err := model.DB.WithContext(c.Request.Context()).Create(&model.Tags{
			Name:       name,
			Link:       link,
			CreateUser: username,
			UpdateUser: username,
			CreateTime: now,
			UpdateTime: now,
		}).Error; err != nil {
			tg.LogAction(c, "新增文章內鏈失敗")
			tg.JSONFail(c, "新增失敗："+err.Error())
			return
		}
		tg.LogAction(c, "新增文章內鏈成功")
		tg.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	common.Render(c, "content/tags.html", gin.H{"action": "add", "C": "tags"})
}

// Del - 刪除內鏈
func (tg *TagsController) Del(c *gin.Context) {
	// 支援 *action 通配符路徑: /del/id/123
	params := helper.ParseWildcardAction(c.Param("action"))
	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	if idStr == "" {
		idStr = c.PostForm("id")
	}
	if idStr == "" {
		tg.LogAction(c, "刪除文章內鏈失敗")
		tg.JSONFail(c, "參數錯誤")
		return
	}
	if err := model.DB.WithContext(c.Request.Context()).Delete(&model.Tags{}, idStr).Error; err != nil {
		tg.LogAction(c, "刪除文章內鏈失敗")
		tg.JSONFail(c, "刪除失敗："+err.Error())
		return
	}
	tg.LogAction(c, "刪除文章內鏈成功")
	tg.JSONOKMsg(c, common.NoticeDelete)
}

// Mod - 修改內鏈
func (tg *TagsController) Mod(c *gin.Context) {
	// 解析 wildcard action 參數：/id/123 或 /123
	params := helper.ParseWildcardAction(c.Param("action"))
	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	if idStr == "" {
		idStr = c.Param("id")
	}
	if idStr == "" {
		tg.JSONFail(c, "參數錯誤")
		return
	}
	id, _ := strconv.Atoi(idStr)

	// 單欄位切換：只從 wildcard action 路徑參數讀取
	// 不從 c.Query 讀取，避免搜索參數 field=link 被誤認為單欄位更新請求
	field := params["field"]
	value := params["value"]
	if field != "" && value != "" {
		// 白名單驗證：Tags 只有 sorting 欄位允許單欄位更新
		if field == "sorting" {
			model.DB.WithContext(c.Request.Context()).Model(&model.Tags{}).Where("id = ?", id).Update(field, value)
			tg.LogAction(c, "修改文章內鏈成功")
			tg.JSONOKMsg(c, common.NoticeModify)
			return
		}
	}

	if c.Request.Method == "POST" {
		name := c.PostForm("name")
		link := c.PostForm("link")

		if name == "" {
			tg.JSONFail(c, "名稱不能為空")
			return
		}

		now := time.Now().Format("2006-01-02 15:04:05")
		// 允許 link 為空（不是所有內鏈都需要連結）
		updates := map[string]interface{}{
			"name":       name,
			"link":       link,
			"update_user": tg.GetAdminUsername(c),
			"update_time": now,
		}
		if err := model.DB.WithContext(c.Request.Context()).Model(&model.Tags{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			tg.JSONFail(c, "修改失敗："+err.Error())
			return
		}
		tg.LogAction(c, "修改文章內鏈成功")
		tg.JSONOKMsg(c, common.NoticeModify)
		return
	}

	var tag model.Tags
	if err := model.DB.WithContext(c.Request.Context()).First(&tag, id).Error; err != nil {
		tg.JSONFail(c, "內容不存在")
		return
	}
	common.Render(c, "content/tags.html", gin.H{"mod": true, "tags": tag, "C": "tags"})
}
