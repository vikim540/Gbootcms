package content

import (
	"pbootcms-go/apps/admin/helper"
	"pbootcms-go/apps/admin/model"
	svc "pbootcms-go/apps/admin/service/content"
	"pbootcms-go/apps/common"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ContentSortController - Content Sort Management Controller
type ContentSortController struct {
	common.BaseController
	svc svc.ContentSortService
}

// sortTemplateData returns the common template data needed by all sort views.
func (csc *ContentSortController) sortTemplateData(sorts []model.ContentSort) gin.H {
	return gin.H{
		"allmodels":   helper.GetAllModelsData(),
		"models":      helper.GetAllModelsData(),
		"tpls":        helper.GetTemplateFiles(),
		"groups":      helper.BuildGroupsData(),
		"sort_select": helper.BuildSortSelectHTML(sorts, ""),
	}
}

// Index - Sort list
func (csc *ContentSortController) Index(c *gin.Context) {
	sorts, _ := csc.svc.ListSorts()
	data := csc.sortTemplateData(sorts)
	data["sorts"] = helper.BuildSortTreeData(sorts)
	data["list"] = true
	common.Render(c, "content/contentsort.html", data)
}

// Add - Add new sort
func (csc *ContentSortController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		multiplename := c.PostForm("multiplename")
		if multiplename != "" {
			if err := csc.svc.BatchAddSorts(multiplename, c.PostForm("pcode")); err != nil {
				csc.JSONFail(c, err.Error())
				return
			}
			csc.JSONOKMsg(c, "批量新增成功")
			return
		}

		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		stype, _ := strconv.Atoi(c.DefaultPostForm("type", "1"))

		// Read filename (template uses name="filename") with urlname fallback
		urlname := c.PostForm("filename")
		if urlname == "" {
			urlname = c.PostForm("urlname")
		}

		sort := model.ContentSort{
			Scode:       c.PostForm("scode"),
			Pcode:       c.PostForm("pcode"),
			Name:        c.PostForm("name"),
			Subname:     c.PostForm("subname"),
			Mcode:       c.PostForm("mcode"),
			Type:        stype,
			ListTpl:     c.PostForm("listtpl"),
			ContentTpl:  c.PostForm("contenttpl"),
			Ico:         c.PostForm("ico"),
			Pic:         c.PostForm("pic"),
			Keywords:    c.PostForm("keywords"),
			Description: c.PostForm("description"),
			Filename:    c.PostForm("filename"), // URL 名稱（PbootCMS 偽靜態用）
			Sort:        sorting,
			URLName:     urlname,
			Outlink:     c.PostForm("outlink"),
			Gid:         c.PostForm("gid"),
			GType:       c.PostForm("gtype"),
			Gnote:       c.PostForm("gnote"),
			Def1:        c.PostForm("def1"),
			Def2:        c.PostForm("def2"),
			Def3:        c.PostForm("def3"),
			Title:       c.PostForm("title"),
			Status:      helper.ParseInt(c.DefaultPostForm("status", "1")),
		}
		if err := csc.svc.CreateSort(&sort); err != nil {
			csc.JSONFail(c, err.Error())
			return
		}
		csc.JSONOKMsg(c, "新增成功")
		return
	}

	sorts, _ := csc.svc.ListSorts()
	data := csc.sortTemplateData(sorts)
	data["sorts"] = helper.BuildSortTreeData(sorts)
	data["list"] = true
	common.Render(c, "content/contentsort.html", data)
}

// Mod - Modify sort
func (csc *ContentSortController) Mod(c *gin.Context) {
	// Parse wildcard action param: /scode/123/field/status/value/0 or /123
	params := helper.ParseWildcardAction(c.Param("action"))

	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	scode := params["scode"]
	if scode == "" {
		scode = c.Query("scode")
	}
	field := params["field"]
	if field == "" {
		field = c.Query("field")
	}
	value := params["value"]
	if value == "" {
		value = c.Query("value")
	}
	mcode := params["mcode"]
	if mcode == "" {
		mcode = c.Query("mcode")
	}
	if field != "" {
		// Try scode-based lookup first, then id-based
		if scode != "" {
			if err := csc.svc.UpdateSortByScodeField(scode, field, value); err != nil {
				csc.JSONFail(c, err.Error())
				return
			}
		} else {
			id, _ := strconv.Atoi(idStr)
			if err := csc.svc.UpdateSortSingleField(id, field, value); err != nil {
				csc.JSONFail(c, err.Error())
				return
			}
		}
		csc.JSONOKMsg(c, "修改成功")
		return
	}

	if csc.IsBatchSort(c) {
		csc.BatchSort(c, &model.ContentSort{}, "sorting", 255)
		return
	}

	if c.Request.Method == "POST" {
		stype, _ := strconv.Atoi(c.DefaultPostForm("type", "1"))
		postScode := c.PostForm("scode")

		// Read filename with urlname fallback
		urlname := c.PostForm("filename")
		if urlname == "" {
			urlname = c.PostForm("urlname")
		}

		updates := map[string]interface{}{
			"pcode":       c.PostForm("pcode"),
			"name":        c.PostForm("name"),
			"subname":     c.PostForm("subname"),
			"mcode":       c.PostForm("mcode"),
			"type":        stype,
			"listtpl":     c.PostForm("listtpl"),
			"contenttpl":  c.PostForm("contenttpl"),
			"ico":         c.PostForm("ico"),
			"pic":         c.PostForm("pic"),
			"keywords":    c.PostForm("keywords"),
			"description": c.PostForm("description"),
			"filename":    c.PostForm("filename"),
			"urlname":     urlname,
			"outlink":     c.PostForm("outlink"),
			"gid":         c.PostForm("gid"),
			"gtype":       c.PostForm("gtype"),
			"gnote":       c.PostForm("gnote"),
			"def1":        c.PostForm("def1"),
			"def2":        c.PostForm("def2"),
			"def3":        c.PostForm("def3"),
			"title":       c.PostForm("title"),
			"status":      helper.ParseInt(c.DefaultPostForm("status", "1")),
		}
		// Only include scode in updates if it was explicitly provided in the form
		// (otherwise we would overwrite the existing scode with an empty string)
		if postScode != "" {
			updates["scode"] = postScode
		}

		// Try scode-based update, then id-based
		var err error
		if scode != "" {
			err = csc.svc.UpdateSortByScode(scode, updates)
		} else {
			id, _ := strconv.Atoi(idStr)
			err = csc.svc.UpdateSort(id, updates)
		}
		if err != nil {
			csc.JSONFail(c, err.Error())
			return
		}

		// If type=1 (list), create initial content if not exists
		// Use the URL scode param as fallback when postScode is empty
		contentScode := postScode
		if contentScode == "" {
			contentScode = scode
		}
		if stype == 1 && contentScode != "" {
			var existing model.Content
			result := model.DB.Where("scode = ?", contentScode).First(&existing)
			if result.Error != nil && c.PostForm("outlink") == "" {
				model.DB.Create(&model.Content{
					Scode:  contentScode,
					Title:  c.PostForm("name"),
					Status: 1,
					Date:   time.Now(),
				})
			}
		}

		csc.JSONOKMsg(c, "修改成功")
		return
	}

	// GET: render mod form
	// If the wildcard marker says "lookup by scode" (e.g. URL ".../123,scode"),
	// promote idStr → scode so we use the scode-based lookup below.
	if params["_lookup_by"] == "scode" && scode == "" {
		scode = idStr
		idStr = ""
	}
	var sort *model.ContentSort
	var err error
	if scode != "" {
		sort, err = csc.svc.GetSortByScode(scode)
	} else {
		id, _ := strconv.Atoi(idStr)
		sort, err = csc.svc.GetSort(id)
	}
	if err != nil {
		csc.JSONFail(c, err.Error())
		return
	}

	sorts, _ := csc.svc.ListSorts()
	data := csc.sortTemplateData(sorts)
	data["sort"] = sort
	data["sorts"] = helper.BuildSortTreeData(sorts)
	data["sort_select"] = helper.BuildSortSelectHTML(sorts, sort.Pcode)
	data["mod"] = true
	common.Render(c, "content/contentsort.html", data)
}

// Del - Delete sort
func (csc *ContentSortController) Del(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		// Try POST form array for batch delete
		ids := c.PostFormArray("list[]")
		if len(ids) == 0 {
			ids = c.PostFormArray("list")
		}
		for _, scode := range ids {
			csc.svc.DeleteSortByScode(scode)
		}
		csc.JSONOKMsg(c, "刪除成功")
		return
	}
	if err := csc.svc.DeleteSort(idStr); err != nil {
		csc.JSONFail(c, err.Error())
		return
	}
	csc.JSONOKMsg(c, "刪除成功")
}
