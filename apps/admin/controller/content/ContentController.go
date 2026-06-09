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
func (cc *ContentController) contentTemplateData(mcode string, sorts []model.ContentSort) gin.H {
	return gin.H{
		"model_name":     helper.GetModelNameByMcode(mcode),
		"sorts":          sorts,
		"sort_select":    helper.BuildSearchSelectHTML(sorts, mcode),
		"search_select":  helper.BuildSearchSelectHTML(sorts, mcode),
		"subsort_select": helper.BuildSearchSelectHTML(sorts, mcode),
		"extfield":       helper.GetExtFieldsByMcode(mcode),
		"groups":         helper.BuildGroupsData(),
	}
}

// Index - Content list
func (cc *ContentController) Index(c *gin.Context) {
	mcode := c.Query("mcode")
	scode := c.Query("scode")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 15
	if ps := c.Query("pagesize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}

	contents, total, _ := cc.svc.ListContents(scode, page, pageSize)
	sorts, _ := cc.svc.GetAllSorts()

	data := cc.contentTemplateData(mcode, sorts)
	data["contents"] = helper.AddSortName(contents, sorts)
	data["list"] = true
	data["scode"] = scode
	data["total"] = total
	data["page"] = page
	data["pagesize"] = pageSize

	// Build pagination
	baseURL := fmt.Sprintf("/admin/content/index?mcode=%s", mcode)
	if scode != "" {
		baseURL += "&scode=" + scode
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
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
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
			Tags:        c.PostForm("tags"),
			TitleColor:  c.PostForm("titlecolor"),
			Enclosure:   c.PostForm("enclosure"),
			Gid:         c.PostForm("gid"),
			GType:       c.PostForm("gtype"),
			Gnote:       c.PostForm("gnote"),
		}
		if err := cc.svc.CreateContent(&doc); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "Added successfully")
		return
	}

	sorts, _ := cc.svc.GetAllSorts()
	data := cc.contentTemplateData(mcode, sorts)
	data["list"] = true
	common.Render(c, "content/content.html", data)
}

// Mod - Modify content
func (cc *ContentController) Mod(c *gin.Context) {
	// Parse wildcard action param: /mcode/1/id/123 or /id/123/field/status/value/0 or /123
	params := helper.ParseWildcardAction(c.Param("action"))

	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

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

	submit := c.PostForm("submit")

	if submit == "sorting" {
		idList := c.PostFormArray("id")
		sortingList := c.PostFormArray("sorting")
		idSortingMap := map[string]int{}
		for i, sid := range idList {
			if i < len(sortingList) {
				s, _ := strconv.Atoi(sortingList[i])
				idSortingMap[sid] = s
			}
		}
		if err := cc.svc.UpdateSorting(idSortingMap); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "Sort order modified successfully")
		return
	}

	if submit == "field" {
		field := c.PostForm("field")
		value := c.PostForm("value")
		if err := cc.svc.UpdateSingleField(id, field, value); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "Modified successfully")
		return
	}

	// Handle single field update via query params (already parsed from wildcard)
	if field != "" {
		if err := cc.svc.UpdateSingleField(id, field, value); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "Modified successfully")
		return
	}

	if submit == "copy" {
		if err := cc.svc.CopyContent(id, c.PostForm("scode")); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "Copied successfully")
		return
	}

	if submit == "move" {
		if err := cc.svc.MoveContent(id, c.PostForm("scode")); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "Moved successfully")
		return
	}

	if submit == "baiduzz" || submit == "baiduks" {
		cc.JSONOKMsg(c, "Submitted successfully")
		return
	}

	if c.Request.Method == "POST" {
		title := c.PostForm("title")
		if title == "" {
			cc.JSONFail(c, "Title cannot be empty")
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
			"tags":        c.PostForm("tags"),
			"titlecolor":  c.PostForm("titlecolor"),
			"gnote":       c.PostForm("gnote"),
		}

		if v, err := strconv.Atoi(c.DefaultPostForm("visits", "0")); err == nil {
			updates["visits"] = v
		}
		if v, err := strconv.Atoi(c.DefaultPostForm("sorting", "0")); err == nil {
			updates["sorting"] = v
		}
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

		if err := cc.svc.UpdateContent(id, updates); err != nil {
			cc.JSONFail(c, err.Error())
			return
		}
		cc.JSONOKMsg(c, "Modified successfully")
		return
	}

	doc, err := cc.svc.GetContent(id)
	if err != nil {
		cc.JSONFail(c, err.Error())
		return
	}

	// Get mcode from content's sort if not in query
	if mcode == "" {
		var sort model.ContentSort
		if model.DB.Where("scode = ?", doc.Scode).First(&sort).Error == nil {
			mcode = sort.Mcode
		}
	}

	sorts, _ := cc.svc.GetAllSorts()
	data := cc.contentTemplateData(mcode, sorts)
	data["content"] = doc
	data["sort_select"] = helper.BuildSortSelectWithSelected(sorts, mcode, doc.Scode)
	data["mod"] = true
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
			cc.JSONOKMsg(c, "Deleted successfully")
			return
		}
		cc.JSONFail(c, "No items selected")
		return
	}
	ids := strings.Split(idStr, ",")
	if err := cc.svc.DeleteContent(ids); err != nil {
		cc.JSONFail(c, err.Error())
		return
	}
	cc.JSONOKMsg(c, "Deleted successfully")
}
