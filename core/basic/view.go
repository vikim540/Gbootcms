package basic

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/flosch/pongo2/v6"
)

var (
	viewOnce      sync.Once
	viewDir       string
	adminViewDir  string
	templateCache map[string]*pongo2.Template
	viewMu        sync.RWMutex
)

var (
	reInclude         = regexp.MustCompile(`\{include\s+file=['"]?([^'"{}\s]+)['"]?\}`)
	rePhpBlock        = regexp.MustCompile(`(?s)\{php\}.*?\{/php\}`)
	reDynamicDollar   = regexp.MustCompile(`\{\$([\w]+)->\{\$([\w]+)->([\w]+)\}\}`)
	reDynamicDollar2  = regexp.MustCompile(`\{\$([\w]+)->\$([\w]+)\}`)
	reForeachSon      = regexp.MustCompile(`\{foreach\s+\$([\w_]+)->son\((\w+),(\w+)(?:,(\w+))?\)\}`)
	reForeach         = regexp.MustCompile(`\{foreach\s+\$([\w_]+)\((\w+),(\w+)(?:,(\w+))?\)\}`)
	reUrlPhpConcat    = regexp.MustCompile(`\{url\.'?\.?\$([\w]+)->(\w+)'?\.?\}`)
	reUrlPath         = regexp.MustCompile(`\{url\.(/[^}'\s]+)\}`)
	reUrlMcode        = regexp.MustCompile(`\{url\./admin/(\w+)/index/mcode/'\.\$(\[\w]+)->mcode'\.\}`)
	reFunUrl          = regexp.MustCompile(`\{fun=url\('([^']+)'(?:,[^)]*)?\)\}`)
	reFunLong2ip      = regexp.MustCompile(`\{fun=long2ip\(\[([^\]]+)\]\)\}`)
	reFunGeneric      = regexp.MustCompile(`\{fun=([^}]+)\}`)
	reDollarArrow     = regexp.MustCompile(`\{\$([\w]+)->([\w]+)\}`)
	reDollarSession   = regexp.MustCompile(`\{\$session\.([\w]+)\}`)
	reDollarGet       = regexp.MustCompile(`\{\$get\.([\w]+)\}`)
	reDollarVar       = regexp.MustCompile(`\{\$([\w]+)\}`)
	reBracketArrow       = regexp.MustCompile(`\[([\w]+)->([\w]+)\]`)
	reBracketDollarArrow = regexp.MustCompile(`\[\$([\w]+)->([\w]+)\]`)
	reBracketVar         = regexp.MustCompile(`\[\$([\w]+)\]`)
)

func InitViewEngine(dir string, adminDir string) {
	viewOnce.Do(func() {
		viewDir = dir
		adminViewDir = adminDir
		templateCache = make(map[string]*pongo2.Template)
		registerPongo2Filters()
	})
}

func registerPongo2Filters() {
	pongo2.RegisterFilter("long2ip", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		return pongo2.AsValue(fmt.Sprintf("%v", in.Interface())), nil
	})
	pongo2.RegisterFilter("safe", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		return pongo2.AsSafeValue(in.Interface()), nil
	})
	pongo2.RegisterFilter("truncate", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		s := in.String()
		maxLen := param.Integer()
		if maxLen <= 0 {
			maxLen = 15
		}
		runes := []rune(s)
		if len(runes) > maxLen {
			return pongo2.AsValue(string(runes[:maxLen]) + "..."), nil
		}
		return pongo2.AsValue(s), nil
	})
	pongo2.RegisterFilter("add", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		return pongo2.AsValue(in.Integer() + param.Integer()), nil
	})
	// fix_url: 外鏈圖片路徑修正。http(s):// 或 // 開頭直接返回，其餘前綴 /
	pongo2.RegisterFilter("fix_url", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		s := in.String()
		if s == "" {
			return pongo2.AsValue(s), nil
		}
		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "//") || strings.HasPrefix(s, "/") {
			return pongo2.AsValue(s), nil
		}
		return pongo2.AsValue("/" + s), nil
	})
}

func GetAdminView(tplPath string) (*pongo2.Template, error) {
	viewMu.RLock()
	if t, ok := templateCache[tplPath]; ok {
		viewMu.RUnlock()
		return t, nil
	}
	viewMu.RUnlock()

	viewMu.Lock()
	defer viewMu.Unlock()

	if t, ok := templateCache[tplPath]; ok {
		return t, nil
	}

	t, err := compileAdminView(tplPath)
	if err != nil {
		return nil, err
	}
	templateCache[tplPath] = t
	return t, nil
}

// ClearTemplateCache 清除所有已編譯的模板內存快取
// 用於後台「清理緩存」功能，清除後下次請求會重新讀取並編譯模板
func ClearTemplateCache() {
	viewMu.Lock()
	defer viewMu.Unlock()
	templateCache = make(map[string]*pongo2.Template)
}

func compileAdminView(tplPath string) (*pongo2.Template, error) {
	// 使用adminViewDir載入後台模板
	content, err := os.ReadFile(filepath.Join(adminViewDir, tplPath))
	if err != nil {
		return nil, fmt.Errorf("讀取模板失敗 %s: %w", tplPath, err)
	}

	htmlStr := resolveViewIncludes(string(content))

	htmlStr = convertPbootToPongo2(htmlStr)

	debugPath := filepath.Join("runtime", "pongo2_debug_"+strings.ReplaceAll(tplPath, "/", "_"))
	os.MkdirAll(filepath.Dir(debugPath), 0755)
	os.WriteFile(debugPath, []byte(htmlStr), 0644)

	tmpl, err := pongo2.FromString(htmlStr)
	if err != nil {
		return nil, fmt.Errorf("Pongo2 編譯失敗 %s: %w", tplPath, err)
	}

	return tmpl, nil
}

func resolveViewIncludes(content string) string {
	for i := 0; i < 5; i++ {
		if !reInclude.MatchString(content) {
			break
		}
		content = reInclude.ReplaceAllStringFunc(content, func(match string) string {
			subs := reInclude.FindStringSubmatch(match)
			if len(subs) < 2 {
				return match
			}
			includePath := subs[1]
			// 使用adminViewDir查找包含檔案
			fullPath := filepath.Join(adminViewDir, includePath)
			data, err := os.ReadFile(fullPath)
			if err != nil {
				return fmt.Sprintf("<!-- include error: %s -->", includePath)
			}
			return string(data)
		})
	}
	return content
}

func RenderAdminView(tplPath string, ctx pongo2.Context) (string, error) {
	tmpl, err := GetAdminView(tplPath)
	if err != nil {
		return "", err
	}
	return tmpl.Execute(ctx)
}

func convertPbootToPongo2(html string) string {
	html = rePhpBlock.ReplaceAllString(html, "")

	// Fix template typo: {{if( should be {if(
	html = strings.ReplaceAll(html, "{{if(", "{if(")

	html = renameReservedVars(html)

	// Process simple {url./path} BEFORE constants (processMcodeUrls needs raw patterns)
	html = processMcodeUrls(html)

	// Handle $configs.xxx.value patterns BEFORE other processing
	html = processConfigVars(html)

	// First URL concat pass: handle {url.'/path/'.CONST.'/'.($var->field).'/more'}
	html = processUrlConcat(html)

	html = processPongo2Constants(html)

	html = processPongo2Foreach(html)

	html = processDynamicVars(html)

	// Pre-process bracket dynamic vars [$var1->$var2] before processPongo2Fun
	// so that decode_string([$content->$name]) works correctly
	html = processBracketDynamicVars(html)

	html = processPongo2Url(html)

	html = processPongo2Fun(html)

		// Second URL concat pass: handle URLs generated by fun handlers (get_btn_del/mod)
	html = processUrlConcat(html)

	html = processPongo2If(html)

	html = strings.ReplaceAll(html, "{/if}", "{% endif %}")
	html = strings.ReplaceAll(html, "{else}", "{% else %}")

	html = processPongo2BracketVars(html)

	html = processPongo2DollarVars(html)

	// Post-process: fix remaining -> syntax outside of pongo2 tags
	html = fixRemainingArrowSyntax(html)

	// Post-process: merge {{ SiteDir }}{{ Var }} → {{ Var|fix_url }}
	// 外鏈圖片（http://, https://, // 開頭）不會被前綴 / 破壞
	reSiteDirVar := regexp.MustCompile(`\{\{\s*SiteDir\s*\}\}\{\{\s*([^}]+?)\s*\}\}`)
	html = reSiteDirVar.ReplaceAllString(html, `{{ $1|fix_url }}`)

	return html
}

func fixRemainingArrowSyntax(html string) string {
	// Find {% if ... %} blocks and fix -> inside them
	re := regexp.MustCompile(`\{%\s*(if|elif)\s+([^%]+)%\}`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		subs := re.FindStringSubmatch(match)
		if len(subs) < 3 {
			return match
		}
		tag := subs[1]
		cond := subs[2]
		// Fix Word->Word patterns
		arrowRe := regexp.MustCompile(`(\w+)->(\w+)`)
		cond = arrowRe.ReplaceAllStringFunc(cond, func(m string) string {
			aSubs := arrowRe.FindStringSubmatch(m)
			if len(aSubs) < 3 {
				return m
			}
			return aSubs[1] + "." + SnakeToPascal(aSubs[2])
		})
		return fmt.Sprintf("{%% %s %s %%}", tag, cond)
	})
	return html
}

func processConfigVars(html string) string {
	// Handle [$configs.name.value] inside {if} conditions → Config.Name
	reBracketConfigValue := regexp.MustCompile(`\[\$configs\.([\w]+)\.value\]`)
	html = reBracketConfigValue.ReplaceAllStringFunc(html, func(match string) string {
		subs := reBracketConfigValue.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		return "Config." + SnakeToPascal(subs[1])
	})

	// Handle {$configs.name.value} in output → {{ Config.Name }}
	reDollarConfigValue := regexp.MustCompile(`\{\$configs\.([\w]+)\.value\}`)
	html = reDollarConfigValue.ReplaceAllStringFunc(html, func(match string) string {
		subs := reDollarConfigValue.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		return fmt.Sprintf("{{ Config.%s }}", SnakeToPascal(subs[1]))
	})

	// Handle remaining $configs.name.value in conditions (without brackets) → Config.Name
	reConfigDotValue := regexp.MustCompile(`\$configs\.([\w]+)\.value`)
	html = reConfigDotValue.ReplaceAllStringFunc(html, func(match string) string {
		subs := reConfigDotValue.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		return "Config." + SnakeToPascal(subs[1])
	})

	return html
}

func processDynamicVars(html string) string {
	html = reDynamicDollar.ReplaceAllStringFunc(html, func(match string) string {
		subs := reDynamicDollar.FindStringSubmatch(match)
		if len(subs) < 4 {
			return match
		}
		objName := SnakeToPascal(subs[1])
		keyVar := subs[2]
		keyField := SnakeToPascal(subs[3])
		if isLoopVar(keyVar) {
			return fmt.Sprintf("{{ %s[%s.%s] }}", objName, keyVar, keyField)
		}
		return fmt.Sprintf("{{ %s[%s.%s] }}", objName, SnakeToPascal(keyVar), keyField)
	})

	html = reDynamicDollar2.ReplaceAllStringFunc(html, func(match string) string {
		subs := reDynamicDollar2.FindStringSubmatch(match)
		if len(subs) < 3 {
			return match
		}
		objName := SnakeToPascal(subs[1])
		keyVar := subs[2]
		if isLoopVar(keyVar) {
			return fmt.Sprintf("{{ %s[%s] }}", objName, keyVar)
		}
		return fmt.Sprintf("{{ %s[%s] }}", objName, SnakeToPascal(keyVar))
	})

	return html
}

// processBracketDynamicVars converts [$var1->$var2] inside {fun=...} arguments to pongo2 bracket access.
// Only converts within {fun=...} tags to avoid breaking other contexts like {if} and JS code.
func processBracketDynamicVars(html string) string {
	reFunBracketDyn := regexp.MustCompile(`\{fun=([^}]*?)\[\$([\w]+)->\$([\w]+)\]([^}]*?)\}`)
	return reFunBracketDyn.ReplaceAllStringFunc(html, func(match string) string {
		subs := reFunBracketDyn.FindStringSubmatch(match)
		if len(subs) < 5 {
			return match
		}
		objName := SnakeToPascal(subs[2])
		keyName := SnakeToPascal(subs[3])
		replacement := fmt.Sprintf("%s[%s]", objName, keyName)
		return "{" + "fun=" + subs[1] + replacement + subs[4] + "}"
	})
}

func renameReservedVars(html string) string {
	replacements := []struct{ from, to string }{
		{"{url.'.$value2->url.'}", "{{ val2.URL }}"},
		{"{url.'.$value3->url.'}", "{{ val2.URL }}"},
		{"{url.'.$value->url.'}", "{{ val1.URL }}"},
		{"$value3->son", "$val2->son"},
		{"$value3->", "$val2->"},
		{"$value2->son", "$val2->son"},
		{"$value2->", "$val2->"},
		{"$value->son", "$val1->son"},
		{"$value->", "$val1->"},
		{"[value3->", "[val2->"},
		{"[value2->", "[val2->"},
		{"[value->", "[val1->"},
		{"(key3,value3)", "(key3,val2)"},
		{"(key2,value2)", "(key2,val2)"},
		{"(key,value)", "(key,val1)"},
		{"$value3]", "$val2]"},
		{"$value2]", "$val2]"},
		{"$value]", "$val1]"},
		{"!isset($value3->status)", "!isset($val2->status)"},
		{"!isset($value2->status)", "!isset($val2->status)"},
		{"!isset($value->status)", "!isset($val1->status)"},
	}
	for _, r := range replacements {
		html = strings.ReplaceAll(html, r.from, r.to)
	}
	return html
}

func processPongo2Constants(html string) string {
	replacements := map[string]string{
		"{CMSNAME}":       "{{ CmsName }}",
		"{APP_VERSION}":   "{{ AppVersion }}",
		"{RELEASE_TIME}":  "{{ ReleaseTime }}",
		"{APP_THEME_DIR}": "{{ AppThemeDir }}",
		"{CORE_DIR}":      "{{ CoreDir }}",
		"{SITE_DIR}":      "{{ SiteDir }}",
		"{URL}":           "{{ URL }}",
		"{C}":             "{{ C }}",
	}
	for k, v := range replacements {
		html = strings.ReplaceAll(html, k, v)
	}
	return html
}

func processPongo2Foreach(html string) string {
	depth := 0
	varStack := []string{}
	var result strings.Builder
	lines := strings.Split(html, "\n")
	for _, line := range lines {
		if reForeachSon.MatchString(line) {
			curVar := fmt.Sprintf("val%d", depth+1)
			parentVar := "val"
			if depth > 0 {
				parentVar = varStack[depth-1]
			}
			varStack = append(varStack, curVar)
			line = reForeachSon.ReplaceAllStringFunc(line, func(match string) string {
				return fmt.Sprintf("{%% for %s in %s.Son %%}", curVar, parentVar)
			})
			depth++
		} else if reForeach.MatchString(line) {
			curVar := fmt.Sprintf("val%d", depth+1)
			varStack = append(varStack, curVar)
			line = reForeach.ReplaceAllStringFunc(line, func(match string) string {
				subs := reForeach.FindStringSubmatch(match)
				if len(subs) < 4 {
					return match
				}
				collectionName := subs[1]
				return fmt.Sprintf("{%% for %s in %s %%}", curVar, SnakeToPascal(collectionName))
			})
			depth++
		}
		// Replace [value] → {{ valN }} and [key] → {{ keyN }} inside foreach loops
		if depth > 0 && len(varStack) > 0 {
			curVar := varStack[depth-1]
			line = strings.ReplaceAll(line, "[value]", "{{ "+curVar+" }}")
			line = strings.ReplaceAll(line, "[key]", "{{ key }}")
			// 序號：分頁時需加 PageStart 偏移量（對齊 PbootCMS: (PAGE-1)*PAGESIZE+$num）
			line = strings.ReplaceAll(line, "[num]", "{{ forloop.Counter|add:PageStart }}")
			// Replace standalone 'value' in PbootCMS conditions (e.g. {if(value!=X)}) → valN
			// But NOT inside HTML attributes like value="..." or value='...'
			// At this stage conditions are still {if(...)} not {% if ... %}
			reIfValue := regexp.MustCompile(`(\{(if|elseif)\(.*?)\bvalue\b(.*?\)\})`)
			line = reIfValue.ReplaceAllString(line, "${1}"+curVar+"${3}")
		}
		if strings.Contains(line, "{/foreach}") {
			depth--
			if depth < 0 {
				depth = 0
			}
			if len(varStack) > 0 {
				varStack = varStack[:len(varStack)-1]
			}
			line = strings.ReplaceAll(line, "{/foreach}", "{% endfor %}")
		}
		result.WriteString(line)
		result.WriteString("\n")
	}
	return result.String()
}

func processPongo2Url(html string) string {
	count := 0
	html = reUrlPhpConcat.ReplaceAllStringFunc(html, func(match string) string {
		count++
		subs := reUrlPhpConcat.FindStringSubmatch(match)
		if len(subs) < 3 {
			return match
		}
		varName := subs[1]
		field := SnakeToPascal(subs[2])
		result := fmt.Sprintf("{{ %s.%s }}", varName, field)
		return result
	})

	// Handle URLs with {{ }} template variables (lowercase only literal path parts)
	reUrlWithTpl := regexp.MustCompile(`\{url\.([^}]*\{\{[^}]+\}\}[^}]*)\}`)
	html = reUrlWithTpl.ReplaceAllStringFunc(html, func(match string) string {
		subs := reUrlWithTpl.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		path := subs[1]
		reTpl := regexp.MustCompile(`\{\{[^}]+\}\}`)
		parts := reTpl.Split(path, -1)
		tpls := reTpl.FindAllString(path, -1)
		var sb strings.Builder
		for i, part := range parts {
			sb.WriteString(strings.ToLower(part))
			if i < len(tpls) {
				sb.WriteString(tpls[i])
			}
		}
		return sb.String()
	})

	// Handle simple URLs without template variables
	html = reUrlPath.ReplaceAllStringFunc(html, func(match string) string {
		subs := reUrlPath.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		// Skip if contains {{ (already handled above)
		if strings.Contains(subs[1], "{{") {
			return match
		}
		return strings.ToLower(subs[1])
	})

	return html
}

func processPongo2Fun(html string) string {
	html = reFunUrl.ReplaceAllStringFunc(html, func(match string) string {
		subs := reFunUrl.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		return strings.ToLower(subs[1])
	})

	html = reFunLong2ip.ReplaceAllStringFunc(html, func(match string) string {
		subs := reFunLong2ip.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		raw := subs[1]
		if strings.HasPrefix(raw, "$") {
			parts := strings.SplitN(raw[1:], "->", 2)
			if len(parts) == 2 {
				return fmt.Sprintf("{{ %s.%s }}", SnakeToPascal(parts[0]), SnakeToPascal(parts[1]))
			}
			return fmt.Sprintf("{{ %s }}", SnakeToPascal(raw[1:]))
		}
		return fmt.Sprintf("{{ %s }}", SnakeToPascal(raw))
	})

	// --- Specific function handlers (before generic catch-all) ---

	// get_btn_back() → back button
	reBtnBack := regexp.MustCompile(`\{fun=get_btn_back\(\)\}`)
	html = reBtnBack.ReplaceAllString(html, `<a href="javascript:history.back();" class="layui-btn layui-btn-primary">返回</a>`)

	// get_btn_del($var->field) or get_btn_del($var->field,'fieldname') → delete link
	// 生成真實刪除 URL：呼叫 Del 控制器
	// - scode 欄位: /del?id={value}（POST/GET 均可）
	// - id 欄位:   /del/id/{value}（萬用路由）
	reBtnDel := regexp.MustCompile(`\{fun=get_btn_del\(([^)]+)\)\}`)
	html = reBtnDel.ReplaceAllStringFunc(html, func(match string) string {
		subs := reBtnDel.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		args := strings.SplitN(subs[1], ",", 2)
		valExpr := strings.TrimSpace(args[0])
		fieldName := "id" // 默認按 id 查
		if len(args) >= 2 {
			fieldName = strings.Trim(strings.TrimSpace(args[1]), "'\"")
		}
		var inner string
		if fieldName == "scode" {
			inner = "/admin/{{ C }}/del?id=" + valExpr
		} else {
			inner = "/admin/{{ C }}/del/id/" + valExpr
		}
		return `<a href="javascript:;" class="layui-btn layui-btn-xs layui-btn-danger btn-del" data-url="{url.` + inner + `}">刪除</a>`
	})

	// get_btn_mod($var->field) or get_btn_mod($var->field,'fieldname'[, 'btntext']) → edit link
	// PbootCMS 慣例: 第二個參數 'fieldname' 是類型標記,生成的 URL 為
	//   /mod/{fieldname}/{value}
	// 例如 get_btn_mod($value->scode,'scode') → /mod/scode/{scode}
	reBtnMod := regexp.MustCompile(`\{fun=get_btn_mod\(([^)]+)\)\}`)
	html = reBtnMod.ReplaceAllStringFunc(html, func(match string) string {
		subs := reBtnMod.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		// 解析最多 3 個逗號分隔的參數
		parts := strings.Split(subs[1], ",")
		valExpr := strings.TrimSpace(parts[0])
		fieldName := "id"
		btnText := "修改"
		if len(parts) >= 2 {
			fieldName = strings.Trim(strings.TrimSpace(parts[1]), "'\"")
		}
		if len(parts) >= 3 {
			btnText = strings.Trim(strings.TrimSpace(parts[2]), "'\"")
		}
		// 生成 /admin/C/mod/{fieldName}/{valExpr}
		// Btnqs 已含 ? 前綴（如 ?gid=1），不可再加 ?，否則產生 ??gid=1 導致查詢參數 key 變成 ?gid
		inner := "/admin/{{ C }}/mod/" + fieldName + "/" + valExpr
		return `<a href="{url.` + inner + `}{{ Btnqs }}" class="layui-btn layui-btn-xs">` + btnText + `</a>`
	})

	// check_level('xxx') → CheckLevelXxx (boolean variable injected by Render)
	// 對齊 PbootCMS PHP check_level()：超級管理員(uid=1)擁有所有權限，
	// 其他用戶根據 session('levels') 中的 URL 判斷是否有權限
	reCheckLevel := regexp.MustCompile(`check_level\('(\w+)'\)`)
	html = reCheckLevel.ReplaceAllStringFunc(html, func(match string) string {
		subs := reCheckLevel.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		action := subs[1]
		// 轉為 PascalCase 變數名：mod → CheckLevelMod, del → CheckLevelDel
		return "CheckLevel" + strings.ToUpper(action[:1]) + action[1:]
	})

	// date('Y-m-d H:i:s') → current datetime
	reDate := regexp.MustCompile(`\{fun=date\([^)]*\)\}`)
	html = reDate.ReplaceAllString(html, "{{ Now }}")

	// substr_both($var->field,start,len) → pongo2 truncate filter
	reSubstr := regexp.MustCompile(`\{fun=substr_both\(([^,]+),\s*(\d+),\s*(\d+)\)\}`)
	html = reSubstr.ReplaceAllStringFunc(html, func(match string) string {
		subs := reSubstr.FindStringSubmatch(match)
		if len(subs) < 4 {
			return match
		}
		arg := convertArrowToDot(strings.TrimSpace(subs[1]))
		return fmt.Sprintf("{{ %s|truncate:%s }}", arg, subs[3])
	})

	// decode_string($var) or decode_string([$var1->$var2]) → pongo2 safe filter
	reDecode := regexp.MustCompile(`\{fun=decode_string\(([\w.$\[\]>_-]+)\)\}`)
	html = reDecode.ReplaceAllStringFunc(html, func(match string) string {
		subs := reDecode.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		arg := strings.TrimSpace(subs[1])
		// Strip outer brackets: [$content->content] → $content->content
		if strings.HasPrefix(arg, "[") && strings.HasSuffix(arg, "]") {
			arg = arg[1 : len(arg)-1]
		}
		arg = convertArrowToDot(arg)
		return fmt.Sprintf("{{ %s|safe }}", arg)
	})

	// Generic catch-all: clear remaining unknown functions
	html = reFunGeneric.ReplaceAllString(html, "")

	return html
}

func processPongo2If(html string) string {
	html = replacePongo2IfTag(html, "{if(", "{% if ", " %}")
	html = replacePongo2IfTag(html, "{elseif(", "{% elif ", " %}")
	return html
}

func replacePongo2IfTag(html, prefix, outPrefix, outSuffix string) string {
	var result strings.Builder
	i := 0
	for i < len(html) {
		idx := strings.Index(html[i:], prefix)
		if idx < 0 {
			result.WriteString(html[i:])
			break
		}
		result.WriteString(html[i : i+idx])
		start := i + idx + len(prefix)
		depth := 1
		j := start
		for j < len(html) && depth > 0 {
			ch := html[j]
			if ch == '(' {
				depth++
			} else if ch == ')' {
				depth--
				if depth == 0 {
					break
				}
			}
			j++
		}
		if j >= len(html) {
			result.WriteString(html[i+idx:])
			break
		}
		condition := html[start:j]
		afterParen := j + 1
		if afterParen < len(html) && html[afterParen] == '}' {
			goCond := convertPongo2Condition(condition)
			result.WriteString(outPrefix)
			result.WriteString(goCond)
			result.WriteString(outSuffix)
			i = afterParen + 1
		} else {
			result.WriteString(html[i+idx : afterParen])
			i = afterParen
		}
	}
	return result.String()
}

func convertPongo2Condition(cond string) string {
	cond = strings.TrimSpace(cond)

	if strings.Contains(cond, "==") || strings.Contains(cond, "!=") {
	} else if strings.Contains(cond, "not ") || strings.Contains(cond, "and ") || strings.Contains(cond, "or ") {
		return cond
	}

	cond = strings.ReplaceAll(cond, "!isset($val2->status)|| $val2->status==1", "not val2.Status or val2.Status == 1")
	cond = strings.ReplaceAll(cond, "!isset($val2->status) || $val2->status==1", "not val2.Status or val2.Status == 1")
	cond = strings.ReplaceAll(cond, "!isset($val2->status)", "not val2.Status")
	cond = strings.ReplaceAll(cond, "!isset($val1->status)", "not val1.Status")

	// check_level('xxx') → true (permission check placeholder)
	cond = regexp.MustCompile(`check_level\([^)]*\)`).ReplaceAllString(cond, "true")

	cond = strings.ReplaceAll(cond, "LICENSE", "License")
	cond = strings.ReplaceAll(cond, "CMSNAME", "CmsName")

	// Handle [$var->field] patterns (dollar sign inside brackets)
	bracketDollarArrowRe := regexp.MustCompile(`(!?)\[\$([\w]+)->([\w]+)\]`)
	cond = bracketDollarArrowRe.ReplaceAllStringFunc(cond, func(m string) string {
		subs := bracketDollarArrowRe.FindStringSubmatch(m)
		if len(subs) < 4 {
			return m
		}
		negated := subs[1] == "!"
		varName := subs[2]
		fieldName := SnakeToPascal(subs[3])
		var result string
		if isLoopVar(varName) {
			result = fmt.Sprintf("%s.%s", varName, fieldName)
		} else {
			result = fmt.Sprintf("%s.%s", SnakeToPascal(varName), fieldName)
		}
		if negated {
			return "not " + result
		}
		return result
	})

	bracketArrowRe := regexp.MustCompile(`(!?)\[([\w]+)->([\w]+)\]`)
	cond = bracketArrowRe.ReplaceAllStringFunc(cond, func(m string) string {
		subs := bracketArrowRe.FindStringSubmatch(m)
		if len(subs) < 4 {
			return m
		}
		negated := subs[1] == "!"
		result := fmt.Sprintf("%s.%s", subs[2], SnakeToPascal(subs[3]))
		if negated {
			return "not " + result
		}
		return result
	})

	bracketRe := regexp.MustCompile(`(!?)\[\$([\w.]+)\]`)
	cond = bracketRe.ReplaceAllStringFunc(cond, func(m string) string {
		subs := bracketRe.FindStringSubmatch(m)
		if len(subs) < 3 {
			return m
		}
		negated := subs[1] == "!"
		result := pongo2DataKey(subs[2])
		if negated {
			return "not " + result
		}
		return result
	})

	sessionFuncRe := regexp.MustCompile(`session\('(\w+)'\)`)
	cond = sessionFuncRe.ReplaceAllStringFunc(cond, func(m string) string {
		subs := sessionFuncRe.FindStringSubmatch(m)
		if len(subs) < 2 {
			return m
		}
		// 必須轉 PascalCase，與 flattenData 注入的 session_xxx 鍵名一致
		return SnakeToPascal("session_" + subs[1])
	})

	// Handle get('xxx') function calls → GetXxx (pongo2 variable for GET params)
	getFuncRe := regexp.MustCompile(`get\(['"](\w+)['"]\)`)
	cond = getFuncRe.ReplaceAllStringFunc(cond, func(m string) string {
		subs := getFuncRe.FindStringSubmatch(m)
		if len(subs) < 2 {
			return m
		}
		return SnakeToPascal("get_" + subs[1])
	})

	dollarFieldRe := regexp.MustCompile(`\$([\w]+)->(\w+)`)
	cond = dollarFieldRe.ReplaceAllStringFunc(cond, func(m string) string {
		subs := dollarFieldRe.FindStringSubmatch(m)
		if len(subs) < 3 {
			return m
		}
		varName := subs[1]
		if isLoopVar(varName) {
			return fmt.Sprintf("%s.%s", varName, SnakeToPascal(subs[2]))
		}
		return fmt.Sprintf("%s.%s", SnakeToPascal(varName), SnakeToPascal(subs[2]))
	})

	dollarDotRe := regexp.MustCompile(`\$([\w]+)\.(\w+)`)
	cond = dollarDotRe.ReplaceAllStringFunc(cond, func(m string) string {
		subs := dollarDotRe.FindStringSubmatch(m)
		if len(subs) < 3 {
			return m
		}
		return fmt.Sprintf("%s.%s", subs[1], SnakeToPascal(subs[2]))
	})

	// Handle bare $variable (not already handled by ->field or .field patterns)
	bareDollarRe := regexp.MustCompile(`\$([\w]+)`)
	cond = bareDollarRe.ReplaceAllStringFunc(cond, func(m string) string {
		subs := bareDollarRe.FindStringSubmatch(m)
		if len(subs) < 2 {
			return m
		}
		varName := subs[1]
		if isLoopVar(varName) {
			return varName
		}
		return SnakeToPascal(varName)
	})

	cond = strings.ReplaceAll(cond, "'", "\"")
	cond = strings.ReplaceAll(cond, "&&", " and ")
	cond = strings.ReplaceAll(cond, "||", " or ")

	// Strip @ prefix (PbootCMS raw output marker, not valid in pongo2)
	cond = strings.ReplaceAll(cond, "@", "")

	// Handle remaining [fun=...] patterns (function checks inside conditions)
	funBracketRe := regexp.MustCompile(`\[fun=[^\]]*\]`)
	cond = funBracketRe.ReplaceAllString(cond, "true")

	// Handle remaining [Word.Word] brackets (e.g., leftover from partial conversion)
	residualBracketRe := regexp.MustCompile(`\[([\w.]+)\]`)
	cond = residualBracketRe.ReplaceAllStringFunc(cond, func(m string) string {
		subs := residualBracketRe.FindStringSubmatch(m)
		if len(subs) < 2 {
			return m
		}
		return subs[1]
	})

	// Handle remaining -> syntax (e.g., Content->Name → Content.Name)
	arrowRe := regexp.MustCompile(`(\w+)->(\w+)`)
	cond = arrowRe.ReplaceAllStringFunc(cond, func(m string) string {
		subs := arrowRe.FindStringSubmatch(m)
		if len(subs) < 3 {
			return m
		}
		return subs[1] + "." + SnakeToPascal(subs[2])
	})

	cond = strings.TrimSpace(cond)
	for strings.Contains(cond, "  ") {
		cond = strings.ReplaceAll(cond, "  ", " ")
	}

	return cond
}

func processPongo2BracketVars(html string) string {
	// Handle [$xxx.yyy] bracket-dot syntax (e.g. [$get.scode] → {{ get_scode }})
	reBracketDot := regexp.MustCompile(`\[\$([\w]+)\.([\w]+)\]`)
	html = reBracketDot.ReplaceAllStringFunc(html, func(match string) string {
		subs := reBracketDot.FindStringSubmatch(match)
		if len(subs) < 3 {
			return match
		}
		return fmt.Sprintf("{{ %s }}", pongo2DataKey(subs[1]+"."+subs[2]))
	})

	html = reBracketArrow.ReplaceAllStringFunc(html, func(match string) string {
		subs := reBracketArrow.FindStringSubmatch(match)
		if len(subs) < 3 {
			return match
		}
		varName := subs[1]
		fieldName := SnakeToPascal(subs[2])
		if isLoopVar(varName) {
			return fmt.Sprintf("{{ %s.%s }}", varName, fieldName)
		}
		return fmt.Sprintf("{{ %s.%s }}", SnakeToPascal(varName), fieldName)
	})

	// Handle [$var->field] bracket-dollar-arrow syntax (e.g. [$form->fcode] → {{ Form.Fcode }})
	html = reBracketDollarArrow.ReplaceAllStringFunc(html, func(match string) string {
		subs := reBracketDollarArrow.FindStringSubmatch(match)
		if len(subs) < 3 {
			return match
		}
		varName := subs[1]
		fieldName := SnakeToPascal(subs[2])
		if isLoopVar(varName) {
			return fmt.Sprintf("{{ %s.%s }}", varName, fieldName)
		}
		return fmt.Sprintf("{{ %s.%s }}", SnakeToPascal(varName), fieldName)
	})

	html = reBracketVar.ReplaceAllStringFunc(html, func(match string) string {
		subs := reBracketVar.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		return fmt.Sprintf("{{ %s }}", pongo2DataKey(subs[1]))
	})

	return html
}

func processPongo2DollarVars(html string) string {
	html = reDollarArrow.ReplaceAllStringFunc(html, func(match string) string {
		subs := reDollarArrow.FindStringSubmatch(match)
		if len(subs) < 3 {
			return match
		}
		varName := subs[1]
		fieldName := SnakeToPascal(subs[2])
		if isLoopVar(varName) {
			return fmt.Sprintf("{{ %s.%s }}", varName, fieldName)
		}
		return fmt.Sprintf("{{ %s.%s }}", SnakeToPascal(varName), fieldName)
	})

	html = reDollarSession.ReplaceAllStringFunc(html, func(match string) string {
		subs := reDollarSession.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		return fmt.Sprintf("{{ %s }}", SnakeToPascal("session_"+subs[1]))
	})

	html = reDollarGet.ReplaceAllStringFunc(html, func(match string) string {
		subs := reDollarGet.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		return fmt.Sprintf("{{ %s }}", SnakeToPascal("get_"+subs[1]))
	})

	html = reDollarVar.ReplaceAllStringFunc(html, func(match string) string {
		subs := reDollarVar.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		varName := subs[1]
		if isLoopVar(varName) {
			return "{{ " + varName + " }}"
		}
		return fmt.Sprintf("{{ %s }}", SnakeToPascal(varName))
	})

	return html
}

func isLoopVar(name string) bool {
	loopVars := map[string]bool{"val": true, "val1": true, "val2": true, "val3": true, "val4": true, "value": true, "key": true, "v": true, "k": true}
	return loopVars[name]
}

func processMcodeUrls(html string) string {
	reMcode := regexp.MustCompile(`\{url\./admin/(\w+)/[^}]*?\$([\w]+)->mcode[^}]*\}`)
	return reMcode.ReplaceAllStringFunc(html, func(match string) string {
		subs := reMcode.FindStringSubmatch(match)
		if len(subs) < 3 {
			return match
		}
		module := strings.ToLower(subs[1])
		varName := subs[2]
		return fmt.Sprintf("/admin/%s/index/mcode/{{ %s.Mcode }}", module, varName)
	})
}

func pongo2DataKey(s string) string {
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "session.") {
		field := strings.TrimPrefix(s, "session.")
		return SnakeToPascal("session_" + field)
	}
	if strings.HasPrefix(s, "get.") {
		field := strings.TrimPrefix(s, "get.")
		return SnakeToPascal("get_" + field)
	}
	if strings.HasPrefix(s, "config.") {
		field := strings.TrimPrefix(s, "config.")
		return "Config." + SnakeToPascal(field)
	}
	return SnakeToPascal(s)
}

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
		"os":   "OS",
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
		"gid":         "GID",
		"icp":         "ICP",
		"os":          "OS",
		"picmobile":   "PicMobile",
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

// convertArrowToDot converts $var->field to val.Field for pongo2
func convertArrowToDot(s string) string {
	re := regexp.MustCompile(`\$([\w]+)->(\w+)`)
	return re.ReplaceAllStringFunc(s, func(m string) string {
		subs := re.FindStringSubmatch(m)
		if len(subs) < 3 {
			return m
		}
		varName := subs[1]
		if isLoopVar(varName) {
			return fmt.Sprintf("%s.%s", varName, SnakeToPascal(subs[2]))
		}
		return fmt.Sprintf("%s.%s", SnakeToPascal(varName), SnakeToPascal(subs[2]))
	})
}

// processUrlConcat handles PHP-concat style URL tags: {url./path/'.CONST.'/'.($var->field).'/more'}
// Uses brace-aware parsing to handle nested {{ }} inside URL tags.
func processUrlConcat(html string) string {
	var result strings.Builder
	i := 0
	for i < len(html) {
		idx := strings.Index(html[i:], "{url.")
		if idx < 0 {
			result.WriteString(html[i:])
			break
		}
		result.WriteString(html[i : i+idx])
		start := i + idx
		// Find closing } with brace depth tracking (quote-aware: { } inside '...' are literal)
		depth := 0
		inQuote := false
		j := start
		for j < len(html) {
			if html[j] == '\'' {
				inQuote = !inQuote
			} else if !inQuote {
				if html[j] == '{' {
					depth++
				} else if html[j] == '}' {
					depth--
					if depth == 0 {
						break
					}
				}
			}
			j++
		}
		if j >= len(html) {
			result.WriteString(html[start:])
			break
		}
		inner := html[start+5 : j] // content between {url. and }
		// Check if it contains PHP concat indicators: ' or $ (not inside {{ }})
		hasPhpConcat := false
		for ci := 0; ci < len(inner); ci++ {
			if inner[ci] == '{' {
				// skip {{ ... }}
				if ci+1 < len(inner) && inner[ci+1] == '{' {
					for ci < len(inner) {
						if inner[ci] == '}' {
							if ci+1 < len(inner) && inner[ci+1] == '}' {
								ci += 2
								break
							}
						}
						ci++
					}
					continue
				}
			} else if inner[ci] == '\'' || inner[ci] == '$' {
				hasPhpConcat = true
				break
			}
		}
		if !hasPhpConcat {
			// Not a PHP concat URL, leave for reUrlPath or other handlers
			result.WriteString(html[start : j+1])
		} else {
			// Pre-replace get('xxx') with {{ GetXxx }} to avoid nested quote issues in splitUrlSegments
			reGetFn := regexp.MustCompile(`get\(['"](\w+)['"]\)`)
			inner = reGetFn.ReplaceAllStringFunc(inner, func(m string) string {
				subs := reGetFn.FindStringSubmatch(m)
				if len(subs) < 2 {
					return m
				}
				return "{{ " + SnakeToPascal("get_" + subs[1]) + " }}"
			})
			// Parse dot-separated segments and convert
			segments := splitUrlSegments(inner)
			var parts []string
			for _, seg := range segments {
				parts = append(parts, convertUrlSegment(seg))
			}
			result.WriteString(strings.Join(parts, ""))
		}
		i = j + 1
	}
	return result.String()
}

// splitUrlSegments splits a PHP-concat URL into tokens.
// In PHP, '.' is the string concatenation operator. So in:
//   /admin/'.C.'/mod/id/'.$value->id.'/field/status/value/0
// The dots between '/admin/' and 'C' are PHP concat, NOT path separators.
// Tokens: literals, 'quoted strings', $var->field, $var, func('args')
func splitUrlSegments(s string) []string {
	var segments []string
	i := 0
	for i < len(s) {
		// Skip PHP concat dots (dots between PHP tokens)
		if s[i] == '.' {
			i++
			continue
		}
		// Quoted string: 'content' → extract content (PHP constant or path literal)
		// Inside quotes, '.' is a literal character (PHP concat dots don't apply).
		if s[i] == '\'' {
			i++ // skip opening quote
			start := i
			for i < len(s) && s[i] != '\'' {
				i++
			}
			quoted := s[start:i]
			// Strip leading/trailing '.' inside quoted string (PHP concat artifacts)
			quoted = strings.Trim(quoted, ".")
			segments = append(segments, quoted)
			if i < len(s) {
				i++ // skip closing quote
			}
			// After closing quote, the next '.' is a PHP concat (skip it)
			if i < len(s) && s[i] == '.' {
				i++
			}
			continue
		}
		// Variable: $var->field or $var (allow '-' and '>' for arrow operator)
		if s[i] == '$' {
			start := i
			i++ // skip $
			for i < len(s) && (isWordChar(s[i]) || s[i] == '>' || s[i] == '-') {
				i++
			}
			segments = append(segments, s[start:i])
			continue
		}
		// Function call: get('mcode') or get(mcode)
		if i+4 <= len(s) && s[i:i+3] == "get" && s[i+3] == '(' {
			start := i
			depth := 0
			for i < len(s) {
				if s[i] == '(' {
					depth++
				} else if s[i] == ')' {
					depth--
					if depth == 0 {
						i++
						break
					}
				}
				i++
			}
			segments = append(segments, s[start:i])
			continue
		}
		// {{ ... }} pongo2 variable (from earlier processing)
		if s[i] == '{' && i+1 < len(s) && s[i+1] == '{' {
			start := i
			for i < len(s) {
				if s[i] == '}' && i+1 < len(s) && s[i+1] == '}' {
					i += 2
					break
				}
				i++
			}
			segments = append(segments, s[start:i])
			continue
		}
		// Literal character (path segment)
		start := i
		for i < len(s) && s[i] != '.' && s[i] != '\'' && s[i] != '$' && s[i] != '{' {
			// Also stop at function calls
			if i+4 <= len(s) && s[i:i+3] == "get" && s[i+3] == '(' {
				break
			}
			i++
		}
		if i > start {
			segments = append(segments, s[start:i])
		}
	}
	return segments
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// convertUrlSegment converts a single URL segment (PHP token) to pongo2-compatible output
func convertUrlSegment(seg string) string {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return ""
	}
	// {{ VAR }} - already pongo2 (from constant replacement)
	if strings.Contains(seg, "{{") {
		return seg
	}
	// $var->field
	if matched, _ := regexp.MatchString(`^\$\w+->\w+$`, seg); matched {
		return "{{ " + convertArrowToDot(seg) + " }}"
	}
	// $var (bare variable)
	if matched, _ := regexp.MatchString(`^\$\w+$`, seg); matched {
		varName := seg[1:]
		if isLoopVar(varName) {
			return "{{ " + varName + " }}"
		}
		return "{{ " + SnakeToPascal(varName) + " }}"
	}
	// get('xxx') or get(xxx) — function call
	if matched, _ := regexp.MatchString(`^get\(['"]?(\w+)['"]?\)$`, seg); matched {
		reFn := regexp.MustCompile(`get\(['"]?(\w+)['"]?\)`)
		subs := reFn.FindStringSubmatch(seg)
		return "{{ get_" + subs[1] + " }}"
	}
	// [$var1->$var2] — dynamic bracket variable access
	if matched, _ := regexp.MatchString(`^\[\$(\w+)->\$(\w+)\]$`, seg); matched {
		reDyn := regexp.MustCompile(`^\[\$(\w+)->\$(\w+)\]$`)
		subs := reDyn.FindStringSubmatch(seg)
		obj := SnakeToPascal(subs[1])
		key := SnakeToPascal(subs[2])
		return "{{ " + obj + "[" + key + "] }}"
	}
	// [$var->field] — bracket dollar arrow variable (e.g. [$form->fcode] → {{ Form.Fcode }})
	if matched, _ := regexp.MatchString(`^\[\$(\w+)->(\w+)\]$`, seg); matched {
		reBda := regexp.MustCompile(`^\[\$(\w+)->(\w+)\]$`)
		subs := reBda.FindStringSubmatch(seg)
		obj := subs[1]
		field := SnakeToPascal(subs[2])
		if isLoopVar(obj) {
			return fmt.Sprintf("{{ %s.%s }}", obj, field)
		}
		return fmt.Sprintf("{{ %s.%s }}", SnakeToPascal(obj), field)
	}
	// [$var.field] — bracket dot variable
	if matched, _ := regexp.MatchString(`^\[\$(\w+)\.(\w+)\]$`, seg); matched {
		reBd := regexp.MustCompile(`^\[\$(\w+)\.(\w+)\]$`)
		subs := reBd.FindStringSubmatch(seg)
		return "{{ " + pongo2DataKey(subs[1]+"."+subs[2]) + " }}"
	}
	// [$var] — bare bracket variable
	if matched, _ := regexp.MatchString(`^\[\$(\w+)\]$`, seg); matched {
		reBv := regexp.MustCompile(`^\[\$(\w+)\]$`)
		subs := reBv.FindStringSubmatch(seg)
		return "{{ " + pongo2DataKey(subs[1]) + " }}"
	}
	// PHP constant: all uppercase letter(s) like C, URL, etc.
	if matched, _ := regexp.MatchString(`^[A-Z][A-Z_]*$`, seg); matched {
		// Check if it's a known PbootCMS constant
		knownConsts := map[string]bool{"C": true, "URL": true, "SITE_DIR": true, "CORE_DIR": true, "APP_THEME_DIR": true}
		if knownConsts[seg] {
			return "{{ " + seg + " }}"
		}
	}
	// Literal path segment
	return seg
}
