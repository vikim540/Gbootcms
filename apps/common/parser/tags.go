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
		"pre":           `(?s)\{pboot:pre\}(.*?)\{\/pboot:pre\}`,
		"include":       `\{include\s+file\s?=\s?["']?([\w.\-\/@]+)["']?\s*\}`,
		"site":          `\{pboot:site(\w+)(?:\s+([^}]+))?\}`,
		"company":       `\{pboot:company(\w+)(?:\s+([^}]+))?\}`,
		"label":         `\{label:(\w+)(?:\s+([^}]+))?\}`,
		"user":          `\{user:(\w+)(?:\s+([^}]+))?\}`,
		"sort_single":   `\{sort:(\w+)(?:\s+([^}]+))?\}`,
		"content_single": `\{content:(\w+)(?:\s+([^}]+))?\}`,
		"page":          `\{page:(\w+)\}`,
		"pboot_single":  `\{pboot:(\w+)(?:\s+([^}]+))?\}`,
		"position":      `\{pboot:position(?:\s+([^}]+))?\}`,
		"selectall":     `\{pboot:selectall(?:\s+([^}]+))?\}`,
		"qrcode":        `\{pboot:qrcode(?:\s+([^}]+))?\}`,
		"form_single":   `\{pboot:form(?:\s+([^}]+))?\}`,
		"nav":           `(?s)\{pboot:nav(?:\s+([^}]+))?\}(.*?)\{\/pboot:nav\}`,
		"sort_loop":     `(?s)\{pboot:sort(?:\s+([^}]+))?\}(.*?)\{\/pboot:sort\}`,
		"list":          `(?s)\{pboot:list(?:\s+([^}]+))?\}(.*?)\{\/pboot:list\}`,
		"content_loop":  `(?s)\{pboot:content(?:\s+([^}]+))?\}(.*?)\{\/pboot:content\}`,
		"pics":          `(?s)\{pboot:pics(?:\s+([^}]+))?\}(.*?)\{\/pboot:pics\}`,
		"checkbox":      `(?s)\{pboot:checkbox(?:\s+([^}]+))?\}(.*?)\{\/pboot:checkbox\}`,
		"tags":          `(?s)\{pboot:tags(?:\s+([^}]+))?\}(.*?)\{\/pboot:tags\}`,
		"slide":         `(?s)\{pboot:slide(?:\s+([^}]+))?\}(.*?)\{\/pboot:slide\}`,
		"link":          `(?s)\{pboot:link(?:\s+([^}]+))?\}(.*?)\{\/pboot:link\}`,
		"message":       `(?s)\{pboot:message(?:\s+([^}]+))?\}(.*?)\{\/pboot:message\}`,
		"formlist":      `(?s)\{pboot:formlist(?:\s+([^}]+))?\}(.*?)\{\/pboot:formlist\}`,
		"search":        `(?s)\{pboot:search(?:\s+([^}]+))?\}(.*?)\{\/pboot:search\}`,
		"comment":       `(?s)\{pboot:comment(?:\s+([^}]+))?\}(.*?)\{\/pboot:comment\}`,
		"commentsub":    `(?s)\{pboot:commentsub(?:\s+([^}]+))?\}(.*?)\{\/pboot:commentsub\}`,
		"mycomment":     `(?s)\{pboot:mycomment(?:\s+([^}]+))?\}(.*?)\{\/pboot:mycomment\}`,
		"loop":          `(?s)\{pboot:loop(?:\s+([^}]+))?\}(.*?)\{\/pboot:loop\}`,
		"select":        `(?s)\{pboot:select(?:\s+([^}]+))?\}(.*?)\{\/pboot:select\}`,
		"pboot_if":      `(?s)\{pboot:if\(([^}]+)\)\}(.*?)(?:\{else\}(.*?))?\{\/pboot:if\}`,
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

	// Pre-resolve single tags inside pair tag params (e.g. {pboot:list scode={sort:scode}})
	content = p.preResolveSingleInPairParams(content)
	content = p.processPairTags(content)
	content = p.processIfTags(content)
	content = p.processSingleTags(content)

	for i, block := range p.preBlocks {
		content = strings.Replace(content, fmt.Sprintf("{__PRE_%d__}", i), block, 1)
	}

	return content
}

// RenderWithoutInclude æ¸²æŸ“æ¨¡æ¿ä½†ä¸å¤„ç† include æ ‡ç­¾ï¼ˆinclude å·²ç”±ä¸Šå±‚å¤„ç†ï¼?
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
	// Pre-resolve single tags inside pair tag params (e.g. {pboot:list scode={sort:scode}})
	content = p.preResolveSingleInPairParams(content)
	content = p.processPairTags(content)
	content = p.processIfTags(content)
	content = p.processSingleTags(content)

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

	re := p.re("pboot_single")
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

	for depth := 3; depth >= 0; depth-- {
		var re *regexp.Regexp
		if depth > 0 {
			prefix := strconv.Itoa(depth)
			pattern := fmt.Sprintf(
				`(?s)\{pboot:%sif\(([^}]+)\)\}(.*?)(?:\{else\}(.*?))?\{\/pboot:%sif\}`,
				prefix, prefix,
			)
			var err error
			re, err = regexp.Compile(pattern)
			if err != nil {
				continue
			}
		} else {
			re = p.re("pboot_if")
		}
		if re == nil {
			continue
		}

		content = re.ReplaceAllStringFunc(content, func(match string) string {
			subs := re.FindStringSubmatch(match)
			if len(subs) < 2 {
				return match
			}
			cond := subs[1]
			trueBranch := ""
			falseBranch := ""
			if len(subs) > 2 {
				trueBranch = subs[2]
			}
			if len(subs) > 3 {
				falseBranch = subs[3]
			}
			params := map[string]string{
				"condition": cond, "true": trueBranch, "false": falseBranch,
			}
			return providerCall(pr, "if", params, "")
		})
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
// e.g. {pboot:list scode={sort:scode} num=15} → {pboot:list scode=5 num=15}
func (p *TagParser) preResolveSingleInPairParams(content string) string {
	// Match pair tag openings and capture the params section
	pairNames := []string{
		"list", "nav", "sort_loop", "search", "message", "tags",
		"slide", "link", "pics", "checkbox", "formlist", "comment",
		"commentsub", "mycomment", "loop", "select",
	}
	for _, name := range pairNames {
		// Match opening like {pboot:NAME ...params...}
		// The params end at the first } that closes the opening tag
		pattern := regexp.MustCompile(`\{pboot:` + name + `\s+([^}]+)\}`)
		content = pattern.ReplaceAllStringFunc(content, func(match string) string {
			subs := pattern.FindStringSubmatch(match)
			if len(subs) < 2 {
				return match
			}
			paramStr := subs[1]
			// Resolve single tags within the params
			resolved := p.resolveSingleTagsInString(paramStr)
			if resolved != paramStr {
				return "{pboot:" + name + " " + resolved + "}"
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
