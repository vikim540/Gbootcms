package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gbootcms/apps/admin/model"
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/common"
	"gbootcms/core/acodeplugin"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// dbCtx 返回帶當前請求 context 的 DB 實例，使 AcodePlugin 自動注入 acode 過濾
func dbCtx(c *gin.Context) *gorm.DB {
	return model.DB.WithContext(c.Request.Context())
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
	var site model.Site
	if err := dbCtx(c).First(&site).Error; err != nil {
		slog.Warn("API GetSite 查詢失敗", "error", err, "acode", acodeplugin.GetAcode(c.Request.Context()))
	}

	apiOK(c, gin.H{
		"title":       site.Title,
		"subtitle":    site.Subtitle,
		"domain":      site.Domain,
		"logo":        site.Logo,
		"keywords":    site.Keywords,
		"description": site.Description,
		"icp":         site.ICP,
		"theme":       site.Theme,
		"acode":       acodeplugin.GetAcode(c.Request.Context()),
	})
}

// GetCompany 公司資訊
// GET /api/v1/company
func GetCompany(c *gin.Context) {
	var company model.Company
	if err := dbCtx(c).First(&company).Error; err != nil {
		slog.Warn("API GetCompany 查詢失敗", "error", err, "acode", acodeplugin.GetAcode(c.Request.Context()))
	}

	apiOK(c, gin.H{
		"name":    company.Name,
		"address": company.Address,
		"phone":   company.Phone,
		"mobile":  company.Mobile,
		"email":   company.Email,
		"qq":      company.Qq,
		"wechat":  company.Weixin,
	})
}

// ListSorts 欄目列表
// GET /api/v1/sorts?scode=&mcode=&status=1
func ListSorts(c *gin.Context) {
	query := dbCtx(c).Model(&model.ContentSort{})

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

	var sorts []model.ContentSort = []model.ContentSort{}
	if err := query.Order("sorting ASC, id ASC").Find(&sorts).Error; err != nil {
		slog.Warn("API ListSorts 查詢失敗", "error", err)
	}

	apiOK(c, sorts)
}

// GetSort 欄目詳情
// GET /api/v1/sorts/:scode
func GetSort(c *gin.Context) {
	scode := c.Param("scode")
	if scode == "" {
		apiFail(c, http.StatusBadRequest, "請提供欄目編號 scode")
		return
	}
	var sort model.ContentSort
	if err := dbCtx(c).Where("scode = ? OR filename = ? OR urlname = ?", scode, scode, scode).First(&sort).Error; err != nil {
		apiFail(c, http.StatusNotFound, "欄目不存在")
		return
	}
	apiOK(c, sort)
}

// ListNav 導航樹
// GET /api/v1/nav?scode=
func ListNav(c *gin.Context) {
	query := dbCtx(c).Model(&model.ContentSort{}).Where("status = 1")

	if scode := c.Query("scode"); scode != "" {
		query = query.Where("scode = ? OR pcode = ?", scode, scode)
	}

	var sorts []model.ContentSort = []model.ContentSort{}
	if err := query.Order("sorting ASC, id ASC").Find(&sorts).Error; err != nil {
		slog.Warn("API ListNav 查詢失敗", "error", err)
	}
	type navItem struct {
		ID       uint      `json:"id"`
		Scode    string    `json:"scode"`
		Pcode    string    `json:"pcode"`
		Name     string    `json:"name"`
		Filename string    `json:"filename"`
		URLName  string    `json:"urlname"`
		Mcode    string    `json:"mcode"`
		Listtpl  string    `json:"listtpl"`
		Content  string    `json:"contenttpl"`
		ICO      string    `json:"ico"`
		Pic      string    `json:"pic"`
		Sorting  int       `json:"sorting"`
		Children []navItem `json:"children,omitempty"`
	}

	var buildNav func(items []model.ContentSort, parentCode string) []navItem
	buildNav = func(items []model.ContentSort, parentCode string) []navItem {
		result := []navItem{}
		for _, s := range items {
			if s.Pcode == parentCode {
				item := navItem{
					ID:       s.ID,
					Scode:    s.Scode,
					Pcode:    s.Pcode,
					Name:     s.Name,
					Filename: s.Filename,
					URLName:  s.URLName,
					Mcode:    s.Mcode,
					Listtpl:  s.ListTpl,
					Content:  s.ContentTpl,
					ICO:      s.Ico,
					Pic:      s.Pic,
					Sorting:  s.Sort,
				}
				item.Children = buildNav(items, s.Scode)
				result = append(result, item)
			}
		}
		return result
	}

	tree := buildNav(sorts, "0")
	apiOK(c, tree)
}

// ListContents 內容列表
// GET /api/v1/contents?scode=&mcode=&keyword=&page=&pagesize=&istop=&isrecommend=&order=
func ListContents(c *gin.Context) {
	page, pagesize := parsePagination(c)

	query := dbCtx(c).Model(&model.Content{}).Where("status = 1 AND date <= ?", time.Now())

	if scode := c.Query("scode"); scode != "" {
		// 遞迴 CTE 查詢欄目及其所有子孫欄目（與前台 findAllChildScodes 行為一致）
		allScodes := findAllChildScodesAPI(c, scode)
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

	// Count 時不需要 ORDER BY，提升效能
	var total int64
	if err := query.Session(&gorm.Session{}).Order("").Count(&total).Error; err != nil {
		slog.Warn("API ListContents Count 失敗", "error", err)
	}

	var contents []model.Content
	if err := query.Offset((page - 1) * pagesize).Limit(pagesize).Find(&contents).Error; err != nil {
		slog.Warn("API ListContents Find 失敗", "error", err)
	}

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
	items := []gin.H{}
	for _, ct := range contents {
		item := gin.H{
			"id":          ct.ID,
			"title":       ct.Title,
			"subtitle":    ct.Subtitle,
			"date":        formatTime(ct.Date),
			"ico":         ct.Ico,
			"description": ct.Description,
			"keywords":    ct.Keywords,
			"visits":      ct.Visits,
			"likes":       ct.Likes,
			"scode":       ct.Scode,
			"istop":       ct.IsTop,
			"isrecommend": ct.IsRecommend,
			"isheadline":  ct.IsHeadline,
			"url":         buildContentURL(&ct),
			"create_time": formatTime(ct.CreateTime),
			"update_time": formatTime(ct.UpdateTime),
			"ext":         extMap[ct.ID],
		}
		items = append(items, item)
	}

	apiOKWithMeta(c, items, &apiMeta{Page: page, Pagesize: pagesize, Total: total})
}

// GetContent 內容詳情
// GET /api/v1/contents/:id?track=1
// track=1 時累加訪問量（預設不計數，避免 API 輪詢污染統計）
func GetContent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		apiFail(c, http.StatusBadRequest, "無效的 ID")
		return
	}

	var ct model.Content
	if err := dbCtx(c).Where("id = ? AND status = 1 AND date <= ?", id, time.Now()).First(&ct).Error; err != nil {
		apiFail(c, http.StatusNotFound, "內容不存在")
		return
	}

	// 可選訪問量追蹤（使用 .Exec() 繞過 GORM 回調，避免觸發快取失效）
	if c.Query("track") == "1" {
		// 使用 .Exec() 原始 SQL 繞過 GORM 回調（避免觸發快取失效）
		if err := dbCtx(c).Exec("UPDATE ay_content SET visits = visits + 1 WHERE id = ?", ct.ID).Error; err != nil {
			slog.Warn("API 訪問量更新失敗", "content_id", ct.ID, "error", err)
		}
	}

	// 載入擴展字段
	extData := content.GetContentExtByContentIDs([]uint{ct.ID})[ct.ID]
	if extData == nil {
		extData = make(map[string]interface{})
	}

	// 上一篇/下一篇
	var prev, next model.Content
	hasPrev := dbCtx(c).Where("status = 1 AND date <= ? AND id < ? AND scode = ?", time.Now(), ct.ID, ct.Scode).
		Order("id DESC").Limit(1).First(&prev).Error == nil
	hasNext := dbCtx(c).Where("status = 1 AND date <= ? AND id > ? AND scode = ?", time.Now(), ct.ID, ct.Scode).
		Order("id ASC").Limit(1).First(&next).Error == nil

	// 欄目資訊
	var sort model.ContentSort
	if err := dbCtx(c).Where("scode = ?", ct.Scode).First(&sort).Error; err != nil {
		slog.Warn("API GetContent 欄目查詢失敗", "scode", ct.Scode, "error", err)
	}

	// 構建 prev/next 回應（不存在時為 null）
	var prevData, nextData interface{}
	if hasPrev {
		prevData = gin.H{"id": prev.ID, "title": prev.Title, "url": buildContentURL(&prev)}
	}
	if hasNext {
		nextData = gin.H{"id": next.ID, "title": next.Title, "url": buildContentURL(&next)}
	}

	apiOK(c, gin.H{
		"id":          ct.ID,
		"title":       ct.Title,
		"subtitle":    ct.Subtitle,
		"titlecolor":  ct.TitleColor,
		"author":      ct.Author,
		"source":      ct.Source,
		"date":        formatTime(ct.Date),
		"ico":         ct.Ico,
		"pics":        ct.Pics,
		"content":     ct.Content,
		"tags":        ct.Tags,
		"keywords":    ct.Keywords,
		"description": ct.Description,
		"visits":      ct.Visits,
		"likes":       ct.Likes,
		"scode":       ct.Scode,
		"istop":       ct.IsTop,
		"isrecommend": ct.IsRecommend,
		"isheadline":  ct.IsHeadline,
		"url":         buildContentURL(&ct),
		"create_time": formatTime(ct.CreateTime),
		"update_time": formatTime(ct.UpdateTime),
		"ext":         extData,
		"sort":        sort,
		"prev":        prevData,
		"next":        nextData,
	})
}

// GetContentImages 內容附件圖片
// GET /api/v1/contents/:id/images
func GetContentImages(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		apiFail(c, http.StatusBadRequest, "無效的 ID")
		return
	}

	var ct model.Content
	if err := dbCtx(c).Where("id = ? AND status = 1", id).First(&ct).Error; err != nil {
		apiFail(c, http.StatusNotFound, "內容不存在")
		return
	}

	var images []string = []string{}
	if ct.Pics != "" {
		for _, p := range strings.Split(ct.Pics, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				images = append(images, p)
			}
		}
	}

	apiOK(c, gin.H{
		"id":     ct.ID,
		"title":  ct.Title,
		"ico":    ct.Ico,
		"images": images,
	})
}

// SearchContent 搜索內容
// GET /api/v1/search?keyword=&field=title|keywords|description&fuzzy=1&page=&pagesize=
func SearchContent(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		apiFail(c, http.StatusBadRequest, "請提供搜索關鍵字")
		return
	}

	page, pagesize := parsePagination(c)

	// 嘗試使用 MeiliSearch（如果已配置）
	if meiliAvailable {
		results, err := meiliSearch(keyword, acodeplugin.GetAcode(c.Request.Context()), page, pagesize)
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

	// 解析搜索字段（預設 title+keywords+description）
	field := c.DefaultQuery("field", "title|keywords|description")
	fuzzy := c.DefaultQuery("fuzzy", "1")

	var whereClause string
	var args []interface{}

	if fuzzy == "0" {
		// 精準匹配
		fields := strings.Split(field, "|")
		conditions := make([]string, 0, len(fields))
		for _, f := range fields {
			f = strings.TrimSpace(f)
			if f == "title" || f == "keywords" || f == "description" {
				conditions = append(conditions, f+" = ?")
				args = append(args, keyword)
			}
		}
		if len(conditions) > 0 {
			whereClause = "(" + strings.Join(conditions, " OR ") + ")"
		} else {
			whereClause = "title = ?"
			args = append(args, keyword)
		}
	} else {
		// 模糊匹配
		like := "%" + keyword + "%"
		fields := strings.Split(field, "|")
		conditions := make([]string, 0, len(fields))
		for _, f := range fields {
			f = strings.TrimSpace(f)
			if f == "title" || f == "keywords" || f == "description" {
				conditions = append(conditions, f+" LIKE ?")
				args = append(args, like)
			}
		}
		if len(conditions) > 0 {
			whereClause = "(" + strings.Join(conditions, " OR ") + ")"
		} else {
			whereClause = "title LIKE ?"
			args = append(args, like)
		}
	}

	query := dbCtx(c).Model(&model.Content{}).Where("status = 1 AND date <= ? AND "+whereClause, append([]interface{}{time.Now()}, args...)...)

	var total int64
	if err := query.Session(&gorm.Session{}).Order("").Count(&total).Error; err != nil {
		slog.Warn("API SearchContent Count 失敗", "error", err)
	}

	var contents []model.Content
	if err := query.Order("date DESC, id DESC").
		Offset((page - 1) * pagesize).
		Limit(pagesize).
		Find(&contents).Error; err != nil {
		slog.Warn("API SearchContent Find 失敗", "error", err)
	}

	items := []gin.H{}
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
	// 速率限制：同一 IP 60 秒內最多 3 次留言
	if !checkMessageRate(c) {
		apiFail(c, http.StatusTooManyRequests, "提交過於頻繁，請稍後再試")
		return
	}

	var req struct {
		Contacts string `json:"contacts" binding:"required"`
		Mobile   string `json:"mobile"`
		Content  string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apiFail(c, http.StatusBadRequest, "請求參數錯誤")
		return
	}

	// XSS 過濾 + 敏感詞替換（與前台邏輯完全統一）
	contacts := common.FilterSensitiveWords(common.FilterUserInput(req.Contacts))
	mobile := common.FilterUserInput(req.Mobile)
	contentText := common.FilterSensitiveWords(common.FilterUserInput(req.Content))

	// 採集訪客基礎資訊
	clientIP := c.ClientIP()
	ua := c.Request.UserAgent()
	chPlatformVer := c.GetHeader("Sec-CH-UA-Platform-Version")
	osName, bsName := common.ParseUserAgent(ua, chPlatformVer)

	// 會員 UID（如果攜帶了 JWT Token）
	uid := 0
	if v, exists := c.Get("api_uid"); exists {
		uid = v.(int)
	}

	msg := model.Message{
		Contacts:   contacts,
		Mobile:     mobile,
		Content:    contentText,
		IP:         clientIP,
		OS:         osName,
		Browser:    bsName,
		UID:        uid,
		Status:     0,
		CreateTime: time.Now(),
	}
	if err := dbCtx(c).Create(&msg).Error; err != nil {
		apiFail(c, http.StatusInternalServerError, "提交失敗")
		return
	}

	apiOK(c, gin.H{"id": msg.ID})
}

// ListMessages 留言列表（需認證）
// GET /api/v1/messages?page=&pagesize=&status=
func ListMessages(c *gin.Context) {
	page, pagesize := parsePagination(c)

	query := dbCtx(c).Model(&model.Message{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		apiFail(c, http.StatusInternalServerError, "查詢失敗")
		return
	}

	var messages []model.Message = []model.Message{}
	if err := query.Order("id DESC").
		Offset((page - 1) * pagesize).
		Limit(pagesize).
		Find(&messages).Error; err != nil {
		apiFail(c, http.StatusInternalServerError, "查詢失敗")
		return
	}

	// 過濾敏感字段，只返回前端需要的資訊
	items := []gin.H{}
	for _, m := range messages {
		items = append(items, gin.H{
			"id":          m.ID,
			"contacts":    m.Contacts,
			"mobile":      m.Mobile,
			"content":     m.Content,
			"status":      m.Status,
			"recontent":   m.ReContent,
			"uid":         m.UID,
			"create_time": formatTime(m.CreateTime),
			"update_time": formatTime(m.UpdateTime),
		})
	}

	apiOKWithMeta(c, items, &apiMeta{Page: page, Pagesize: pagesize, Total: total})
}

// ListFormFields 表單字段定義（需認證）
// GET /api/v1/forms/:fcode/fields
func ListFormFields(c *gin.Context) {
	fcode := c.Param("fcode")
	if fcode == "" {
		apiFail(c, http.StatusBadRequest, "請提供表單編號 fcode")
		return
	}

	form := content.GetFormByCode(fcode)
	if form == nil {
		apiFail(c, http.StatusNotFound, "表單不存在")
		return
	}

	fields := content.GetFormFieldByCode(fcode)

	apiOK(c, gin.H{
		"form":   form,
		"fields": fields,
	})
}

// ListFormData 表單數據列表（需認證）
// GET /api/v1/forms/:fcode/data?page=&pagesize=
func ListFormData(c *gin.Context) {
	fcode := c.Param("fcode")
	if fcode == "" {
		apiFail(c, http.StatusBadRequest, "請提供表單編號 fcode")
		return
	}

	page, pagesize := parsePagination(c)

	form := content.GetFormByCode(fcode)
	if form == nil {
		apiFail(c, http.StatusNotFound, "表單不存在")
		return
	}

	tableName := form.TableName
	if tableName == "" {
		apiFail(c, http.StatusNotFound, "表單未配置數據表")
		return
	}

	// 安全驗證表名（白名單）
	if !common.CheckVarType(tableName) {
		apiFail(c, http.StatusBadRequest, "非法表名")
		return
	}

	var total int64
	if err := dbCtx(c).Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&total).Error; err != nil {
		apiFail(c, http.StatusInternalServerError, "查詢失敗")
		return
	}

	offset := (page - 1) * pagesize
	results := []map[string]interface{}{}
	if err := dbCtx(c).Raw(fmt.Sprintf("SELECT * FROM %s ORDER BY id DESC LIMIT ? OFFSET ?", tableName), pagesize, offset).Scan(&results).Error; err != nil {
		apiFail(c, http.StatusInternalServerError, "查詢失敗")
		return
	}

	apiOKWithMeta(c, results, &apiMeta{Page: page, Pagesize: pagesize, Total: total})
}

// ListSlides 幻燈片列表
// GET /api/v1/slides?gid=1
func ListSlides(c *gin.Context) {
	query := dbCtx(c).Model(&model.Slide{})
	if gid := c.Query("gid"); gid != "" {
		query = query.Where("gid = ?", gid)
	}
	var slides []model.Slide = []model.Slide{}
	if err := query.Order("sorting ASC, id ASC").Find(&slides).Error; err != nil {
		slog.Warn("API ListSlides 查詢失敗", "error", err)
	}
	apiOK(c, slides)
}

// ListLinks 友情連結列表
// GET /api/v1/links?gid=1
func ListLinks(c *gin.Context) {
	query := dbCtx(c).Model(&model.Link{})
	if gid := c.Query("gid"); gid != "" {
		query = query.Where("gid = ?", gid)
	}
	var links []model.Link = []model.Link{}
	if err := query.Order("sorting ASC, id ASC").Find(&links).Error; err != nil {
		slog.Warn("API ListLinks 查詢失敗", "error", err)
	}
	apiOK(c, links)
}

// ListTags 標籤列表
// GET /api/v1/tags
func ListTags(c *gin.Context) {
	var tags []model.Tags = []model.Tags{}
	if err := dbCtx(c).Order("id DESC").Find(&tags).Error; err != nil {
		slog.Warn("API ListTags 查詢失敗", "error", err)
	}
	apiOK(c, tags)
}

// findAllChildScodesAPI 使用遞迴 CTE 查詢指定欄目及其所有子孫欄目的 scode
// 與前台 parser.findAllChildScodes 邏輯一致，確保 API 和前台返回相同的內容範圍
func findAllChildScodesAPI(c *gin.Context, parentScode string) []string {
	query := `WITH RECURSIVE descendants AS (
		SELECT scode FROM ay_content_sort WHERE scode = ? AND status = 1
		UNION ALL
		SELECT s.scode FROM ay_content_sort s
		INNER JOIN descendants d ON s.pcode = d.scode
		WHERE s.status = 1
	)
	SELECT scode FROM descendants`

	var rows []struct {
		Scode string
	}
	if err := dbCtx(c).Raw(query, parentScode).Scan(&rows).Error; err != nil {
		slog.Warn("API findAllChildScodesAPI 查詢失敗", "error", err, "parent_scode", parentScode)
	}

	result := make([]string, 0, len(rows))
	for _, r := range rows {
		result = append(result, r.Scode)
	}
	if len(result) == 0 {
		return []string{parentScode}
	}
	return result
}
