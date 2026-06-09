package common

import (
	"fmt"
	"html"
	"strings"
	"time"

	"pbootcms-go/apps/admin/model"
	"pbootcms-go/core/basic"

	"github.com/flosch/pongo2/v6"
	"github.com/gin-gonic/gin"
)

var BuildVersion = "build-20260522-02-pongo2-engine"

// Render renders an admin template using the pongo2 engine with compiled template cache.
// All controllers call this function with the same interface: common.Render(c, tpl, data).
func Render(c *gin.Context, tpl string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}

	// Inject session variables (session_xxx format for pongo2)
	injectSessionData(c, data)

	// Inject CMS constants
	data["CmsName"] = "gbootcms"
	data["CoreVersion"] = "1.8.1"
	data["AppVersion"] = "3.2.12"
	data["ReleaseTime"] = "2025-04-24"
	data["SiteDir"] = ""
	data["AppThemeDir"] = "/static/admin"
	data["CoreDir"] = "/static/admin"
	data["Formcheck"] = "1"
	data["BuildVersion"] = BuildVersion
	data["License"] = 3
	// 使用原始 PbootCMS URL（若经 NoRoute 重写则取请求头中的原始路径，否则取当前路径）
	pageURL := c.Request.URL.Path
	if origPath := c.GetHeader("X-Original-Path"); origPath != "" {
		pageURL = origPath
	}
	data["URL"] = pageURL
	data["C"] = extractController(pageURL)

	// Load config from DB into nested Config map
	loadConfigToData(data)

	// Inject GET query parameters for template access (e.g. {$get.mcode} → Get_mcode after flatten)
	for key, values := range c.Request.URL.Query() {
		data["get_"+key] = values[0]
	}

	// Inject backurl / pathinfo / btnqs (used by PbootCMS mod-form action URLs)
	qParams := c.Request.URL.Query()
	var backParts, btnParts []string
	var pathinfoBuilder strings.Builder
	for key, values := range qParams {
		v := values[0]
		backParts = append(backParts, key+"="+v)
		btnParts = append(btnParts, key+"="+v)
		pathinfoBuilder.WriteString(fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, key, html.EscapeString(v)))
	}
	if len(backParts) > 0 {
		data["backurl"] = "&" + strings.Join(backParts, "&")
		data["btnqs"] = "?" + strings.Join(btnParts, "&")
	} else {
		data["backurl"] = ""
		data["btnqs"] = ""
	}
	data["pathinfo"] = pongo2.AsSafeValue(pathinfoBuilder.String())

	// Inject current datetime for {fun=date(...)} replacements
	data["now"] = time.Now().Format("2006-01-02 15:04:05")

	// Flatten struct fields for template access
	data = flattenData(data)

	// Inject menu tree and models for logged-in users
	uid := GetSessionInt(c, "admin_uid")
	if uid > 0 {
		if _, exists := data["MenuTree"]; !exists {
			data["MenuTree"] = buildMenuTree()
		}
		if _, exists := data["MenuModels"]; !exists {
			data["MenuModels"] = buildMenuModels()
		}
	}

	// pongo2 render with compiled template cache
	tmpl, err := basic.GetAdminView("admin/" + tpl)
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(500, "模板加载失败: %v", err)
		return
	}
	output, err := tmpl.Execute(pongo2.Context(data))
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(500, "模板渲染错误: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, output)
}

// injectSessionData injects session values as session_xxx keys for pongo2 templates.
// Template syntax {$session.ucode} is converted to {{ session_ucode }}.
func injectSessionData(c *gin.Context, data gin.H) {
	data["session_uid"] = GetSessionInt(c, "admin_uid")
	data["session_username"] = GetSession(c, "admin_username")
	data["session_realname"] = GetSession(c, "admin_realname")
	data["session_ucode"] = GetSession(c, "admin_ucode")
	data["session_rcodes"] = GetSession(c, "admin_rcodes")
	pwsecurity := GetSession(c, "pwsecurity")
	data["session_pwsecurity"] = pwsecurity == "1"
}

// loadConfigToData loads ay_config entries into data["Config"] as a nested map.
// Config keys use PascalCase: e.g. config name "page_size" → Config.PageSize.
func loadConfigToData(data gin.H) {
	configMap := make(map[string]interface{})
	var configs []model.Config
	model.DB.Find(&configs)
	for _, cfg := range configs {
		configMap[SnakeToPascal(cfg.Name)] = cfg.Value
	}
	data["Config"] = configMap
}

// flattenData converts data keys from snake_case to PascalCase for pongo2 template access.
// pongo2 handles struct field access and map access natively via dot notation,
// so no value flattening is needed (unlike the old regex engine).
func flattenData(data gin.H) gin.H {
	result := gin.H{}
	for k, v := range data {
		result[SnakeToPascal(k)] = v
	}
	return result
}

// MenuNode represents a menu item with children (for sidebar tree rendering).
type MenuNode struct {
	Mcode   string
	Pcode   string
	Name    string
	URL     string
	Ico     string
	Sorting int
	Status  int
	Type    int
	Son     []MenuNode
}

// ContentModelNode represents a content model entry (for sidebar model list).
type ContentModelNode struct {
	Mcode string
	Name  string
	Type  int
}

// buildMenuTree builds the sidebar menu tree from the ay_menu table.
func buildMenuTree() []MenuNode {
	var menus []model.Menu
	model.DB.Where("status = 1").Order("sorting ASC, id ASC").Find(&menus)

	menuMap := make(map[string][]MenuNode)
	for _, m := range menus {
		node := MenuNode{
			Mcode:   m.Mcode,
			Pcode:   m.Pcode,
			Name:    m.Name,
			URL:     m.URL,
			Ico:     m.Ico,
			Sorting: m.Sorting,
			Status:  m.Status,
			Type:    m.Type,
		}
		menuMap[m.Pcode] = append(menuMap[m.Pcode], node)
	}

	var tree []MenuNode
	for _, item := range menuMap[""] {
		item.Son = menuMap[item.Mcode]
		tree = append(tree, item)
	}
	return tree
}

// buildMenuModels returns all active content models for the sidebar.
func buildMenuModels() []ContentModelNode {
	var models []model.ContentModel
	model.DB.Where("status = 1").Order("sorting ASC").Find(&models)
	var result []ContentModelNode
	for _, m := range models {
		result = append(result, ContentModelNode{
			Mcode: m.Mcode,
			Name:  m.Name,
			Type:  m.Type,
		})
	}
	if result == nil {
		result = []ContentModelNode{}
	}
	return result
}

// SnakeToPascal converts snake_case to PascalCase.
// Handles special abbreviations: IP, ID, URL, API, DB, CMS, HTML.
func SnakeToPascal(s string) string {
	if s == "" {
		return ""
	}
	upperWords := map[string]string{
		"ip":   "IP",
		"id":   "ID",
		"url":  "URL",
		"api":  "API",
		"db":   "DB",
		"cms":  "CMS",
		"html": "HTML",
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

// extractController 从 PbootCMS 风格 URL 提取控制器名称
// 例: /admin/company/index → "company"
//      /admin/contentsort/mod/1 → "contentsort"
//      /admin/single/index/mcode/M13001 → "single"
//      /admin/index/home → "index"
func extractController(path string) string {
	lower := strings.ToLower(path)
	lower = strings.TrimPrefix(lower, "/admin/")
	parts := strings.SplitN(lower, "/", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "index"
}
