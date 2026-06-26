package system

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"

	"github.com/gin-gonic/gin"
)

// ConfigController - System Configuration Controller
// Corresponds to PHP: apps/admin/controller/ConfigController.php
type ConfigController struct {
	common.BaseController
}

// Index - Configuration page
func (cf *ConfigController) Index(c *gin.Context) {
	if c.Request.Method == "POST" {
		submit := c.PostForm("submit")
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
			"baidu_zz_token", "baidu_ks_token",
			"api_open", "api_auth", "api_appid", "api_secret",
			"watermark_open", "watermark_text", "watermark_text_font",
			"watermark_text_size", "watermark_text_color", "watermark_pic",
			"watermark_position",
		}
		for _, name := range names {
			val := c.PostForm(name)
			if submit != "" && name == "submit" {
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
