package controller

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"net/http"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/admin/model/content"
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

// 留言驗證碼存儲（前台用，與後台 checkCodeStore 分離）
var messageCodeStore = make(map[string]string)

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

func (fc *FrontController) Index(c *gin.Context) {
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render("index.html")
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) ListPage(c *gin.Context) {
	path := c.Param("path")
	path = trimSuffix(path)

	// 優先查 filename（欄目自定義 URL 名稱），fallback 查 urlname（向後兼容）
	var sort model.ContentSort
	if err := model.DB.Where("filename = ?", path).First(&sort).Error; err != nil {
		if err2 := model.DB.Where("urlname = ?", path).First(&sort).Error; err2 != nil {
			c.String(http.StatusNotFound, "404")
			return
		}
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
	var content model.Content
	if err := model.DB.Where("filename = ? AND status = 1", path).First(&content).Error; err != nil {
		if err2 := model.DB.Where("urlname = ? AND status = 1", path).First(&content).Error; err2 != nil {
			var sort model.ContentSort
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

	ctx := fc.buildContext(c)
	ctx.Content = &content

	var sort model.ContentSort
	if model.DB.Where("scode = ?", content.Scode).First(&sort).Error == nil {
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
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) Tags(c *gin.Context) {
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render("tags.html")
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

		// 驗證碼校驗（message_check_code 非 '0' 時啟用）
		if model.GetConfigValue("message_check_code", "1") != "0" {
			checkcode := strings.ToLower(c.PostForm("checkcode"))
			if checkcode == "" {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "驗證碼不能為空"})
				return
			}
			cookie, _ := c.Cookie("PbootGo")
			savedCode, ok := messageCodeStore[cookie]
			if !ok || strings.ToLower(savedCode) != checkcode {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "驗證碼錯誤"})
				return
			}
			delete(messageCodeStore, cookie)
		}

		msg := model.Message{
			Contacts: filterGbootIf(c.PostForm("contacts")),
			Mobile:   filterGbootIf(c.PostForm("mobile")),
			Content:  filterGbootIf(c.PostForm("content")),
			IP:       clientIP,
			OS:       parseUserOS(c.Request.UserAgent()),
			Browser:  parseUserBrowser(c.Request.UserAgent()),
			AskDate:  time.Now(),
			Status:   0,
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
			webhook.Send("在線留言", msg.IP, msg.OS, msg.Browser, notifyFields)
		}

		messageRateLimit[clientIP] = time.Now()
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "提交成功"})
		return
	}
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render("message.html")
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

// handleFormSubmit 處理自定義表單提交（fcode≥2，寫入動態表 ay_diy_*）
func (fc *FrontController) handleFormSubmit(c *gin.Context, fcode, clientIP string, filterGbootIf func(string) string) {
	// 查 ay_form 獲取 table_name
	form := content.GetFormByCode(fcode)
	if form == nil || form.Table == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "表單不存在"})
		return
	}
	tableName := form.Table

	// 驗證碼校驗（form_check_code 非 '0' 時啟用，預設關閉）
	if model.GetConfigValue("form_check_code", "0") != "0" {
		checkcode := strings.ToLower(c.PostForm("checkcode"))
		if checkcode == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "驗證碼不能為空"})
			return
		}
		cookie, _ := c.Cookie("PbootGo")
		savedCode, ok := messageCodeStore[cookie]
		if !ok || strings.ToLower(savedCode) != checkcode {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "驗證碼錯誤"})
			return
		}
		delete(messageCodeStore, cookie)
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
		webhook.Send(form.FormName, clientIP, "", "", notifyFields)
	}

	messageRateLimit[clientIP] = time.Now()
	c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "提交成功"})
}

func (fc *FrontController) Visits(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	if id > 0 {
		model.DB.Model(&model.Content{}).Where("id = ?", id).
			UpdateColumn("visits", gorm.Expr("visits + 1"))
	}
	c.String(http.StatusOK, "ok")
}

// CheckCode 前台驗證碼生成（存入 messageCodeStore，供留言校驗）
func (fc *FrontController) CheckCode(c *gin.Context) {
	a := randInt(9) + 1
	b := randInt(9) + 1
	op := randInt(2)
	var expr string
	var answer int
	if op == 0 {
		expr = fmt.Sprintf("%d + %d = ?", a, b)
		answer = a + b
	} else {
		if a < b {
			a, b = b, a
		}
		expr = fmt.Sprintf("%d - %d = ?", a, b)
		answer = a - b
	}

	sessionID := fc.getCookie(c, "PbootGo")
	if sessionID == "" {
		sessionID = fc.generateSessionID()
		fc.setCookie(c, "PbootGo", sessionID, 86400)
	}
	messageCodeStore[sessionID] = fmt.Sprintf("%d", answer)

	width := 200
	height := 70
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	bgColor := color.RGBA{245, 250, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)
	for i := 0; i < 80; i++ {
		x := randInt(width)
		y := randInt(height)
		r := uint8(180 + randInt(60))
		g := uint8(180 + randInt(60))
		b2 := uint8(180 + randInt(60))
		img.Set(x, y, color.RGBA{r, g, b2, 255})
	}
	for i := 0; i < 4; i++ {
		x1 := randInt(width)
		y1 := randInt(height)
		x2 := randInt(width)
		y2 := randInt(height)
		lineColor := color.RGBA{uint8(100 + randInt(100)), uint8(100 + randInt(100)), uint8(100 + randInt(100)), 255}
		drawLine(img, x1, y1, x2, y2, lineColor)
	}
	addLabel(img, expr, width, height)

	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	png.Encode(c.Writer, img)
}

// SortByScode renders the list page for a sort identified by its scode
// (the stable PbootCMS code, never changes once assigned).
// Used as the dynamic URL "/sort/{scode}" generated by sortToMap
// when a sort has no urlname set.
func (fc *FrontController) SortByScode(c *gin.Context) {
	scode := c.Param("scode")
	var sort model.ContentSort
	if err := model.DB.Where("scode = ?", scode).First(&sort).Error; err != nil {
		c.String(http.StatusNotFound, "404")
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
	var content model.Content
	if err := model.DB.Where("id = ? AND status = 1", id).First(&content).Error; err != nil {
		c.String(http.StatusNotFound, "404")
		return
	}
	var sort model.ContentSort
	_ = model.DB.Where("scode = ?", content.Scode).First(&sort).Error

	ctx := fc.buildContext(c)
	ctx.Content = &content
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

func (fc *FrontController) renderSortPage(c *gin.Context, sort *model.ContentSort) {
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
			var content model.Content
			if model.DB.Where("scode = ? AND status = 1", sort.Scode).Order("id DESC").First(&content).Error == nil {
				ctx.Content = &content
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
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
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

	return ctx
}

func trimSuffix(s string) string {
	return strings.TrimSuffix(strings.TrimSuffix(s, ".html"), ".htm")
}

// randInt 返回 0~max-1 的隨機整數
func randInt(max int) int {
	return rand.Intn(max)
}

// getCookie 安全讀取 cookie
func (fc *FrontController) getCookie(c *gin.Context, name string) string {
	if cookie, err := c.Cookie(name); err == nil {
		return cookie
	}
	return ""
}

// setCookie 設置 cookie
func (fc *FrontController) setCookie(c *gin.Context, name, value string, maxAge int) {
	c.SetCookie(name, value, maxAge, "/", "", false, true)
}

// generateSessionID 生成唯一 session ID
func (fc *FrontController) generateSessionID() string {
	return fmt.Sprintf("%d%d", time.Now().UnixNano(), rand.Intn(1000000))
}

// addLabel 在圖片上繪製文字（簡易版，用 basicfont）
func addLabel(img *image.RGBA, label string, width, height int) {
	// 用簡易點陣字體繪製，每個字符約 12px 寬
	charWidth := 12
	startX := (width - len(label)*charWidth) / 2
	startY := height / 2
	for i, ch := range label {
		drawChar(img, startX+i*charWidth, startY-10, byte(ch))
	}
}

// drawChar 繪製單個字符（簡易 5x7 點陣）
func drawChar(img *image.RGBA, x, y int, ch byte) {
	color := color.RGBA{30, 80, 160, 255}
	// 簡易實現：用固定字體繪製
	font := getBasicFont()
	rows, ok := font[ch]
	if !ok {
		return
	}
	for ry, row := range rows {
		for cx := 0; cx < len(row); cx++ {
			if row[cx] == '1' {
				for dy := 0; dy < 2; dy++ {
					for dx := 0; dx < 2; dx++ {
						px := x + cx*2 + dx
						py := y + ry*2 + dy
						if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
							img.Set(px, py, color)
						}
					}
				}
			}
		}
	}
}

// drawLine 繪製干擾線
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.Color) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	sy := 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy
	for {
		img.Set(x1, y1, col)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// getBasicFont 返回簡易 5x7 點陣字體（數字和運算符）
func getBasicFont() map[byte][]string {
	return map[byte][]string{
		'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
		'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
		'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
		'3': {"11111", "00010", "00100", "00010", "00001", "10001", "01110"},
		'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
		'5': {"11111", "10000", "11110", "00001", "00001", "10001", "01110"},
		'6': {"00110", "01000", "10000", "11110", "10001", "10001", "01110"},
		'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
		'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
		'9': {"01110", "10001", "10001", "01111", "00001", "00010", "01100"},
		' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
		'+': {"00000", "00100", "00100", "11111", "00100", "00100", "00000"},
		'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
		'=': {"00000", "00000", "11111", "00000", "11111", "00000", "00000"},
		'?': {"01110", "10001", "00001", "00010", "00100", "00000", "00100"},
	}
}
