package content

import (
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"gbootcms/apps/common/middleware"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RedirectController 301 重定向管理
type RedirectController struct {
	common.BaseController
}

// Index 重定向規則列表
func (rc *RedirectController) Index(c *gin.Context) {
	page, pageSize, _ := rc.Paginate(c)
	keyword := c.Query("keyword")

	query := model.DB.Model(&model.Redirect{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("old_url LIKE ? OR new_url LIKE ?", like, like)
	}

	var total int64
	query.Count(&total)

	var items []model.Redirect
	query.Order("sorting ASC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items)

	baseURL := "/admin/content/redirect/index?"
	if keyword != "" {
		baseURL += "keyword=" + keyword + "&"
	}

	common.Render(c, "content/redirect.html", gin.H{
		"list":     true,
		"items":    items,
		"C":        "content/redirect",
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
		"keyword":  keyword,
	})
}

// Add 新增重定向規則
func (rc *RedirectController) Add(c *gin.Context) {
	if c.Request.Method == "GET" {
		common.Render(c, "content/redirect.html", gin.H{
			"mod": false,
			"C":   "content/redirect",
		})
		return
	}

	oldURL := strings.TrimSpace(c.PostForm("old_url"))
	newURL := strings.TrimSpace(c.PostForm("new_url"))
	if oldURL == "" || newURL == "" {
		rc.JSONFail(c, "舊URL和新URL不能為空")
		return
	}

	matchType, _ := strconv.Atoi(c.PostForm("match_type"))
	if matchType != 1 && matchType != 2 {
		matchType = 1
	}
	sorting, _ := strconv.Atoi(c.PostForm("sorting"))
	status, _ := strconv.Atoi(c.PostForm("status"))
	if status != 0 && status != 1 {
		status = 1
	}

	username := rc.GetAdminUsername(c)
	now := time.Now()
	rule := model.Redirect{
		OldURL:     oldURL,
		NewURL:     newURL,
		MatchType:  matchType,
		Status:     status,
		Sorting:    sorting,
		CreateUser: username,
		UpdateUser: username,
		CreateTime: now,
		UpdateTime: now,
	}
	if err := model.DB.Create(&rule).Error; err != nil {
		rc.JSONFail(c, "新增失敗: "+err.Error())
		return
	}

	middleware.RefreshRedirectRules()
	rc.LogAction(c, "新增301重定向規則: "+oldURL+" → "+newURL)
	rc.JSONOKMsgTourl(c, common.NoticeAdd, "/admin/content/redirect/index")
}

// Mod 修改重定向規則
func (rc *RedirectController) Mod(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))
	idStr := params["id"]
	field := params["field"]
	value := params["value"]

	// 單欄位切換（狀態開關）
	if idStr != "" && field != "" && value != "" {
		id, _ := strconv.Atoi(idStr)
		if err := model.DB.Model(&model.Redirect{}).Where("id = ?", id).Update(field, value).Error; err != nil {
			rc.JSONFail(c, err.Error())
			return
		}
		middleware.RefreshRedirectRules()
		rc.JSONOKMsg(c, common.NoticeModify)
		return
	}

	if c.Request.Method == "GET" {
		id, _ := strconv.Atoi(idStr)
		var rule model.Redirect
		if err := model.DB.First(&rule, id).Error; err != nil {
			rc.JSONFail(c, "規則不存在")
			return
		}
		common.Render(c, "content/redirect.html", gin.H{
			"mod":  true,
			"item": rule,
			"C":    "content/redirect",
		})
		return
	}

	// POST 修改
	id, _ := strconv.Atoi(idStr)
	var rule model.Redirect
	if err := model.DB.First(&rule, id).Error; err != nil {
		rc.JSONFail(c, "規則不存在")
		return
	}

	oldURL := strings.TrimSpace(c.PostForm("old_url"))
	newURL := strings.TrimSpace(c.PostForm("new_url"))
	if oldURL == "" || newURL == "" {
		rc.JSONFail(c, "舊URL和新URL不能為空")
		return
	}

	matchType, _ := strconv.Atoi(c.PostForm("match_type"))
	if matchType != 1 && matchType != 2 {
		matchType = 1
	}
	sorting, _ := strconv.Atoi(c.PostForm("sorting"))
	status, _ := strconv.Atoi(c.PostForm("status"))
	if status != 0 && status != 1 {
		status = 1
	}

	// 髒檢查
	changed := rule.OldURL != oldURL || rule.NewURL != newURL ||
		rule.MatchType != matchType || rule.Sorting != sorting || rule.Status != status
	if !changed {
		rc.JSONOKMsg(c, common.NoticeNoChange)
		return
	}

	rule.OldURL = oldURL
	rule.NewURL = newURL
	rule.MatchType = matchType
	rule.Sorting = sorting
	rule.Status = status
	rule.UpdateUser = rc.GetAdminUsername(c)
	rule.UpdateTime = time.Now()

	if err := model.DB.Save(&rule).Error; err != nil {
		rc.JSONFail(c, "修改失敗: "+err.Error())
		return
	}

	middleware.RefreshRedirectRules()
	rc.LogAction(c, "修改301重定向規則: "+oldURL+" → "+newURL)
	rc.JSONOKMsgTourl(c, common.NoticeModify, "/admin/content/redirect/index")
}

// Del 刪除重定向規則
func (rc *RedirectController) Del(c *gin.Context) {
	// 支援 *action 通配符路徑: /del/id/123
	params := helper.ParseWildcardAction(c.Param("action"))
	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	if idStr == "" {
		ids := c.PostFormArray("list[]")
		if len(ids) == 0 {
			ids = c.PostFormArray("list")
		}
		if len(ids) > 0 {
			if err := model.DB.Where("id IN ?", ids).Delete(&model.Redirect{}).Error; err != nil {
				rc.JSONFail(c, err.Error())
				return
			}
			middleware.RefreshRedirectRules()
			rc.LogAction(c, "刪除301重定向規則")
			rc.JSONOKMsg(c, common.NoticeDelete)
			return
		}
		rc.JSONFail(c, "未選擇任何項目")
		return
	}
	ids := strings.Split(idStr, ",")
	if err := model.DB.Where("id IN ?", ids).Delete(&model.Redirect{}).Error; err != nil {
		rc.JSONFail(c, err.Error())
		return
	}
	middleware.RefreshRedirectRules()
	rc.LogAction(c, "刪除301重定向規則")
	rc.JSONOKMsg(c, common.NoticeDelete)
}
