package parser

import (
	"regexp"
	"strings"
)

var dangerousPatterns = []string{
	"eval", "system", "exec", "passthru", "shell_exec",
	"base64_decode", "base64_encode", "phpinfo",
	"$_GET", "$_POST", "$_REQUEST", "$_SERVER",
	"file_get_contents", "file_put_contents", "fopen",
	"unlink", "rmdir", "mkdir", "chmod",
}

// 預編譯正則表達式（避免每次 resolveCondVars 調用都重新編譯）
var (
	reContentVar = regexp.MustCompile(`\[content:(\w+)\]`)
	reSortVar    = regexp.MustCompile(`\[sort:(\w+)\]`)
	reSortBrace  = regexp.MustCompile(`\{sort:(\w+)\}`)
	reGbootVar   = regexp.MustCompile(`\{gboot:(\w+)\}`)
)

func EvalIfCondition(cond string, data map[string]interface{}) bool {
	cond = strings.TrimSpace(cond)
	for _, pat := range dangerousPatterns {
		if strings.Contains(strings.ToLower(cond), strings.ToLower(pat)) {
			return false
		}
	}

	cond = resolveCondVars(cond, data)

	if strings.Contains(cond, "&&") || strings.Contains(cond, "||") {
		return evalLogicalExpr(cond)
	}

	return evalSimpleExpr(cond)
}

func resolveCondVars(cond string, data map[string]interface{}) string {
	// [content:xxx] 方括號格式
	cond = reContentVar.ReplaceAllStringFunc(cond, func(match string) string {
		subs := reContentVar.FindStringSubmatch(match)
		if len(subs) > 1 {
			if val, ok := data[subs[1]]; ok {
				return ValToStr(val)
			}
		}
		return "0"
	})

	// [sort:xxx] 方括號格式
	cond = reSortVar.ReplaceAllStringFunc(cond, func(match string) string {
		subs := reSortVar.FindStringSubmatch(match)
		if len(subs) > 1 {
			if val, ok := data[subs[1]]; ok {
				return ValToStr(val)
			}
		}
		return "0"
	})

	// {sort:xxx} 花括號格式（已被 processSingleTags 解析，但保底處理）
	cond = reSortBrace.ReplaceAllStringFunc(cond, func(match string) string {
		subs := reSortBrace.FindStringSubmatch(match)
		if len(subs) > 1 {
			if val, ok := data[subs[1]]; ok {
				return ValToStr(val)
			}
		}
		return "0"
	})

	// {gboot:xxx} 花括號格式 - 通用處理
	cond = reGbootVar.ReplaceAllStringFunc(cond, func(match string) string {
		subs := reGbootVar.FindStringSubmatch(match)
		if len(subs) > 1 {
			key := subs[1]
			// 直接從 data map 查找
			if val, ok := data[key]; ok {
				return ValToStr(val)
			}
			// 特殊映射
			switch key {
			case "islogin":
				if data["islogin"] != nil {
					return ValToStr(data["islogin"])
				}
				return "0"
			}
		}
		return "0"
	})

	return cond
}

func evalLogicalExpr(expr string) bool {
	orParts := strings.Split(expr, "||")
	for _, orPart := range orParts {
		andParts := strings.Split(orPart, "&&")
		allTrue := true
		for _, andPart := range andParts {
			if !evalSimpleExpr(strings.TrimSpace(andPart)) {
				allTrue = false
				break
			}
		}
		if allTrue {
			return true
		}
	}
	return false
}

func evalSimpleExpr(expr string) bool {
	expr = strings.TrimSpace(expr)
	expr = strings.Trim(expr, "'\"")

	ops := []string{"==", "!=", ">=", "<=", ">", "<", "%"}
	for _, op := range ops {
		if strings.Contains(expr, op) {
			parts := strings.SplitN(expr, op, 2)
			if len(parts) != 2 {
				continue
			}
			left := strings.TrimSpace(strings.Trim(parts[0], "'\""))
			right := strings.TrimSpace(strings.Trim(parts[1], "'\""))

			switch op {
			case "==":
				return left == right
			case "!=":
				return left != right
			case ">=":
				return compareNum(left, right) >= 0
			case "<=":
				return compareNum(left, right) <= 0
			case ">":
				return compareNum(left, right) > 0
			case "<":
				return compareNum(left, right) < 0
			case "%":
				return left != "" && right != ""
			}
		}
	}

	return expr != "" && expr != "0" && expr != "false"
}

func compareNum(a, b string) int {
	ai, errA := parseIntSafe(a)
	bi, errB := parseIntSafe(b)
	if errA == nil && errB == nil {
		switch {
		case ai > bi:
			return 1
		case ai < bi:
			return -1
		default:
			return 0
		}
	}
	af, errA := parseFloatSafe(a)
	bf, errB := parseFloatSafe(b)
	if errA == nil && errB == nil {
		switch {
		case af > bf:
			return 1
		case af < bf:
			return -1
		default:
			return 0
		}
	}
	return strings.Compare(a, b)
}
