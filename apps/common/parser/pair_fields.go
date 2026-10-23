package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/member"
	"gbootcms/core/acodeplugin"
	"gbootcms/core/db"
)

func getContentField(ctx *Context, field string, params map[string]string) string {
	c := ctx.Content
	if c == nil {
		return ""
	}
	switch field {
	case "title":
		return AdjustValue(c.Title, params)
	case "titlecolor":
		return c.TitleColor
	case "subtitle":
		return c.Subtitle
	case "keywords":
		return c.Keywords
	case "description":
		return c.Description
	case "content":
		// 內鏈替換 + 外鏈 nofollow + 敏感詞過濾
		raw := c.Content
		raw = replaceContentTags(raw, ctx.Ctx)
		raw = replaceKeyword(raw)
		raw = addNofollowToExternalLinks(raw)
		return raw
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
	case "date":
		if style, ok := params["style"]; ok {
			return c.Date.Format(phpToGoFormat(style))
		}
		return c.Date.Format("2006-01-02")
	case "id":
		return strconv.Itoa(int(c.ID))
	case "link":
		return contentURL(ctx, c)
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
		var prev content.Content
		if err := db.DB.WithContext(ctx.Ctx).Where("scode = ? AND id < ? AND status = 1 AND date <= ?", c.Scode, c.ID, time.Now()).Order("id desc").First(&prev).Error; err == nil {
			return fmt.Sprintf("<a href=\"%s\">%s</a>", contentURL(ctx, &prev), prev.Title)
		}
		return "沒有了"
	case "nextcontent":
		// 下一篇：同欄目下 ID 大於當前的最小記錄
		var next content.Content
		if err := db.DB.WithContext(ctx.Ctx).Where("scode = ? AND id > ? AND status = 1 AND date <= ?", c.Scode, c.ID, time.Now()).Order("id asc").First(&next).Error; err == nil {
			return fmt.Sprintf("<a href=\"%s\">%s</a>", contentURL(ctx, &next), next.Title)
		}
		return "沒有了"
	default:
		if strings.HasPrefix(field, "ext_") {
			// 從 ay_content_ext 表讀取擴展字段（對齊 PHP: $data->ext_xxx 動態屬性）
			// 使用請求級快取避免同一頁面多次存取 ext_ 欄位時重複查詢
			if ctx.ContentExt == nil {
				ctx.ContentExt = content.GetContentExtByContentID(c.ID)
				if ctx.ContentExt == nil {
					ctx.ContentExt = map[string]interface{}{} // 標記為已載入但無數據
				}
			}
			if v, ok := ctx.ContentExt[field]; ok {
				return fmt.Sprintf("%v", v)
			}
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

// parseIntConfig 從 DB 讀取整數配置（簡化 GetConfigValue+Atoi 組合）
func parseIntConfig(name string, defaultVal int) int {
	v, err := strconv.Atoi(db.GetConfigValue(name, ""))
	if err != nil {
		return defaultVal
	}
	return v
}

// commentToMap 將 CommentView 轉為模板欄位 map
func commentToMap(c *member.CommentView, index int) map[string]interface{} {
	nickname := c.Nickname
	if nickname == "" {
		nickname = c.Username
	}
	if nickname == "" {
		nickname = "遊客"
	}
	pnickname := c.Pnickname
	if pnickname == "" {
		pnickname = c.Pusername
	}
	if pnickname == "" {
		pnickname = "遊客"
	}
	headpic := c.Headpic
	if headpic == "" {
		headpic = "/static/admin/images/logo.png"
	}
	dateStr := ""
	if !c.CreateTime.IsZero() {
		dateStr = c.CreateTime.Format("2006-01-02 15:04:05")
	}
	// replyaction: comment/add?contentid=X&pid=Y&puid=Z
	pid := c.Pid
	if pid == 0 {
		pid = c.ID
	}
	replyaction := fmt.Sprintf("/comment/add?contentid=%d&pid=%d&puid=%d", c.Contentid, pid, c.Uid)

	return map[string]interface{}{
		"i":           strconv.Itoa(index),
		"n":           strconv.Itoa(index - 1),
		"id":          strconv.FormatUint(uint64(c.ID), 10),
		"comment":     c.Comment,
		"contentid":   strconv.FormatUint(uint64(c.Contentid), 10),
		"nickname":    nickname,
		"pnickname":   pnickname,
		"username":    c.Username,
		"headpic":     headpic,
		"date":        dateStr,
		"ip":          c.UserIP,
		"os":          c.UserOS,
		"bs":          c.UserBS,
		"replyaction": replyaction,
	}
}

// contentToMap 將 Content 模型轉為模板可用的 map。
// extMap 為批量預載入的擴展字段（避免 N+1），nil 時回退到單條查詢。
func contentToMap(ctx *Context, c *content.Content, index int, extMap map[uint]map[string]interface{}) map[string]interface{} {
	// URL 生成統一使用 contentURL（與前台路由邏輯一致）
	link := contentURL(ctx, c)
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
	m := map[string]interface{}{
		"n":           index,
		"i":           index + 1,
		"id":          c.ID,
		"scode":       c.Scode,
		"title":       c.Title,
		"titlecolor":  c.TitleColor,
		"subtitle":    c.Subtitle,
		"keywords":    c.Keywords,
		"description": c.Description,
		"content":     c.Content,
		"ico":         ico,
		"pics":        pics,
		"source":      c.Source,
		"author":      c.Author,
		"visits":      c.Visits,
		"date":        c.Date.Format("2006-01-02"),
		"istop":       c.IsTop,
		"isrecommend": c.IsRecommend,
		"isheadline":  c.IsHeadline,
		"link":        link,
	}

	// 注入擴展字段（ext_前綴），供列表頁 [list:ext_xxx] 使用
	// 優先使用批量預載入的 extMap，避免 N+1 查詢
	var ext map[string]interface{}
	if extMap != nil {
		ext = extMap[c.ID]
	} else {
		ext = content.GetContentExtByContentID(c.ID)
	}
	if ext != nil {
		for k, v := range ext {
			if strings.HasPrefix(k, "ext_") {
				m[k] = fmt.Sprintf("%v", v)
			}
		}
	}
	return m
}

func sortToMap(ctx context.Context, s *content.ContentSort, index int, countMap map[string]int) map[string]interface{} {
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
	// 優先使用批量預載入的 countMap，避免 N+1 查詢
	var rowCount int
	if countMap != nil {
		rowCount = countMap[s.Scode]
	} else {
		childScodes := findAllChildScodes(ctx, s.Scode)
		var c int64
		fallbackQ := db.DB.WithContext(ctx).Model(&content.Content{}).
			Where("scode IN ? AND status = 1 AND date <= ?", childScodes, time.Now())
		if acode := acodeplugin.GetAcode(ctx); acode != "" {
			fallbackQ = fallbackQ.Where("acode = ?", acode)
		}
		fallbackQ.Count(&c)
		rowCount = int(c)
	}
	return map[string]interface{}{
		"n":       index,
		"i":       index + 1,
		"scode":   s.Scode,
		"name":    s.Name,
		"subname": s.Subname,
		"ico":     ico,
		"pic":     pic,
		"link":    link,
		"rows":    rowCount,
	}
}

// buildSortCountMap 批量預載入所有欄目及其子欄目的內容數量。
// 一次性載入全部 ContentSort 和內容計數，在記憶體中建構樹狀結構，
// 將 N 次遞迴查詢降為 2 欝查詢（全部欄目 + 全部內容計數）。
func buildSortCountMap(ctx context.Context) map[string]int {
	// 1. 載入所有啟用的欄目
	var allSorts []content.ContentSort
	q := db.DB.WithContext(ctx).Where("status = 1")
	if acode := acodeplugin.GetAcode(ctx); acode != "" {
		q = q.Where("acode = ?", acode)
	}
	q.Find(&allSorts)

	// 2. 建構 parent → children 記憶體映射
	childrenMap := make(map[string][]string)
	for _, s := range allSorts {
		childrenMap[s.Pcode] = append(childrenMap[s.Pcode], s.Scode)
	}

	// 3. 批量查詢每個 scode 的內容數量（單次 GROUP BY）
	type scodeCount struct {
		Scode string
		Cnt   int64
	}
	var counts []scodeCount
	countQ := db.DB.WithContext(ctx).Model(&content.Content{}).
		Select("scode, count(*) as cnt").
		Where("status = 1 AND date <= ?", time.Now())
	if acode := acodeplugin.GetAcode(ctx); acode != "" {
		countQ = countQ.Where("acode = ?", acode)
	}
	countQ.Group("scode").Scan(&counts)
	directCount := make(map[string]int)
	for _, c := range counts {
		directCount[c.Scode] = int(c.Cnt)
	}

	// 4. 遞迴計算每個欄目的總數量（自身 + 所有子孫）
	result := make(map[string]int, len(allSorts))
	var computeTotal func(scode string) int
	computeTotal = func(scode string) int {
		total := directCount[scode]
		for _, child := range childrenMap[scode] {
			total += computeTotal(child)
		}
		return total
	}
	for _, s := range allSorts {
		result[s.Scode] = computeTotal(s.Scode)
	}
	return result
}

// findAllChildScodes 遞歸查找指定 scode 及其所有子欄目的 scode 列表
// 使用 SQLite 遞迴 CTE 單次查詢取代 N 次遞迴 DB 查詢（配合 idx_sort_pcode 索引）
func findAllChildScodes(ctx context.Context, parentScode string) []string {
	acode := acodeplugin.GetAcode(ctx)

	query := `WITH RECURSIVE descendants AS (
		SELECT scode FROM ay_content_sort WHERE scode = ? AND status = 1`
	if acode != "" {
		query += ` AND acode = ?`
	}
	query += `
		UNION ALL
		SELECT s.scode FROM ay_content_sort s
		INNER JOIN descendants d ON s.pcode = d.scode
		WHERE s.status = 1`
	if acode != "" {
		query += ` AND s.acode = ?`
	}
	query += `
	)
	SELECT scode FROM descendants`

	var rows []struct {
		Scode string
	}
	if acode != "" {
		db.DB.WithContext(ctx).Raw(query, parentScode, acode, acode).Scan(&rows)
	} else {
		db.DB.WithContext(ctx).Raw(query, parentScode).Scan(&rows)
	}

	if len(rows) == 0 {
		return []string{parentScode}
	}
	scodes := make([]string, len(rows))
	for i, r := range rows {
		scodes[i] = r.Scode
	}
	return scodes
}

// ─── 標題樣式解析（對齊 PHP index_title/list_title/content_title/about_title/other_title 配置） ───


