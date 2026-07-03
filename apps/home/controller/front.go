package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/common"
	"pbootcms-go/apps/common/mail"
	"pbootcms-go/apps/common/parser"
	"pbootcms-go/apps/common/webhook"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 統一驗證碼已移至 apps/common/captcha.go

// 留言防頻繁提交：IP → 上次提交時間
var messageRateLimit = make(map[string]time.Time)

// gboot:if 安全過濾正則（遞歸清除模板標籤注入）
var gbootIfRegex = regexp.MustCompile(`(?i)gboot:if`)

// parseUserOS 從 User-Agent 解析操作系統
func parseUserOS(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "windows nt 10"):
		return "Windows 10"
	case strings.Contains(ua, "windows nt 6.3"):
		return "Windows 8.1"
	case strings.Contains(ua, "windows nt 6.2"):
		return "Windows 8"
	case strings.Contains(ua, "windows nt 6.1"):
		return "Windows 7"
	case strings.Contains(ua, "windows nt 6.0"):
		return "Windows Vista"
	case strings.Contains(ua, "windows nt 5.1"):
		return "Windows XP"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone"):
		return "iPhone"
	case strings.Contains(ua, "ipad"):
		return "iPad"
	case strings.Contains(ua, "mac"):
		return "Mac"
	case strings.Contains(ua, "linux"):
		return "Linux"
	default:
		return "Other"
	}
}

// parseUserBrowser 從 User-Agent 解析瀏覽器
func parseUserBrowser(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "micromessenger"):
		return "Weixin"
	case strings.Contains(ua, "qq"):
		return "QQ"
	case strings.Contains(ua, "weibo"):
		return "Weibo"
	case strings.Contains(ua, "alipayclient"):
		return "Alipay"
	case strings.Contains(ua, "edg"):
		return "Edge"
	case strings.Contains(ua, "firefox"):
		return "Firefox"
	case strings.Contains(ua, "chrome") || strings.Contains(ua, "android"):
		return "Chrome"
	case strings.Contains(ua, "safari"):
		return "Safari"
	default:
		return "Other"
	}
}

type FrontController struct {
	Store *parser.TemplateStore
}

func NewFrontController(store *parser.TemplateStore) *FrontController {
	return &FrontController{Store: store}
}

// checkMustLogin 檢查模板是否含 {gboot:mustlogin} 或 {pboot:mustlogin}
// 若含且未登入，跳轉登入頁，回傳 false（呼叫者應 return）
func (fc *FrontController) checkMustLogin(c *gin.Context, content string) bool {
	if !strings.Contains(content, "mustlogin") {
		return true
	}
	// 檢查 {gboot:mustlogin} 或 {pboot:mustlogin}
	if strings.Contains(content, "{gboot:mustlogin}") || strings.Contains(content, "{pboot:mustlogin}") {
		uid := common.GetSessionInt(c, "pboot_uid")
		if uid == 0 {
			currentURL := c.Request.URL.String()
			c.Redirect(http.StatusFound, "/login?backurl="+url.QueryEscape(currentURL))
			return false
		}
	}
	return true
}

func (fc *FrontController) Index(c *gin.Context) {
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render("index.html")
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) ListPage(c *gin.Context) {
	path := c.Param("path")
	path = trimSuffix(path)

	// 優先查 filename（欄目自定義 URL 名稱），fallback 查 urlname（向後兼容）
	var sort content.ContentSort
	if err := model.DB.Where("filename = ?", path).First(&sort).Error; err != nil {
		if err2 := model.DB.Where("urlname = ?", path).First(&sort).Error; err2 != nil {
			c.String(http.StatusNotFound, "404")
			return
		}
	}

	// 欄目瀏覽權限檢查
	if !fc.checkSortPermission(c, &sort) {
		return
	}

	ctx := fc.buildContext(c)
	ctx.Sort = &sort
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		ctx.CurrentPage = p
	}
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)

	tpl := sort.ListTpl
	if tpl == "" {
		tpl = "list.html"
	}
	content := fc.Store.Render(tpl)
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) ContentPage(c *gin.Context) {
	path := c.Request.URL.Path
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimRight(path, "/")
	path = trimSuffix(path)

	if path == "" {
		fc.Index(c)
		return
	}

	// 優先查 filename（自定義 URL 名稱），fallback 查 urlname
	var ct content.Content
	if err := model.DB.Where("filename = ? AND status = 1", path).First(&ct).Error; err != nil {
		if err2 := model.DB.Where("urlname = ? AND status = 1", path).First(&ct).Error; err2 != nil {
			var sort content.ContentSort
			if err3 := model.DB.Where("filename = ?", path).First(&sort).Error; err3 != nil {
				if err4 := model.DB.Where("urlname = ?", path).First(&sort).Error; err4 != nil {
					c.String(http.StatusNotFound, "404")
					return
				}
			}
			fc.renderSortPage(c, &sort)
			return
		}
	}

	// 查欄目並做欄目權限檢查
	var sort content.ContentSort
	if model.DB.Where("scode = ?", ct.Scode).First(&sort).Error == nil {
		if !fc.checkSortPermission(c, &sort) {
			return
		}
	}

	// 內容權限檢查
	if !fc.checkContentPermission(c, &ct) {
		return
	}

	ctx := fc.buildContext(c)
	ctx.Content = &ct
	if sort.ID != 0 {
		ctx.Sort = &sort
	}

	p := parser.New()
	parser.RegisterAllProviders(p, ctx)

	tpl := "content.html"
	if ctx.Sort != nil && ctx.Sort.ContentTpl != "" {
		tpl = ctx.Sort.ContentTpl
	}
	html := fc.Store.Render(tpl)
	html = p.Render(html)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (fc *FrontController) Search(c *gin.Context) {
	ctx := fc.buildContext(c)
	ctx.Keyword = c.Query("keyword")
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		ctx.CurrentPage = p
	}
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render("search.html")
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) Tags(c *gin.Context) {
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render("tags.html")
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) Message(c *gin.Context) {
	if c.Request.Method == "POST" {
		// 留言開關檢查
		if model.GetConfigValue("message_status", "1") == "0" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "留言功能已關閉"})
			return
		}

		// 防頻繁提交：同一 IP 10 秒內只能提交一次
		clientIP := c.ClientIP()
		if submitTime, ok := messageRateLimit[clientIP]; ok {
			if time.Since(submitTime) < 10*time.Second {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "提交太頻繁，請稍後再試"})
				return
			}
		}

		// 蜜罐 + 時間陷阱反垃圾檢查
		if !fc.checkAntispam(c) {
			return
		}

		// pboot:if 安全過濾
		filterGbootIf := func(s string) string {
			return gbootIfRegex.ReplaceAllString(s, "")
		}

		// 區分 fcode：fcode=1 或無 fcode → 留言(ay_message)，fcode≥2 → 自定義表單(ay_diy_*)
		fcode := c.PostForm("fcode")
		if fcode != "" && fcode != "1" {
			// 自定義表單提交
			fc.handleFormSubmit(c, fcode, clientIP, filterGbootIf)
			return
		}

		// 留言需登錄檢查（message_rqlogin 配置啟用時，未登錄會員不可留言）
		if model.GetConfigValue("message_rqlogin", "0") == "1" {
			uid := common.GetSessionInt(c, "pboot_uid")
			if uid == 0 {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "請先註冊登錄後再留言！"})
				return
			}
		}

		// 統一驗證碼校驗（message_check_code 默認啟用）
		if !common.VerifyCaptcha(c, "message_check_code", "1") {
			return
		}

		msg := model.Message{
			Acode:      "cn",
			Contacts:   filterGbootIf(c.PostForm("contacts")),
			Mobile:     filterGbootIf(c.PostForm("mobile")),
			Content:    filterGbootIf(c.PostForm("content")),
			IP:         clientIP,
			OS:         parseUserOS(c.Request.UserAgent()),
			Browser:    parseUserBrowser(c.Request.UserAgent()),
			CreateTime: time.Now(),
			Status:     0,
			CreateUser: "guest",
			UpdateUser: "guest",
		}
		// 審核狀態：message_verify='0' 時直接通過
		if model.GetConfigValue("message_verify", "1") == "0" {
			msg.Status = 1
		}

		if err := model.DB.Create(&msg).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "提交失敗"})
			return
		}

		// 郵件通知 + Webhook 推送（message_send_mail=1 時啟用）
		if model.GetConfigValue("message_send_mail", "0") == "1" {
			notifyFields := []map[string]string{
				{"label": "聯繫人", "value": msg.Contacts},
				{"label": "手機", "value": msg.Mobile},
				{"label": "內容", "value": msg.Content},
				{"label": "IP", "value": msg.IP},
				{"label": "操作系統", "value": msg.OS},
				{"label": "瀏覽器", "value": msg.Browser},
			}
			mail.SendNotifyMail("在線留言", notifyFields)
			webhook.SendIf("message", "在線留言", msg.IP, msg.OS, msg.Browser, notifyFields)
		}

		messageRateLimit[clientIP] = time.Now()
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "提交成功"})
		return
	}
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render("message.html")
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

// handleFormSubmit 處理自定義表單提交（fcode≥2，寫入動態表 ay_diy_*）
func (fc *FrontController) handleFormSubmit(c *gin.Context, fcode, clientIP string, filterGbootIf func(string) string) {
	// 蜜罐 + 時間陷阱反垃圾檢查（與留言共用）
	if !fc.checkAntispam(c) {
		return
	}

	// 查 ay_form 獲取 table_name
	form := content.GetFormByCode(fcode)
	if form == nil || form.Table == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "表單不存在"})
		return
	}
	tableName := form.Table

	// 統一驗證碼校驗（form_check_code 默認關閉）
	if !common.VerifyCaptcha(c, "form_check_code", "0") {
		return
	}

	// 查 ay_form_field 獲取字段定義
	fields := content.GetFormFieldByCode(fcode)
	if len(fields) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "表單字段不存在"})
		return
	}

	// 動態收集 POST 數據 + 必填校驗
	cols := []string{"create_time"}
	vals := []interface{}{time.Now().Format("2006-01-02 15:04:05")}
	placeholders := []string{"?"}
	var notifyFields []map[string]string

	for _, f := range fields {
		fieldName := f.Name
		if fieldName == "" {
			continue
		}
		val := filterGbootIf(c.PostForm(fieldName))
		// 多選框轉逗號分隔
		if arr := c.PostFormArray(fieldName + "[]"); len(arr) > 0 {
			val = strings.Join(arr, ",")
		}
		if f.Required == 1 && val == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": f.Field + "不能為空"})
			return
		}
		cols = append(cols, fieldName)
		vals = append(vals, val)
		placeholders = append(placeholders, "?")
		notifyFields = append(notifyFields, map[string]string{"label": f.Field, "value": val})
	}

	// 動態 INSERT（參數化查詢）
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName, strings.Join(cols, ","), strings.Join(placeholders, ","))
	if err := model.DB.Exec(sql, vals...).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "提交失敗"})
		return
	}

	// 郵件通知 + Webhook 推送（form_send_mail=1 時啟用）
	if model.GetConfigValue("form_send_mail", "0") == "1" {
		mail.SendNotifyMail(form.FormName, notifyFields)
		webhook.SendIf("form", form.FormName, clientIP, "", "", notifyFields)
	}

	messageRateLimit[clientIP] = time.Now()
	c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "提交成功"})
}

func (fc *FrontController) Visits(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	if id > 0 {
		model.DB.Model(&content.Content{}).Where("id = ?", id).
			UpdateColumn("visits", gorm.Expr("visits + 1"))
	}
	c.String(http.StatusOK, "ok")
}

// checkAntispam 蜜罐 + 時間陷阱通用反垃圾檢查
// 返回 true 表示通過（非垃圾），false 表示已攔截（已寫入 JSON 響應）
// 適用於留言、自定義表單等公開提交場景；後台登錄不適用（蜜罐易被密碼管理器誤填）
func (fc *FrontController) checkAntispam(c *gin.Context) bool {
	// 蜜罐欄位：機器人會自動填充隱藏欄位，正常用戶不會
	if honeypot := c.PostForm("website"); honeypot != "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "提交失敗"})
		return false
	}
	// 時間陷阱：提交間隔 <3 秒判定為機器人
	if loadts := c.PostForm("_loadts"); loadts != "" {
		if ts, err := strconv.ParseInt(loadts, 10, 64); err == nil {
			if time.Now().Unix()-ts < 3 {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "提交失敗"})
				return false
			}
		}
	}
	return true
}

// CheckCode 前台驗證碼生成（存入 messageCodeStore，供留言校驗）
func (fc *FrontController) CheckCode(c *gin.Context) {
	common.GenerateCaptcha(c)
}

// SortByScode renders the list page for a sort identified by its scode
// (the stable PbootCMS code, never changes once assigned).
// Used as the dynamic URL "/sort/{scode}" generated by sortToMap
// when a sort has no urlname set.
func (fc *FrontController) SortByScode(c *gin.Context) {
	scode := c.Param("scode")
	var sort content.ContentSort
	if err := model.DB.Where("scode = ?", scode).First(&sort).Error; err != nil {
		c.String(http.StatusNotFound, "404")
		return
	}
	// 欄目瀏覽權限檢查
	if !fc.checkSortPermission(c, &sort) {
		return
	}
	fc.renderSortPage(c, &sort)
}

// ContentByID renders the content detail page for a content identified by id.
// Used as the dynamic URL "/content/{id}" generated by contentToMap
// when a content has no urlname set.
func (fc *FrontController) ContentByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.String(http.StatusNotFound, "404")
		return
	}
	var ct content.Content
	if err := model.DB.Where("id = ? AND status = 1", id).First(&ct).Error; err != nil {
		c.String(http.StatusNotFound, "404")
		return
	}
	var sort content.ContentSort
	_ = model.DB.Where("scode = ?", ct.Scode).First(&sort).Error

	// 欄目權限檢查
	if sort.ID != 0 {
		if !fc.checkSortPermission(c, &sort) {
			return
		}
	}

	// 內容權限檢查
	if !fc.checkContentPermission(c, &ct) {
		return
	}

	ctx := fc.buildContext(c)
	ctx.Content = &ct
	if sort.ID != 0 {
		ctx.Sort = &sort
	}
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)

	tpl := "content.html"
	if sort.ID != 0 && sort.ContentTpl != "" {
		tpl = sort.ContentTpl
	}
	html := fc.Store.Render(tpl)
	html = p.Render(html)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (fc *FrontController) renderSortPage(c *gin.Context, sort *content.ContentSort) {
	// 欄目權限檢查（renderSortPage 被 SortByScode/ContentPage 內部調用時可能已檢查過，
	// 但 ContentPage 的 fallback 路徑未檢查，此處統一確保）
	// 注意：SortByScode 已在調用前檢查，此處重複檢查不影響正確性（gid=0 時直接通過）
	if !fc.checkSortPermission(c, sort) {
		return
	}

	ctx := fc.buildContext(c)
	ctx.Sort = sort
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		ctx.CurrentPage = p
	}
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)

	// 通過 mcode 查 ay_model 獲取 type
	var tpl string
	var contentModel model.ContentModel
	if sort.Mcode != "" && model.DB.Where("mcode = ?", sort.Mcode).First(&contentModel).Error == nil {
		if contentModel.Type == 1 {
			// 單頁模型 → 用 ContentTpl (如 about.html)
			tpl = sort.ContentTpl
			// 單頁需要加載內容數據
			var ct content.Content
			if model.DB.Where("scode = ? AND status = 1", sort.Scode).Order("id DESC").First(&ct).Error == nil {
				// 單頁內容權限檢查
				if !fc.checkContentPermission(c, &ct) {
					return
				}
				ctx.Content = &ct
			}
		} else {
			// 列表模型 → 用 ListTpl
			tpl = sort.ListTpl
		}
	} else {
		tpl = sort.ListTpl
	}
	if tpl == "" {
		tpl = "list.html"
	}

	content := fc.Store.Render(tpl)
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

// loadGcode 透過 gid 查詢 ay_member_group 取得 gcode（等級編號）
func loadGcode(gid string) string {
	if gid == "" || gid == "0" {
		return ""
	}
	var gcode string
	model.DB.Table("ay_member_group").Where("id = ?", gid).Select("gcode").Row().Scan(&gcode)
	return gcode
}

// checkPageLevel 檢查頁面瀏覽權限（對應 PHP IndexController::checkPageLevel）
// requiredGcode: 欄目/內容要求的等級編號（透過 JOIN ay_member_group 取得）
// gtype: 比較運算子（1:<= / 2:< / 3:!= / 4:> / 5:>=，預設4）
// gnote: 權限不足提示文字
// 回傳 true 表示通過，false 表示被拒絕（已寫入 response）
func (fc *FrontController) checkPageLevel(c *gin.Context, requiredGcode, gtype, gnote string) bool {
	if requiredGcode == "" || requiredGcode == "0" {
		return true // 無權限要求
	}

	// 訪客的等級編號
	visitorGcode := common.GetSessionInt(c, "pboot_gcode")
	uid := common.GetSessionInt(c, "pboot_uid")

	// gtype 預設 4
	gt, _ := strconv.Atoi(gtype)
	if gt == 0 {
		gt = 4
	}
	required, _ := strconv.Atoi(requiredGcode)

	deny := false
	switch gt {
	case 1:
		if required <= visitorGcode {
			deny = true
		}
	case 2:
		if required < visitorGcode {
			deny = true
		}
	case 3:
		if required != visitorGcode {
			deny = true
		}
	case 4:
		if required > visitorGcode {
			deny = true
		}
	case 5:
		if required >= visitorGcode {
			deny = true
		}
	}

	if !deny {
		return true
	}

	// 權限不足
	if gnote == "" {
		gnote = "您的權限不足，無法瀏覽本頁面！"
	}

	if uid > 0 {
		// 已登入但權限不足 → 顯示提示
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, fmt.Sprintf(`<div style="text-align:center;padding:80px 20px;"><h3>%s</h3><p><a href="/">返回首頁</a></p></div>`, gnote))
	} else {
		// 未登入 → 跳轉登入頁，帶 backurl（URL-encode 避免查詢參數被截斷）
		currentURL := c.Request.URL.String()
		c.Redirect(http.StatusFound, "/login?backurl="+url.QueryEscape(currentURL))
	}
	return false
}

// checkSortPermission 檢查欄目權限（載入 gcode + 呼叫 checkPageLevel）
func (fc *FrontController) checkSortPermission(c *gin.Context, sort *content.ContentSort) bool {
	if sort.Gid == "" || sort.Gid == "0" {
		return true
	}
	sort.Gcode = loadGcode(sort.Gid)
	return fc.checkPageLevel(c, sort.Gcode, sort.GType, sort.Gnote)
}

// checkContentPermission 檢查內容權限（載入 gcode + 呼叫 checkPageLevel）
func (fc *FrontController) checkContentPermission(c *gin.Context, ct *content.Content) bool {
	if ct.Gid == "" || ct.Gid == "0" {
		return true
	}
	ct.Gcode = loadGcode(ct.Gid)
	return fc.checkPageLevel(c, ct.Gcode, ct.GType, ct.Gnote)
}

func (fc *FrontController) buildContext(c *gin.Context) *parser.Context {
	ctx := &parser.Context{
		Page:    make(map[string]interface{}),
		Filters: make(map[string]string),
	}

	// 收集 ext_ 查詢參數供篩選使用
	for key, vals := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "ext_") && len(vals) > 0 && vals[0] != "" {
			ctx.Filters[key] = vals[0]
		}
	}

	var site model.Site
	if model.DB.First(&site).Error == nil {
		ctx.Site = &site
	}

	var company model.Company
	if model.DB.First(&company).Error == nil {
		ctx.Company = &company
	}

	// 載入已登錄會員資訊
	if uid := common.GetSessionInt(c, "pboot_uid"); uid > 0 {
		var member model.Member
		if model.DB.First(&member, uid).Error == nil {
			ctx.Member = &member
			ctx.Gcode = common.GetSessionInt(c, "pboot_gcode")
			ctx.Ucode = common.GetSessionString(c, "pboot_ucode")
		}
	}

	return ctx
}

func trimSuffix(s string) string {
	return strings.TrimSuffix(strings.TrimSuffix(s, ".html"), ".htm")
}

// === 以下驗證碼繪製輔助函數已移至 apps/common/captcha.go 統一管理 ===
