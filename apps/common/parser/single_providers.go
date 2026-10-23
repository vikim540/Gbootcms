package parser

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/member"
	"gbootcms/core/acodeplugin"
	"gbootcms/core/db"
)

func registerSingleProviders(p *TagParser, ctx *Context) {
	p.Register("site", func(tagName string, params map[string]string, inner string) string {
		if ctx.Site == nil {
			return ""
		}
		field := params["_field"]
		switch field {
		case "title":
			return ctx.Site.Title
		case "subtitle":
			return ctx.Site.Subtitle
		case "keywords":
			return ctx.Site.Keywords
		case "description":
			return ctx.Site.Description
		case "logo":
			logo := ctx.Site.Logo
			if logo != "" && !strings.HasPrefix(logo, "/") && !strings.HasPrefix(logo, "http") {
				logo = "/" + logo
			}
			return logo
		case "icp":
			return ctx.Site.ICP
		case "copyright":
			return ctx.Site.Copyright
		case "statistical":
			return ctx.Site.Statistical
		case "tplpath":
			theme := ctx.Site.Theme
			if theme == "" {
				theme = "default"
			}
			return "/template/" + theme + "/static"
		case "index":
			return "/"
		case "path":
			return "/"
		default:
			return ""
		}
	})

	// GBoot 兼容: {gboot:sitepath}, {gboot:sitetitle}, {gboot:sitetplpath} 等
	p.Register("gboot", func(tagName string, params map[string]string, inner string) string {
		if ctx.Site == nil {
			return ""
		}
		field := params["_field"]
		switch field {
		case "sitetitle":
			return ctx.Site.Title
		case "sitesubtitle":
			return ctx.Site.Subtitle
		case "sitekeywords":
			return ctx.Site.Keywords
		case "sitedescription":
			return ctx.Site.Description
		case "sitelogo":
			logo := ctx.Site.Logo
			if logo != "" && !strings.HasPrefix(logo, "/") && !strings.HasPrefix(logo, "http") {
				logo = "/" + logo
			}
			return logo
		case "siteicp":
			return ctx.Site.ICP
		case "sitecopyright":
			return ctx.Site.Copyright
		case "sitestatistical":
			return ctx.Site.Statistical
		case "sitetplpath":
			// 靜態資源（CSS/JS/圖片）所有語言共用 default 目錄，避免重複和同步問題
			return "/template/default/static"
		case "sitepath":
			return currentHomePath(ctx)
		case "pagetitle":
			return resolvePageTitle(ctx)
		case "pagekeywords":
			return resolvePageKeywords(ctx)
		case "pagedescription":
			return resolvePageDescription(ctx)
		// 會員相關標籤
		case "loginstatus":
			return db.GetConfigValue("login_status", "1")
		case "login":
			return "/login"
		case "registerstatus":
			return db.GetConfigValue("register_status", "1")
		case "register":
			return "/register"
		case "ucenter":
			return "/ucenter"
		case "umodify":
			return "/umodify"
		case "logout":
			return "/logout"
		case "retrieve":
			return "/retrieve"
		case "islogin":
			if ctx.Member != nil {
				return "1"
			}
			return "0"
		case "mustlogin":
			// 標籤本身回傳空字串（權限檢查在 controller 層的 checkMustLogin 處理）
			return ""
		case "commentstatus":
			return db.GetConfigValue("comment_status", "1")
		case "commentcodestatus":
			return db.GetConfigValue("comment_check_code", "1")
		case "msgcodestatus":
			return db.GetConfigValue("message_check_code", "1")
		case "msgturnstilestatus":
		return db.GetConfigValue("message_turnstile", "0")
	case "turnstile_sitekey":
		return db.GetConfigValue("turnstile_sitekey", "")
	case "httpurl":
		// 對齊 PbootCMS get_http_url(): 動態返回請求 URL，非配置值
		if ctx != nil && ctx.HttpURL != "" {
			return ctx.HttpURL
		}
		return db.GetConfigValue("httpurl", "/")
		// Company 字段路由: {gboot:companyname} → company.name
		case "companyname":
			if ctx.Company != nil {
				return ctx.Company.Name
			}
			return ""
		case "companyaddress":
			if ctx.Company != nil {
				return ctx.Company.Address
			}
			return ""
		case "companypostcode":
			if ctx.Company != nil {
				return ctx.Company.Postcode
			}
			return ""
		case "companycontact":
			if ctx.Company != nil {
				return ctx.Company.Contact
			}
			return ""
		case "companyphone":
			if ctx.Company != nil {
				return ctx.Company.Phone
			}
			return ""
		case "companymobile":
			if ctx.Company != nil {
				return ctx.Company.Mobile
			}
			return ""
		case "companyfax":
			if ctx.Company != nil {
				return ctx.Company.Fax
			}
			return ""
		case "companyemail":
			if ctx.Company != nil {
				return ctx.Company.Email
			}
			return ""
		case "companyqq":
			if ctx.Company != nil {
				return ctx.Company.Qq
			}
			return ""
		case "companyweixin":
			if ctx.Company != nil {
				return addLeadingSlash(ctx.Company.Weixin)
			}
			return ""
		case "companyicp":
			if ctx.Company != nil {
				return ctx.Company.ICP
			}
			return ""
		case "companyblicense":
			if ctx.Company != nil {
				return ctx.Company.Blicense
			}
			return ""
		case "companylegal":
			if ctx.Company != nil {
				return ctx.Company.Legal
			}
			return ""
		case "companybusiness":
			if ctx.Company != nil {
				return ctx.Company.Business
			}
			return ""
		case "companyother":
			if ctx.Company != nil {
				return ctx.Company.Other
			}
			return ""
		// Site info alias: {gboot:siteurl}
		case "siteurl":
			return "/"
		default:
			return ""
		}
	})

	p.Register("company", func(tagName string, params map[string]string, inner string) string {
		if ctx.Company == nil {
			return ""
		}
		field := params["_field"]
		switch field {
		case "name":
			return ctx.Company.Name
		case "address":
			return ctx.Company.Address
		case "postcode":
			return ctx.Company.Postcode
		case "contact":
			return ctx.Company.Contact
		case "phone":
			return ctx.Company.Phone
		case "mobile":
			return ctx.Company.Mobile
		case "fax":
			return ctx.Company.Fax
		case "email":
			return ctx.Company.Email
		case "qq":
			return ctx.Company.Qq
		case "weixin":
			return ctx.Company.Weixin
		case "icp":
			return ctx.Company.ICP
		case "blicense":
			return ctx.Company.Blicense
		case "legal":
			return ctx.Company.Legal
		case "business":
			return ctx.Company.Business
		case "other":
			return ctx.Company.Other
		default:
			return ""
		}
	})

	// QRCode 標籤: {gboot:qrcode string=xxx}
	p.Register("qrcode", func(tagName string, params map[string]string, inner string) string {
		str := params["string"]
		if str == "" {
			str = params["_field"]
		}
		if str == "" {
			return ""
		}
		// 返回一個簡單的 QR code 圖片（使用在線 API）
		// URL 編碼 str 防止 XSS（避免雙引號截斷 src 屬性）
		return fmt.Sprintf("<img src=\"https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=%s\" alt=\"QR Code\">", url.QueryEscape(str))
	})

	p.Register("sort", func(tagName string, params map[string]string, inner string) string {
		if ctx.Sort == nil {
			return ""
		}
		field := params["_field"]
		return getSortField(ctx.Ctx, ctx.Sort, field)
	})

	p.Register("content", func(tagName string, params map[string]string, inner string) string {
		if ctx.Content == nil {
			return ""
		}
		field := params["_field"]
		return getContentField(ctx, field, params)
	})

	p.Register("page", func(tagName string, params map[string]string, inner string) string {
		if ctx.Page == nil {
			return ""
		}
		field := params["_field"]
		if field == "numbar" {
			// 生成數字分頁條: [1] 2 3 4 5
			current := 1
			totalPages := 1
			total := 0
			if v, ok := ctx.Page["current"]; ok {
				current, _ = v.(int)
			} else if v, ok := ctx.Page["current_page"]; ok {
				current, _ = v.(int)
			}
			if v, ok := ctx.Page["count"]; ok {
				totalPages, _ = v.(int)
			} else if v, ok := ctx.Page["total_pages"]; ok {
				totalPages, _ = v.(int)
			}
			if v, ok := ctx.Page["rows"]; ok {
				total, _ = v.(int)
			} else if v, ok := ctx.Page["total"]; ok {
				total, _ = v.(int)
			}
			if total == 0 {
			return ""
		}
		// 讀取分頁數字條數量配置（對齊 PHP Paging::pageNumBar，預設 5）
		pageNum := parseIntConfig("pagenum", 5)
		var sb strings.Builder
		// 計算顯示範圍（對齊 PHP pageNumBar 邏輯：當前頁前後各顯示一半）
		halfL := pageNum / 2
		halfU := (pageNum + 1) / 2
		start := 1
		end := totalPages
		if totalPages > pageNum {
			if current > halfU {
				start = current - halfL
			}
			if current+halfL <= totalPages {
				end = current + halfU
			} else {
				end = totalPages
				start = totalPages - pageNum + 1
			}
			if start < 1 {
				start = 1
			}
			if end > totalPages {
				end = totalPages
			}
		}
		for i := start; i <= end; i++ {
			link := fmt.Sprintf("?page=%d", i)
			if baseP, ok := ctx.Page["basePath"]; ok {
				if bp, ok2 := baseP.(string); ok2 && bp != "" {
					link = bp + "page=" + strconv.Itoa(i)
				}
			}
			if i == current {
				sb.WriteString(fmt.Sprintf("<a class=\"page-num page-num-current\" href=\"%s\">%d</a>", link, i))
			} else {
				sb.WriteString(fmt.Sprintf("<a class=\"page-num\" href=\"%s\">%d</a>", link, i))
			}
		}
		return sb.String()
		}
		if val, ok := ctx.Page[field]; ok {
			return ValToStr(val)
		}
		return ""
	})

	p.Register("user", func(tagName string, params map[string]string, inner string) string {
		if ctx.Member == nil {
			return ""
		}
		field := params["_field"]
		switch field {
		case "username":
			return ctx.Member.Username
		case "nickname":
			return ctx.Member.Nickname
		case "headpic":
			return ctx.Member.Headpic
		case "email":
			return ctx.Member.Email
		case "useremail":
			return ctx.Member.Useremail
		case "usermobile":
			return ctx.Member.Usermobile
		case "ucode":
			return ctx.Member.Ucode
		case "uid", "id":
			return strconv.FormatUint(uint64(ctx.Member.ID), 10)
		case "gid":
			return ctx.Member.GID
		case "gcode":
			// 透過 gid 查 ay_member_group 取 gcode（對應 PHP getUser() JOIN）
			if ctx.Member.GID != "" && ctx.Member.GID != "0" {
				var group member.MemberGroup
				db.DB.WithContext(ctx.Ctx).Where("id = ?", ctx.Member.GID).First(&group)
				return group.Gcode
			}
			return ""
		case "score":
			return strconv.Itoa(ctx.Member.Score)
		case "logincount":
			return strconv.Itoa(ctx.Member.LoginCount)
		case "register_time":
			t := ctx.Member.RegisterTime
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02 15:04:05")
		case "last_login_time":
			return ctx.Member.LastLoginTime
		case "last_login_ip":
			return ctx.Member.LastLoginIP
		case "sex":
			return ctx.Member.Sex
		case "birthday":
			return ctx.Member.Birthday
		case "telephone":
			return ctx.Member.Telephone
		case "qq":
			return ctx.Member.QQ
		case "gname":
			if ctx.Member != nil && ctx.Member.GID != "" {
				var group member.MemberGroup
				db.DB.WithContext(ctx.Ctx).Where("id = ?", ctx.Member.GID).First(&group)
				return group.Gname
			}
			return ""
		case "registertime":
			t := ctx.Member.RegisterTime
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02 15:04:05")
		case "lastloginip":
			return ctx.Member.LastLoginIP
		case "lastlogintime":
			return ctx.Member.LastLoginTime
		default:
			return ""
		}
	})

	p.Register("label", func(tagName string, params map[string]string, inner string) string {
		field := params["_field"]
		var labels []content.Label
		db.DB.WithContext(ctx.Ctx).Where("name = ?", field).Find(&labels)
		if len(labels) > 0 {
			return labels[0].Value
		}
		return ""
	})

	p.Register("i18n", func(tagName string, params map[string]string, inner string) string {
		field := params["_field"]
		if field == "" {
			field = params["field"]
		}
		if field == "" {
			return ""
		}
		acode := acodeplugin.GetAcode(ctx.Ctx)
		if acode == "" {
			acode = "default"
		}
		return ResolveI18nText(field, acode)
	})

	// buildSortChain 從當前欄目向上遞歸到根欄目，返回從根到當前的有序鏈
	buildSortChain := func(scode string) []content.ContentSort {
		var chain []content.ContentSort
		current := scode
		for current != "" && current != "0" {
			var s content.ContentSort
			if err := db.DB.WithContext(ctx.Ctx).Where("scode = ?", current).First(&s).Error; err != nil {
				break
			}
			chain = append(chain, s)
			current = s.Pcode
		}
		for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
			chain[i], chain[j] = chain[j], chain[i]
		}
		return chain
	}

	// sortLink 生成欄目鏈接：filename > urlname > /sort/scode
	sortLink := func(s *content.ContentSort) string {
		if s.Filename != "" {
			return "/" + s.Filename
		}
		if s.URLName != "" {
			return "/" + s.URLName
		}
		return fmt.Sprintf("/sort/%s", s.Scode)
	}

	p.Register("position", func(tagName string, params map[string]string, inner string) string {
		sep := params["separator"]
		if sep == "" {
			sep = "/"
		}
		idxText := params["indextext"]
		if idxText == "" {
			idxText = "首頁"
		}
		parts := []string{fmt.Sprintf(`<a href="/">%s</a>`, idxText)}
		if ctx.Sort != nil && ctx.Sort.Name != "" {
			// 遍歷父級欄目鏈（當前欄目 → pcode 向上 → 根欄目）
			chain := buildSortChain(ctx.Sort.Scode)
			// 去重：根欄目可能和當前欄目相同
			seen := map[string]bool{}
			for _, s := range chain {
				if seen[s.Scode] {
					continue
				}
				seen[s.Scode] = true
				parts = append(parts, fmt.Sprintf(`<a href="%s">%s</a>`, sortLink(&s), s.Name))
			}
		}
		return strings.Join(parts, sep)
	})

	p.Register("pagetitle", func(tagName string, params map[string]string, inner string) string {
		return resolvePageTitle(ctx)
	})

	p.Register("pagekeywords", func(tagName string, params map[string]string, inner string) string {
		return resolvePageKeywords(ctx)
	})

	p.Register("pagedescription", func(tagName string, params map[string]string, inner string) string {
		return resolvePageDescription(ctx)
	})

	p.Register("httpurl", func(tagName string, params map[string]string, inner string) string {
		// 對齊 PbootCMS get_http_url(): 動態返回請求 URL
		if ctx != nil && ctx.HttpURL != "" {
			return ctx.HttpURL
		}
		return db.GetConfigValue("httpurl", "/")
	})

	p.Register("pageurl", func(tagName string, params map[string]string, inner string) string {
		if ctx.Content != nil && ctx.Content.URLName != "" {
			return "/" + ctx.Content.URLName + ".html"
		}
		return "/"
	})

	p.Register("islogin", func(tagName string, params map[string]string, inner string) string {
		if ctx.Member != nil {
			return "1"
		}
		return "0"
	})

	p.Register("checkcode", func(tagName string, params map[string]string, inner string) string {
		return "/api/checkcode"
	})

	p.Register("msgaction", func(tagName string, params map[string]string, inner string) string {
		return "/message"
	})

	p.Register("scaction", func(tagName string, params map[string]string, inner string) string {
		return "/search"
	})

	// 時間戳：用於前端時間陷阱反垃圾
	p.Register("timestamp", func(tagName string, params map[string]string, inner string) string {
		return fmt.Sprintf("%d", time.Now().Unix())
	})

	// Form provider: {gboot:form fcode=X} → 表單提交 URL
	p.Register("form", func(tagName string, params map[string]string, inner string) string {
		fcode := params["fcode"]
		if fcode != "" {
			return "/message?fcode=" + fcode
		}
		return "/message"
	})

	p.Register("keyword", func(tagName string, params map[string]string, inner string) string {
		return html.EscapeString(ctx.Keyword)
	})

	p.Register("commentaction", func(tagName string, params map[string]string, inner string) string {
		// 帶上 contentid 參數（與 PHP 原版一致）
		if ctx.Content != nil {
			return "/comment/add?contentid=" + strconv.FormatUint(uint64(ctx.Content.ID), 10)
		}
		return "/comment/add"
	})

	p.Register("lgpath", func(tagName string, params map[string]string, inner string) string {
		return "/home/index/area"
	})

	// acode — 當前語言區域代碼（用於模板條件判斷）
	p.Register("acode", func(tagName string, params map[string]string, inner string) string {
		return acodeplugin.GetAcode(ctx.Ctx)
	})

	// homename — 導航菜單首項文字（本地化「首頁」），站點標題請用 {gboot:sitetitle}
	p.Register("homename", func(tagName string, params map[string]string, inner string) string {
		acode := acodeplugin.GetAcode(ctx.Ctx)
		switch acode {
		case "sc":
			return "首页"
		case "en":
			return "Home"
		default:
			return "首頁"
		}
	})

	// morename — 當前語言的「查看更多」文字
	p.Register("morename", func(tagName string, params map[string]string, inner string) string {
		acode := acodeplugin.GetAcode(ctx.Ctx)
		switch acode {
		case "sc":
			return "查看更多"
		case "en":
			return "View More"
		default:
			return "查看更多"
		}
	})

	// 會員相關 URL 標籤
	p.Register("login", func(tagName string, params map[string]string, inner string) string {
		return "/login"
	})
	p.Register("register", func(tagName string, params map[string]string, inner string) string {
		return "/register"
	})
	p.Register("logout", func(tagName string, params map[string]string, inner string) string {
		return "/logout"
	})
	p.Register("ucenter", func(tagName string, params map[string]string, inner string) string {
		return "/ucenter"
	})
	p.Register("umodify", func(tagName string, params map[string]string, inner string) string {
		return "/umodify"
	})
	p.Register("retrieve", func(tagName string, params map[string]string, inner string) string {
		return "/retrieve"
	})

	// 驗證碼狀態標籤（返回 "1" 或 "0"）
	p.Register("logincodestatus", func(tagName string, params map[string]string, inner string) string {
		if db.GetConfigValue("login_check_code", "1") != "0" {
			return "1"
		}
		return "0"
	})
	p.Register("registercodestatus", func(tagName string, params map[string]string, inner string) string {
		if db.GetConfigValue("register_check_code", "1") != "0" {
			return "1"
		}
		return "0"
	})
}

