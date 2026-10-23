package parser

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/member"
	"gbootcms/apps/admin/model/system"
	"gbootcms/core/acodeplugin"
	"gbootcms/core/db"
)

func RegisterAllProviders(p *TagParser, ctx *Context) {
	registerSingleProviders(p, ctx)
	registerPairProviders(p, ctx)
	registerIfProvider(p, ctx)
	registerJSONLDProvider(p, ctx)
	p.SetCtx(ctx) // 用於 checkLabelLevel 權限檢查
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

		query := db.DB.WithContext(ctx.Ctx).Where("status = 1 AND date <= ?", time.Now())
		if scode != "" {
			// 遞歸查找當前欄目及其所有子欄目的 scode
			childScodes := findAllChildScodes(ctx.Ctx, scode)
			query = query.Where("scode IN ?", childScodes)
		}
		// 排序優先級（對齊 PbootCMS ParserController 默認排序邏輯）
		// istop > isrecommend > isheadline > sorting > date > id
		switch order {
		case "date":
			query = query.Order("istop DESC, date DESC, isrecommend DESC, isheadline DESC, sorting ASC, id DESC")
		case "sorting":
			query = query.Order("sorting ASC, istop DESC, isrecommend DESC, isheadline DESC, date DESC, id DESC")
		case "istop":
			query = query.Order("istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "isrecommend":
			query = query.Order("isrecommend DESC, istop DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "isheadline":
			query = query.Order("isrecommend DESC, istop DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "visits":
			query = query.Order(order+" DESC, istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "id":
			query = query.Order("id DESC, istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC")
		case "random":
			query = query.Order("RANDOM()")
		default:
			// 自定義欄位排序
			query = query.Order(order)
		}

		// Pagination support
		pageEnabled := params["page"] == "1"
		var total int64
		currentPage := 1

		// ext_ 篩選：使用 JOIN 方式過濾（比子查詢在 GORM 中更可靠）
		if len(ctx.Filters) > 0 {
			query = query.Joins("JOIN ay_content_ext ON ay_content.id = ay_content_ext.contentid")
			for field, value := range ctx.Filters {
				if !IsSafeFieldName(field) {
					continue // 跳過不安全的字段名，防止 SQL 注入
				}
				query = query.Where("ay_content_ext."+field+" LIKE ?", "%"+value+"%")
			}
		}

		// 先取數，再獨立取總數（避免 GORM Count 污染查詢狀態）
		var contents []content.Content
		if pageEnabled {
			if ctx.CurrentPage > 0 {
				currentPage = ctx.CurrentPage
			}
			offset := (currentPage - 1) * num
			query.Offset(offset).Limit(num).Find(&contents)

			// 獨立查詢取總記錄數（含 scode 過濾 + ext_ 篩選）
			countQuery := db.DB.WithContext(ctx.Ctx).Model(&content.Content{}).
				Where("status = 1 AND date <= ?", time.Now())
			if acode := acodeplugin.GetAcode(ctx.Ctx); acode != "" {
				countQuery = countQuery.Where("acode = ?", acode)
			}
			if scode != "" {
				countQuery = countQuery.Where("scode IN (?)", findAllChildScodes(ctx.Ctx, scode))
			}
			if len(ctx.Filters) > 0 {
			countQuery = countQuery.Joins("JOIN ay_content_ext ON ay_content.id = ay_content_ext.contentid")
			for field, value := range ctx.Filters {
				if !IsSafeFieldName(field) {
					continue
				}
				countQuery = countQuery.Where("ay_content_ext."+field+" LIKE ?", "%"+value+"%")
			}
		}
			countQuery.Count(&total)
		} else {
			query.Limit(num).Find(&contents)
			// 獨立查詢取總記錄數（含 scode 過濾 + ext_ 篩選）
			countQuery := db.DB.WithContext(ctx.Ctx).Model(&content.Content{}).
				Where("status = 1 AND date <= ?", time.Now())
			if acode := acodeplugin.GetAcode(ctx.Ctx); acode != "" {
				countQuery = countQuery.Where("acode = ?", acode)
			}
			if scode != "" {
				countQuery = countQuery.Where("scode IN (?)", findAllChildScodes(ctx.Ctx, scode))
			}
			if len(ctx.Filters) > 0 {
				countQuery = countQuery.Joins("JOIN ay_content_ext ON ay_content.id = ay_content_ext.contentid")
				for field, value := range ctx.Filters {
					if !IsSafeFieldName(field) {
						continue
					}
					countQuery = countQuery.Where("ay_content_ext."+field+" LIKE ?", "%"+value+"%")
				}
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
		// 保留 ext_ 篩選參數，使分頁鏈接攜帶當前篩選條件（驗證字段名安全）
		for k, v := range ctx.Filters {
			if !IsSafeFieldName(k) {
				continue
			}
			basePath += k + "=" + urlEncode(v) + "&"
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
		// 批量預載入擴展字段，避免 N+1 查詢
		contentIDs := make([]uint, len(contents))
		for i, c := range contents {
			contentIDs[i] = c.ID
		}
		extMap := content.GetContentExtByContentIDs(contentIDs)
		// 批量預載入欄目路徑，避免 contentURL 的 N+1 查詢
		preloadSortPaths(ctx, contents)
		for i, c := range contents {
			data := contentToMap(ctx, &c, i, extMap)
			row := ReplaceInnerTags(inner, "list", data)
			row = processInnerIfTags(row)
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

		var sorts []content.ContentSort
		query := db.DB.WithContext(ctx.Ctx).Where("status = 1")
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

		// 批量預載入欄目內容數量，避免 N+1 查詢
		countMap := buildSortCountMap(ctx.Ctx)
		var sb strings.Builder
		for i, s := range sorts {
			data := sortToMap(ctx.Ctx, &s, i, countMap)
			row := ReplaceInnerTags(inner, "nav", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("sort_loop", func(tagName string, params map[string]string, inner string) string {
		scode := params["scode"]
		if scode == "" {
			return ""
		}
		var sorts []content.ContentSort
		db.DB.WithContext(ctx.Ctx).Where("scode IN (?)", strings.Split(scode, ",")).Order("sorting ASC").Find(&sorts)

		// 批量預載入欄目內容數量，避免 N+1 查詢
		countMap := buildSortCountMap(ctx.Ctx)
		var sb strings.Builder
		for i, s := range sorts {
			data := sortToMap(ctx.Ctx, &s, i, countMap)
			row := ReplaceInnerTags(inner, "sort", data)
			row = processInnerIfTags(row)
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
		var c content.Content
		query := db.DB.WithContext(ctx.Ctx).Where("status = 1 AND date <= ?", time.Now())
		if id != "" {
			query = query.Where("id = ?", id)
		}
		if scode != "" {
			query = query.Where("scode = ?", scode)
		}
		if query.First(&c).Error != nil {
			return ""
		}

		data := contentToMap(ctx, &c, 0, nil)
		return ReplaceInnerTags(inner, "content", data)
	})

	p.Register("slide", func(tagName string, params map[string]string, inner string) string {
		gid := params["gid"]
		num := 5
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		var slides []content.Slide
		query := db.DB.WithContext(ctx.Ctx).Order("gid ASC, sorting ASC, id ASC")
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
				"n":           i,
				"i":           i + 1,
				"src":         src,
				"pic":         pic,
				"picmobile":   picmobile,
				"link":        s.Link,
				"title":       s.Title,
				"subtitle":    s.Subtitle,
				"button_text": s.ButtonText,
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
		var links []content.Link
		query := db.DB.WithContext(ctx.Ctx).Order("sorting ASC")
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
				"title": l.Name,
			}
			row := ReplaceInnerTags(inner, "link", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	// language — 多語言切換標籤
	// 用法：{gboot:language}[language:name]{/gboot:language}
	// 切換時保持當前頁面路徑（如 /sc/article → /tc/article → /en/article）
	p.Register("language", func(tagName string, params map[string]string, inner string) string {
		areas := system.GetCachedAreas()
		if len(areas) <= 1 {
			return ""
		}

		// 當前 acode
		currentAcode := acodeplugin.GetAcode(ctx.Ctx)

		// 找默認區域
		defaultAcode := ""
		for _, a := range areas {
			if a.IsDefault == "1" {
				defaultAcode = a.Acode
				break
			}
		}
		if defaultAcode == "" && len(areas) > 0 {
			defaultAcode = areas[0].Acode
		}

		// 當前頁面路徑（已剝離 acode 前綴），確保首頁為 /
		currentPath := ctx.CurrentPath
		if currentPath == "" {
			currentPath = "/"
		}
		if !strings.HasPrefix(currentPath, "/") {
			currentPath = "/" + currentPath
		}

		var sb strings.Builder
		for i, a := range areas {
			// 構建保持當前頁面的語言切換連結
			// 默認區域: /{currentPath}，非默認: /{acode}{currentPath}
			link := currentPath
			if a.Acode != defaultAcode {
				link = "/" + a.Acode + currentPath
			}
			data := map[string]interface{}{
				"i":      i + 1,
				"n":      i,
				"acode":  a.Acode,
				"name":   a.Name,
				"link":   link,
				"active": a.Acode == currentAcode,
			}
			row := ReplaceInnerTags(inner, "language", data)
			row = processInnerIfTags(row)
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
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("tags", func(tagName string, params map[string]string, inner string) string {
		num := 0
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		var contents []content.Content
		// 限制最多載入 2000 條記錄，防止大型站點 OOM
		db.DB.WithContext(ctx.Ctx).Where("status = 1 AND keywords != '' AND date <= ?", time.Now()).Order("id DESC").Limit(2000).Find(&contents)
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
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	// hreflang — SEO 多語言替代連結標籤（自引用 + 雙向對稱 + x-default）
	// 用法：在 <head> 中放置 {gboot:hreflang}
	// 生成：<link rel="alternate" hreflang="zh-Hant" href="..." />
	p.Register("hreflang", func(tagName string, params map[string]string, inner string) string {
		return buildHreflang(ctx)
	})

	// canonical — SEO 標準連結標籤，指向當前頁面的標準 URL
	// 用法：在 <head> 中放置 {gboot:canonical}
	p.Register("canonical", func(tagName string, params map[string]string, inner string) string {
		return buildCanonical(ctx)
	})

	// htmllang — 當前語言的 HTML lang 屬性值（zh-Hans / zh-Hant / en）
	// 用法：<html lang="{gboot:htmllang}">
	p.Register("htmllang", func(tagName string, params map[string]string, inner string) string {
		return acodeToHreflang(acodeplugin.GetAcode(ctx.Ctx))
	})

	// og — Open Graph 社交分享 meta 標籤
	// 用法：在 <head> 中放置 {gboot:og}
	p.Register("og", func(tagName string, params map[string]string, inner string) string {
		return buildOpenGraph(ctx)
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
			// 確保路徑以 / 開頭
			if !strings.HasPrefix(pic, "/") {
				pic = "/" + pic
			}
			data := map[string]interface{}{
				"n":   i,
				"i":   i + 1,
				"src": pic,
			}
			row := ReplaceInnerTags(inner, "pics", data)
			row = processInnerIfTags(row)
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
		var messages []content.Message
		db.DB.WithContext(ctx.Ctx).Where("status = 1").Order("id DESC").Limit(num).Find(&messages)

		// 批量查會員信息（LEFT JOIN ay_member），匹配 PHP 原版 getMessage()
		uidSet := map[int]bool{}
		for _, m := range messages {
			if m.UID > 0 {
				uidSet[m.UID] = true
			}
		}
		type memberInfo struct {
			Nickname string
			HeadPic  string
		}
		memberMap := map[int]memberInfo{}
		if len(uidSet) > 0 {
			uids := make([]int, 0, len(uidSet))
			for uid := range uidSet {
				uids = append(uids, uid)
			}
			var members []member.Member
			db.DB.WithContext(ctx.Ctx).Where("id IN ?", uids).Find(&members)
			for _, mem := range members {
				memberMap[int(mem.ID)] = memberInfo{Nickname: mem.Nickname, HeadPic: mem.Headpic}
			}
		}

		var sb strings.Builder
		for i, m := range messages {
			// nickname: 有會員暱稱用暱稱，無會員顯示"匿名用戶"（PHP 原版邏輯）
			nickname := "匿名用戶"
			headpic := "/static/admin/images/logo.png"
			if m.UID > 0 {
				if mi, ok := memberMap[m.UID]; ok {
					if mi.Nickname != "" {
						nickname = mi.Nickname
					}
					if mi.HeadPic != "" {
						headpic = mi.HeadPic
					}
				}
			}
			// replydate: 零值時間顯示為空（避免顯示 0001-01-01）
			replyDate := ""
			if !m.UpdateTime.IsZero() {
				replyDate = m.UpdateTime.Format("2006-01-02")
			}
			data := map[string]interface{}{
				"n":          i,
				"i":          i + 1,
				"contacts":   m.Contacts,
				"mobile":     m.Mobile,
				"content":    m.Content,
				"create_time": m.CreateTime.Format("2006-01-02"),
				"askdate":    m.CreateTime.Format("2006-01-02"),
				"status":     m.Status,
				"recontent":  m.ReContent,
				"update_time": m.UpdateTime.Format("2006-01-02"),
				"replydate":  replyDate,
				"os":         m.OS,
				"bs":         m.Browser,
				"ip":         m.IP,
				"nickname":   nickname,
				"headpic":    headpic,
			}
			row := ReplaceInnerTags(inner, "message", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("formlist", func(tagName string, params map[string]string, inner string) string {
		fcode := params["fcode"]
		if fcode == "" {
			return ""
		}
		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		// 查 ay_form 獲取 table_name
		tableName := content.GetFormTableByCode(fcode)
		if tableName == "" {
			return ""
		}
		// 查動態表數據
		var rows []map[string]interface{}
		// SQL 注入防護：驗證表名（表名從 DB 讀取，但額外驗證以防資料被篡改）
		if !safeTableRe.MatchString(tableName) {
			return ""
		}
		db.DB.WithContext(ctx.Ctx).Raw("SELECT * FROM `" + tableName + "` ORDER BY id DESC LIMIT " + strconv.Itoa(num)).Scan(&rows)
		if len(rows) == 0 {
			return ""
		}
		var sb strings.Builder
		for i, row := range rows {
			data := make(map[string]interface{})
			for k, v := range row {
				data[k] = fmt.Sprintf("%v", v)
			}
			data["i"] = strconv.Itoa(i + 1)
			data["date"] = data["create_time"]
			rowHTML := ReplaceInnerTags(inner, "form", data)
			rowHTML = processInnerIfTags(rowHTML)
			sb.WriteString(rowHTML)
		}
		return sb.String()
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
		like := "%" + keyword + "%"
		query := db.DB.WithContext(ctx.Ctx).Where("status = 1 AND date <= ? AND (title LIKE ? OR keywords LIKE ? OR description LIKE ?)", time.Now(), like, like, like)

		// scode 過濾（可選）
		scode := params["scode"]
		if scode != "" {
			query = query.Where("scode IN ?", findAllChildScodes(ctx.Ctx, scode))
		}

		order := params["order"]
		if order == "" {
			order = "date"
		}
		// 排序優先級（對齊 PbootCMS ParserController 默認排序邏輯）
		switch order {
		case "date":
			query = query.Order("istop DESC, date DESC, isrecommend DESC, isheadline DESC, sorting ASC, id DESC")
		case "sorting":
			query = query.Order("sorting ASC, istop DESC, isrecommend DESC, isheadline DESC, date DESC, id DESC")
		case "istop":
			query = query.Order("istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "isrecommend":
			query = query.Order("isrecommend DESC, istop DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "isheadline":
			query = query.Order("isrecommend DESC, istop DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "visits":
			query = query.Order(order+" DESC, istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "id":
			query = query.Order("id DESC, istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC")
		default:
			query = query.Order("istop DESC, date DESC, isrecommend DESC, isheadline DESC, sorting ASC, id DESC")
		}

		// 分頁支援
		pageEnabled := params["page"] == "1"
		var total int64
		currentPage := 1

		var contents []content.Content
		if pageEnabled {
			if ctx.CurrentPage > 0 {
				currentPage = ctx.CurrentPage
			}
			offset := (currentPage - 1) * num
			query.Offset(offset).Limit(num).Find(&contents)

			countQuery := db.DB.WithContext(ctx.Ctx).Model(&content.Content{}).
			Where("status = 1 AND date <= ? AND (title LIKE ? OR keywords LIKE ? OR description LIKE ?)", time.Now(), like, like, like)
		if acode := acodeplugin.GetAcode(ctx.Ctx); acode != "" {
			countQuery = countQuery.Where("acode = ?", acode)
		}
		if scode != "" {
			countQuery = countQuery.Where("scode IN ?", findAllChildScodes(ctx.Ctx, scode))
		}
		countQuery.Count(&total)
	} else {
		query.Limit(num).Find(&contents)
		countQuery := db.DB.WithContext(ctx.Ctx).Model(&content.Content{}).
			Where("status = 1 AND date <= ? AND (title LIKE ? OR keywords LIKE ? OR description LIKE ?)", time.Now(), like, like, like)
		if acode := acodeplugin.GetAcode(ctx.Ctx); acode != "" {
			countQuery = countQuery.Where("acode = ?", acode)
		}
			if scode != "" {
				countQuery = countQuery.Where("scode IN ?", findAllChildScodes(ctx.Ctx, scode))
			}
			countQuery.Count(&total)
		}

		// 設置分頁資訊
		totalPages := int(total) / num
		if int(total)%num > 0 {
			totalPages++
		}
		if totalPages < 1 {
			totalPages = 1
		}

		basePath := "/search?keyword=" + urlEncode(keyword) + "&"
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
		// 批量預載入擴展字段，避免 N+1 查詢
		contentIDs := make([]uint, len(contents))
		for i, c := range contents {
			contentIDs[i] = c.ID
		}
		extMap := content.GetContentExtByContentIDs(contentIDs)
		// 批量預載入欄目路徑，避免 contentURL 的 N+1 查詢
		preloadSortPaths(ctx, contents)
		for i, c := range contents {
			data := contentToMap(ctx, &c, i, extMap)
			row := ReplaceInnerTags(inner, "search", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("comment", func(tagName string, params map[string]string, inner string) string {
		// 取得 contentid：優先用 params，其次用 ctx.Content
		contentid := params["contentid"]
		cid, _ := strconv.Atoi(contentid)
		if cid == 0 && ctx.Content != nil {
			cid = int(ctx.Content.ID)
		}
		if cid == 0 {
			return ""
		}

		num := 20
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}

		// 查詢主評論（pid=0, status=1）
		var comments []member.CommentView
		db.DB.WithContext(ctx.Ctx).Table("ay_member_comment a").
			Select("a.*, b.username, b.nickname, b.headpic, c.username as pusername, c.nickname as pnickname, c.headpic as pheadpic").
			Joins("LEFT JOIN ay_member b ON a.uid=b.id").
			Joins("LEFT JOIN ay_member c ON a.puid=c.id").
			Where("a.contentid = ? AND a.pid = 0 AND a.status = 1", cid).
			Order("a.id DESC").
			Limit(num).
			Find(&comments)

		// 提取 commentsub 塊的內容
		subMatch := commentSubRe.FindStringSubmatch(inner)
		subInner := ""
		if len(subMatch) >= 3 {
			subInner = subMatch[2]
		}
		// 移除 inner 中的 commentsub 塊（避免被外層重複渲染）
		innerWithoutSub := commentSubRe.ReplaceAllString(inner, "{__COMMENTSUB__}")

		var sb strings.Builder

		// 批量預載入所有子評論，避免 N+1 查詢（20 條主評論 = 原本 21 次 DB 查詢 → 現在 2 次）
		subMap := map[uint][]member.CommentView{}
		if subInner != "" && len(comments) > 0 {
			parentIDs := make([]uint, len(comments))
			for i, cm := range comments {
				parentIDs[i] = cm.ID
			}
			var allSubs []member.CommentView
			db.DB.WithContext(ctx.Ctx).Table("ay_member_comment a").
				Select("a.*, b.username, b.nickname, b.headpic, c.username as pusername, c.nickname as pnickname, c.headpic as pheadpic").
				Joins("LEFT JOIN ay_member b ON a.uid=b.id").
				Joins("LEFT JOIN ay_member c ON a.puid=c.id").
				Where("a.contentid = ? AND a.pid IN ? AND a.status = 1", cid, parentIDs).
				Order("a.pid ASC, a.id ASC").
				Limit(100*len(comments)).
				Find(&allSubs)
			for i := range allSubs {
				subMap[allSubs[i].Pid] = append(subMap[allSubs[i].Pid], allSubs[i])
			}
		}

		for i, cm := range comments {
			data := commentToMap(&cm, i+1)
			row := ReplaceInnerTags(innerWithoutSub, "comment", data)
			// 處理 commentsub 塊（從預載入的 subMap 中取，無需再查 DB）
			if subInner != "" {
				subs := subMap[cm.ID]

				var subSB strings.Builder
				for j, sc := range subs {
					subData := commentToMap(&sc, j+1)
					subRow := ReplaceInnerTags(subInner, "commentsub", subData)
					subRow = processInnerIfTags(subRow)
					subSB.WriteString(subRow)
				}
				row = strings.Replace(row, "{__COMMENTSUB__}", subSB.String(), 1)
			} else {
				row = strings.Replace(row, "{__COMMENTSUB__}", "", 1)
			}
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("commentsub", func(tagName string, params map[string]string, inner string) string {
		// commentsub 由 comment provider 內部處理，此處為兜底返回空
		return ""
	})

	p.Register("mycomment", func(tagName string, params map[string]string, inner string) string {
		// 需要從 session 取得 uid，但 provider 無法直接存取 gin.Context
		// 透過 ctx.Member 取得（renderMemberPage 時已設定）
		if ctx.Member == nil {
			return ""
		}
		uid := ctx.Member.ID
		if uid == 0 {
			return ""
		}

		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}

		var comments []member.CommentView
		db.DB.WithContext(ctx.Ctx).Table("ay_member_comment a").
			Select("a.*, b.username, b.nickname, b.headpic, c.username as pusername, c.nickname as pnickname, c.headpic as pheadpic, d.title").
			Joins("LEFT JOIN ay_member b ON a.uid=b.id").
			Joins("LEFT JOIN ay_member c ON a.puid=c.id").
			Joins("LEFT JOIN ay_content d ON a.contentid=d.id").
			Where("a.uid = ?", uid).
			Order("a.id DESC").
			Limit(num).
			Find(&comments)

		var sb strings.Builder
		for i, cm := range comments {
			data := commentToMap(&cm, i+1)
			data["delaction"] = "/comment/del?id=" + strconv.FormatUint(uint64(cm.ID), 10)
			data["title"] = cm.Title
			data["status"] = strconv.Itoa(cm.Status)
			if !cm.CreateTime.IsZero() {
				data["date"] = cm.CreateTime.Format("2006-01-02 15:04:05")
			} else {
				data["date"] = ""
			}
			row := ReplaceInnerTags(inner, "mycomment", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	// selectall: 生成「全部」連結（清除該欄位篩選）
	p.Register("selectall", func(tagName string, params map[string]string, inner string) string {
		field := params["field"]
		if field == "" {
			return ""
		}
		text := params["text"]
		if text == "" {
			text = "全部"
		}
		class := params["class"]
		active := params["active"]
		if active == "" {
			active = class
		}
		// 當前是否有該欄位的篩選
		currentVal := ctx.Filters[field]
		cssClass := class
		if currentVal == "" {
			cssClass = active
		}
		// 生成清除該欄位的 URL（保留其他篩選參數）
		link := buildFilterURL(ctx, field, "")
		return fmt.Sprintf(`<a href="%s" class="%s">%s</a>`, link, cssClass, text)
	})

	// select: 遍歷擴展欄位的選項，渲染篩選按鈕
	p.Register("select", func(tagName string, params map[string]string, inner string) string {
		field := params["field"]
		if field == "" || inner == "" {
			return ""
		}
		// 查 ay_extfield 取選項（表名 ay_extfield，非 GORM 默認推斷）
		var ef content.ExtField
		if err := db.DB.WithContext(ctx.Ctx).Raw("SELECT * FROM ay_extfield WHERE field = ? LIMIT 1", field).Scan(&ef).Error; err != nil {
			return ""
		}
		if ef.Value == "" {
			return ""
		}
		options := strings.Split(ef.Value, ",")
		currentVal := ctx.Filters[field]
		var buf strings.Builder
		for _, opt := range options {
			opt = strings.TrimSpace(opt)
			if opt == "" {
				continue
			}
			link := buildFilterURL(ctx, field, opt)
			// 替換 inner 中的 [select:link] [select:value] [select:current]
			row := inner
			row = strings.ReplaceAll(row, "[select:link]", link)
			row = strings.ReplaceAll(row, "[select:value]", opt)
			row = strings.ReplaceAll(row, "[select:current]", currentVal)
			buf.WriteString(row)
		}
		return buf.String()
	})
}

// processInnerIfTags 處理循環體內的 {gboot:if(...)} 標籤
// 使用正則匹配，避免 processIfTags 的字符串索引 bug
func processInnerIfTags(content string) string {
	for innerIfRe.MatchString(content) {
		content = innerIfRe.ReplaceAllStringFunc(content, func(match string) string {
			subs := innerIfRe.FindStringSubmatch(match)
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
// 對齊 PbootCMS PHP symbol() 函數，支援 == != > >= < <= 運算符
func evalInnerCondition(cond string) bool {
	cond = strings.TrimSpace(cond)

	// 處理 && (AND) 邏輯運算符
	if idx := strings.Index(cond, "&&"); idx > 0 {
		left := strings.TrimSpace(cond[:idx])
		right := strings.TrimSpace(cond[idx+2:])
		return evalInnerCondition(left) && evalInnerCondition(right)
	}
	// 處理 || (OR) 邏輯運算符
	if idx := strings.Index(cond, "||"); idx > 0 {
		left := strings.TrimSpace(cond[:idx])
		right := strings.TrimSpace(cond[idx+2:])
		return evalInnerCondition(left) || evalInnerCondition(right)
	}

	// 比較運算符（按長度降序匹配，避免 >= 被 > 截獲）
	for _, op := range []string{">=", "<=", "!=", "==", ">", "<"} {
		if idx := strings.Index(cond, op); idx > 0 {
			left := strings.TrimSpace(strings.Trim(cond[:idx], "'\" "))
			right := strings.TrimSpace(strings.Trim(cond[idx+len(op):], "'\" "))

			// 嘗試數值比較
			leftNum, leftErr := strconv.Atoi(left)
			rightNum, rightErr := strconv.Atoi(right)

			if leftErr == nil && rightErr == nil {
				// 數值比較
				switch op {
				case "==":
					return leftNum == rightNum
				case "!=":
					return leftNum != rightNum
				case ">":
					return leftNum > rightNum
				case ">=":
					return leftNum >= rightNum
				case "<":
					return leftNum < rightNum
				case "<=":
					return leftNum <= rightNum
				}
			} else {
				// 字串比較
				switch op {
				case "==":
					return left == right
				case "!=":
					return left != right
				case ">":
					return left > right
				case ">=":
					return left >= right
				case "<":
					return left < right
				case "<=":
					return left <= right
				}
			}
		}
	}

	// 處理 % 取模運算（如 [pics:n]%2==0 判斷偶數）
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
	// 快取：同一頁面渲染期間 config 值不變，首次建構後直接返回
	if ctx.ifContext != nil {
		return ctx.ifContext
	}
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
	// 會員驗證碼狀態（供 {gboot:if(logincodestatus)} 等條件判斷使用）
	if db.GetConfigValue("login_check_code", "1") != "0" {
		data["logincodestatus"] = 1
	} else {
		data["logincodestatus"] = 0
	}
	if db.GetConfigValue("register_check_code", "1") != "0" {
		data["registercodestatus"] = 1
	} else {
		data["registercodestatus"] = 0
	}
	// 評論功能開關（與 commentstatus provider 一致，空值默認啟用）
	if db.GetConfigValue("comment_status", "1") != "0" {
		data["commentstatus"] = 1
	} else {
		data["commentstatus"] = 0
	}
	// 評論驗證碼開關
	if db.GetConfigValue("comment_check_code", "1") != "0" {
		data["commentcodestatus"] = 1
	} else {
		data["commentcodestatus"] = 0
	}
	// 會員登入開關
	if db.GetConfigValue("login_status", "1") != "0" {
		data["loginstatus"] = 1
	} else {
		data["loginstatus"] = 0
	}
	// 會員註冊開關
	if db.GetConfigValue("register_status", "1") != "0" {
		data["registerstatus"] = 1
	} else {
		data["registerstatus"] = 0
	}
	// 留言驗證碼開關
	if db.GetConfigValue("message_check_code", "1") != "0" {
		data["msgcodestatus"] = 1
	} else {
		data["msgcodestatus"] = 0
	}
	ctx.ifContext = data // 快取供後續 {gboot:if} 標籤復用
	return data
}

func getSortField(ctx context.Context, s *content.ContentSort, field string) string {
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
			var parent content.ContentSort
			if err := db.DB.WithContext(ctx).Where("scode = ?", pcode).First(&parent).Error; err != nil {
				break
			}
			pcode = parent.Pcode
		}
		return code
	case "toplink":
		// 頂級父欄目的鏈接
		tcode := getSortField(ctx, s, "tcode")
		var topSort content.ContentSort
		if err := db.DB.WithContext(ctx).Where("scode = ?", tcode).First(&topSort).Error; err == nil {
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
		tcode := getSortField(ctx, s, "tcode")
		allScodes := findAllChildScodes(ctx, tcode)
		var cnt int64
		topRowsQ := db.DB.WithContext(ctx).Model(&content.Content{}).
			Where("scode IN ? AND status = 1 AND date <= ?", allScodes, time.Now())
		if acode := acodeplugin.GetAcode(ctx); acode != "" {
			topRowsQ = topRowsQ.Where("acode = ?", acode)
		}
		topRowsQ.Count(&cnt)
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


