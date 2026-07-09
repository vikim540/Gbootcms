package common

import (
	"strconv"
	"strings"
)

// ParseUserAgent 從 User-Agent 解析操作系統和瀏覽器（完整對齊 PbootCMS handle.php get_user_os + get_user_bs）
// 前後台共用，消除重複造輪子
// chPlatformVer 為 Sec-CH-UA-Platform-Version 標頭值，用於區分 Win10/Win11
func ParseUserAgent(ua string, chPlatformVer string) (osName, bsName string) {
	uaLower := strings.ToLower(ua)

	// === 操作系統（對齊 PbootCMS get_user_os 完整列表）===
	switch {
	case strings.Contains(uaLower, "windows nt 5.0"):
		osName = "Windows 2000"
	case strings.Contains(uaLower, "windows nt 9"):
		osName = "Windows 9X"
	case strings.Contains(uaLower, "windows nt 5.1"):
		osName = "Windows XP"
	case strings.Contains(uaLower, "windows nt 5.2"):
		osName = "Windows 2003"
	case strings.Contains(uaLower, "windows nt 6.0"):
		osName = "Windows Vista"
	case strings.Contains(uaLower, "windows nt 6.1"):
		osName = "Windows 7"
	case strings.Contains(uaLower, "windows nt 6.2"):
		osName = "Windows 8"
	case strings.Contains(uaLower, "windows nt 6.3"):
		osName = "Windows 8.1"
	case strings.Contains(uaLower, "windows nt 10"):
		// Win11 的 UA 仍是 "Windows NT 10.0"，用 Client Hints 區分
		// Win11 的 Sec-CH-UA-Platform-Version >= 13.0.0
		if chPlatformVer != "" {
			parts := strings.Split(chPlatformVer, ".")
			if len(parts) > 0 {
				if major, err := strconv.Atoi(parts[0]); err == nil && major >= 13 {
					osName = "Windows 11"
				} else {
					osName = "Windows 10"
				}
			} else {
				osName = "Windows 10"
			}
		} else {
			osName = "Windows 10"
		}
	case strings.Contains(uaLower, "windows phone"):
		osName = "Windows Phone"
	case strings.Contains(uaLower, "android"):
		osName = "Android"
	case strings.Contains(uaLower, "iphone"):
		osName = "iPhone"
	case strings.Contains(uaLower, "ipad"):
		osName = "iPad"
	case strings.Contains(uaLower, "mac"):
		osName = "Mac"
	case strings.Contains(uaLower, "sunos"):
		osName = "Sun OS"
	case strings.Contains(uaLower, "bsd"):
		osName = "BSD"
	case strings.Contains(uaLower, "ubuntu"):
		osName = "Ubuntu"
	case strings.Contains(uaLower, "linux"):
		osName = "Linux"
	case strings.Contains(uaLower, "unix"):
		osName = "Unix"
	default:
		osName = "Other"
	}

	// === 瀏覽器（對齊 PbootCMS get_user_bs 完整列表）===
	switch {
	case strings.Contains(uaLower, "micromessenger"):
		bsName = "Weixin"
	case strings.Contains(uaLower, "qq"):
		bsName = "QQ"
	case strings.Contains(uaLower, "weibo"):
		bsName = "Weibo"
	case strings.Contains(uaLower, "alipayclient"):
		bsName = "Alipay"
	case strings.Contains(uaLower, "trident/7.0"):
		bsName = "IE11"
	case strings.Contains(uaLower, "trident/6.0"):
		bsName = "IE10"
	case strings.Contains(uaLower, "trident/5.0"):
		bsName = "IE9"
	case strings.Contains(uaLower, "trident/4.0"):
		bsName = "IE8"
	case strings.Contains(uaLower, "msie 7.0"):
		bsName = "IE7"
	case strings.Contains(uaLower, "msie 6.0"):
		bsName = "IE6"
	case strings.Contains(uaLower, "edg"):
		bsName = "Edge"
	case strings.Contains(uaLower, "edge"):
		bsName = "Edge"
	case strings.Contains(uaLower, "firefox"):
		bsName = "Firefox"
	case strings.Contains(uaLower, "chrome"), strings.Contains(uaLower, "android"):
		bsName = "Chrome"
	case strings.Contains(uaLower, "safari"):
		bsName = "Safari"
	case strings.Contains(uaLower, "mj12bot"):
		bsName = "MJ12bot"
	default:
		bsName = "Other"
	}
	return
}
