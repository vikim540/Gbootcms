package system

import (
	"fmt"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"pbootcms-go/apps/common/mail"
	"regexp"

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
			cf.JSONFail(c, common.NoticeMailTestEmpty)
			return
		}
		// 郵箱格式驗證（比 PHP 更嚴謹）
		if !isValidEmail(to) {
			cf.JSONFail(c, common.NoticeMailTestBadFormat+"："+to)
			return
		}
		// 檢查 SMTP 配置是否完整
		if model.GetConfigValue("smtp_server", "") == "" || model.GetConfigValue("smtp_username", "") == "" {
			cf.JSONFail(c, common.NoticeMailTestNoSMTP)
			return
		}
		if err := mail.SendTestMail(to); err != nil {
			// 原始錯誤打到終端日誌，前端只顯示友好提示
			fmt.Printf("[SMTP] 測試郵件發送失敗 to=%s: %v\n", to, err)
			cf.JSONFail(c, mail.FriendlyErr(err))
			return
		}
		cf.JSONOKMsg(c, common.NoticeMailTestSent(to))
		return
	}

	if c.Request.Method == "POST" {
		names := []string{
			"close_site", "close_site_note", "open_wap", "wap_domain",
			"tpl_html_cache", "tpl_html_cache_time", "gzip", "session_in_sitepath",
			"lgautosw", "spiderlog", "to_https", "to_main_domain", "main_domain",
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
			"baidu_zz_token", "baidu_ks_token",
			"api_open", "api_auth", "api_appid", "api_secret",
			"watermark_open", "watermark_text", "watermark_text_font",
			"watermark_text_size", "watermark_text_color", "watermark_pic",
			"watermark_position",
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
			var config model.Config
			result := model.DB.Where("name = ?", name).First(&config)
			if result.Error != nil {
				model.DB.Create(&model.Config{Name: name, Value: val})
			} else {
				model.DB.Model(&config).Update("value", val)
			}
		}
		cf.JSONOKMsg(c, common.NoticeSave)
		return
	}

	var configs []model.Config
	model.DB.Find(&configs)
	configMap := make(gin.H)
	for _, cfg := range configs {
		configMap[cfg.Name] = cfg.Value
	}
	common.Render(c, "system/config.html", gin.H{"configs": configMap})
}
