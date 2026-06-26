package content

import (
	"fmt"
	"pbootcms-go/apps/admin/helper"
	"pbootcms-go/apps/admin/model"
	svc "pbootcms-go/apps/admin/service/content"
	"pbootcms-go/apps/common"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ContentController - Content Management Controller
type ContentController struct {
	common.BaseController
	svc svc.ContentService
}

// contentTemplateData returns common template data for content views.
func (cc *ContentController) contentTemplateData(mcode string, sorts []model.ContentSort, contentMap map[string]interface{}) gin.H {
	return gin.H{
		"mcode":          mcode,
		"model_name":     helper.GetModelNameByMcode(mcode),
		"sorts":          sorts,
		"sort_select":    helper.BuildSearchSelectHTML(sorts, mcode),
		"search_select":  helper.BuildSearchSelectHTML(sorts, mcode),
		"subsort_select": helper.BuildSearchSelectHTML(sorts, mcode),
		"extfield":       cc.svc.BuildExtFieldTemplateData(mcode, contentMap),
		"groups":         helper.BuildGroupsData(),
	}
}

// Index - Content list
func (cc *ContentController) Index(c *gin.Context) {
	mcode := c.Query("mcode")
	scode := c.Query("scode")
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 15
	if ps := c.Query("pagesize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}

	contents, total, _ := cc.svc.ListContents(mcode, scode, keyword, page, pageSize)
	sorts, _ := cc.svc.GetAllSorts()

	data := cc.contentTemplateData(mcode, sorts, nil)
	data["contents"] = helper.AddSortName(contents, sorts)
	data["list"] = true
	data["scode"] = scode
	data["keyword"] = keyword
	data["total"] = total
	data["page"] = page
	data["pagesize"] = pageSize

	// Build pagination
	baseURL := fmt.Sprintf("/admin/content/index?mcode=%s", mcode)
	if scode != "" {
		baseURL += "&scode=" + scode
	}
	if keyword != "" {
		baseURL += "&keyword=" + keyword
	}
	data["pagebar"] = helper.BuildPagebarHTML(total, page, pageSize, baseURL)

	common.Render(c, "content/content.html", data)
}

// Add - Add new content
func (cc *ContentController) Add(c *gin.Context) {
	mcode := c.Query("mcode")
	if mcode == "" {
		mcode = c.Param("mcode")
	}

	if c.Request.Method == "POST" {
		mcode = c.PostForm("mcode")
		if mcode == "" {
			mcode = c.Query("mcode")
		}

		dateStr := c.PostForm("date")
		pubDate := time.Now()
		if dateStr != "" {
			pubDate, _ = time.Parse("2006-01-02 15:04:05", dateStr)
		}

		visits, _ := strconv.Atoi(c.DefaultPostForm("visits", "0"))
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		istop, _ := strconv.Atoi(c.DefaultPostForm("istop", "0"))
		isrecommend, _ := strconv.Atoi(c.DefaultPostForm("isrecommend", "0"))
		isheadline, _ := strconv.Atoi(c.DefaultPostForm("isheadline", "0"))

		// Read filename with urlname fallback
		urlname := c.PostForm("filename")
		if urlname == "" {
			urlname = c.PostForm("urlname")
		}

		doc := model.Content{
			Scode:       c.PostForm("scode"),
			Subscode:    c.PostForm("subscode"),
			Title:       c.PostForm("title"),
			Subtitle:    c.PostForm("subtitle"),
			Keywords:    c.PostForm("keywords"),
			Description: c.PostForm("description"),
			Content:     c.PostForm("content"),
			Ico:         c.PostForm("ico"),
			Pics:        c.PostForm("pics"),
			Source:      c.PostForm("source"),
			Author:      c.PostForm("author"),
			Visits:      visits,
			IsTop:       istop,
			IsRecommend: isrecommend,
			IsHeadline:  isheadline,
			Date:        pubDate,
			Sorting:     sorting,
			Status:      helper.ParseInt(c.DefaultPostForm("status", "1")),
			URLName:     urlname,
			Outlink:     c.PostForm("outlink"),
			Tags:        strings.ReplaceAll(c.PostForm("tags"), "，", ","),
			TitleColor:  c.PostForm("titlecolor"),
			Enclosure:   c.PostForm("enclosure"),
			Gid:         c.PostForm("gid"),
			GType:       c.PostForm("gtype"),
			Gnote:       c.PostForm("gnote"),
		}

		// 收集擴展字段數據
		extFields := helper.GetExtFieldsByMcode(mcode)
		extData := cc.svc.CollectExtFieldData(extFields,
			func(key string) string { return c.PostForm(key) },
			func(key string) []string { return c.PostFormArray(key) },
		)

		if err := cc.svc.CreateContent(&doc, extData); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "新增成功")
		return
	}

	sorts, _ := cc.svc.GetAllSorts()
	data := cc.contentTemplateData(mcode, sorts, nil)
	data["list"] = true
	common.Render(c, "content/content.html", data)
}

// Mod - Modify content
func (cc *ContentController) Mod(c *gin.Context) {
	// Parse wildcard action param: /mcode/1/id/123 or /id/123/field/status/value/0 or /123
	params := helper.ParseWildcardAction(c.Param("action"))
	// Resolve "get(xxx)" URL path literals against actual GET/POST params
	params = helper.ResolveActionGetParams(params, c)

	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	mcode := params["mcode"]
	if mcode == "" {
		mcode = c.Query("mcode")
	}
	if mcode == "" {
		mcode = c.PostForm("mcode")
	}

	// 確保 mcode 可用：從內容的欄目反推（MOD GET/POST 通用）
	if mcode == "" && id > 0 {
		mcode = cc.resolveMcodeByContentID(id)
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

	submit := c.PostForm("submit")

	if cc.IsBatchSort(c) {
		cc.BatchSort(c, &model.Content{}, "sorting", 255)
		return
	}

	if submit == "field" {
		field := c.PostForm("field")
		value := c.PostForm("value")
		if err := cc.svc.UpdateSingleField(id, field, value); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "修改成功")
		return
	}

	// Handle single field update via query params (already parsed from wildcard)
	if field != "" {
		if err := cc.svc.UpdateSingleField(id, field, value); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "修改成功")
		return
	}

	if submit == "copy" {
		if err := cc.svc.CopyContent(id, c.PostForm("scode")); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "複製成功")
		return
	}

	if submit == "move" {
		if err := cc.svc.MoveContent(id, c.PostForm("scode")); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "移動成功")
		return
	}

	if submit == "baiduzz" || submit == "baiduks" {
		cc.JSONOKMsg(c, "提交成功")
		return
	}

	if c.Request.Method == "POST" {
		title := c.PostForm("title")
		if title == "" {
			cc.JSONFail(c, "標題不能為空")
			return
		}

		// Read filename with urlname fallback
		urlname := c.PostForm("filename")
		if urlname == "" {
			urlname = c.PostForm("urlname")
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
			"urlname":     urlname,
			"outlink":     c.PostForm("outlink"),
			"enclosure":   c.PostForm("enclosure"),
			"tags":        strings.ReplaceAll(c.PostForm("tags"), "，", ","),
			"titlecolor":  c.PostForm("titlecolor"),
			"gnote":       c.PostForm("gnote"),
		}

		if v, err := strconv.Atoi(c.DefaultPostForm("visits", "0")); err == nil {
			updates["visits"] = v
		}
		// 注意：詳情編輯不改排序值（與 PbootCMS PHP 一致）
		// 排序僅通過列表頁批量排序修改
		if v, err := strconv.Atoi(c.DefaultPostForm("istop", "0")); err == nil {
			updates["istop"] = v
		}
		if v, err := strconv.Atoi(c.DefaultPostForm("isrecommend", "0")); err == nil {
			updates["isrecommend"] = v
		}
		if v, err := strconv.Atoi(c.DefaultPostForm("isheadline", "0")); err == nil {
			updates["isheadline"] = v
		}
		if v := c.PostForm("status"); v != "" {
			updates["status"] = helper.ParseInt(v)
		}
		// picstitle: PHP 用 implode(',', $picstitle) 處理數組
		if pts := c.PostFormArray("picstitle"); len(pts) > 0 {
			updates["picstitle"] = strings.Join(pts, ",")
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

		// 收集擴展字段數據
		extFields := helper.GetExtFieldsByMcode(mcode)
		extData := cc.svc.CollectExtFieldData(extFields,
			func(key string) string { return c.PostForm(key) },
			func(key string) []string { return c.PostFormArray(key) },
		)

		if err := cc.svc.UpdateContent(id, updates, extData); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "修改成功")
		return
	}

	// GET: 加載內容及擴展數據用於表單回填
	contentMap, err := cc.svc.GetContentWithExt(id)
	if err != nil {
		cc.JSONFail(c, err.Error())
		return
	}

	// Get mcode from content's sort if not in query (已由上方 resolveMcodeByContentID 處理)

	sorts, _ := cc.svc.GetAllSorts()
	data := cc.contentTemplateData(mcode, sorts, contentMap)
	data["content"] = contentMap
	scodeVal, _ := contentMap["Scode"].(string)
	data["sort_select"] = helper.BuildSortSelectWithSelected(sorts, mcode, scodeVal)
	// 預處理 pics/picstitle 供模板多圖循環（替代殘留的 {php} explode/foreach）
	if picsStr, ok := contentMap["Pics"].(string); ok && picsStr != "" {
		data["pics"] = strings.Split(picsStr, ",")
	} else {
		data["pics"] = []string{}
	}
	if picstitleStr, ok := contentMap["Picstitle"].(string); ok && picstitleStr != "" {
		data["picstitle"] = strings.Split(picstitleStr, ",")
	} else {
		data["picstitle"] = []string{}
	}
	data["mod"] = true
	data["mcode"] = mcode
	// 預格式化日期供模板使用（pongo2 無 date 過濾器）
	if dateVal, ok := contentMap["Date"].(time.Time); ok && !dateVal.IsZero() {
		data["dateStr"] = dateVal.Format("2006-01-02 15:04:05")
	} else {
		data["dateStr"] = ""
	}
	common.Render(c, "content/content.html", data)
}

// Del - Delete content
func (cc *ContentController) Del(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		// Try POST form array for batch delete
		ids := c.PostFormArray("list[]")
		if len(ids) == 0 {
			ids = c.PostFormArray("list")
		}
		if len(ids) > 0 {
			if err := cc.svc.DeleteContent(ids); err != nil {
				cc.JSONFail(c, err.Error())
				return
			}
			cc.JSONOKMsg(c, "刪除成功")
			return
		}
		cc.JSONFail(c, "未選擇任何項目")
		return
	}
	ids := strings.Split(idStr, ",")
	if err := cc.svc.DeleteContent(ids); err != nil {
		cc.JSONFail(c, err.Error())
		return
	}
	cc.JSONOKMsg(c, "刪除成功")
}

// applyPathAction 將 /key/value/key2/value2 路徑參數轉為 query 參數
func (cc *ContentController) applyPathAction(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))
	for k, v := range params {
		if c.Request.URL.RawQuery != "" {
			c.Request.URL.RawQuery += "&"
		}
		c.Request.URL.RawQuery += k + "=" + v
	}
}

// IndexCatchAll handles /content/index/*action paths like /mcode/2D1
func (cc *ContentController) IndexCatchAll(c *gin.Context) {
	cc.applyPathAction(c)
	cc.Index(c)
}

// resolveMcodeByContentID 從內容的欄目反推 mcode，確保 MOD GET/POST 都能獲取擴展字段定義
func (cc *ContentController) resolveMcodeByContentID(id int) string {
	var doc model.Content
	if model.DB.Select("scode").Where("id = ?", id).First(&doc).Error != nil {
		return ""
	}
	var sort model.ContentSort
	if model.DB.Select("mcode").Where("scode = ?", doc.Scode).First(&sort).Error != nil {
		return ""
	}
	return sort.Mcode
}

// AddCatchAll handles /content/add/*action paths like /mcode/2D1
func (cc *ContentController) AddCatchAll(c *gin.Context) {
	cc.applyPathAction(c)
	cc.Add(c)
}

// DelCatchAll handles /content/del/*action paths like /id/42
func (cc *ContentController) DelCatchAll(c *gin.Context) {
	cc.applyPathAction(c)
	cc.Del(c)
}
