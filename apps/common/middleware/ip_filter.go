package middleware

import (
	"net"
	"gbootcms/apps/admin/model"
	"strings"

	"github.com/gin-gonic/gin"
)

// IPFilter IP 黑白名單中間件
// 對齊 PHP 原版 HomeController.__construct() 的 IP 過濾邏輯
// 生效範圍：前台（排除 /admin /static /template，對齊 PHP 原版後台不受限）
func IPFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// 後台和靜態資源不受 IP 過濾限制（對齊 PHP 原版：後台控制器不繼承 HomeController）
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/static") || strings.HasPrefix(path, "/template") {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		// IPv6 本地迴環地址轉 IPv4（對齊 PHP get_user_ip() 將 ::1 轉為 127.0.0.1）
		if clientIP == "::1" {
			clientIP = "127.0.0.1"
		}
		ip := net.ParseIP(clientIP)
		if ip == nil {
			c.Next()
			return
		}

		// 黑名單檢查（優先級高於白名單）
		ipDeny := model.GetConfigValue("ip_deny", "")
		if ipDeny != "" {
			for _, rule := range strings.Split(ipDeny, ",") {
				rule = strings.TrimSpace(rule)
				if rule == "" {
					continue
				}
				if matchIP(ip, rule) {
					c.String(403, "本站啟用了黑名單功能，您的IP("+clientIP+")不允許訪問！")
					c.Abort()
					return
				}
			}
		}

		// 白名單檢查（配置了白名單時，不在白名單內的 IP 被阻擋）
		ipAllow := model.GetConfigValue("ip_allow", "")
		if ipAllow != "" {
			allowed := false
			for _, rule := range strings.Split(ipAllow, ",") {
				rule = strings.TrimSpace(rule)
				if rule == "" {
					continue
				}
				if matchIP(ip, rule) {
					allowed = true
					break
				}
			}
			if !allowed {
				c.String(403, "本站啟用了白名單功能，您的IP("+clientIP+")不在允許範圍！")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// matchIP 判斷 IP 是否匹配規則（支援單 IP 和 CIDR）
// 對齊 PHP 原版 network_match()：支援 192.168.1.100 和 192.168.1.0/24
func matchIP(ip net.IP, rule string) bool {
	// CIDR 格式：192.168.1.0/24
	if strings.Contains(rule, "/") {
		_, ipNet, err := net.ParseCIDR(rule)
		if err != nil {
			return false
		}
		return ipNet.Contains(ip)
	}
	// 單 IP 精確匹配
	ruleIP := net.ParseIP(rule)
	if ruleIP != nil {
		return ip.Equal(ruleIP)
	}
	return false
}
