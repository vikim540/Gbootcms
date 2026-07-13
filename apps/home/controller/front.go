package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/common"
	"gbootcms/apps/common/mail"
	"gbootcms/apps/common/middleware"
	"gbootcms/apps/common/parser"
	"gbootcms/apps/common/webhook"
	"gbootcms/core/acodeplugin"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 統一驗證碼已移至 apps/common/captcha.go

// rateLimitInterval 頻率限制間隔（秒）
const rateLimitInterval = 60

// rateLimitTTL 頻率限制條目存活時間（過期後自動清理，防止內存洩漏）
const rateLimitTTL = 10 * time.Minute

// messageRateLimit 留言防頻繁提交：IP → 上次提交時間（併發安全）
var messageRateLimit sync.Map

// cleanupRateLimit 清理過期的頻率限制條目，防止內存洩漏
func cleanupRateLimit() {
	now := time.Now()
	messageRateLimit.Range(func(key, value any) bool {
		if t, ok := value.(time.Time); ok && now.Sub(t) > rateLimitTTL {
			messageRateLimit.Delete(key)
		}
		return true
	})
}

// parseUserOS 和 parseUserBrowser 已合併到 common.ParseUserAgent（消除重複造輪子）

type FrontController struct {
	Store  *parser.TemplateStore // 預設模板（default）
	stores map[string]*parser.TemplateStore // 各語言模板（按 theme 目錄名索引）
	mu     sync.RWMutex
}

func NewFrontController(store *parser.TemplateStore) *FrontController {
	return &FrontController{Store: store, stores: make(map[string]*parser.TemplateStore)}
}

// getStore 根據當前請求的 acode 取得對應的 TemplateStore
// 邏輯：acode → 查 ay_site.theme → 如果 theme 目錄有對應 TemplateStore 則用它，否則用 default
func (fc *FrontController) getStore(c *gin.Context) *parser.TemplateStore {
	acode := acodeplugin.GetAcode(c.Request.Context())
	if acode == "" {
		return fc.Store
	}

	// 查詢當前 acode 對應的 theme
	var site model.Site
	if model.DB.WithContext(c.Request.Context()).First(&site).Error != nil {
		return fc.Store
	}
	theme := site.Theme
	if theme == "" || theme == "default" {
		return fc.Store
	}

	// 檢查是否已有快取的 store
	fc.mu.RLock()
	s, ok := fc.stores[theme]
	fc.mu.RUnlock()
	if ok {
		return s
	}

	// 嘗試載入該 theme 的模板目錄
	themeDir := "template/" + theme
	if _, err := os.Stat(themeDir); os.IsNotExist(err) {
		// theme 目錄不存在，fallback 到 default
		return fc.Store
	}

	tagParser := parser.New()
	newStore, err := parser.NewTemplateStore(themeDir, tagParser)
	if err != nil || newStore == nil {
		return fc.Store
	}

	fc.mu.Lock()
	fc.stores[theme] = newStore
	fc.mu.Unlock()
	return newStore
}

// langPath 根據當前請求的語言區域，為路徑添加語言前綴
// 默認區域返回原路徑（如 "/" → "/"），非默認區域添加前綴（如 "/" → "/sc/"）
// 用於重定向 URL 和 AJAX tourl，確保語言上下文在導航中不丟失
func langPath(c *gin.Context, path string) string {
	acode := acodeplugin.GetAcode(c.Request.Context())
	if acode == "" || acode == middleware.GetDefaultAcode() {
		return path
	}
	// 路徑已含前綴則不重複添加
	if strings.HasPrefix(path, "/"+acode+"/") || strings.HasPrefix(path, "/"+acode+"?") {
		return path
	}
	return "/" + acode + path
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
			backurl := langPath(c, currentURL)
			c.Redirect(http.StatusFound, langPath(c, "/login")+"?backurl="+url.QueryEscape(backurl))
			return false
		}
	}
	return true
}

func (fc *FrontController) Index(c *gin.Context) {
	// url_index_404 配置：非標準首頁 URL 的處理（對齊 PHP urlJump 邏輯）
	// 當訪問 /index、/index.php 等首頁變體時，根據配置決定 404 或 301 重定向
	reqPath := c.Request.URL.Path
	if reqPath != "/" && (reqPath == "/index" || reqPath == "/index.php" || reqPath == "/home") {
		if model.GetConfigValue("url_index_404", "0") == "1" {
			c.String(http.StatusNotFound, "404 Page Not Found")
			return
		}
		c.Redirect(http.StatusMovedPermanently, langPath(c, "/"))
		return
	}

	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.getStore(c).Render("index.html")
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	content = postRender(content, c.Request.Context())
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) ListPage(c *gin.Context) {
	path := c.Param("path")
	path = trimSuffix(path)

	// 優先查 filename（欄目自定義 URL 名稱），fallback 查 urlname（向後兼容）
	// 手動添加 acode 過濾（AcodePlugin 對 .First() 不可靠）
	acode := acodeplugin.GetAcode(c.Request.Context())
	var sort content.ContentSort
	sortQuery := model.DB.WithContext(c.Request.Context()).Where("filename = ?", path)
	if acode != "" {
		sortQuery = sortQuery.Where("acode = ?", acode)
	}
	if err := sortQuery.First(&sort).Error; err != nil {
		urlnameQuery := model.DB.WithContext(c.Request.Context()).Where("urlname = ?", path)
		if acode != "" {
			urlnameQuery = urlnameQuery.Where("acode = ?", acode)
		}
		if err2 := urlnameQuery.First(&sort).Error; err2 != nil {
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
	content := fc.getStore(c).Render(tpl)
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	content = postRender(content, c.Request.Context())
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) ContentPage(c *gin.Context) {
	// SEO：301 重定向 .html URL 到無後綴的乾淨 URL
	reqPath := c.Request.URL.Path
	if strings.HasSuffix(reqPath, ".html") || strings.HasSuffix(reqPath, ".htm") {
		cleanPath := strings.TrimSuffix(strings.TrimSuffix(reqPath, ".html"), ".htm")
		c.Redirect(http.StatusMovedPermanently, cleanPath)
		return
	}

	path := strings.TrimPrefix(reqPath, "/")
	path = strings.TrimRight(path, "/")
	path = trimSuffix(path)

	if path == "" {
		fc.Index(c)
		return
	}

	ctAcode := acodeplugin.GetAcode(c.Request.Context())

	// 1. 優先查 content filename（支援多段 slug，如 "test/a"）
	var ct content.Content
	ctQuery := model.DB.WithContext(c.Request.Context()).Where("filename = ? AND status = 1 AND date <= ?", path, time.Now())
	if ctAcode != "" {
		ctQuery = ctQuery.Where("acode = ?", ctAcode)
	}
	if ctQuery.First(&ct).Error == nil {
		fc.renderContentDetail(c, &ct)
		return
	}

	// 2. 查 content urlname
	ctFallback := model.DB.WithContext(c.Request.Context()).Where("urlname = ? AND status = 1 AND date <= ?", path, time.Now())
	if ctAcode != "" {
		ctFallback = ctFallback.Where("acode = ?", ctAcode)
	}
	if ctFallback.First(&ct).Error == nil {
		fc.renderContentDetail(c, &ct)
		return
	}

	// 3. 查欄目 filename
	var sort content.ContentSort
	sortQuery := model.DB.WithContext(c.Request.Context()).Where("filename = ?", path)
	if ctAcode != "" {
		sortQuery = sortQuery.Where("acode = ?", ctAcode)
	}
	if sortQuery.First(&sort).Error == nil {
		fc.renderSortPage(c, &sort)
		return
	}

	// 4. 查欄目 urlname
	sortFallback := model.DB.WithContext(c.Request.Context()).Where("urlname = ?", path)
	if ctAcode != "" {
		sortFallback = sortFallback.Where("acode = ?", ctAcode)
	}
	if sortFallback.First(&sort).Error == nil {
		fc.renderSortPage(c, &sort)
		return
	}

	// 5. 分段匹配：處理 /{sortPath}/{slug}.html 或 /{sortPath}/{id}.html
	//    當 slug 不含 "/" 時，URL 帶欄目前綴；最後一段可能是 filename、urlname 或數字 ID
	if segments := strings.Split(path, "/"); len(segments) > 1 {
		lastSegment := segments[len(segments)-1]
		prefix := strings.Join(segments[:len(segments)-1], "/")

		// 5a. 最後一段為數字 → 按 ID 查詢
		if id, err := strconv.Atoi(lastSegment); err == nil && id > 0 {
			var ctByID content.Content
			idQ := model.DB.WithContext(c.Request.Context()).Where("id = ? AND status = 1 AND date <= ?", id, time.Now())
			if ctAcode != "" {
				idQ = idQ.Where("acode = ?", ctAcode)
			}
			if idQ.First(&ctByID).Error == nil {
				if fc.verifySortPath(c.Request.Context(), ctByID.Scode, prefix, ctAcode) {
					fc.renderContentDetail(c, &ctByID)
					return
				}
			}
		}

		// 5b. 最後一段為 content filename
		var ctByFn content.Content
		fnQ := model.DB.WithContext(c.Request.Context()).Where("filename = ? AND status = 1 AND date <= ?", lastSegment, time.Now())
		if ctAcode != "" {
			fnQ = fnQ.Where("acode = ?", ctAcode)
		}
		if fnQ.First(&ctByFn).Error == nil {
			if fc.verifySortPath(c.Request.Context(), ctByFn.Scode, prefix, ctAcode) {
				fc.renderContentDetail(c, &ctByFn)
				return
			}
		}

		// 5c. 最後一段為 content urlname
		var ctByUn content.Content
		unQ := model.DB.WithContext(c.Request.Context()).Where("urlname = ? AND status = 1 AND date <= ?", lastSegment, time.Now())
		if ctAcode != "" {
			unQ = unQ.Where("acode = ?", ctAcode)
		}
		if unQ.First(&ctByUn).Error == nil {
			if fc.verifySortPath(c.Request.Context(), ctByUn.Scode, prefix, ctAcode) {
				fc.renderContentDetail(c, &ctByUn)
				return
			}
		}
	}

	c.String(http.StatusNotFound, "404")
}

// renderContentDetail 渲染內容詳情頁（ContentPage 和 ContentByID 共用）
func (fc *FrontController) renderContentDetail(c *gin.Context, ct *content.Content) {
	ctAcode := acodeplugin.GetAcode(c.Request.Context())

	// 查欄目並做欄目權限檢查
	var sort content.ContentSort
	sortQ := model.DB.WithContext(c.Request.Context()).Where("scode = ?", ct.Scode)
	if ctAcode != "" {
		sortQ = sortQ.Where("acode = ?", ctAcode)
	}
	if sortQ.First(&sort).Error == nil {
		if !fc.checkSortPermission(c, &sort) {
			return
		}
	}

	// 內容權限檢查
	if !fc.checkContentPermission(c, ct) {
		return
	}

	ctx := fc.buildContext(c)
	ctx.Content = ct
	if sort.ID != 0 {
		ctx.Sort = &sort
	}

	// 累加訪問數（對齊 PHP: 未開靜態快取時同步自增）
	fc.addVisits(c, int(ct.ID))

	p := parser.New()
	parser.RegisterAllProviders(p, ctx)

	tpl := "content.html"
	if ctx.Sort != nil && ctx.Sort.ContentTpl != "" {
		tpl = ctx.Sort.ContentTpl
	}
	html := fc.getStore(c).Render(tpl)
	html = p.Render(html)
	html = postRender(html, c.Request.Context())

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// verifySortPath 驗證路徑前綴是否與欄目的 filename/urlname 匹配
func (fc *FrontController) verifySortPath(ctx context.Context, scode, prefix, acode string) bool {
	if scode == "" || prefix == "" {
		return false
	}
	var sort content.ContentSort
	q := model.DB.WithContext(ctx).Where("scode = ?", scode)
	if acode != "" {
		q = q.Where("acode = ?", acode)
	}
	if err := q.First(&sort).Error; err != nil {
		return false
	}
	return sort.Filename == prefix || sort.URLName == prefix
}

func (fc *FrontController) Search(c *gin.Context) {
	ctx := fc.buildContext(c)
	ctx.Keyword = c.Query("keyword")
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		ctx.CurrentPage = p
	}
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.getStore(c).Render("search.html")
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	content = postRender(content, c.Request.Context())
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) Tags(c *gin.Context) {
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.getStore(c).Render("tags.html")
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	content = postRender(content, c.Request.Context())
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

		// 防頻繁提交：同一 IP 60 秒內只能提交一次
		clientIP := c.ClientIP()
		if v, ok := messageRateLimit.Load(clientIP); ok {
			if submitTime, ok := v.(time.Time); ok && time.Since(submitTime) < rateLimitInterval*time.Second {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "提交太頻繁，請稍後再試"})
				return
			}
		}

		// 蜜罐 + 時間陷阱反垃圾檢查
		if !fc.checkAntispam(c) {
			return
		}

		// XSS 安全過濾（對齊 PbootCMS PHP filter()：過濾 gboot:if/sql 標籤 + HTML 轉義）
		filterGbootIf := func(s string) string {
			return common.FilterUserInput(s)
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

		// Turnstile 人機驗證（message_turnstile 默認關閉，與驗證碼獨立）
		if !common.VerifyTurnstile(c, "message_turnstile", "0") {
			return
		}

		ua := c.Request.UserAgent()
	chPlatformVer := c.GetHeader("Sec-CH-UA-Platform-Version")
	osName, bsName := common.ParseUserAgent(ua, chPlatformVer)

	msg := model.Message{
		Acode:      acodeplugin.GetAcode(c.Request.Context()),
		Contacts:   common.FilterSensitiveWords(filterGbootIf(c.PostForm("contacts"))),
		Mobile:     filterGbootIf(c.PostForm("mobile")),
		Content:    common.FilterSensitiveWords(filterGbootIf(c.PostForm("content"))),
		IP:         clientIP,
		OS:         osName,
		Browser:    bsName,
			CreateTime: time.Now(),
			Status:     0,
			CreateUser: "guest",
			UpdateUser: "guest",
		}
		// 審核狀態：message_verify='0' 時直接通過
		if model.GetConfigValue("message_verify", "1") == "0" {
			msg.Status = 1
		}

		if err := model.DB.WithContext(c.Request.Context()).Create(&msg).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "提交失敗"})
			return
		}

		// 郵件通知（message_send_mail=1 時啟用，與 webhook 獨立判斷）
		if model.GetConfigValue("message_send_mail", "0") == "1" {
			mailFields := []map[string]string{
				{"label": "聯繫人", "value": msg.Contacts},
				{"label": "手機", "value": msg.Mobile},
				{"label": "內容", "value": msg.Content},
				{"label": "IP", "value": msg.IP},
				{"label": "操作系統", "value": msg.OS},
				{"label": "瀏覽器", "value": msg.Browser},
			}
			go func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("[Mail] 留言通知 goroutine panic: %v\n", r)
					}
				}()
				if err := mail.SendNotifyMail("在線留言", mailFields); err != nil {
					model.LogNotify("mail", "error", "留言通知："+err.Error())
				} else {
					model.LogNotify("mail", "success", "留言通知郵件已發送")
				}
			}()
		}

		// Webhook 推送（獨立判斷，webhook_message 開關在 SendIf 內檢查）
		webhookFields := []map[string]string{
			{"label": "聯繫人", "value": msg.Contacts},
			{"label": "手機", "value": msg.Mobile},
			{"label": "內容", "value": msg.Content},
		}
		webhook.SendIf("message", "在線留言", msg.IP, msg.OS, msg.Browser, webhookFields)

		cleanupRateLimit()
		messageRateLimit.Store(clientIP, time.Now())
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "提交成功"})
		return
	}
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.getStore(c).Render("message.html")
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	content = postRender(content, c.Request.Context())
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
	if form == nil || form.TableName == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "表單不存在"})
		return
	}
	tableName := form.TableName

	// 統一驗證碼校驗（form_check_code 默認關閉）
	if !common.VerifyCaptcha(c, "form_check_code", "0") {
		return
	}

	// Turnstile 人機驗證（form_turnstile 默認關閉，與驗證碼獨立）
	if !common.VerifyTurnstile(c, "form_turnstile", "0") {
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
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": f.Description + "不能為空"})
			return
		}
		cols = append(cols, fieldName)
		vals = append(vals, val)
		placeholders = append(placeholders, "?")
		notifyFields = append(notifyFields, map[string]string{"label": f.Description, "value": val})
	}

	// 動態 INSERT（參數化查詢值，表名/欄位名驗證）
	// SQL 注入防護：驗證表名和欄位名（對齊 PbootCMS PHP checkKey）
	safeTableRegex := regexp.MustCompile(`^[\w]+$`)
	if !safeTableRegex.MatchString(tableName) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "提交失敗"})
		return
	}
	for _, col := range cols {
		if !safeTableRegex.MatchString(col) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "提交失敗"})
			return
		}
	}
	sql := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName, "`"+strings.Join(cols, "`,`")+"`", strings.Join(placeholders, ","))
	if err := model.DB.Exec(sql, vals...).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "提交失敗"})
		return
	}

	// 郵件通知（form_send_mail=1 時啟用，與 webhook 獨立判斷）
	if model.GetConfigValue("form_send_mail", "0") == "1" {
		notifyFieldsCopy := make([]map[string]string, len(notifyFields))
		copy(notifyFieldsCopy, notifyFields)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("[Mail] 表單通知 goroutine panic: %v\n", r)
				}
			}()
			if err := mail.SendNotifyMail(form.FormName, notifyFieldsCopy); err != nil {
				model.LogNotify("mail", "error", "表單通知("+form.FormName+")："+err.Error())
			} else {
				model.LogNotify("mail", "success", "表單通知郵件已發送："+form.FormName)
			}
		}()
	}

	// Webhook 推送（獨立判斷，webhook_form 開關在 SendIf 內檢查）
	webhook.SendIf("form", form.FormName, clientIP, "", "", notifyFields)

	cleanupRateLimit()
	messageRateLimit.Store(clientIP, time.Now())
	c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "提交成功"})
}

func (fc *FrontController) Visits(c *gin.Context) {
	// 後台開關：visits_count=0 時關閉訪問量統計
	if model.GetConfigValue("visits_count", "1") == "0" {
		c.String(http.StatusOK, "ok")
		return
	}
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	if id > 0 {
		// cookie 去重：同一訪客對同一文章在有效期內只計一次
		cookieName := fmt.Sprintf("pboot_visited_%d", id)
		if _, err := c.Cookie(cookieName); err == nil {
			c.String(http.StatusOK, "ok")
			return
		}
		model.DB.WithContext(c.Request.Context()).Model(&content.Content{}).Where("id = ?", id).
			UpdateColumn("visits", gorm.Expr("visits + 1"))
		c.SetCookie(cookieName, "1", 1800, "/", "", false, true)
	}
	c.String(http.StatusOK, "ok")
}

// addVisits 在前台詳情頁渲染時累加訪問數（對齊 PHP: 未開靜態快取時同步自增）
// 開啟靜態快取時由前端 <script> 異步請求 Visits 介面累加，此處不自增。
func (fc *FrontController) addVisits(c *gin.Context, id int) {
	if id <= 0 {
		return
	}
	// 後台開關：visits_count=0 時關閉訪問量統計
	if model.GetConfigValue("visits_count", "1") == "0" {
		return
	}
	// 開啟靜態快取時不自增（由前端異步請求處理）
	if model.GetConfigValue("tpl_html_cache", "0") != "0" {
		return
	}
	// cookie 去重：同一訪客對同一文章在有效期內只計一次
	cookieName := fmt.Sprintf("pboot_visited_%d", id)
	if _, err := c.Cookie(cookieName); err == nil {
		return
	}
	model.DB.WithContext(c.Request.Context()).Model(&content.Content{}).Where("id = ?", id).
		UpdateColumn("visits", gorm.Expr("visits + 1"))
	c.SetCookie(cookieName, "1", 1800, "/", "", false, true)
}

// checkReferer 驗證請求來源是否為本站（防止跨站 CSRF）
func checkReferer(c *gin.Context) bool {
	referer := c.Request.Header.Get("Referer")
	if referer == "" {
		// AJAX 請求可能不帶 Referer，檢查 Origin
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			return false
		}
		referer = origin
	}
	host := c.Request.Host
	return strings.Contains(referer, host)
}

// Likes 處理點讚請求（POST + IP限速 + Cookie去重 + Referer驗證）
func (fc *FrontController) Likes(c *gin.Context) {
	if model.GetConfigValue("likes_status", "0") == "0" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "功能已禁用"})
		return
	}
	if !checkReferer(c) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "非法請求來源"})
		return
	}
	id, _ := strconv.Atoi(c.PostForm("id"))
	if id <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "無效的內容ID"})
		return
	}
	cookieName := fmt.Sprintf("pboot_likes_%d", id)
	if _, err := c.Cookie(cookieName); err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "您已經點過讚了"})
		return
	}
	var ct content.Content
	if err := model.DB.WithContext(c.Request.Context()).Where("id = ?", id).First(&ct).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "內容不存在"})
		return
	}
	model.DB.WithContext(c.Request.Context()).Model(&content.Content{}).Where("id = ?", id).
		UpdateColumn("likes", gorm.Expr("likes + 1"))
	c.SetCookie(cookieName, "1", 31536000, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "點讚成功", "likes": ct.Likes + 1})
}

// Oppose 處理反對請求（POST + IP限速 + Cookie去重 + Referer驗證）
func (fc *FrontController) Oppose(c *gin.Context) {
	if model.GetConfigValue("likes_status", "0") == "0" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "功能已禁用"})
		return
	}
	if !checkReferer(c) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "非法請求來源"})
		return
	}
	id, _ := strconv.Atoi(c.PostForm("id"))
	if id <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "無效的內容ID"})
		return
	}
	cookieName := fmt.Sprintf("pboot_oppose_%d", id)
	if _, err := c.Cookie(cookieName); err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "您已經反對過了"})
		return
	}
	var ct content.Content
	if err := model.DB.WithContext(c.Request.Context()).Where("id = ?", id).First(&ct).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "內容不存在"})
		return
	}
	model.DB.WithContext(c.Request.Context()).Model(&content.Content{}).Where("id = ?", id).
		UpdateColumn("oppose", gorm.Expr("oppose + 1"))
	c.SetCookie(cookieName, "1", 31536000, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "反對成功", "oppose": ct.Oppose + 1})
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
	if err := model.DB.WithContext(c.Request.Context()).Where("scode = ?", scode).First(&sort).Error; err != nil {
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
// /content/{id} 是兜底 URL：僅當內容無 slug 且欄目無 pathname 時才是規範 URL。
// 若內容有 slug 或欄目有 pathname，301 重定向到規範 URL（避免重複內容，符合 SEO 標準）。
func (fc *FrontController) ContentByID(c *gin.Context) {
	idStr := c.Param("id")
	// SEO：301 重定向 .html URL 到無後綴的乾淨 URL
	if strings.HasSuffix(idStr, ".html") || strings.HasSuffix(idStr, ".htm") {
		idStr = strings.TrimSuffix(strings.TrimSuffix(idStr, ".html"), ".htm")
		c.Redirect(http.StatusMovedPermanently, langPath(c, "/content/"+idStr))
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.String(http.StatusNotFound, "404")
		return
	}
	var ct content.Content
	if err := model.DB.WithContext(c.Request.Context()).Where("id = ? AND status = 1 AND date <= ?", id, time.Now()).First(&ct).Error; err != nil {
		// 當前語言找不到：嘗試跨語言查詢原始內容，重定向到當前語言的對應頁面
		fc.redirectCrossLangContent(c, id)
		return
	}
	// SEO：若內容有 slug，301 重定向到規範 URL
	var sort content.ContentSort
	_ = model.DB.WithContext(c.Request.Context()).Where("scode = ?", ct.Scode).First(&sort).Error
	sortPath := sort.Filename
	if sortPath == "" {
		sortPath = sort.URLName
	}
	if ct.Filename != "" {
		if strings.Contains(ct.Filename, "/") {
			// 多段 slug → /{slug}
			c.Redirect(http.StatusMovedPermanently, langPath(c, "/"+ct.Filename))
		} else if sortPath != "" {
			// 單段 slug → /{sortPath}/{slug}
			c.Redirect(http.StatusMovedPermanently, langPath(c, "/"+sortPath+"/"+ct.Filename))
		} else {
			c.Redirect(http.StatusMovedPermanently, langPath(c, "/"+ct.Filename))
		}
		return
	}
	if ct.URLName != "" {
		if strings.Contains(ct.URLName, "/") {
			c.Redirect(http.StatusMovedPermanently, langPath(c, "/"+ct.URLName))
		} else if sortPath != "" {
			c.Redirect(http.StatusMovedPermanently, langPath(c, "/"+sortPath+"/"+ct.URLName))
		} else {
			c.Redirect(http.StatusMovedPermanently, langPath(c, "/"+ct.URLName))
		}
		return
	}
	// SEO：若欄目有 pathname，301 重定向到 /{sortPath}/{id}
	if sortPath != "" {
		c.Redirect(http.StatusMovedPermanently, langPath(c, "/"+sortPath+"/"+strconv.Itoa(int(ct.ID))))
		return
	}
	// 無 slug 且欄目無 pathname → /content/{id} 是規範 URL，正常渲染
	fc.renderContentDetail(c, &ct)
}

// redirectCrossLangContent 跨語言內容重定向
// 當 /content/:id 在當前語言中找不到時，嘗試：
// 1. 跨語言查詢原始內容（SkipAcode），取得 filename 和 scode
// 2. 若有 filename，在當前語言中按 filename 查找，重定向到 /{filename}
// 3. 若無 filename，按 scode 查找當前語言的欄目，重定向到欄目列表頁
// 4. 都找不到才返回 404
func (fc *FrontController) redirectCrossLangContent(c *gin.Context, id int) {
	// 跨語言查詢原始內容
	var origContent content.Content
	skipCtx := acodeplugin.SkipAcode(c.Request.Context())
	if err := model.DB.WithContext(skipCtx).Where("id = ?", id).First(&origContent).Error; err != nil {
		c.String(http.StatusNotFound, "404")
		return
	}

	// 嘗試按 filename 在當前語言中查找對應內容
	if origContent.Filename != "" {
		var target content.Content
		if err := model.DB.WithContext(c.Request.Context()).Where("filename = ? AND status = 1 AND date <= ?", origContent.Filename, time.Now()).First(&target).Error; err == nil {
			// 找到對應內容，重定向到其 URL
			targetURL := "/" + target.Filename
			if targetURL == "/" {
				targetURL = fmt.Sprintf("/content/%d", target.ID)
			}
			c.Redirect(http.StatusFound, langPath(c, targetURL))
			return
		}
	}

	// filename 找不到，按 scode 查找當前語言的欄目，重定向到欄目列表頁
	if origContent.Scode != "" {
		var sort content.ContentSort
		if err := model.DB.WithContext(c.Request.Context()).Where("scode = ?", origContent.Scode).First(&sort).Error; err == nil {
			// 優先使用欄目的 filename/urlname 作為重定向目標
			sortURL := "/" + sort.Filename
			if sortURL == "/" {
				sortURL = "/" + sort.URLName
			}
			if sortURL == "/" {
				// 欄目也沒有自定義 URL，重定向到首頁
				c.Redirect(http.StatusFound, langPath(c, "/"))
				return
			}
			c.Redirect(http.StatusFound, langPath(c, sortURL))
			return
		}
	}

	c.String(http.StatusNotFound, "404")
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
	if sort.Mcode != "" && model.DB.WithContext(c.Request.Context()).Where("mcode = ?", sort.Mcode).First(&contentModel).Error == nil {
		if contentModel.Type == 1 {
			// 單頁模型 → 用 ContentTpl (如 about.html)
			tpl = sort.ContentTpl
			// 單頁需要加載內容數據
			var ct content.Content
			if model.DB.WithContext(c.Request.Context()).Where("scode = ? AND status = 1", sort.Scode).Order("id DESC").First(&ct).Error == nil {
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

	content := fc.getStore(c).Render(tpl)
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	content = postRender(content, c.Request.Context())
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
		backurl := langPath(c, currentURL)
		c.Redirect(http.StatusFound, langPath(c, "/login")+"?backurl="+url.QueryEscape(backurl))
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
	// 當前頁面路徑（URL 規範化中間件已剝離 acode 前綴和 .html 後綴）
	currentPath := c.Request.URL.Path
	if currentPath == "" {
		currentPath = "/"
	}

	ctx := &parser.Context{
		Page:        make(map[string]interface{}),
		Filters:     make(map[string]string),
		Ctx:         c.Request.Context(),
		CurrentPath: currentPath,
	}

	// 收集 ext_ 查詢參數供篩選使用（白名單驗證防止 SQL 注入）
	for key, vals := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "ext_") && len(vals) > 0 && vals[0] != "" {
			// 嚴格驗證字段名：只允許 ext_ 前綴 + 字母數字下底線
			if parser.IsSafeFieldName(key) {
				ctx.Filters[key] = vals[0]
			}
		}
	}

	var site model.Site
	if model.DB.WithContext(c.Request.Context()).First(&site).Error == nil {
		ctx.Site = &site
	}

	var company model.Company
	if model.DB.WithContext(c.Request.Context()).First(&company).Error == nil {
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

// postRender 頁面渲染後處理（對齊 PHP ParserController::parserAfter 末尾邏輯）
// 1. 敏感詞過濾
// 2. 語言前綴連結重寫（非默認語言下，所有內部連結自動加 /{acode} 前綴）
func postRender(html string, ctx context.Context) string {
	// 敏感詞過濾（對齊 PHP parserReplaceKeyword）
	keywordReplace := model.GetConfigValue("content_keyword_replace", "")
	if keywordReplace != "" {
		keywords := strings.Split(keywordReplace, ",")
		for _, kw := range keywords {
			kw = strings.TrimSpace(kw)
			if kw == "" {
				continue
			}
			stars := strings.Repeat("*", len([]rune(kw)))
			html = strings.ReplaceAll(html, kw, stars)
		}
	}

	// 語言前綴連結重寫
	html = rewriteLangLinks(html, ctx)

	return html
}

// linkRewriteRe 匹配 href="/path"、action="/path"、data-action="/path" 的內部連結
// 同時重寫這三種屬性，確保表單提交和 AJAX 請求也帶語言前綴
var linkRewriteRe = regexp.MustCompile(`(href|action|data-action)=["'](/[^"']*)["']`)

// langSwitcherRe 匹配語言切換器區塊（保護其內部連結不被重寫）
// 匹配 class 屬性中包含 lang-switch 的 div，含一層巢狀 div（dropdown-menu）
// 非貪婪 .*? 配合 </div>\s*</div> 確保匹配到外層閉合標籤
var langSwitcherRe = regexp.MustCompile(`(?s)<div class="[^"]*lang-switch[^"]*">.*?</div>\s*</div>`)

// scriptBlockRe 匹配 <script>...</script> 區塊（保護 JavaScript 中的 URL 不被重寫）
var scriptBlockRe = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)

// rewriteLangLinks 在非默認語言下，將所有內部連結加上 /{acode} 前綴
// 例如：/aboutus → /en/aboutus，/product → /en/product
func rewriteLangLinks(htmlContent string, ctx context.Context) string {
	acode := acodeplugin.GetAcode(ctx)
	if acode == "" {
		return htmlContent
	}

	// 查默認區域：默認語言不需要重寫
	var areas []model.Area
	model.DB.WithContext(acodeplugin.SkipAcode(ctx)).Find(&areas)
	isDefault := false
	for _, a := range areas {
		if a.Acode == acode && a.IsDefault == "1" {
			isDefault = true
			break
		}
	}
	if isDefault {
		return htmlContent
	}

	// 1. 提取並保護 <script> 區塊（避免重寫 JavaScript 中的 URL）
	var scriptBlocks []string
	htmlContent = scriptBlockRe.ReplaceAllStringFunc(htmlContent, func(match string) string {
		scriptBlocks = append(scriptBlocks, match)
		return fmt.Sprintf("\x00SCRIPT_BLOCK_%d\x00", len(scriptBlocks)-1)
	})

	// 2. 提取並保護語言切換器區塊（其連結已由 language provider 正確生成）
	var switcherBlocks []string
	htmlContent = langSwitcherRe.ReplaceAllStringFunc(htmlContent, func(match string) string {
		switcherBlocks = append(switcherBlocks, match)
		return fmt.Sprintf("\x00LANG_SWITCH_%d\x00", len(switcherBlocks)-1)
	})

	// 構建合法 acode 前綴集合（用於判斷連結是否已有前綴）
	acodeSet := make(map[string]bool)
	for _, a := range areas {
		acodeSet[a.Acode] = true
	}

	// 需要跳過的路徑前綴（不重寫這些連結）
	// 僅跳過真正的非內容路徑：後台、靜態資源、API、錨點
	// sort/、content/、tags、login 等都是前台路由，應該被重寫以保持語言前綴
	skipPrefixes := []string{"//", "admin", "static", "template", "api", "favicon", "#"}

	// 3. 重寫所有內部連結（href、action、data-action）
	htmlContent = linkRewriteRe.ReplaceAllStringFunc(htmlContent, func(match string) string {
		submatches := linkRewriteRe.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		attrName := submatches[1]   // href / action / data-action
		fullPath := submatches[2]   // /path

		pathPart := strings.TrimPrefix(fullPath, "/")
		if pathPart == "" {
			return match
		}

		firstSeg := pathPart
		if idx := strings.Index(pathPart, "/"); idx > 0 {
			firstSeg = pathPart[:idx]
		}
		if acodeSet[firstSeg] {
			return match
		}

		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(pathPart, prefix) {
				return match
			}
		}

		return attrName + `="/` + acode + `/` + pathPart + `"`
	})

	// 4. 還原語言切換器區塊
	for i, block := range switcherBlocks {
		htmlContent = strings.Replace(htmlContent, fmt.Sprintf("\x00LANG_SWITCH_%d\x00", i), block, 1)
	}

	// 5. 還原 <script> 區塊
	for i, block := range scriptBlocks {
		htmlContent = strings.Replace(htmlContent, fmt.Sprintf("\x00SCRIPT_BLOCK_%d\x00", i), block, 1)
	}

	return htmlContent
}

// === 以下驗證碼繪製輔助函數已移至 apps/common/captcha.go 統一管理 ===

