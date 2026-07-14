package system

import (
	"fmt"
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"gbootcms/apps/common/mail"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// isValidEmail 驗證郵箱格式
func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// ConfigController - System Configuration Controller
// Corresponds to PHP: apps/admin/controller/ConfigController.php
type ConfigController struct {
	common.BaseController
}

// Index - Configuration page
func (cf *ConfigController) Index(c *gin.Context) {
	// 測試郵件發送（AJAX，返回 JSON）
	if action := c.Query("action"); action == "sendemail" {
		to := c.Query("to")
		if to == "" {
			to = model.GetConfigValue("smtp_username_test", "")
		}
		if to == "" {
			cf.JSONFail(c, "請先填寫測試收件郵箱")
			return
		}
		// 郵箱格式驗證（比 PHP 更嚴謹）
		if !isValidEmail(to) {
			cf.JSONFail(c, "郵箱格式不正確："+to)
			return
		}
		// 檢查 SMTP 配置是否完整
		if model.GetConfigValue("smtp_server", "") == "" || model.GetConfigValue("smtp_username", "") == "" {
			cf.JSONFail(c, "SMTP 配置不完整，請先填寫伺服器地址和發件帳號")
			return
		}
		if err := mail.SendTestMail(to); err != nil {
			// 原始錯誤打到終端日誌，前端只顯示友好提示
			fmt.Printf("[SMTP] 測試郵件發送失敗 to=%s: %v\n", to, err)
			cf.JSONFail(c, mail.FriendlyErr(err))
			return
		}
		cf.JSONOKMsg(c, "測試郵件已發送至 "+to+"，請查收")
		return
	}

	if c.Request.Method == "POST" {
		names := []string{
			"cmsname",
			"close_site", "close_site_note", "open_wap", "wap_domain",
			"tpl_html_cache", "tpl_html_cache_time", "gzip",
			"spiderlog", "to_https", "to_main_domain", "main_domain",
			"pagenum", "content_tags_replace_num", "content_keyword_replace",
			"sn", "sn_user", "message_status", "message_check_code",
			"message_verify", "message_rqlogin", "form_status", "form_check_code",
			"tpl_html_dir", "ip_deny", "ip_allow", "admin_check_code",
			"lock_count", "lock_time", "url_rule_type", "url_rule_content_path",
			"url_index_404", "index_title", "about_title", "list_title",
			"content_title", "search_title", "register_title", "login_title",
			"ucenter_title", "umodify_title", "other_title",
			"register_status", "register_type", "register_check_code",
			"register_verify", "login_status", "login_check_code",
			"login_no_wait", "comment_status", "comment_anonymous",
			"comment_check_code", "comment_verify", "register_score",
			"login_score", "register_gcode", "home_upload_ext",
			"smtp_server", "smtp_port", "smtp_ssl", "smtp_username",
			"smtp_password", "smtp_username_test", "message_send_mail",
			"form_send_mail", "comment_send_mail", "message_send_to",
			"webhook_url", "webhook_message", "webhook_form", "webhook_comment",
			"baidu_zz_token", "baidu_ks_token", "bing_indexnow_key",
			"api_open", "api_auth", "api_appid", "api_secret",
			"watermark_open", "watermark_text", "watermark_text_font",
			"watermark_text_size", "watermark_text_color", "watermark_pic",
			"watermark_position",
			"image_compress_enable", "image_quality", "image_max_width",
			"image_warn_size",
			"turnstile_sitekey", "turnstile_secret",
			"message_turnstile", "form_turnstile",
			"likes_status",
		}
		for _, name := range names {
			if name == "submit" {
				continue
			}
			// 只更新表單中實際提交的欄位，避免其他 tab 的配置被空值覆蓋
			val, exists := c.GetPostForm(name)
			if !exists {
				continue
			}
			// IP 及敏感詞配置歸一化：換行和中文逗號轉英文逗號（對齊 PHP ConfigController.php:207-215）
			if name == "ip_deny" || name == "ip_allow" || name == "content_keyword_replace" {
				val = strings.ReplaceAll(val, "\r\n", ",")
				val = strings.ReplaceAll(val, "\n", ",")
				val = strings.ReplaceAll(val, "，", ",")
			}
			// 危險擴展名過濾（對齊 PHP ConfigController.php:199-204）
			// 改用精確比對取代正則子串匹配，避免 sh 誤匹配 flash 等合法副檔名
			if name == "home_upload_ext" {
				dangerExts := map[string]bool{
					"php": true, "php3": true, "php4": true, "php5": true, "php7": true,
					"phtml": true, "pht": true, "phar": true, "shtml": true,
					"jsp": true, "jspx": true, "asp": true, "aspx": true, "asa": true, "cer": true,
					"exe": true, "com": true, "bat": true, "cmd": true, "scr": true, "msi": true,
					"sh": true, "bash": true, "csh": true, "zsh": true,
					"vb": true, "vbs": true, "vbe": true,
					"cgi": true, "pl": true, "py": true, "rb": true,
					"htaccess": true, "htc": true,
					"svg": true, "html": true, "htm": true, "js": true, "mjs": true,
					"xml": true, "xsl": true, "xslt": true,
					"ws": true, "wsf": true, "wsh": true, "pif": true,
				}
				for _, ext := range strings.Split(val, ",") {
					ext = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ext), ".")))
					if dangerExts[ext] {
						cf.JSONFail(c, "上傳副檔名包含危險類型（"+ext+"），已拒絕")
						return
					}
				}
			}
			var config model.Config
			result := model.DB.Where("name = ?", name).First(&config)
			if result.Error != nil {
				model.DB.Create(&model.Config{Name: name, Value: val})
			} else {
				model.DB.Model(&config).Update("value", val)
			}
		}
		cf.JSONOKMsg(c, common.NoticeSave)
		// 記錄操作日誌（對齊 PHP $this->log('修改參數配置成功！')）
		cf.LogAction(c, "修改參數配置")
		return
	}

	var configs []model.Config
	model.DB.Find(&configs)
	configMap := make(gin.H)
	for _, cfg := range configs {
		configMap[cfg.Name] = cfg.Value
	}
	// 會員等級數據（對齊 PHP: model('admin.member.MemberGroup')->getSelect()）
	groups := helper.BuildGroupsData()
	common.Render(c, "system/config.html", gin.H{
		"configs": configMap,
		"groups":  groups,
	})
}
