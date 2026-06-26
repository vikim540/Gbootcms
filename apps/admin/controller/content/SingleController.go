package content

import (
	"pbootcms-go/apps/admin/helper"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strings"
	"time"

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
	if mcode == "" {
		mcode = "1" // 默認單頁模型 mcode (對應 ay_model 中 type=1 的記錄)
	}

	// 搜索參數
	field := c.Query("field")
	keyword := c.Query("keyword")

	// 通過 mcode 查詢屬於單頁模型的欄目
	var sorts []model.ContentSort
	sortQuery := model.DB.Where("mcode = ? AND status = 1", mcode)
	if keyword != "" && field == "b.name" {
		sortQuery = sortQuery.Where("name LIKE ?", "%"+keyword+"%")
	}
	sortQuery.Order("sorting ASC").Find(&sorts)

	// 查詢每個欄目下的最新一條內容(單頁每個欄目只有一條)
	var contents []model.Content
	if len(sorts) > 0 {
		var scodes []string
		for _, s := range sorts {
			scodes = append(scodes, s.Scode)
		}
		contentQuery := model.DB.Where("scode IN (?)", scodes).
			Where("id IN (SELECT MAX(id) FROM ay_content WHERE scode IN (?) GROUP BY scode)", scodes)
		if keyword != "" && field == "a.title" {
			contentQuery = contentQuery.Where("title LIKE ?", "%"+keyword+"%")
		}
		contentQuery.Order("scode ASC").Find(&contents)
	}

	common.Render(c, "content/single.html", gin.H{
		"sorts":      sorts,
		"contents":   helper.AddSortName(contents, sorts),
		"list":       true,
		"mcode":      mcode,
		"keyword":    keyword,
		"field":      field,
		"model_name": "欄目",
	})
}

// IndexCatchAll handles /content/single/index/*action paths like /mcode/1
// that come from the NoRoute rewriter when users visit /admin/single/index/mcode/1
func (sg *SingleController) IndexCatchAll(c *gin.Context) {
	action := strings.TrimPrefix(c.Param("action"), "/")
	if strings.HasPrefix(action, "mcode/") {
		mcode := strings.TrimPrefix(action, "mcode/")
		c.Request.URL.RawQuery = "mcode=" + mcode
	}
	sg.Index(c)
}

// Mod - Modify single page
func (sg *SingleController) Mod(c *gin.Context) {
	// Parse wildcard action param: /id/123 or /123/field/status/value/0
	params := helper.ParseWildcardAction(c.Param("action"))
	// Resolve "get(xxx)" URL path literals against actual GET/POST params
	params = helper.ResolveActionGetParams(params, c)

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
			if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
				updates["date"] = t
			}
		}
		if v := c.PostForm("status"); v != "" {
			updates["status"] = helper.ParseInt(v)
		}
		if v := c.PostForm("picstitle"); v != "" {
			updates["picstitle"] = v
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

	// 預格式化日期供模板使用（pongo2 無 date 過濾器）
	dateStr := ""
	if !doc.Date.IsZero() {
		dateStr = doc.Date.Format("2006-01-02 15:04:05")
	}

	common.Render(c, "content/single.html", gin.H{
		"content":    doc,
		"mod":        true,
		"model_name": "单页",
		"mcode":      mcode,
		"extfield":   helper.GetExtFieldsByMcode(mcode),
		"picsList":   splitPics(doc.Pics),
		"dateStr":    dateStr,
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

// splitPics splits comma-separated pics string into a slice for template rendering.
func splitPics(pics string) []string {
	if pics == "" {
		return nil
	}
	parts := strings.Split(pics, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
