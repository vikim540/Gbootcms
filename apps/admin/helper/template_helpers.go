package helper

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"pbootcms-go/apps/admin/model"
	contentModel "pbootcms-go/apps/admin/model/content"
	memberModel "pbootcms-go/apps/admin/model/member"

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

// BuildGroupsData returns member groups with "Gname" alias for template compatibility.
// Templates use [value->gname] but the Go model has Name field.
func BuildGroupsData() []map[string]interface{} {
	var groups []memberModel.MemberGroup
	model.DB.Where("status = 1").Order("id ASC").Find(&groups)
	result := make([]map[string]interface{}, len(groups))
	for i, g := range groups {
		result[i] = map[string]interface{}{
			"ID":    g.ID,
			"Gname": g.Name, // alias: gname → Gname
			"Code":  g.Code,
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
		result[i] = m
	}
	return result
}

// AddSortName adds a computed "SortName" field to each content entry.
// Uses a scode→name mapping from the sorts list.
func AddSortName(contents []model.Content, sorts []model.ContentSort) []map[string]interface{} {
	sortMap := make(map[string]string)
	sortURLMap := make(map[string]string)
	for _, s := range sorts {
		sortMap[s.Scode] = s.Name
		sortURLMap[s.Scode] = s.URLName
	}
	result := make([]map[string]interface{}, len(contents))
	for i, c := range contents {
		m := StructToMap(c)
		m["Sortname"] = sortMap[c.Scode]
		m["SortUrlname"] = sortURLMap[c.Scode]
		// Format date for display
		if !c.Date.IsZero() {
			m["Date"] = c.Date.Format("2006-01-02 15:04:05")
		}
		result[i] = m
	}
	return result
}

// BuildPagebarHTML generates pagination HTML for content lists.
func BuildPagebarHTML(total int64, page, pageSize int, baseURL string) *pongo2.Value {
	if total <= 0 || pageSize <= 0 {
		return pongo2.AsSafeValue("")
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	if totalPages <= 1 {
		return pongo2.AsSafeValue("")
	}
	if page < 1 {
		page = 1
	}

	var sb strings.Builder
	sb.WriteString(`<div class="page">`)

	// First + Previous
	if page > 1 {
		sb.WriteString(fmt.Sprintf(`<a href="%s&page=1">首页</a>`, baseURL))
		sb.WriteString(fmt.Sprintf(`<a href="%s&page=%d">上一页</a>`, baseURL, page-1))
	}

	// Page numbers
	start := page - 3
	if start < 1 {
		start = 1
	}
	end := page + 3
	if end > totalPages {
		end = totalPages
	}
	for i := start; i <= end; i++ {
		if i == page {
			sb.WriteString(fmt.Sprintf(`<span class="current">%d</span>`, i))
		} else {
			sb.WriteString(fmt.Sprintf(`<a href="%s&page=%d">%d</a>`, baseURL, i, i))
		}
	}

	// Next + Last
	if page < totalPages {
		sb.WriteString(fmt.Sprintf(`<a href="%s&page=%d">下一页</a>`, baseURL, page+1))
		sb.WriteString(fmt.Sprintf(`<a href="%s&page=%d">末页</a>`, baseURL, totalPages))
	}

	sb.WriteString(`</div>`)
	return pongo2.AsSafeValue(sb.String())
}

// GetExtFieldsByMcode returns extended fields for a given model code.
func GetExtFieldsByMcode(mcode string) []contentModel.ExtField {
	return contentModel.GetExtFieldsByModelCode(mcode)
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
func snakeToPascal(s string) string {
	if s == "" {
		return ""
	}
	upperWords := map[string]string{
		"ip": "IP", "id": "ID", "url": "URL", "api": "API",
		"db": "DB", "cms": "CMS", "html": "HTML",
	}
	// PbootCMS compound words without underscore separator
	compoundMap := map[string]string{
		"contenttpl":  "ContentTpl",
		"listtpl":     "ListTpl",
		"urlname":     "URLName",
		"outlink":     "Outlink",
		"keywords":    "Keywords",
		"createuser":  "CreateUser",
		"updateuser":  "UpdateUser",
		"createtime":  "CreateTime",
		"updatetime":  "UpdateTime",
		"create_user": "CreateUser",
		"update_user": "UpdateUser",
		"create_time": "CreateTime",
		"update_time": "UpdateTime",
		"subname":     "Subname",
		"gnote":       "Gnote",
		"gtype":       "GType",
		"sortselect":  "SortSelect",
		"menutree":    "MenuTree",
		"menumodels":  "MenuModels",
		"sitedir":     "SiteDir",
		"sitetitle":   "SiteTitle",
	}
	if v, ok := compoundMap[strings.ToLower(s)]; ok {
		return v
	}
	parts := strings.Split(s, "_")
	var result string
	for _, p := range parts {
		if len(p) > 0 {
			if up, ok := upperWords[strings.ToLower(p)]; ok {
				result += up
			} else {
				result += strings.ToUpper(p[:1]) + p[1:]
			}
		}
	}
	return result
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
		seg := parts[0]
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
		result[parts[i]] = parts[i+1]
	}
	return result
}
