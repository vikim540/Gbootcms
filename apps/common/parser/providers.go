package parser

import (
	"encoding/json"
	"fmt"
	"html"
	"pbootcms-go/apps/admin/model"
	"regexp"
	"strconv"
	"strings"
)

type Context struct {
	Sort        *model.ContentSort
	Content     *model.Content
	Site        *model.Site
	Company     *model.Company
	Page        map[string]interface{}
	Member      *model.Member
	Keyword     string
	CurrentPage int
}

// addLeadingSlash 為本地路徑添加前導 /
func addLeadingSlash(path string) string {
	if path != "" && !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "http") {
		return "/" + path
	}
	return path
}

func RegisterAllProviders(p *TagParser, ctx *Context) {
	registerSingleProviders(p, ctx)
	registerPairProviders(p, ctx)
	registerIfProvider(p, ctx)
}

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
			theme := ctx.Site.Theme
			if theme == "" {
				theme = "default"
			}
			return "/template/" + theme + "/static"
		case "sitepath":
			return "/"
		case "pagetitle":
			if ctx.Content != nil {
				return ctx.Content.Title
			}
			if ctx.Sort != nil {
				return ctx.Sort.Name
			}
			return ctx.Site.Title
		case "pagekeywords":
			if ctx.Content != nil && ctx.Content.Keywords != "" {
				return ctx.Content.Keywords
			}
			if ctx.Sort != nil && ctx.Sort.Keywords != "" {
				return ctx.Sort.Keywords
			}
			return ctx.Site.Keywords
		case "pagedescription":
			if ctx.Content != nil && ctx.Content.Description != "" {
				return ctx.Content.Description
			}
			if ctx.Sort != nil && ctx.Sort.Description != "" {
				return ctx.Sort.Description
			}
			return ctx.Site.Description
		// 會員相關標籤 (暫未實現,返回空)
		case "loginstatus":
			return "0" // 0=未登錄
		case "login":
			return "/member/login"
		case "registerstatus":
			return "0" // 0=關閉註冊
		case "register":
			return "/member/register"
		case "ucenter":
			return "/member/center"
		case "islogin":
			return "0"
		case "commentstatus":
			return "1" // 1=開啟評論
		case "commentcodestatus":
			return "0" // 0=關閉評論驗證碼
		case "msgcodestatus":
			return "0" // 0=關閉留言驗證碼
		case "httpurl":
			return "/" // 簡化實現
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
		return fmt.Sprintf("<img src=\"https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=%s\" alt=\"QR Code\">", str)
	})

	p.Register("sort", func(tagName string, params map[string]string, inner string) string {
		if ctx.Sort == nil {
			return ""
		}
		field := params["_field"]
		return getSortField(ctx.Sort, field)
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
			var sb strings.Builder
			for i := 1; i <= totalPages; i++ {
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
			return ctx.Member.HeadPic
		case "email":
			return ctx.Member.Email
		case "logincount":
			return strconv.Itoa(ctx.Member.LoginCount)
		default:
			return ""
		}
	})

	p.Register("label", func(tagName string, params map[string]string, inner string) string {
		field := params["_field"]
		var labels []model.Label
		model.DB.Where("name = ?", field).Find(&labels)
		if len(labels) > 0 {
			return labels[0].Value
		}
		return ""
	})

	p.Register("position", func(tagName string, params map[string]string, inner string) string {
		sep := params["separator"]
		if sep == "" {
			sep = "/"
		}
		idxText := params["indextext"]
		if idxText == "" {
			idxText = "首页"
		}
		parts := []string{fmt.Sprintf(`<a href="/">%s</a>`, idxText)}
		if ctx.Sort != nil && ctx.Sort.Name != "" {
			link := "/" + ctx.Sort.URLName
			if ctx.Sort.URLName == "" {
				link = fmt.Sprintf("/sort/%s", ctx.Sort.Scode)
			}
			parts = append(parts, fmt.Sprintf(`<a href="%s">%s</a>`, link, ctx.Sort.Name))
		}
		return strings.Join(parts, sep)
	})

	p.Register("pagetitle", func(tagName string, params map[string]string, inner string) string {
		if ctx.Site != nil {
			return ctx.Site.Title
		}
		return ""
	})

	p.Register("pagekeywords", func(tagName string, params map[string]string, inner string) string {
		if ctx.Site != nil {
			return ctx.Site.Keywords
		}
		return ""
	})

	p.Register("pagedescription", func(tagName string, params map[string]string, inner string) string {
		if ctx.Site != nil {
			return ctx.Site.Description
		}
		return ""
	})

	p.Register("httpurl", func(tagName string, params map[string]string, inner string) string {
		return "/"
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
		return "/comment/add"
	})

	p.Register("lgpath", func(tagName string, params map[string]string, inner string) string {
		return "/home/index/area"
	})
}

func registerPairProviders(p *TagParser, ctx *Context) {
	p.Register("list", func(tagName string, params map[string]string, inner string) string {
		scode := params["scode"]
		if scode == "" && ctx.Sort != nil {
			scode = ctx.Sort.Scode
		}
		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		order := params["order"]
		if order == "" {
			order = "date"
		}

		query := model.DB.Where("status = 1")
		if scode != "" {
			// 遞歸查找當前欄目及其所有子欄目的 scode
			childScodes := findAllChildScodes(scode)
			query = query.Where("scode IN ?", childScodes)
		}
		switch order {
		case "date":
			query = query.Order("date DESC")
		case "sorting":
			query = query.Order("sorting ASC, date DESC")
		case "visits":
			query = query.Order("visits DESC")
		case "istop":
			query = query.Where("istop = 1").Order("sorting ASC, date DESC")
		case "isrecommend":
			query = query.Where("isrecommend = 1").Order("sorting ASC, date DESC")
		case "isheadline":
			query = query.Where("isheadline = 1").Order("sorting ASC, date DESC")
		default:
			query = query.Order("date DESC")
		}

		// Pagination support
		pageEnabled := params["page"] == "1"
		var total int64
		currentPage := 1

		// 先取數，再獨立取總數（避免 GORM Count 污染查詢狀態）
		var contents []model.Content
		if pageEnabled {
			if ctx.CurrentPage > 0 {
				currentPage = ctx.CurrentPage
			}
			offset := (currentPage - 1) * num
			query.Offset(offset).Limit(num).Find(&contents)

			// 獨立查詢取總記錄數
			model.DB.Model(&model.Content{}).
				Where("status = 1").
				Count(&total)
		} else {
			query.Limit(num).Find(&contents)
			// 獨立查詢取總記錄數（含 scode 過濾）
			countQuery := model.DB.Model(&model.Content{}).
				Where("status = 1")
			if scode != "" {
				countQuery = countQuery.Where("scode IN (?)", findAllChildScodes(scode))
			}
			countQuery.Count(&total)
		}

		// 始終設置分頁資訊（page:rows / page:count 等），供 {gboot:if({page:rows}>0)} 判斷
		totalPages := int(total) / num
		if int(total)%num > 0 {
			totalPages++
		}
		if totalPages < 1 {
			totalPages = 1
		}

		basePath := "?"
		if ctx.Sort != nil {
			if ctx.Sort.Filename != "" {
				basePath = "/" + ctx.Sort.Filename + "/?"
			} else if ctx.Sort.URLName != "" {
				basePath = "/" + ctx.Sort.URLName + "/?"
			}
		}

		ctx.Page["current"] = currentPage
		ctx.Page["count"] = totalPages
		ctx.Page["rows"] = int(total)
		ctx.Page["basePath"] = basePath
		ctx.Page["index"] = basePath + "page=1"
		if currentPage > 1 {
			ctx.Page["pre"] = fmt.Sprintf("%spage=%d", basePath, currentPage-1)
		} else {
			ctx.Page["pre"] = ""
		}
		if currentPage < totalPages {
			ctx.Page["next"] = fmt.Sprintf("%spage=%d", basePath, currentPage+1)
		} else {
			ctx.Page["next"] = ""
		}
		ctx.Page["last"] = fmt.Sprintf("%spage=%d", basePath, totalPages)

		var sb strings.Builder
		for i, c := range contents {
			data := contentToMap(&c, i)
			row := ReplaceInnerTags(inner, "list", data)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("nav", func(tagName string, params map[string]string, inner string) string {
		parent := params["parent"]
		num := 0
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}

		var sorts []model.ContentSort
		query := model.DB.Where("status = 1")
		if parent != "" {
			query = query.Where("pcode = ?", parent)
		} else {
			query = query.Where("pcode = '' OR pcode = '0'")
		}
		query = query.Order("sorting ASC, id ASC")
		if num > 0 {
			query = query.Limit(num)
		}
		query.Find(&sorts)

		var sb strings.Builder
		for i, s := range sorts {
			data := sortToMap(&s, i)
			row := ReplaceInnerTags(inner, "nav", data)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("sort_loop", func(tagName string, params map[string]string, inner string) string {
		scode := params["scode"]
		if scode == "" {
			return ""
		}
		var sorts []model.ContentSort
		model.DB.Where("scode IN (?)", strings.Split(scode, ",")).Order("sorting ASC").Find(&sorts)

		var sb strings.Builder
		for i, s := range sorts {
			data := sortToMap(&s, i)
			row := ReplaceInnerTags(inner, "sort", data)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("content_loop", func(tagName string, params map[string]string, inner string) string {
		id := params["id"]
		scode := params["scode"]
		if id == "" && scode == "" {
			return ""
		}
		var c model.Content
		query := model.DB.Where("status = 1")
		if id != "" {
			query = query.Where("id = ?", id)
		}
		if scode != "" {
			query = query.Where("scode = ?", scode)
		}
		if query.First(&c).Error != nil {
			return ""
		}

		data := contentToMap(&c, 0)
		return ReplaceInnerTags(inner, "content", data)
	})

	p.Register("slide", func(tagName string, params map[string]string, inner string) string {
		gid := params["gid"]
		num := 5
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		var slides []model.Slide
		query := model.DB.Order("gid ASC, sorting ASC, id ASC")
		if gid != "" {
			query = query.Where("gid = ?", gid)
		}
		query.Limit(num).Find(&slides)

		var sb strings.Builder
		for i, s := range slides {
			// 圖片路徑確保以 / 開頭
			src := s.Pic
			if src != "" && !strings.HasPrefix(src, "/") && !strings.HasPrefix(src, "http") {
				src = "/" + src
			}
			pic := s.Pic
			if pic != "" && !strings.HasPrefix(pic, "/") && !strings.HasPrefix(pic, "http") {
				pic = "/" + pic
			}
			picmobile := s.PicMobile
			if picmobile != "" && !strings.HasPrefix(picmobile, "/") && !strings.HasPrefix(picmobile, "http") {
				picmobile = "/" + picmobile
			}
			data := map[string]interface{}{
				"n":         i,
				"i":         i + 1,
				"src":       src,
				"pic":       pic,
				"picmobile": picmobile,
				"link":      s.Link,
				"title":     s.Title,
				"subtitle":  s.Subtitle,
			}
			row := ReplaceInnerTags(inner, "slide", data)
			// 處理內嵌的 {gboot:if(...)} 標籤（在 slide 數據上下文中）
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("link", func(tagName string, params map[string]string, inner string) string {
		gid := params["gid"]
		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		var links []model.Link
		query := model.DB.Order("sorting ASC")
		if gid != "" {
			query = query.Where("gid = ?", gid)
		}
		query.Limit(num).Find(&links)

		var sb strings.Builder
		for i, l := range links {
			logo := l.Logo
			if logo != "" && !strings.HasPrefix(logo, "/") && !strings.HasPrefix(logo, "http") {
				logo = "/" + logo
			}
			data := map[string]interface{}{
				"n":   i,
				"i":   i + 1,
				"logo": logo,
				"link": l.Link,
				"title": l.Title,
			}
			row := ReplaceInnerTags(inner, "link", data)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("loop", func(tagName string, params map[string]string, inner string) string {
		start, _ := strconv.Atoi(params["start"])
		end, _ := strconv.Atoi(params["end"])
		if end <= start {
			return ""
		}

		var sb strings.Builder
		for i := start; i <= end; i++ {
			data := map[string]interface{}{
				"n":     i - start,
				"i":     i - start + 1,
				"index": i,
			}
			row := ReplaceInnerTags(inner, "loop", data)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("tags", func(tagName string, params map[string]string, inner string) string {
		num := 0
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		var contents []model.Content
		model.DB.Where("status = 1 AND keywords != ''").Find(&contents)
		tagSet := map[string]bool{}
		var tagList []string
		for _, c := range contents {
			for _, kw := range strings.Split(c.Keywords, ",") {
				kw = strings.TrimSpace(kw)
				if kw != "" && !tagSet[kw] {
					tagSet[kw] = true
					tagList = append(tagList, kw)
				}
			}
		}
		if num > 0 && num < len(tagList) {
			tagList = tagList[:num]
		}
		var sb strings.Builder
		for i, tag := range tagList {
			data := map[string]interface{}{
				"n":    i,
				"i":    i + 1,
				"tag":  tag,
				"text": tag,
				"link": "/search?keyword=" + tag,
			}
			row := ReplaceInnerTags(inner, "tags", data)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("pics", func(tagName string, params map[string]string, inner string) string {
		if ctx.Content == nil || ctx.Content.Pics == "" {
			return ""
		}
		picList := strings.Split(ctx.Content.Pics, ",")
		var sb strings.Builder
		for i, pic := range picList {
			pic = strings.TrimSpace(pic)
			if pic == "" {
				continue
			}
			data := map[string]interface{}{
				"n":   i,
				"i":   i + 1,
				"src": pic,
			}
			row := ReplaceInnerTags(inner, "pics", data)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("checkbox", func(tagName string, params map[string]string, inner string) string {
		name := params["name"]
		value := params["value"]
		if name == "" || value == "" {
			return ""
		}
		checked := ""
		if ctx.Content != nil {
			// Get field value by name from content
			fieldVal := getContentField(ctx, name, nil)
			if fieldVal != "" {
				for _, v := range strings.Split(fieldVal, ",") {
					if strings.TrimSpace(v) == value {
						checked = " checked"
						break
					}
				}
			}
		}
		return fmt.Sprintf(`<input type="checkbox" name="%s" value="%s"%s>`, name, value, checked)
	})

	p.Register("message", func(tagName string, params map[string]string, inner string) string {
		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		var messages []model.Message
		model.DB.Where("status >= 0").Order("id DESC").Limit(num).Find(&messages)
		var sb strings.Builder
		for i, m := range messages {
			data := map[string]interface{}{
				"n":             i,
				"i":             i + 1,
				"contacts":      m.Contacts,
				"mobile":        m.Mobile,
				"content":       m.Content,
				"askdate":       m.AskDate.Format("2006-01-02"),
				"status":        m.Status,
				"nickname":      m.Nickname,
				"headpic":       m.HeadPic,
				"replycontent":  m.ReplyContent,
				"recontent":     m.ReplyContent, // 兼容模板中 [message:recontent]
				"replydate":     m.ReplyDate.Format("2006-01-02"),
				"os":            m.OS,
				"bs":            m.Browser,
			}
			row := ReplaceInnerTags(inner, "message", data)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("formlist", func(tagName string, params map[string]string, inner string) string {
		return ""
	})

	p.Register("search", func(tagName string, params map[string]string, inner string) string {
		keyword := ctx.Keyword
		if keyword == "" {
			return ""
		}
		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		var contents []model.Content
		like := "%" + keyword + "%"
		model.DB.Where("status = 1 AND (title LIKE ? OR keywords LIKE ? OR description LIKE ?)", like, like, like).
			Order("date DESC").Limit(num).Find(&contents)
		var sb strings.Builder
		for i, c := range contents {
			data := contentToMap(&c, i)
			row := ReplaceInnerTags(inner, "search", data)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("comment", func(tagName string, params map[string]string, inner string) string {
		return ""
	})

	p.Register("commentsub", func(tagName string, params map[string]string, inner string) string {
		return ""
	})

	p.Register("mycomment", func(tagName string, params map[string]string, inner string) string {
		return ""
	})

	p.Register("select", func(tagName string, params map[string]string, inner string) string {
		return ""
	})
}

// processInnerIfTags 處理循環體內的 {gboot:if(...)} 標籤
// 使用正則匹配，避免 processIfTags 的字符串索引 bug
func processInnerIfTags(content string) string {
	re := regexp.MustCompile(`(?s)\{gboot:if\(([^)]+)\)\}(.*?)(?:\{else\}(.*?))?\{/gboot:if\}`)
	for re.MatchString(content) {
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			subs := re.FindStringSubmatch(match)
			if len(subs) < 3 {
				return ""
			}
			cond := strings.TrimSpace(subs[1])
			trueBody := subs[2]
			falseBody := ""
			if len(subs) > 3 {
				falseBody = subs[3]
			}
			// 簡單條件求值
			if evalInnerCondition(cond) {
				return trueBody
			}
			return falseBody
		})
	}
	return content
}

// evalInnerCondition 在循環上下文中求值簡單條件
func evalInnerCondition(cond string) bool {
	// 處理 == 和 != 操作符
	for _, op := range []string{"!=", "=="} {
		if idx := strings.Index(cond, op); idx > 0 {
			left := strings.TrimSpace(strings.Trim(cond[:idx], "'\" "))
			right := strings.TrimSpace(strings.Trim(cond[idx+len(op):], "'\" "))
			if op == "==" {
				return left == right
			}
			return left != right
		}
	}
	// 無操作符：非空即真
	return cond != "" && cond != "0" && cond != "false"
}

func registerIfProvider(p *TagParser, ctx *Context) {
	p.Register("if", func(tagName string, params map[string]string, inner string) string {
		cond := params["condition"]
		trueBranch := params["true"]
		falseBranch := params["false"]

		data := buildIfContext(ctx)
		if EvalIfCondition(cond, data) {
			return trueBranch
		}
		return falseBranch
	})
}

func buildIfContext(ctx *Context) map[string]interface{} {
	data := make(map[string]interface{})
	if ctx.Content != nil {
		data["title"] = ctx.Content.Title
		data["id"] = ctx.Content.ID
		data["visits"] = ctx.Content.Visits
		data["istop"] = ctx.Content.IsTop
		data["isrecommend"] = ctx.Content.IsRecommend
		data["isheadline"] = ctx.Content.IsHeadline
	}
	if ctx.Sort != nil {
		data["scode"] = ctx.Sort.Scode
		data["sortname"] = ctx.Sort.Name
	}
	if ctx.Site != nil {
		data["sitetitle"] = ctx.Site.Title
	}
	if ctx.Member != nil {
		data["islogin"] = 1
	} else {
		data["islogin"] = 0
	}
	return data
}

func getSortField(s *model.ContentSort, field string) string {
	switch field {
	case "name":
		return s.Name
	case "scode":
		return s.Scode
	case "pcode":
		return s.Pcode
	case "tcode":
		// 頂級父欄目代碼：沿 pcode 鏈向上查找
		code := s.Scode
		pcode := s.Pcode
		for pcode != "" && pcode != "0" {
			code = pcode
			var parent model.ContentSort
			if err := model.DB.Where("scode = ?", pcode).First(&parent).Error; err != nil {
				break
			}
			pcode = parent.Pcode
		}
		return code
	case "toplink":
		// 頂級父欄目的鏈接
		tcode := getSortField(s, "tcode")
		var topSort model.ContentSort
		if err := model.DB.Where("scode = ?", tcode).First(&topSort).Error; err == nil {
			if topSort.Outlink != "" {
				return topSort.Outlink
			}
			if topSort.Filename != "" {
				return "/" + topSort.Filename + "/"
			}
			if topSort.URLName != "" {
				return "/" + topSort.URLName + "/"
			}
		}
		return "/" + tcode + "/"
	case "toprows":
		// 頂級父欄目及其所有子欄目的內容總數
		tcode := getSortField(s, "tcode")
		allScodes := findAllChildScodes(tcode)
		var cnt int64
		model.DB.Model(&model.Content{}).
			Where("scode IN ? AND status = 1", allScodes).
			Count(&cnt)
		return strconv.FormatInt(cnt, 10)
	case "link":
		if s.Outlink != "" {
			return s.Outlink
		}
		// 優先用欄目自定義 URL 名稱 (filename)
		if s.Filename != "" {
			return "/" + s.Filename + ".html"
		}
		// fallback 用 model 的 urlname
		if s.URLName != "" {
			return "/" + s.URLName + ".html"
		}
		// 最後 fallback 到動態 scode 路由
		return "/sort/" + s.Scode + ".html"
	case "outlink":
		return s.Outlink
	case "urlname":
		return s.URLName
	case "listtpl":
		return s.ListTpl
	case "contenttpl":
		return s.ContentTpl
	case "ico":
		v := s.Ico
		if v != "" && !strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "http") {
			v = "/" + v
		}
		return v
	case "pic":
		v := s.Pic
		if v != "" && !strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "http") {
			v = "/" + v
		}
		return v
	case "keywords":
		return s.Keywords
	case "description":
		return s.Description
	case "subname":
		return s.Subname
	case "isnav":
		return "1" // 默認導航可見
	case "isblank":
		return "0" // 默認不新窗口
	case "sorttype":
		return fmt.Sprintf("%d", s.Type)
	default:
		return ""
	}
}

func getContentField(ctx *Context, field string, params map[string]string) string {
	c := ctx.Content
	if c == nil {
		return ""
	}
	switch field {
	case "title":
		return AdjustValue(c.Title, params)
	case "subtitle":
		return c.Subtitle
	case "keywords":
		return c.Keywords
	case "description":
		return c.Description
	case "content":
		return c.Content
	case "ico":
		v := c.Ico
		if v != "" && !strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "http") {
			v = "/" + v
		}
		return v
	case "source":
		return c.Source
	case "author":
		return c.Author
	case "visits":
		val := c.Visits
		if operate, ok := params["operate"]; ok {
			operate = strings.TrimSpace(operate)
			if strings.HasPrefix(operate, "+") {
				if n, err := strconv.Atoi(strings.TrimPrefix(operate, "+")); err == nil {
					val += n
				}
			} else if strings.HasPrefix(operate, "-") {
				if n, err := strconv.Atoi(strings.TrimPrefix(operate, "-")); err == nil {
					val -= n
				}
			} else if n, err := strconv.Atoi(operate); err == nil {
				val = n
			}
		}
		return strconv.Itoa(val)
	case "likes":
		return strconv.Itoa(c.Likes)
	case "date":
		if style, ok := params["style"]; ok {
			return c.Date.Format(phpToGoFormat(style))
		}
		return c.Date.Format("2006-01-02")
	case "id":
		return strconv.Itoa(int(c.ID))
	case "link":
		if c.Outlink != "" {
			return c.Outlink
		}
		// 優先用內容自定義 URL 名稱 (filename)
		if c.Filename != "" {
			return "/" + c.Filename + ".html"
		}
		// fallback 用 model 的 urlname
		if c.URLName != "" {
			return "/" + c.URLName + ".html"
		}
		// 最後 fallback 到動態 id 路由
		return "/content/" + strconv.Itoa(int(c.ID)) + ".html"
	case "istop":
		return strconv.Itoa(c.IsTop)
	case "isrecommend":
		return strconv.Itoa(c.IsRecommend)
	case "isheadline":
		return strconv.Itoa(c.IsHeadline)
	case "enclosure":
		return c.Enclosure
	case "precontent":
		// 上一篇：同欄目下 ID 小於當前的最大記錄
		var prev model.Content
		if err := model.DB.Where("scode = ? AND id < ?", c.Scode, c.ID).Order("id desc").First(&prev).Error; err == nil {
			return fmt.Sprintf("<a href=\"%s\">%s</a>", "/"+prev.URLName+".html", prev.Title)
		}
		return "没有了"
	case "nextcontent":
		// 下一篇：同欄目下 ID 大於當前的最小記錄
		var next model.Content
		if err := model.DB.Where("scode = ? AND id > ?", c.Scode, c.ID).Order("id asc").First(&next).Error; err == nil {
			return fmt.Sprintf("<a href=\"%s\">%s</a>", "/"+next.URLName+".html", next.Title)
		}
		return "没有了"
	default:
		if strings.HasPrefix(field, "ext_") {
			// TODO: 擴展字段需要 Content 模型支持 Extra JSON 字段
			// 目前返回空，待 Content 模型增加 Extra 字段後啟用
			return ""
		}
		return ""
	}
}

// parseExtraJSON 解析 content.Extra JSON 字段為 map
func parseExtraJSON(extra string) map[string]string {
	result := make(map[string]string)
	if extra == "" {
		return result
	}
	// 嘗試解析為 JSON
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(extra), &m); err == nil {
		for k, v := range m {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

func contentToMap(c *model.Content, index int) map[string]interface{} {
	// URL 生成規則（對齊 PbootCMS PHP）：
	//   1. 外部鏈接優先
	//   2. 自定義 URL 名稱 (c.Filename)
	//   3. fallback 到 /content/{id}
	link := "/" + c.Filename
	if link == "/" {
		link = fmt.Sprintf("/content/%d", c.ID)
	}
	if c.Outlink != "" {
		link = c.Outlink
	}
	// 圖片路徑標準化：確保以 / 開頭
	ico := c.Ico
	if ico != "" && !strings.HasPrefix(ico, "/") && !strings.HasPrefix(ico, "http") {
		ico = "/" + ico
	}
	// 注意：Content 結構用的是 Pics（複數）不是 Pic
	pics := c.Pics
	if pics != "" && !strings.HasPrefix(pics, "/") && !strings.HasPrefix(pics, "http") {
		pics = "/" + pics
	}
	return map[string]interface{}{
		"n":           index,
		"i":           index + 1,
		"id":          c.ID,
		"scode":       c.Scode,
		"title":       c.Title,
		"subtitle":    c.Subtitle,
		"keywords":    c.Keywords,
		"description": c.Description,
		"content":     c.Content,
		"ico":         ico,
		"pics":        pics,
		"source":      c.Source,
		"author":      c.Author,
		"visits":      c.Visits,
		"likes":       c.Likes,
		"date":        c.Date.Format("2006-01-02"),
		"istop":       c.IsTop,
		"isrecommend": c.IsRecommend,
		"isheadline":  c.IsHeadline,
		"link":        link,
	}
}

func sortToMap(s *model.ContentSort, index int) map[string]interface{} {
	// URL 生成規則（對齊 PbootCMS PHP）：
	//   1. 優先用欄目自定義 URL 名稱 (s.Filename) —— 後台「URL名稱」欄位
	//   2. 退而求其次用模型的 urlname (s.URLName) —— 通常等同於 scode
	//   3. 最後 fallback 到動態 scode 路由
	// 注意：區分 urlname（model 表字段）和 filename（欄目表字段），
	// PbootCMS PHP 中也用 filename 對應「URL名稱」。
	link := "/" + s.Filename + "/"
	if link == "//" {
		link = "/" + s.URLName + "/"
	}
	if link == "//" {
		link = fmt.Sprintf("/sort/%s", s.Scode)
	}
	// 圖片路徑標準化：確保以 / 開頭
	ico := s.Ico
	if ico != "" && !strings.HasPrefix(ico, "/") && !strings.HasPrefix(ico, "http") {
		ico = "/" + ico
	}
	pic := s.Pic
	if pic != "" && !strings.HasPrefix(pic, "/") && !strings.HasPrefix(pic, "http") {
		pic = "/" + pic
	}
	// 計算該欄目及其子欄目的內容數量
	childScodes := findAllChildScodes(s.Scode)
	var rowCount int64
	model.DB.Model(&model.Content{}).
		Where("scode IN ? AND status = 1", childScodes).
		Count(&rowCount)
	return map[string]interface{}{
		"n":       index,
		"i":       index + 1,
		"scode":   s.Scode,
		"name":    s.Name,
		"subname": s.Subname,
		"ico":     ico,
		"pic":     pic,
		"link":    link,
		"rows":    int(rowCount),
	}
}

// findAllChildScodes 遞歸查找指定 scode 及其所有子欄目的 scode 列表
func findAllChildScodes(parentScode string) []string {
	result := []string{parentScode}
	childScodes := getDirectChildScodes(parentScode)
	for _, child := range childScodes {
		// 遞歸：查找孫子、曾孫...
		grandchildren := findAllChildScodes(child)
		result = append(result, grandchildren...)
	}
	return result
}

// getDirectChildScodes 查找指定 scode 的直接子欄目 scode 列表
func getDirectChildScodes(parentScode string) []string {
	var children []struct {
		Scode string
	}
	model.DB.Table("ay_content_sort").
		Select("scode").
		Where("pcode = ? AND status = 1", parentScode).
		Find(&children)
	scodes := make([]string, len(children))
	for i, c := range children {
		scodes[i] = c.Scode
	}
	return scodes
}
