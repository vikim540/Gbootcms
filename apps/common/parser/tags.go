package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DataProvider func(tagName string, params map[string]string, inner string) string

type TagParser struct {
	providers  map[string]DataProvider
	regexes    map[string]*regexp.Regexp
	mu         sync.RWMutex
	preBlocks  []string
}

func New() *TagParser {
	p := &TagParser{
		providers: make(map[string]DataProvider),
		regexes:   make(map[string]*regexp.Regexp),
	}
	p.initRegexes()
	return p
}

func (p *TagParser) Register(name string, provider DataProvider) {
	p.mu.Lock()
	p.providers[name] = provider
	p.mu.Unlock()
}

func (p *TagParser) provider(name string) (DataProvider, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pr, ok := p.providers[name]
	return pr, ok
}

func (p *TagParser) initRegexes() {
	defs := map[string]string{
		"pre":           `(?s)\{gboot:pre\}(.*?)\{\/gboot:pre\}`,
		"include":       `\{include\s+file\s?=\s?["']?([\w.\-\/@]+)["']?\s*\}`,
		"site":          `\{gboot:site(\w+)(?:\s+([^}]+))?\}`,
		"company":       `\{gboot:company(\w+)(?:\s+([^}]+))?\}`,
		"label":         `\{label:(\w+)(?:\s+([^}]+))?\}`,
		"user":          `\{user:(\w+)(?:\s+([^}]+))?\}`,
		"sort_single":   `\{sort:(\w+)(?:\s+([^}]+))?\}`,
		"content_single": `\{content:(\w+)(?:\s+([^}]+))?\}`,
		"page":          `\{page:(\w+)\}`,
		"gboot_single":  `\{gboot:(\w+)(?:\s+([^}]+))?\}`,
		"position":      `\{gboot:position(?:\s+([^}]+))?\}`,
		"selectall":     `\{gboot:selectall(?:\s+([^}]+))?\}`,
		"qrcode":        `\{gboot:qrcode(?:\s+([^}]+))?\}`,
		"form_single":   `\{gboot:form(?:\s+([^}]+))?\}`,
		"nav":           `(?s)\{gboot:nav(?:\s+([^}]+))?\}(.*?)\{\/gboot:nav\}`,
		"sort_loop":     `(?s)\{gboot:sort(?:\s+([^}]+))?\}(.*?)\{\/gboot:sort\}`,
		"list":          `(?s)\{gboot:list(?:\s+([^}]+))?\}(.*?)\{\/gboot:list\}`,
		"content_loop":  `(?s)\{gboot:content(?:\s+([^}]+))?\}(.*?)\{\/gboot:content\}`,
		"pics":          `(?s)\{gboot:pics(?:\s+([^}]+))?\}(.*?)\{\/gboot:pics\}`,
		"checkbox":      `(?s)\{gboot:checkbox(?:\s+([^}]+))?\}(.*?)\{\/gboot:checkbox\}`,
		"tags":          `(?s)\{gboot:tags(?:\s+([^}]+))?\}(.*?)\{\/gboot:tags\}`,
		"slide":         `(?s)\{gboot:slide(?:\s+([^}]+))?\}(.*?)\{\/gboot:slide\}`,
		"link":          `(?s)\{gboot:link(?:\s+([^}]+))?\}(.*?)\{\/gboot:link\}`,
		"message":       `(?s)\{gboot:message(?:\s+([^}]+))?\}(.*?)\{\/gboot:message\}`,
		"formlist":      `(?s)\{gboot:formlist(?:\s+([^}]+))?\}(.*?)\{\/gboot:formlist\}`,
		"search":        `(?s)\{gboot:search(?:\s+([^}]+))?\}(.*?)\{\/gboot:search\}`,
		"comment":       `(?s)\{gboot:comment(?:\s+([^}]+))?\}(.*?)\{\/gboot:comment\}`,
		"commentsub":    `(?s)\{gboot:commentsub(?:\s+([^}]+))?\}(.*?)\{\/gboot:commentsub\}`,
		"mycomment":     `(?s)\{gboot:mycomment(?:\s+([^}]+))?\}(.*?)\{\/gboot:mycomment\}`,
		"loop":          `(?s)\{gboot:loop(?:\s+([^}]+))?\}(.*?)\{\/gboot:loop\}`,
		"select":        `(?s)\{gboot:select(?:\s+([^}]+))?\}(.*?)\{\/gboot:select\}`,
		"gboot_if":      `(?s)\{gboot:if\(([^}]+)\)\}(.*?)(?:\{else\}(.*?))?\{\/gboot:if\}`,
	}
	for name, pattern := range defs {
		if re, err := regexp.Compile(pattern); err == nil {
			p.regexes[name] = re
		}
	}
}

func (p *TagParser) re(name string) *regexp.Regexp {
	return p.regexes[name]
}

func ParseParams(s string) map[string]string {
	m := make(map[string]string)
	if s == "" {
		return m
	}
	re := regexp.MustCompile(`(\w+)\s?=\s*["']([^"']*)["']|(\w+)\s?=\s*(\S+)`)
	for _, sub := range re.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			m[sub[1]] = sub[2]
		} else if sub[3] != "" {
			m[sub[3]] = sub[4]
		}
	}
	return m
}

func (p *TagParser) Render(content string) string {
	p.preBlocks = nil

	if re := p.re("pre"); re != nil {
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			subs := re.FindStringSubmatch(match)
			if len(subs) > 1 {
				idx := len(p.preBlocks)
				p.preBlocks = append(p.preBlocks, subs[1])
				return fmt.Sprintf("{__PRE_%d__}", idx)
			}
			return match
		})
	}

	if re := p.re("include"); re != nil {
		content = p.processInclude(content, re)
	}

	// Pre-resolve single tags inside pair tag params (e.g. {gboot:list scode={sort:scode}})
	content = p.preResolveSingleInPairParams(content)
	content = p.processSingleTags(content)
	content = p.processIfTags(content)
	content = p.processPairTags(content)

	for i, block := range p.preBlocks {
		content = strings.Replace(content, fmt.Sprintf("{__PRE_%d__}", i), block, 1)
	}

	return content
}

// RenderWithoutInclude 渲染模板但不处理 include 标签（include 已由上层处理）
func (p *TagParser) RenderWithoutInclude(content string) string {
	p.preBlocks = nil

	if re := p.re("pre"); re != nil {
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			subs := re.FindStringSubmatch(match)
			if len(subs) > 1 {
				idx := len(p.preBlocks)
				p.preBlocks = append(p.preBlocks, subs[1])
				return fmt.Sprintf("{__PRE_%d__}", idx)
			}
			return match
		})
	}

	// Skip include tags
	// Pre-resolve single tags inside pair tag params (e.g. {gboot:list scode={sort:scode}})
	content = p.preResolveSingleInPairParams(content)
	content = p.processPairTags(content)
	content = p.processSingleTags(content) // single 必須在 if 之前，否則 if 中的 {sort:xxx} 無法解析
	content = p.processIfTags(content)

	for i, block := range p.preBlocks {
		content = strings.Replace(content, fmt.Sprintf("{__PRE_%d__}", i), block, 1)
	}

	return content
}

func (p *TagParser) processInclude(content string, re *regexp.Regexp) string {
	return re.ReplaceAllStringFunc(content, func(match string) string {
		subs := re.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		pr, ok := p.provider("include")
		if !ok {
			return match
		}
		return providerCall(pr, "include", map[string]string{"file": subs[1]}, "")
	})
}

func (p *TagParser) processSingleTags(content string) string {
	singles := []struct {
		reKey    string
		provKey  string
	}{
		{"site", "site"}, {"company", "company"}, {"label", "label"},
		{"user", "user"}, {"sort_single", "sort"}, {"content_single", "content"},
		{"page", "page"}, {"position", "position"}, {"selectall", "selectall"},
		{"qrcode", "qrcode"}, {"form_single", "form"},
	}

	for _, s := range singles {
		re := p.re(s.reKey)
		if re == nil {
			continue
		}
		pr, ok := p.provider(s.provKey)
		if !ok {
			continue
		}
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			subs := re.FindStringSubmatch(match)
			field := ""
			paramStr := ""
			if len(subs) > 1 {
				field = subs[1]
			}
			if len(subs) > 2 {
				paramStr = subs[2]
			}
			params := ParseParams(paramStr)
			params["_field"] = field
			return providerCall(pr, s.provKey, params, "")
		})
	}

	re := p.re("gboot_single")
	if re != nil {
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			subs := re.FindStringSubmatch(match)
			if len(subs) < 2 {
				return match
			}
			name := subs[1]
			pr, ok := p.provider(name)
			if !ok {
				return match
			}
			params := map[string]string{}
			if len(subs) > 2 && subs[2] != "" {
				params = ParseParams(subs[2])
			}
			return providerCall(pr, name, params, "")
		})
	}

	return content
}

func (p *TagParser) processPairTags(content string) string {
	pairs := []struct {
		reKey   string
		provKey string
	}{
		{"nav", "nav"}, {"sort_loop", "sort_loop"}, {"list", "list"},
		{"content_loop", "content_loop"}, {"pics", "pics"}, {"checkbox", "checkbox"},
		{"tags", "tags"}, {"slide", "slide"}, {"link", "link"}, {"message", "message"},
		{"formlist", "formlist"}, {"search", "search"}, {"comment", "comment"},
		{"commentsub", "commentsub"}, {"mycomment", "mycomment"}, {"loop", "loop"},
		{"select", "select"},
	}

	for _, pt := range pairs {
		re := p.re(pt.reKey)
		if re == nil {
			continue
		}
		pr, ok := p.provider(pt.provKey)
		if !ok {
			continue
		}
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			subs := re.FindStringSubmatch(match)
			params := map[string]string{}
			inner := ""
			if len(subs) > 1 && subs[1] != "" {
				params = ParseParams(strings.TrimSpace(subs[1]))
			}
			if len(subs) > 2 {
				inner = subs[2]
			}
			return providerCall(pr, pt.provKey, params, inner)
		})
	}

	return content
}

func (p *TagParser) processIfTags(content string) string {
	pr, ok := p.provider("if")
	if !ok {
		return content
	}

	// 自定義解析器: 手動匹配括號深度
	for depth := 3; depth >= 0; depth-- {
		prefix := ""
		if depth > 0 {
			prefix = strconv.Itoa(depth)
		}
		openTag := fmt.Sprintf("{gboot:%sif(", prefix)
		closeTag := fmt.Sprintf("{/gboot:%sif}", prefix)

		for {
			startIdx := strings.Index(content, openTag)
			if startIdx == -1 {
				break
			}
			// 找到條件結束的匹配 ')'
			condStart := startIdx + len(openTag)
			parenDepth := 1
			i := condStart
			for i < len(content) && parenDepth > 0 {
				if content[i] == '(' {
					parenDepth++
				} else if content[i] == ')' {
					parenDepth--
				}
				i++
			}
			if parenDepth != 0 {
				break // 括號不匹配
			}
			condEnd := i - 1 // ')' 的位置
			cond := content[condStart:condEnd]

			// 找到 {/gboot:Xif}
			afterCond := content[condEnd+1:]
			closeIdx := strings.Index(afterCond, closeTag)
			if closeIdx == -1 {
				break
			}
			fullContent := afterCond[:closeIdx]
			remainder := afterCond[closeIdx+len(closeTag):]

			// 分割 true/false 分支
			trueBranch := fullContent
			falseBranch := ""
			elseTag := "{else}"
			elseIdx := strings.Index(fullContent, elseTag)
			if elseIdx != -1 {
				trueBranch = fullContent[:elseIdx]
				falseBranch = fullContent[elseIdx+len(elseTag):]
			}

			params := map[string]string{
				"condition": cond, "true": trueBranch, "false": falseBranch,
			}
			result := providerCall(pr, "if", params, "")
			content = content[:startIdx] + result + remainder
		}
	}

	return content
}

func providerCall(pr DataProvider, name string, params map[string]string, inner string) string {
	defer func() {
		recover()
	}()
	return pr(name, params, inner)
}

func ReplaceInnerTags(content string, prefix string, data map[string]interface{}) string {
	re := regexp.MustCompile(`\[` + regexp.QuoteMeta(prefix) + `:(\w+)(?:\s+([^\]]+))?\]`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		subs := re.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		field := subs[1]
		params := map[string]string{}
		if len(subs) > 2 && subs[2] != "" {
			params = ParseParams(subs[2])
		}
		if val, ok := data[field]; ok {
			return AdjustValue(ValToStr(val), params)
		}
		return match
	})
}

func AdjustValue(val string, params map[string]string) string {
	if len(params) == 0 {
		return val
	}
	if l, err := strconv.Atoi(params["len"]); err == nil && l > 0 {
		runes := []rune(val)
		if len(runes) > l {
			more := params["more"]
			val = string(runes[:l]) + more
		}
	}
	if l, err := strconv.Atoi(params["lencn"]); err == nil && l > 0 {
		runes := []rune(val)
		total := 0
		end := 0
		for i, r := range runes {
			if r > 127 {
				total += 2
			} else {
				total++
			}
			if total > l {
				break
			}
			end = i + 1
		}
		if end < len(runes) {
			more := params["more"]
			val = string(runes[:end]) + more
		}
	}
	if params["drophtml"] == "1" {
		re := regexp.MustCompile(`<[^>]*>`)
		val = re.ReplaceAllString(val, "")
	}
	if style, ok := params["style"]; ok && style != "" {
		val = FormatDate(val, style)
	}
	if params["decode"] == "1" {
		val = strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&#39;", "'").Replace(val)
	}
	if sub, ok := params["substr"]; ok && sub != "" {
		parts := strings.Split(sub, ",")
		if len(parts) == 2 {
			start, _ := strconv.Atoi(parts[0])
			length, _ := strconv.Atoi(parts[1])
			runes := []rune(val)
			if start < len(runes) {
				end := start + length
				if end > len(runes) {
					end = len(runes)
				}
				val = string(runes[start:end])
			}
		}
	}
	return val
}

func FormatDate(val string, style string) string {
	if val == "" || style == "" {
		return val
	}
	// Try to parse val as a time
	var t time.Time
	var err error
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC3339,
		"2006/01/02 15:04:05",
		"2006/01/02",
	} {
		t, err = time.Parse(layout, val)
		if err == nil {
			break
		}
	}
	if err != nil {
		return val
	}
	// Convert PHP date format to Go format
	goFmt := phpToGoFormat(style)
	return t.Format(goFmt)
}

// phpToGoFormat converts PHP date format chars to Go reference time
// Y=2006, y=06, m=01, n=1, d=02, j=2, H=15, i=04, s=05
func phpToGoFormat(php string) string {
	var sb strings.Builder
	for i := 0; i < len(php); i++ {
		switch php[i] {
		case 'Y':
			sb.WriteString("2006")
		case 'y':
			sb.WriteString("06")
		case 'm':
			sb.WriteString("01")
		case 'n':
			sb.WriteString("1")
		case 'd':
			sb.WriteString("02")
		case 'j':
			sb.WriteString("2")
		case 'H':
			sb.WriteString("15")
		case 'i':
			sb.WriteString("04")
		case 's':
			sb.WriteString("05")
		default:
			sb.WriteByte(php[i])
		}
	}
	return sb.String()
}

// preResolveSingleInPairParams resolves single tags ({sort:xxx}, {content:xxx}, etc.)
// that appear inside pair tag parameter sections.
// e.g. {gboot:list scode={sort:scode} num=15} → {gboot:list scode=5 num=15}
func (p *TagParser) preResolveSingleInPairParams(content string) string {
	// Match pair tag openings and capture the params section
	pairNames := []string{
		"list", "nav", "sort_loop", "search", "message", "tags",
		"slide", "link", "pics", "checkbox", "formlist", "comment",
		"commentsub", "mycomment", "loop", "select",
	}
	for _, name := range pairNames {
		// Match opening like {gboot:NAME ...params...}
		// The params end at the first } that closes the opening tag
		pattern := regexp.MustCompile(`\{gboot:` + name + `\s+([^}]+)\}`)
		content = pattern.ReplaceAllStringFunc(content, func(match string) string {
			subs := pattern.FindStringSubmatch(match)
			if len(subs) < 2 {
				return match
			}
			paramStr := subs[1]
			// Resolve single tags within the params
			resolved := p.resolveSingleTagsInString(paramStr)
			if resolved != paramStr {
				return "{gboot:" + name + " " + resolved + "}"
			}
			return match
		})
	}
	return content
}

// resolveSingleTagsInString resolves single tag patterns within a given string
func (p *TagParser) resolveSingleTagsInString(s string) string {
	// {sort:xxx} patterns
	reSort := regexp.MustCompile(`\{sort:(\w+)\}`)
	s = reSort.ReplaceAllStringFunc(s, func(match string) string {
		subs := reSort.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		pr, ok := p.provider("sort")
		if !ok {
			return match
		}
		return providerCall(pr, "sort", map[string]string{"_field": subs[1]}, "")
	})

	// {content:xxx} patterns
	reContent := regexp.MustCompile(`\{content:(\w+)\}`)
	s = reContent.ReplaceAllStringFunc(s, func(match string) string {
		subs := reContent.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		pr, ok := p.provider("content")
		if !ok {
			return match
		}
		return providerCall(pr, "content", map[string]string{"_field": subs[1]}, "")
	})

	// {site:xxx} patterns
	reSite := regexp.MustCompile(`\{site:(\w+)\}`)
	s = reSite.ReplaceAllStringFunc(s, func(match string) string {
		subs := reSite.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		pr, ok := p.provider("site")
		if !ok {
			return match
		}
		return providerCall(pr, "site", map[string]string{"_field": subs[1]}, "")
	})

	// {label:xxx} patterns
	reLabel := regexp.MustCompile(`\{label:(\w+)\}`)
	s = reLabel.ReplaceAllStringFunc(s, func(match string) string {
		subs := reLabel.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		pr, ok := p.provider("label")
		if !ok {
			return match
		}
		return providerCall(pr, "label", map[string]string{"_field": subs[1]}, "")
	})

	return s
}

func ValToStr(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		if val {
			return "1"
		}
		return "0"
	case []byte:
		return string(val)
	case nil:
		return ""
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return strings.Trim(string(b), `"`)
	}
}
