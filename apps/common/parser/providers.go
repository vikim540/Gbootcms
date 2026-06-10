package parser

import (
	"fmt"
	"html"
	"pbootcms-go/apps/admin/model"
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
			return ctx.Site.Logo
		case "icp":
			return ctx.Site.ICP
		case "copyright":
			return ctx.Site.Copyright
		case "statistical":
			return ctx.Site.Statistical
		case "tplpath":
			return "/template/" + ctx.Site.Theme
		case "index":
			return "/"
		case "path":
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
		case "phone":
			return ctx.Company.Phone
		case "fax":
			return ctx.Company.Fax
		case "email":
			return ctx.Company.Email
		case "weixin":
			return ctx.Company.Weixin
		case "icp":
			return ctx.Company.ICP
		default:
			return ""
		}
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
			// 動態URL: 與 sortToMap 保持一致
			link := "/" + ctx.Sort.URLName + ".html"
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
		return ""
	})

	p.Register("pageurl", func(tagName string, params map[string]string, inner string) string {
		return ""
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
			query = query.Where("scode = ? OR subscode = ?", scode, scode)
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

		if pageEnabled {
			query.Count(&total)
			if ctx.CurrentPage > 0 {
				currentPage = ctx.CurrentPage
			}
		}

		var contents []model.Content
		if pageEnabled {
			offset := (currentPage - 1) * num
			query.Offset(offset).Limit(num).Find(&contents)

			// Fill page data
			totalPages := int(total) / num
			if int(total)%num > 0 {
				totalPages++
			}
			if totalPages < 1 {
				totalPages = 1
			}

			basePath := "?"
			if ctx.Sort != nil && ctx.Sort.URLName != "" {
				basePath = "/" + ctx.Sort.URLName + ".html?"
			}

			ctx.Page["current"] = currentPage
			ctx.Page["count"] = totalPages
			ctx.Page["rows"] = int(total)
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
		} else {
			query.Limit(num).Find(&contents)
		}

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
		query := model.DB.Order("sorting ASC")
		if gid != "" {
			query = query.Where("gid = ?", gid)
		}
		query.Limit(num).Find(&slides)

		var sb strings.Builder
		for i, s := range slides {
			data := map[string]interface{}{
				"n":   i,
				"i":   i + 1,
				"src": s.Pic,
				"link": s.Link,
				"title": s.Title,
			}
			row := ReplaceInnerTags(inner, "slide", data)
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
			data := map[string]interface{}{
				"n":   i,
				"i":   i + 1,
				"logo": l.Logo,
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
				"replydate":     m.ReplyDate.Format("2006-01-02"),
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
	case "link":
		if s.Outlink != "" {
			return s.Outlink
		}
		return "/" + s.URLName + ".html"
	case "ico":
		return s.Ico
	case "pic":
		return s.Pic
	case "keywords":
		return s.Keywords
	case "description":
		return s.Description
	case "subname":
		return s.Subname
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
		return c.Ico
	case "source":
		return c.Source
	case "author":
		return c.Author
	case "visits":
		return strconv.Itoa(c.Visits)
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
		return "/" + c.URLName + ".html"
	case "istop":
		return strconv.Itoa(c.IsTop)
	case "isrecommend":
		return strconv.Itoa(c.IsRecommend)
	case "isheadline":
		return strconv.Itoa(c.IsHeadline)
	case "enclosure":
		return c.Enclosure
	default:
		if strings.HasPrefix(field, "ext_") {
			return ""
		}
		return ""
	}
}

func contentToMap(c *model.Content, index int) map[string]interface{} {
	link := "/" + c.URLName + ".html"
	if c.URLName == "" {
		// 動態URL: PbootCMS 慣例是 /content/{id} (對應 router Sort 動態分派)
		link = fmt.Sprintf("/content/%d", c.ID)
	}
	if c.Outlink != "" {
		link = c.Outlink
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
		"ico":         c.Ico,
		"pics":        c.Pics,
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
	link := "/" + s.URLName + ".html"
	if s.URLName == "" {
		// 動態URL: PbootCMS 慣例是 /sort/{scode} (scode 比 id 更具語義,不會隨資料遷移而改變)
		link = fmt.Sprintf("/sort/%s", s.Scode)
	}
	return map[string]interface{}{
		"n":      index,
		"i":      index + 1,
		"scode":  s.Scode,
		"name":   s.Name,
		"subname": s.Subname,
		"ico":    s.Ico,
		"pic":    s.Pic,
		"link":   link,
	}
}
