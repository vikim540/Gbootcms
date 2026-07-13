package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gbootcms/apps/admin/model"
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/common/middleware"

	"github.com/gin-gonic/gin"
)

// getAcode 取得語言代碼，預設使用系統預設語言
func getAcode(c *gin.Context) string {
	acode := c.Query("acode")
	if acode == "" {
		acode = middleware.GetDefaultAcode()
	}
	return acode
}

// buildContentURL 構建內容 URL
func buildContentURL(ct *model.Content) string {
	if ct.Outlink != "" {
		return ct.Outlink
	}
	if ct.Filename != "" {
		return "/" + ct.Filename + ".html"
	}
	if ct.URLName != "" {
		return "/" + ct.URLName
	}
	return fmt.Sprintf("/content/%d.html", ct.ID)
}

// formatTime 格式化時間
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// GetSite 站點資訊
// GET /api/v1/site
func GetSite(c *gin.Context) {
	acode := getAcode(c)
	var site model.Site
	model.DB.Where("acode = ?", acode).First(&site)
	var company model.Company
	model.DB.Where("acode = ?", acode).First(&company)

	apiOK(c, gin.H{
		"site": gin.H{
			"title":       site.Title,
			"subtitle":    site.Subtitle,
			"domain":      site.Domain,
			"logo":        site.Logo,
			"keywords":    site.Keywords,
			"description": site.Description,
			"icp":         site.ICP,
			"theme":       site.Theme,
		},
		"company": gin.H{
			"name":    company.Name,
			"address": company.Address,
			"phone":   company.Phone,
			"mobile":  company.Mobile,
			"email":   company.Email,
			"qq":      company.Qq,
			"wechat":  company.Weixin,
		},
	})
}

// ListSorts 欄目列表
// GET /api/v1/sorts?scode=&mcode=&status=1
func ListSorts(c *gin.Context) {
	acode := getAcode(c)
	query := model.DB.Model(&model.ContentSort{}).Where("acode = ?", acode)

	if scode := c.Query("scode"); scode != "" {
		query = query.Where("scode = ? OR pcode = ?", scode, scode)
	}
	if mcode := c.Query("mcode"); mcode != "" {
		query = query.Where("mcode = ?", mcode)
	}
	status := c.Query("status")
	if status == "" {
		status = "1"
	}
	if status != "-1" {
		query = query.Where("status = ?", status)
	}

	var sorts []model.ContentSort
	query.Order("sorting ASC, id ASC").Find(&sorts)

	apiOK(c, sorts)
}

// GetSort 欄目詳情
// GET /api/v1/sorts/:scode
func GetSort(c *gin.Context) {
	acode := getAcode(c)
	scode := c.Param("scode")
	var sort model.ContentSort
	if err := model.DB.Where("acode = ? AND (scode = ? OR filename = ? OR urlname = ?)", acode, scode, scode, scode).First(&sort).Error; err != nil {
		apiFail(c, http.StatusNotFound, "欄目不存在")
		return
	}
	apiOK(c, sort)
}

// ListContents 內容列表
// GET /api/v1/contents?scode=&mcode=&keyword=&page=&pagesize=&status=1
func ListContents(c *gin.Context) {
	acode := getAcode(c)
	page, pagesize := parsePagination(c)

	query := model.DB.Model(&model.Content{}).Where("acode = ? AND status = 1 AND date <= ?", acode, time.Now())

	if scode := c.Query("scode"); scode != "" {
		// 查找欄目及其子欄目
		var childScodes []string
		model.DB.Model(&model.ContentSort{}).Where("pcode = ?", scode).Pluck("scode", &childScodes)
		allScodes := append([]string{scode}, childScodes...)
		query = query.Where("scode IN ?", allScodes)
	}
	if mcode := c.Query("mcode"); mcode != "" {
		query = query.Where("scode IN (SELECT scode FROM ay_content_sort WHERE mcode = ?)", mcode)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR keywords LIKE ? OR description LIKE ?", like, like, like)
	}
	if istop := c.Query("istop"); istop != "" {
		query = query.Where("istop = ?", istop)
	}
	if isrecommend := c.Query("isrecommend"); isrecommend != "" {
		query = query.Where("isrecommend = ?", isrecommend)
	}

	// 排序
	order := c.DefaultQuery("order", "date")
	switch order {
	case "visits":
		query = query.Order("visits DESC, id DESC")
	case "sorting":
		query = query.Order("sorting ASC, id DESC")
	case "date":
		fallthrough
	default:
		query = query.Order("date DESC, id DESC")
	}

	var total int64
	query.Count(&total)

	var contents []model.Content
	query.Offset((page - 1) * pagesize).Limit(pagesize).Find(&contents)

	// 批量載入擴展字段
	extMap := make(map[uint]map[string]interface{})
	if len(contents) > 0 {
		var contentIDs []uint
		for _, ct := range contents {
			contentIDs = append(contentIDs, ct.ID)
		}
		extMap = content.GetContentExtByContentIDs(contentIDs)
		if extMap == nil {
			extMap = make(map[uint]map[string]interface{})
		}
	}

	// 構建回應
	var items []gin.H
	for _, ct := range contents {
		item := gin.H{
			"id":           ct.ID,
			"title":        ct.Title,
			"subtitle":     ct.Subtitle,
			"date":         formatTime(ct.Date),
			"ico":          ct.Ico,
			"description":  ct.Description,
			"keywords":     ct.Keywords,
			"visits":       ct.Visits,
			"likes":        ct.Likes,
			"scode":        ct.Scode,
			"istop":        ct.IsTop,
			"isrecommend":  ct.IsRecommend,
			"isheadline":   ct.IsHeadline,
			"url":          buildContentURL(&ct),
			"create_time":  formatTime(ct.CreateTime),
			"update_time":  formatTime(ct.UpdateTime),
			"ext":          extMap[ct.ID],
		}
		items = append(items, item)
	}

	apiOKWithMeta(c, items, &apiMeta{Page: page, Pagesize: pagesize, Total: total})
}

// GetContent 內容詳情
// GET /api/v1/contents/:id
func GetContent(c *gin.Context) {
	acode := getAcode(c)
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		apiFail(c, http.StatusBadRequest, "無效的 ID")
		return
	}

	var ct model.Content
	if err := model.DB.Where("acode = ? AND id = ? AND status = 1 AND date <= ?", acode, id, time.Now()).First(&ct).Error; err != nil {
		apiFail(c, http.StatusNotFound, "內容不存在")
		return
	}

	// 載入擴展字段
	extData := content.GetContentExtByContentIDs([]uint{ct.ID})[ct.ID]
	if extData == nil {
		extData = make(map[string]interface{})
	}

	// 上一篇/下一篇
	var prev, next model.Content
	model.DB.Where("acode = ? AND status = 1 AND date <= ? AND id < ? AND scode = ?", acode, time.Now(), ct.ID, ct.Scode).
		Order("id DESC").Limit(1).First(&prev)
	model.DB.Where("acode = ? AND status = 1 AND date <= ? AND id > ? AND scode = ?", acode, time.Now(), ct.ID, ct.Scode).
		Order("id ASC").Limit(1).First(&next)

	// 欄目資訊
	var sort model.ContentSort
	model.DB.Where("scode = ?", ct.Scode).First(&sort)

	apiOK(c, gin.H{
		"id":           ct.ID,
		"title":        ct.Title,
		"subtitle":     ct.Subtitle,
		"titlecolor":   ct.TitleColor,
		"author":       ct.Author,
		"source":       ct.Source,
		"date":         formatTime(ct.Date),
		"ico":          ct.Ico,
		"pics":         ct.Pics,
		"content":      ct.Content,
		"tags":         ct.Tags,
		"keywords":     ct.Keywords,
		"description":  ct.Description,
		"visits":       ct.Visits,
		"likes":        ct.Likes,
		"scode":        ct.Scode,
		"istop":        ct.IsTop,
		"isrecommend":  ct.IsRecommend,
		"isheadline":   ct.IsHeadline,
		"url":          buildContentURL(&ct),
		"create_time":  formatTime(ct.CreateTime),
		"update_time":  formatTime(ct.UpdateTime),
		"ext":          extData,
		"sort":         sort,
		"prev":         prev.ID,
		"prev_title":   prev.Title,
		"prev_url":     buildContentURL(&prev),
		"next":         next.ID,
		"next_title":   next.Title,
		"next_url":     buildContentURL(&next),
	})
}

// SearchContent 搜索內容
// GET /api/v1/search?keyword=&page=&pagesize=
func SearchContent(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		apiFail(c, http.StatusBadRequest, "請提供搜索關鍵字")
		return
	}

	acode := getAcode(c)
	page, pagesize := parsePagination(c)

	// 嘗試使用 MeiliSearch（如果已配置）
	if meiliAvailable {
		results, err := meiliSearch(keyword, acode, page, pagesize)
		if err == nil && results != nil {
			apiOKWithMeta(c, results.Hits, &apiMeta{
				Page:     page,
				Pagesize: pagesize,
				Total:    int64(results.EstimatedTotalHits),
			})
			return
		}
		// MeiliSearch 失敗則降級到 SQL LIKE
	}

	// SQL LIKE 降級搜索
	like := "%" + keyword + "%"
	query := model.DB.Model(&model.Content{}).Where("acode = ? AND status = 1 AND date <= ? AND (title LIKE ? OR keywords LIKE ? OR description LIKE ?)", acode, time.Now(), like, like, like)

	var total int64
	query.Count(&total)

	var contents []model.Content
	query.Order("date DESC, id DESC").
		Offset((page - 1) * pagesize).
		Limit(pagesize).
		Find(&contents)

	var items []gin.H
	for _, ct := range contents {
		items = append(items, gin.H{
			"id":          ct.ID,
			"title":       ct.Title,
			"description": ct.Description,
			"ico":         ct.Ico,
			"date":        formatTime(ct.Date),
			"visits":      ct.Visits,
			"scode":       ct.Scode,
			"url":         buildContentURL(&ct),
		})
	}

	apiOKWithMeta(c, items, &apiMeta{Page: page, Pagesize: pagesize, Total: total})
}

// CreateMessage 提交留言
// POST /api/v1/messages
func CreateMessage(c *gin.Context) {
	var req struct {
		Contacts string `json:"contacts" binding:"required"`
		Mobile   string `json:"mobile"`
		Content  string `json:"content" binding:"required"`
		Acode    string `json:"acode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apiFail(c, http.StatusBadRequest, "請求參數錯誤")
		return
	}

	if req.Acode == "" {
		req.Acode = getAcode(c)
	}

	msg := model.Message{
		Contacts:   req.Contacts,
		Mobile:     req.Mobile,
		Content:    req.Content,
		Acode:      req.Acode,
		Status:     0,
		CreateTime: time.Now(),
	}
	if err := model.DB.Create(&msg).Error; err != nil {
		apiFail(c, http.StatusInternalServerError, "提交失敗")
		return
	}

	apiOK(c, gin.H{"id": msg.ID})
}

// ListSlides 幻燈片列表
// GET /api/v1/slides?gid=1
func ListSlides(c *gin.Context) {
	acode := getAcode(c)
	query := model.DB.Model(&model.Slide{}).Where("acode = ? AND status = 1", acode)
	if gid := c.Query("gid"); gid != "" {
		query = query.Where("gid = ?", gid)
	}
	var slides []model.Slide
	query.Order("sorting ASC, id ASC").Find(&slides)
	apiOK(c, slides)
}

// ListLinks 友情連結列表
// GET /api/v1/links?gid=1
func ListLinks(c *gin.Context) {
	acode := getAcode(c)
	query := model.DB.Model(&model.Link{}).Where("acode = ? AND status = 1", acode)
	if gid := c.Query("gid"); gid != "" {
		query = query.Where("gid = ?", gid)
	}
	var links []model.Link
	query.Order("sorting ASC, id ASC").Find(&links)
	apiOK(c, links)
}

// ListTags 標籤列表
// GET /api/v1/tags
func ListTags(c *gin.Context) {
	acode := getAcode(c)
	var tags []model.Tags
	model.DB.Where("acode = ?", acode).Order("id DESC").Find(&tags)
	apiOK(c, tags)
}
