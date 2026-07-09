package content

import (
	"context"
	"fmt"
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	svc "gbootcms/apps/admin/service/content"
	"gbootcms/apps/common"
	"gbootcms/apps/common/push"
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
func (cc *ContentController) contentTemplateData(ctx context.Context, mcode, scode string, sorts []model.ContentSort, contentMap map[string]interface{}) gin.H {
	return gin.H{
		"mcode":             mcode,
		"model_name":        helper.GetModelNameByMcode(mcode),
		"sorts":             sorts,
		"sort_select":       helper.BuildSearchSelectHTML(sorts, mcode),
		"search_select":     helper.BuildSearchSelectHTML(sorts, mcode),
		"subsort_select":    helper.BuildSearchSelectHTML(sorts, mcode),
		"extfield":          cc.svc.BuildExtFieldTemplateData(ctx, mcode, scode, contentMap),
		"groups":            helper.BuildGroupsData(),
		"baidu_zz_token":    model.GetConfigValue("baidu_zz_token", ""),
		"baidu_ks_token":    model.GetConfigValue("baidu_ks_token", ""),
		"bing_indexnow_key": model.GetConfigValue("bing_indexnow_key", ""),
	}
}

// Index - Content list
func (cc *ContentController) Index(c *gin.Context) {
	// 用 c.Request.URL.Query() 而非 c.Query()，因為 IndexCatchAll 修改了 RawQuery
	q := c.Request.URL.Query()
	mcode := q.Get("mcode")
	scode := q.Get("scode")
	keyword := q.Get("keyword")
	page, pageSize, _ := cc.Paginate(c)

	contents, total, _ := cc.svc.ListContents(c.Request.Context(), mcode, scode, keyword, page, pageSize)
	sorts, _ := cc.svc.GetAllSorts(c.Request.Context())

	data := cc.contentTemplateData(c.Request.Context(), mcode, scode, sorts, nil)
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
			CreateUser:  cc.GetAdminUsername(c),
			UpdateUser:  cc.GetAdminUsername(c),
		}

		// 收集擴展字段數據
		extFields := helper.GetExtFieldsByMcode(mcode)
		extData := cc.svc.CollectExtFieldData(c.Request.Context(), extFields,
			func(key string) string { return c.PostForm(key) },
			func(key string) []string { return c.PostFormArray(key) },
		)

		if err := cc.svc.CreateContent(c.Request.Context(), &doc, extData); err != nil {
			cc.LogAction(c, "新增文章失敗")
			cc.JSONFail(c, err.Error())
			return
		}
		cc.LogAction(c, "新增文章成功")
		cc.JSONOKMsg(c, common.NoticeAdd)
		return
	}

	sorts, _ := cc.svc.GetAllSorts(c.Request.Context())
	data := cc.contentTemplateData(c.Request.Context(), mcode, "", sorts, nil)
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
		mcode = cc.resolveMcodeByContentID(c.Request.Context(), id)
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
		cc.LogAction(c, "修改內容排序成功")
		cc.BatchSort(c, &model.Content{}, "sorting", 255)
		return
	}

	if submit == "field" {
		field := c.PostForm("field")
		value := c.PostForm("value")
		if err := cc.svc.UpdateSingleField(c.Request.Context(), id, field, value); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// Handle single field update via query params (already parsed from wildcard)
	if field != "" {
		if err := cc.svc.UpdateSingleField(c.Request.Context(), id, field, value); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, common.NoticeModify)
		return
	}

	if submit == "copy" {
		// 批量複製（對齊 PHP: post('list') 接收陣列）
		list := c.PostFormArray("list[]")
		if len(list) == 0 {
			list = c.PostFormArray("list")
		}
		scode := c.PostForm("scode")
		if len(list) == 0 {
			cc.JSONFail(c, "請選擇要複製的內容")
			return
		}
		if scode == "" {
			cc.JSONFail(c, "請選擇目標欄目")
			return
		}
		var failCount int
		for _, idStr := range list {
			idInt, err := strconv.Atoi(idStr)
			if err != nil {
				failCount++
				continue
			}
			if err := cc.svc.CopyContent(c.Request.Context(), idInt, scode); err != nil {
				failCount++
			}
		}
		cc.LogAction(c, fmt.Sprintf("複製內容成功（共%d條，失敗%d條）", len(list), failCount))
		cc.JSONOKMsg(c, common.NoticeCopy)
		return
	}

	if submit == "move" {
		// 批量移動（對齊 PHP: post('list') 接收陣列）
		list := c.PostFormArray("list[]")
		if len(list) == 0 {
			list = c.PostFormArray("list")
		}
		scode := c.PostForm("scode")
		if len(list) == 0 {
			cc.JSONFail(c, "請選擇要移動的內容")
			return
		}
		if scode == "" {
			cc.JSONFail(c, "請選擇目標欄目")
			return
		}
		var failCount int
		for _, idStr := range list {
			idInt, err := strconv.Atoi(idStr)
			if err != nil {
				failCount++
				continue
			}
			if err := cc.svc.MoveContent(c.Request.Context(), idInt, scode); err != nil {
				failCount++
			}
		}
		cc.LogAction(c, fmt.Sprintf("移動內容成功（共%d條，失敗%d條）", len(list), failCount))
		cc.JSONOKMsg(c, common.NoticeMove)
		return
	}

	if submit == "baiduzz" || submit == "baiduks" || submit == "bingpush" || submit == "googlepush" {
		cc.handlePush(c, submit)
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
			"update_user": cc.GetAdminUsername(c),
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
		extData := cc.svc.CollectExtFieldData(c.Request.Context(), extFields,
			func(key string) string { return c.PostForm(key) },
			func(key string) []string { return c.PostFormArray(key) },
		)

		if err := cc.svc.UpdateContent(c.Request.Context(), id, updates, extData); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.LogAction(c, "修改文章成功")
		cc.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// GET: 加載內容及擴展數據用於表單回填
	contentMap, err := cc.svc.GetContentWithExt(c.Request.Context(), id)
	if err != nil {
		cc.JSONFail(c, err.Error())
		return
	}

	// Get mcode from content's sort if not in query (已由上方 resolveMcodeByContentID 處理)

	sorts, _ := cc.svc.GetAllSorts(c.Request.Context())
	scodeVal, _ := contentMap["Scode"].(string)
	data := cc.contentTemplateData(c.Request.Context(), mcode, scodeVal, sorts, contentMap)
	data["content"] = contentMap
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
			if err := cc.svc.DeleteContent(c.Request.Context(), ids); err != nil {
				cc.LogAction(c, "刪除文章失敗")
				cc.JSONFail(c, err.Error())
				return
			}
			cc.LogAction(c, "刪除文章成功")
			cc.JSONOKMsg(c, common.NoticeDelete)
			return
		}
		cc.LogAction(c, "刪除文章失敗")
		cc.JSONFail(c, "未選擇任何項目")
		return
	}
	ids := strings.Split(idStr, ",")
	if err := cc.svc.DeleteContent(c.Request.Context(), ids); err != nil {
		cc.LogAction(c, "刪除文章失敗")
		cc.JSONFail(c, err.Error())
		return
	}
	cc.LogAction(c, "刪除文章成功")
	cc.JSONOKMsg(c, common.NoticeDelete)
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
func (cc *ContentController) resolveMcodeByContentID(ctx context.Context, id int) string {
	var doc model.Content
	if model.DB.WithContext(ctx).Select("scode").Where("id = ?", id).First(&doc).Error != nil {
		return ""
	}
	var sort model.ContentSort
	if model.DB.WithContext(ctx).Select("mcode").Where("scode = ?", doc.Scode).First(&sort).Error != nil {
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

// handlePush 處理百度/Bing/Google 推送（對齊 PHP ContentController::mod 的 baiduzz/baiduks 分支）
func (cc *ContentController) handlePush(c *gin.Context, pushType string) {
	// 取得選中的內容 ID 列表
	ids := c.PostFormArray("list[]")
	if len(ids) == 0 {
		ids = c.PostFormArray("list")
	}
	if len(ids) == 0 {
		cc.JSONFail(c, "請選擇要推送的內容")
		return
	}

	// 查詢內容
	var contents []model.Content
	model.DB.WithContext(c.Request.Context()).Where("id IN ?", ids).Find(&contents)
	if len(contents) == 0 {
		cc.JSONFail(c, "未找到可推送的內容")
		return
	}

	// 排除外鏈內容（對齊 PHP: 鏈接類型不允許推送）
	var validContents []model.Content
	for _, ct := range contents {
		if ct.Outlink == "" {
			validContents = append(validContents, ct)
		}
	}
	if len(validContents) == 0 {
		cc.JSONFail(c, "所選內容均為外鏈類型，無法推送")
		return
	}

	// 取得站點域名
	domain := push.GetSiteDomain(c.Request.Host)

	// 構建推送 URL 列表
	var urls []string
	for _, ct := range validContents {
		urlPath := buildContentPath(c.Request.Context(), &ct)
		urls = append(urls, push.BuildFullURL(domain, urlPath))
	}

	switch pushType {
	case "baiduzz":
		token := model.GetConfigValue("baidu_zz_token", "")
		if token == "" {
			cc.JSONFail(c, "請先到系統配置中填寫百度普通收錄推送 token")
			return
		}
		api := fmt.Sprintf("http://data.zz.baidu.com/urls?site=%s&token=%s", domain, token)
		result, err := push.PushBaidu(api, urls)
		if err != nil {
			cc.JSONFail(c, "百度普通收錄推送失敗: "+err.Error())
			return
		}
		if result.Success {
			cc.JSONOKMsg(c, "百度普通收錄"+result.Message)
		} else {
			cc.JSONFail(c, result.Message)
		}

	case "baiduks":
		token := model.GetConfigValue("baidu_ks_token", "")
		if token == "" {
			cc.JSONFail(c, "請先到系統配置中填寫百度快速收錄推送 token")
			return
		}
		api := fmt.Sprintf("http://data.zz.baidu.com/urls?site=%s&token=%s&type=daily", domain, token)
		result, err := push.PushBaidu(api, urls)
		if err != nil {
			cc.JSONFail(c, "百度快速收錄推送失敗: "+err.Error())
			return
		}
		if result.Success {
			cc.JSONOKMsg(c, "百度快速收錄"+result.Message)
		} else {
			cc.JSONFail(c, result.Message)
		}

	case "bingpush":
		key := model.GetConfigValue("bing_indexnow_key", "")
		if key == "" {
			cc.JSONFail(c, "請先到系統配置中填寫 Bing IndexNow 密鑰")
			return
		}
		result, err := push.PushBing(domain, key, urls)
		if err != nil {
			cc.JSONFail(c, "Bing 推送失敗: "+err.Error())
			return
		}
		if result.Success {
			cc.JSONOKMsg(c, result.Message)
		} else {
			cc.JSONFail(c, result.Message)
		}

	case "googlepush":
		sitemapURL := push.BuildFullURL(domain, "/sitemap.xml")
		result, err := push.PushGoogle(sitemapURL)
		if err != nil {
			cc.JSONFail(c, "Google 推送失敗: "+err.Error())
			return
		}
		if result.Success {
			cc.JSONOKMsg(c, result.Message)
		} else {
			cc.JSONFail(c, result.Message)
		}

	default:
		cc.JSONFail(c, "未知的推送類型")
	}
}

// buildContentPath 構建內容的相對 URL 路徑（與前台 contentURL 邏輯一致）
func buildContentPath(ctx context.Context, c *model.Content) string {
	if c.Outlink != "" {
		return c.Outlink
	}

	// 短路徑模式
	if model.GetConfigValue("url_rule_content_path", "0") == "1" {
		if c.Filename != "" {
			return "/" + c.Filename + ".html"
		}
		if c.URLName != "" {
			return "/" + c.URLName + ".html"
		}
		return "/content/" + strconv.Itoa(int(c.ID)) + ".html"
	}

	// 帶欄目路徑模式
	var sortPath string
	if c.Scode != "" {
		var s model.ContentSort
		if model.DB.WithContext(ctx).Where("scode = ?", c.Scode).First(&s).Error == nil {
			if s.Filename != "" {
				sortPath = s.Filename
			} else if s.URLName != "" {
				sortPath = s.URLName
			}
		}
	}

	if c.Filename != "" {
		if sortPath != "" {
			return "/" + sortPath + "/" + c.Filename + ".html"
		}
		return "/" + c.Filename + ".html"
	}
	if c.URLName != "" {
		if sortPath != "" {
			return "/" + sortPath + "/" + c.URLName + ".html"
		}
		return "/" + c.URLName + ".html"
	}
	if sortPath != "" {
		return "/" + sortPath + "/" + strconv.Itoa(int(c.ID)) + ".html"
	}
	return "/content/" + strconv.Itoa(int(c.ID)) + ".html"
}
