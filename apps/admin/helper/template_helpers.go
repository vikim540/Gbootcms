package helper

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"gbootcms/apps/admin/model"
	contentModel "gbootcms/apps/admin/model/content"
	memberModel "gbootcms/apps/admin/model/member"
	"gbootcms/core/basic"

	"github.com/flosch/pongo2/v6"
)

// StructToMap converts a struct to map[string]interface{} with PascalCase keys.
// Uses JSON tag names (converted to PascalCase) as map keys for pongo2 compatibility.
func StructToMap(obj interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return result
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		// Use JSON tag name, converted to PascalCase
		tag := field.Tag.Get("json")
		key := field.Name
		if tag != "" && tag != "-" {
			name := strings.Split(tag, ",")[0]
			if name != "" {
				key = snakeToPascal(name)
			}
		}
		result[key] = v.Field(i).Interface()
	}
	return result
}

// SnakeToPascal 導出版本，供外部包（如 Controller/Service）使用
func SnakeToPascal(s string) string {
	return snakeToPascal(s)
}

// GetAllModelsData returns all content models as a slice of maps (for $allmodels).
func GetAllModelsData() []map[string]interface{} {
	models := contentModel.GetAllModels()
	result := make([]map[string]interface{}, len(models))
	for i, m := range models {
		result[i] = StructToMap(m)
	}
	return result
}

// GetTemplateFiles scans the template/default/ directory for .html files (excluding admin/).
// Returns a list of template filenames for dropdown selection.
func GetTemplateFiles() []string {
	var files []string
	tplDir := "template/default"
	entries, err := os.ReadDir(tplDir)
	if err != nil {
		return files
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".html") {
			files = append(files, name)
		}
	}
	if files == nil {
		files = []string{}
	}
	return files
}

// BuildGroupsData returns member groups with Gname/Gcode for template compatibility.
// Templates use [value->gname] / [value->gcode] matching the model fields.
func BuildGroupsData() []map[string]interface{} {
	var groups []memberModel.MemberGroup
	model.DB.Where("status = 1").Order("id ASC").Find(&groups)
	result := make([]map[string]interface{}, len(groups))
	for i, g := range groups {
		result[i] = map[string]interface{}{
			"ID":    g.ID,
			"Gname": g.Gname,
			"Gcode": g.Gcode,
		}
	}
	if result == nil {
		result = []map[string]interface{}{}
	}
	return result
}

// BuildSortSelectHTML generates <option> HTML for parent sort selection dropdowns.
// Builds hierarchical options based on pcode/scode tree structure.
func BuildSortSelectHTML(sorts []model.ContentSort, selected string) *pongo2.Value {
	var sb strings.Builder
	buildSortOptions(&sb, sorts, "0", "", selected, 0)
	return pongo2.AsSafeValue(sb.String())
}

func buildSortOptions(sb *strings.Builder, sorts []model.ContentSort, parentCode, prefix, selected string, depth int) {
	indent := strings.Repeat("&nbsp;&nbsp;", depth)
	if depth > 0 {
		indent = strings.Repeat("&nbsp;&nbsp;", depth-1) + "├─"
	}
	for _, s := range sorts {
		if s.Pcode == parentCode || (parentCode == "0" && (s.Pcode == "" || s.Pcode == "0")) {
			sel := ""
			if s.Scode == selected {
				sel = " selected"
			}
			sb.WriteString(fmt.Sprintf(`<option value="%s"%s>%s%s</option>`, s.Scode, sel, indent, s.Name))
			// Recurse for children
			buildSortOptions(sb, sorts, s.Scode, prefix, selected, depth+1)
		}
	}
}

// BuildSearchSelectHTML generates <option> HTML for search/filter dropdowns.
// Optionally filters by model code (mcode).
func BuildSearchSelectHTML(sorts []model.ContentSort, mcode string) *pongo2.Value {
	var sb strings.Builder
	for _, s := range sorts {
		if mcode != "" && s.Mcode != mcode {
			continue
		}
		sb.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, s.Scode, s.Name))
	}
	return pongo2.AsSafeValue(sb.String())
}

// BuildSubsortSelectHTML generates <option> HTML for sub-sort selection.
// Excludes the given scode (current sort) from the options.
func BuildSubsortSelectHTML(sorts []model.ContentSort, excludeScode string) *pongo2.Value {
	var sb strings.Builder
	for _, s := range sorts {
		if s.Scode == excludeScode {
			continue
		}
		sb.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, s.Scode, s.Name))
	}
	return pongo2.AsSafeValue(sb.String())
}

// BuildSubsortSelectWithSelected generates <option> HTML for sub-sort selection with selected value.
// Used in content mod forms where the content's subscode should be selected.
func BuildSubsortSelectWithSelected(sorts []model.ContentSort, selectedSubscode string) *pongo2.Value {
	var sb strings.Builder
	for _, s := range sorts {
		sel := ""
		if s.Scode == selectedSubscode {
			sel = " selected"
		}
		sb.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`, s.Scode, sel, s.Name))
	}
	return pongo2.AsSafeValue(sb.String())
}

// AddSonField adds a computed "Son" boolean field to each sort entry.
// Son=true means the sort has child sorts (used for folder icon display).
func AddSonField(sorts []model.ContentSort) []map[string]interface{} {
	// Build set of all pcodes to check if a scode is a parent
	pcodeSet := make(map[string]bool)
	for _, s := range sorts {
		if s.Pcode != "" && s.Pcode != "0" {
			pcodeSet[s.Pcode] = true
		}
	}
	result := make([]map[string]interface{}, len(sorts))
	for i, s := range sorts {
		m := StructToMap(s)
		m["Son"] = pcodeSet[s.Scode]
		result[i] = m
	}
	return result
}

// BuildSortTreeData reorders sorts into a proper tree hierarchy
// (parent rows always before their children) AND adds the Son field.
// This is critical for jQuery treetable which requires parent-before-child
// ordering in the DOM. The DB ORDER BY sorting,id does NOT guarantee this.
func BuildSortTreeData(sorts []model.ContentSort) []map[string]interface{} {
	// 1. Build children map: pcode → []ContentSort
	children := make(map[string][]model.ContentSort)
	for _, s := range sorts {
		pcode := s.Pcode
		if pcode == "" {
			pcode = "0"
		}
		children[pcode] = append(children[pcode], s)
	}

	// 2. Recursively flatten: parent before children
	var ordered []model.ContentSort
	var walk func(pcode string)
	walk = func(pcode string) {
		for _, s := range children[pcode] {
			ordered = append(ordered, s)
			walk(s.Scode)
		}
	}
	walk("0")

	// 3. Add Son field (has children?)
	pcodeSet := make(map[string]bool)
	for _, s := range sorts {
		if s.Pcode != "" && s.Pcode != "0" {
			pcodeSet[s.Pcode] = true
		}
	}

	result := make([]map[string]interface{}, len(ordered))
	for i, s := range ordered {
		m := StructToMap(s)
		m["Son"] = pcodeSet[s.Scode]
		// 預計算前台 URL（替代原 {php} parserLink 邏輯）
		m["FrontUrl"] = buildSortFrontURL(s)
		result[i] = m
	}
	return result
}

// buildSortFrontURL 生成欄目前台連結（對齊 PHP parserLink for list type）
func buildSortFrontURL(s model.ContentSort) string {
	if s.Outlink != "" {
		return s.Outlink
	}
	if s.Filename != "" {
		return "/" + s.Filename + "/"
	}
	if s.URLName != "" {
		return "/" + s.URLName + "/"
	}
	return "/sort/" + s.Scode
}

// AddSortName adds a computed "SortName" field to each content entry.
// Uses a scode→name mapping from the sorts list.
func AddSortName(contents []model.Content, sorts []model.ContentSort) []map[string]interface{} {
	sortMap := make(map[string]string)
	sortURLMap := make(map[string]string)
	sortFilenameMap := make(map[string]string)
	for _, s := range sorts {
		sortMap[s.Scode] = s.Name
		sortURLMap[s.Scode] = s.URLName
		sortFilenameMap[s.Scode] = s.Filename
	}

	// 讀取 URL 規則配置
	contentPathMode := model.GetConfigValue("url_rule_content_path", "0")

	result := make([]map[string]interface{}, len(contents))
	for i, c := range contents {
		m := StructToMap(c)
		m["Sortname"] = sortMap[c.Scode]
		m["SortUrlname"] = sortURLMap[c.Scode]

		// 預計算前台 URL（替代原 {php} parserLink 邏輯）
		m["FrontUrl"] = buildFrontURL(c, sortFilenameMap[c.Scode], sortURLMap[c.Scode], contentPathMode)

		// Format date for display: zero time shows empty string
		if !c.Date.IsZero() {
			m["Date"] = c.Date.Format("2006-01-02 15:04:05")
		} else {
			m["Date"] = ""
		}
		if !c.UpdateTime.IsZero() {
			m["UpdateTime"] = c.UpdateTime.Format("2006-01-02 15:04:05")
		} else {
			m["UpdateTime"] = ""
		}
		if !c.CreateTime.IsZero() {
			m["CreateTime"] = c.CreateTime.Format("2006-01-02 15:04:05")
		} else {
			m["CreateTime"] = ""
		}
		result[i] = m
	}
	return result
}

// buildFrontURL 生成內容前台連結（Google SEO 標準：無 .html 副檔名）
// URL 規則：
//   1. 多段 slug（含 /，如 test/a/b）→ /test/a/b（自定義完整路徑）
//   2. 單段 slug（如 my-article）→ /{sortPath}/{slug}（欄目路徑 + slug）
//   3. 無 slug，欄目有 pathname → /{sortPath}/{id}
//   4. 無 slug，欄目無 pathname → /content/{id}（兜底）
func buildFrontURL(c model.Content, sortFilename, sortURLName, contentPathMode string) string {
	if c.Outlink != "" {
		return c.Outlink
	}
	sortPath := sortFilename
	if sortPath == "" {
		sortPath = sortURLName
	}
	// 多段 slug（含 /）→ 直接作為完整路徑
	if c.Filename != "" {
		if strings.Contains(c.Filename, "/") {
			return "/" + c.Filename
		}
		// 單段 slug → 欄目路徑 + slug
		if sortPath != "" {
			return "/" + sortPath + "/" + c.Filename
		}
		return "/" + c.Filename
	}
	if c.URLName != "" {
		if strings.Contains(c.URLName, "/") {
			return "/" + c.URLName
		}
		if sortPath != "" {
			return "/" + sortPath + "/" + c.URLName
		}
		return "/" + c.URLName
	}
	// 無 slug，欄目有 pathname → /{sortPath}/{id}
	if sortPath != "" {
		return "/" + sortPath + "/" + strconv.Itoa(int(c.ID))
	}
	// 兜底
	return "/content/" + strconv.Itoa(int(c.ID))
}

// BuildPagebarHTML generates pagination HTML for content lists.
func BuildPagebarHTML(total int64, page, pageSize int, baseURL string) *pongo2.Value {
	return buildPagebarImpl(total, page, pageSize, baseURL, "page")
}

// BuildPagebarHTMLEx 同 BuildPagebarHTML 但支援自訂分頁參數名
func BuildPagebarHTMLEx(total int64, page, pageSize int, baseURL, pageParam string) *pongo2.Value {
	return buildPagebarImpl(total, page, pageSize, baseURL, pageParam)
}

func buildPagebarImpl(total int64, page, pageSize int, baseURL, pageParam string) *pongo2.Value {
	if total <= 0 || pageSize <= 0 {
		return pongo2.AsSafeValue("<div class=\"page\"><span class=\"page-none\" style=\"color:#999\">未查詢到任何數據!</span></div>")
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	if page < 1 {
		page = 1
	}

	// buildPath 對齊 PbootCMS Paging::buildPath：帶分頁參數的 URL
	buildPath := func(p int) string {
		if p <= 0 {
			return baseURL
		}
		// 判斷 baseURL 是否已有查詢參數
		if strings.Contains(baseURL, "?") {
			return fmt.Sprintf("%s&%s=%d", baseURL, pageParam, p)
		}
		return fmt.Sprintf("%s?%s=%d", baseURL, pageParam, p)
	}

	var sb strings.Builder
	sb.WriteString(`<div class="page">`)

	// 無數據
	if totalPages == 0 {
		sb.WriteString(`<span class="page-none" style="color:#999">未查詢到任何數據!</span>`)
		sb.WriteString(`</div>`)
		return pongo2.AsSafeValue(sb.String())
	}

	// page-status：共X条 当前X/X页（對齊 PbootCMS pageStatus）
	sb.WriteString(fmt.Sprintf(`<span class="page-status">共%d條 當前%d/%d頁</span>`, total, page, totalPages))

	// page-index：首頁（對齊 PbootCMS pageIndex）
	sb.WriteString(fmt.Sprintf(`<span class="page-index"><a href="%s">首頁</a></span>`, buildPath(1)))

	// page-pre：前一頁（對齊 PbootCMS pagePre，當前頁為 1 時連結為 baseURL）
	prePage := baseURL
	if page > 1 {
		prePage = buildPath(page - 1)
	}
	sb.WriteString(fmt.Sprintf(`<span class="page-pre"><a href="%s">前一頁</a></span>`, prePage))

	// page-numbar：數字分頁（對齊 PbootCMS pageNumBar，後台固定顯示 5 個數字）
	sb.WriteString(`<span class="page-numbar">`)
	numTotal := 5
	halfl := numTotal / 2
	halfu := (numTotal + 1) / 2

	if page > halfu {
		sb.WriteString(`<span class="page-num">···</span>`)
	}

	if page <= halfl || totalPages < numTotal {
		for i := 1; i <= numTotal; i++ {
			if i > totalPages {
				break
			}
			if page == i {
				sb.WriteString(fmt.Sprintf(`<a href="%s" class="page-num page-num-current">%d</a>`, buildPath(i), i))
			} else {
				sb.WriteString(fmt.Sprintf(`<a href="%s" class="page-num">%d</a>`, buildPath(i), i))
			}
		}
	} else if page+halfl >= totalPages {
		for i := totalPages - numTotal + 1; i <= totalPages; i++ {
			if page == i {
				sb.WriteString(fmt.Sprintf(`<a href="%s" class="page-num page-num-current">%d</a>`, buildPath(i), i))
			} else {
				sb.WriteString(fmt.Sprintf(`<a href="%s" class="page-num">%d</a>`, buildPath(i), i))
			}
		}
	} else {
		for i := page - halfl; i <= page+halfl; i++ {
			if page == i {
				sb.WriteString(fmt.Sprintf(`<a href="%s" class="page-num page-num-current">%d</a>`, buildPath(i), i))
			} else {
				sb.WriteString(fmt.Sprintf(`<a href="%s" class="page-num">%d</a>`, buildPath(i), i))
			}
		}
	}

	if totalPages > numTotal && page < totalPages-halfl {
		sb.WriteString(`<span class="page-num">···</span>`)
	}
	sb.WriteString(`</span>`)

	// page-next：後一頁（對齊 PbootCMS pageNext，當前頁為最後一頁時連結為 baseURL）
	nextPage := baseURL
	if page < totalPages {
		nextPage = buildPath(page + 1)
	}
	sb.WriteString(fmt.Sprintf(`<span class="page-next"><a href="%s">後一頁</a></span>`, nextPage))

	// page-last：尾頁（對齊 PbootCMS pageLast）
	sb.WriteString(fmt.Sprintf(`<span class="page-last"><a href="%s">尾頁</a></span>`, buildPath(totalPages)))

	sb.WriteString(`</div>`)
	return pongo2.AsSafeValue(sb.String())
}

// GetExtFieldsByMcode returns extended fields for a given model code.
func GetExtFieldsByMcode(mcode string) []contentModel.ExtField {
	return contentModel.GetExtFieldsByModelCode(mcode)
}

// GetExtFieldsByMcodeAndScode returns extended fields for a given model code and scode.
// Returns fields where scode is empty (全展示) OR scode matches the given scode.
func GetExtFieldsByMcodeAndScode(mcode, scode string) []contentModel.ExtField {
	return contentModel.GetExtFieldsByModelCodeAndScode(mcode, scode)
}

// GetModelNameByMcode returns the model name for a given mcode.
func GetModelNameByMcode(mcode string) string {
	if mcode == "" {
		return "内容"
	}
	m := contentModel.GetModelByMcode(mcode)
	if m.Name != "" {
		return m.Name
	}
	return "内容"
}

// BuildSortSelectWithSelected generates sort select HTML with the current sort pre-selected.
// Used in content mod forms where the content's scode should be selected.
func BuildSortSelectWithSelected(sorts []model.ContentSort, mcode string, selectedScode string) *pongo2.Value {
	var sb strings.Builder
	for _, s := range sorts {
		if mcode != "" && s.Mcode != mcode {
			continue
		}
		sel := ""
		if s.Scode == selectedScode {
			sel = " selected"
		}
		sb.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`, s.Scode, sel, s.Name))
	}
	return pongo2.AsSafeValue(sb.String())
}

// snakeToPascal converts snake_case to PascalCase.
// snakeToPascal 委託到 core/basic 的統一實作，確保轉譯器、模板渲染、服務層使用同一份轉換規則
func snakeToPascal(s string) string {
	return basic.SnakeToPascal(s)
}

// ParseInt safely parses a string to int, returning 0 on error.
func ParseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// ParseWildcardAction parses a gin wildcard param (*action) into a map.
// Supports two URL conventions produced by PbootCMS PHP templates:
//   e.g. "/scode/123/field/status/value/0" → map[scode:123 field:status value:0]
//   e.g. "/mcode/1/id/456"                  → map[mcode:1 id:456]
//   e.g. "/123"                            → map[id:123]  (single ID segment)
//   e.g. "/123,scode"                      → map[id:123 scode_marker:""]
//        (the ",scode" suffix is a PbootCMS PHP template artifact — it tells
//         the controller the *previous* segment should be looked up by scode,
//         not by primary key. We preserve the marker so the caller can decide.)
func ParseWildcardAction(action string) map[string]string {
	result := map[string]string{}
	action = strings.TrimPrefix(action, "/")
	if action == "" {
		return result
	}
	parts := strings.Split(action, "/")
	if len(parts) == 1 {
		// Single segment: may be "123" or "123,scode" or "123,id"
		seg := strings.TrimSpace(parts[0])
		// Strip trailing Backurl artifact ("12&mcode=3" → "12")
		if idx := strings.IndexAny(seg, "&?"); idx >= 0 {
			seg = seg[:idx]
		}
		if i := strings.Index(seg, ","); i >= 0 {
			// "123,scode" → id=123, scode_marker present
			result["id"] = seg[:i]
			marker := seg[i+1:]
			if marker == "scode" || marker == "id" {
				// Marker indicates the controller should treat 'id' as a scode lookup
				result["_lookup_by"] = marker
			}
			return result
		}
		// Pure ID
		result["id"] = seg
		return result
	}
	// Key-value pairs
	for i := 0; i+1 < len(parts); i += 2 {
		val := strings.TrimSpace(parts[i+1])
		// Strip trailing Backurl artifact: when the template appends
		// {$backurl} (e.g. "&mcode=3") to the form action URL without
		// a '?' prefix, the '&...' becomes part of the URL path segment
		// instead of query string. This breaks strconv.Atoi on numeric IDs.
		// Example: "12&mcode=3" → "12"
		if idx := strings.IndexAny(val, "&?"); idx >= 0 {
			val = val[:idx]
		}
		// Resolve function-call literals like "get(mcode)" — these are PbootCMS PHP
		// URL-generation artifacts where the template wrote get(mcode) to mean
		// "use the current GET parameter value". Keep the raw string; the caller
		// (or ResolveActionGetParams) will resolve them against actual query params.
		result[parts[i]] = val
	}
	return result
}

// ResolveActionGetParams resolves values in params that look like "get(X)" by
// reading the actual value from the given gin request query/post/params.
// For example, if params["mcode"] = "get(mcode)", it replaces it with c.Query("mcode").
// This handles the PbootCMS PHP URL convention where {url./admin/.../mcode/get(mcode)/...}
// produces literal "get(mcode)" text in the URL path.
func ResolveActionGetParams(params map[string]string, c interface{ GetQuery(key string) (string, bool); PostForm(key string) string; Param(key string) string }) map[string]string {
	getRe := regexp.MustCompile(`^get\((\w+)\)$`)
	for k, v := range params {
		if m := getRe.FindStringSubmatch(v); len(m) == 2 {
			innerKey := m[1]
			// Try query first, then post form, then path param
			if qv, ok := c.GetQuery(innerKey); ok && qv != "" {
				params[k] = qv
			} else if pv := c.PostForm(innerKey); pv != "" {
				params[k] = pv
			} else if pv := c.Param(innerKey); pv != "" {
				params[k] = pv
			}
		}
	}
	return params
}
