package seed

import (
	"crypto/md5"
	"fmt"
	"time"

	"gbootcms/apps/admin/model"
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/member"
	"gbootcms/apps/admin/model/system"

	"gorm.io/gorm"
)

func Init() {
	SeedData()
}

func SeedData() {
	// 確保 ay_content_ext 基礎表存在（冪等操作，每次啟動都執行）
	content.EnsureContentExtTable()

	var user system.AdminUser
	if model.DB.Where("1 = 1").First(&user).Error != gorm.ErrRecordNotFound {
		// 用戶表非空：跳過首次種子，但仍要確保選單是最新版本
		ensureMenuVersion()
		ensureMemberConfigs()
		return
	}

	now := time.Now()

	seedAdminUser(now)
	seedSite()
	seedCompany()
	seedMenus(now)
	seedRoles(now)
	seedMemberGroups()
	seedContentModels()
	seedConfigs()
}

// ensureMenuVersion 檢查選單資料是否與最新 seedMenus 版本一致。
// 偵測標誌：是否存在 mcode='M1008'（移除類型管理後的版本標記）。若不存在則清空重建。
func ensureMenuVersion() {
	var count int64
	model.DB.Model(&system.Menu{}).Where("mcode = ?", "M1008").Count(&count)
	if count > 0 {
		return
	}
	// 舊版選單結構 → 清空重建
	model.DB.Exec("DELETE FROM ay_menu")
	seedMenus(time.Now())
	// 寫入版本標記（隱藏選單，不作導航用）
	model.DB.Create(&system.Menu{
		Mcode:      "M1008",
		Pcode:      "",
		Name:       "版本標記",
		URL:        "#",
		Sorting:    0,
		Status:     0,
		Type:       0,
		Shortcut:   0,
		UpdateUser: "system",
		UpdateTime: time.Now(),
	})
}

func seedAdminUser(now time.Time) {
	model.DB.Create(&system.AdminUser{
		Ucode:      "10001",
		Username:   "admin",
		Password:   fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%x", md5.Sum([]byte("admin")))))),
		Realname:   "管理員",
		Rcodes:     "R100",
		Acodes:     "",
		Status:     1,
		CreateTime: now,
		UpdateTime: now,
	})
}

func seedSite() {
	model.DB.Create(&content.Site{
		Name:        "Gbootcms",
		Title:       "Gbootcms",
		Subtitle:    "",
		Keywords:    "Gbootcms",
		Description: "Gbootcms",
		Logo:        "",
		ICP:         "",
		Copyright:   "",
		Statistical: "",
		Theme:       "default",
		Lang:        "zh",
	})
}

func seedCompany() {
	model.DB.Create(&content.Company{
		Name:     "Gbootcms",
		Address:  "",
		Phone:    "",
		Fax:      "",
		Email:    "",
		Weixin:   "",
		ICP:      "",
		Legal:    "",
		Business: "",
	})
}

func seedMenus(now time.Time) {
	// 與 PbootCMS-3.2.12 原版 ay_menu 表結構 1:1 對齊
	// 資料來源：PbootCMS-3.2.12/static/backup/sql/pbootcms_v3211.sql
	menus := []system.Menu{
		// ============ 一級選單 ============
		{Mcode: "M156", Pcode: "", Name: "全局配置", URL: "/admin/M156/index", Ico: "fa-globe", Sorting: 200, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M110", Pcode: "", Name: "基礎內容", URL: "/admin/M110/index", Ico: "fa-sliders", Sorting: 300, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M130", Pcode: "", Name: "文章內容", URL: "/admin/M130/index", Ico: "fa-file-text-o", Sorting: 400, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M157", Pcode: "", Name: "擴展內容", URL: "/admin/M157/index", Ico: "fa-arrows-alt", Sorting: 500, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1001", Pcode: "", Name: "會員中心", URL: "/admin/M1001/index", Ico: "fa-user-o", Sorting: 600, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M101", Pcode: "", Name: "系統管理", URL: "/admin/M101/index", Ico: "fa-cog", Sorting: 900, Status: 1, Shortcut: 0, Type: 1},

		// ============ 全局配置 子選單 (M156) ============
		{Mcode: "M153", Pcode: "M156", Name: "配置參數", URL: "/admin/Config/index", Ico: "fa-sliders", Sorting: 201, Status: 1, Shortcut: 1, Type: 1},
		{Mcode: "M155", Pcode: "M156", Name: "模型管理", URL: "/admin/Model/index", Ico: "fa-codepen", Sorting: 204, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M158", Pcode: "M156", Name: "模型欄位", URL: "/admin/ExtField/index", Ico: "fa-external-link", Sorting: 205, Status: 1, Shortcut: 0, Type: 1},

		// ============ 基礎內容 子選單 (M110) ============
		{Mcode: "M112", Pcode: "M110", Name: "站點資訊", URL: "/admin/Site/index", Ico: "fa-cog", Sorting: 301, Status: 1, Shortcut: 1, Type: 1},
		{Mcode: "M113", Pcode: "M110", Name: "公司資訊", URL: "/admin/Company/index", Ico: "fa-copyright", Sorting: 302, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M129", Pcode: "M110", Name: "內容欄目", URL: "/admin/ContentSort/index", Ico: "fa-bars", Sorting: 303, Status: 1, Shortcut: 1, Type: 1},

		// ============ 文章內容 子選單 (M130) ============
		{Mcode: "M131", Pcode: "M130", Name: "欄目", URL: "/admin/Single/index", Ico: "fa-file-o", Sorting: 401, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M132", Pcode: "M130", Name: "列表內容", URL: "/admin/Content/index", Ico: "fa-file-text-o", Sorting: 402, Status: 1, Shortcut: 1, Type: 1},

		// ============ 擴展內容 子選單 (M157) ============
		{Mcode: "M150", Pcode: "M157", Name: "留言資訊", URL: "/admin/Message/index", Ico: "fa-question-circle-o", Sorting: 501, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M151", Pcode: "M157", Name: "輪播圖片", URL: "/admin/Slide/index", Ico: "fa-picture-o", Sorting: 502, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M152", Pcode: "M157", Name: "友情連結", URL: "/admin/Link/index", Ico: "fa-link", Sorting: 503, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M160", Pcode: "M157", Name: "自訂表單", URL: "/admin/Form/index", Ico: "fa-plus-square-o", Sorting: 504, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1000", Pcode: "M157", Name: "文章內鏈", URL: "/admin/Tags/index", Ico: "fa-random", Sorting: 505, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1007", Pcode: "M157", Name: "媒體庫", URL: "/admin/Media/index", Ico: "fa-photo", Sorting: 506, Status: 1, Shortcut: 0, Type: 1},

		// ============ 會員中心 子選單 (M1001) ============
		{Mcode: "M1002", Pcode: "M1001", Name: "會員等級", URL: "/admin/MemberGroup/index", Ico: "fa-signal", Sorting: 601, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1003", Pcode: "M1001", Name: "會員欄位", URL: "/admin/MemberField/index", Ico: "fa-wpforms", Sorting: 602, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1004", Pcode: "M1001", Name: "會員管理", URL: "/admin/Member/index", Ico: "fa-users", Sorting: 603, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1005", Pcode: "M1001", Name: "文章評論", URL: "/admin/MemberComment/index", Ico: "fa-commenting-o", Sorting: 604, Status: 1, Shortcut: 0, Type: 1},

		// ============ 系統管理 子選單 (M101) ============
		{Mcode: "M102", Pcode: "M101", Name: "資料區域", URL: "/admin/Area/index", Ico: "fa-sitemap", Sorting: 901, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M103", Pcode: "M101", Name: "系統選單", URL: "/admin/Menu/index", Ico: "fa-bars", Sorting: 902, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M104", Pcode: "M101", Name: "系統角色", URL: "/admin/Role/index", Ico: "fa-hand-stop-o", Sorting: 903, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M105", Pcode: "M101", Name: "系統用戶", URL: "/admin/User/index", Ico: "fa-users", Sorting: 904, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M106", Pcode: "M101", Name: "系統日誌", URL: "/admin/Syslog/index", Ico: "fa-history", Sorting: 905, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M108", Pcode: "M101", Name: "資料庫管理", URL: "/admin/Database/index", Ico: "fa-database", Sorting: 907, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M109", Pcode: "M101", Name: "伺服器資訊", URL: "/admin/Site/server", Ico: "fa-info-circle", Sorting: 908, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1101", Pcode: "M101", Name: "媒體庫", URL: "/admin/content/media/index", Ico: "fa-photo", Sorting: 909, Status: 1, Shortcut: 0, Type: 1},
	}
	for _, m := range menus {
		m.CreateUser = "admin"
		m.UpdateUser = "admin"
		m.CreateTime = now
		m.UpdateTime = now
		model.DB.Create(&m)
	}
}

func seedRoles(now time.Time) {
	model.DB.Create(&system.Role{
		Code:        "R100",
		Rcode:       "R100",
		Name:        "超級管理員",
		Description: "",
		Levels:      "",
		Status:      1,
		CreateUser:  "admin",
		UpdateUser:  "admin",
		CreateTime:  now,
		UpdateTime:  now,
	})
}

func seedMemberGroups() {
	groups := []member.MemberGroup{
		{Gcode: "G1", Gname: "普通會員", Status: 1},
		{Gcode: "G2", Gname: "VIP會員", Status: 1},
	}
	for _, g := range groups {
		model.DB.Create(&g)
	}
}

func seedContentModels() {
	models := []content.Model{
		{Mcode: "3D1", Name: "文章模型", URLName: "list", Status: 1, Type: 2, Issystem: 0},
		{Mcode: "3D2", Name: "單頁模型", URLName: "about", Status: 1, Type: 1, Issystem: 0},
	}
	for _, m := range models {
		model.DB.Create(&m)
	}
}

func seedConfigs() {
	configs := []system.Config{
		{Name: "close_site", Value: "0"},
		{Name: "open_wap", Value: "0"},
		{Name: "admin_check_code", Value: "1"},
		{Name: "message_check_code", Value: "1"},
		{Name: "form_check_code", Value: "0"},
		{Name: "message_status", Value: "1"},
		{Name: "message_verify", Value: "1"},
		{Name: "message_rqlogin", Value: "0"},
		{Name: "url_rule_type", Value: "2"},
		{Name: "tpl_html_cache", Value: "0"},
		{Name: "tpl_html_cache_time", Value: "900"},
		{Name: "content_sort_count", Value: "100"},
		{Name: "page_size", Value: "15"},
		{Name: "page_status", Value: "1"},
		{Name: "login_error_count", Value: "5"},
		{Name: "login_error_wait", Value: "300"},
		{Name: "session_time", Value: "1800"},
		// 會員系統配置
		{Name: "register_status", Value: "1"},
		{Name: "register_type", Value: "1"},
		{Name: "register_check_code", Value: "1"},
		{Name: "register_verify", Value: "0"},
		{Name: "register_score", Value: "0"},
		{Name: "register_gcode", Value: ""},
		{Name: "register_title", Value: "會員註冊"},
		{Name: "login_status", Value: "1"},
		{Name: "login_check_code", Value: "1"},
		{Name: "login_title", Value: "會員登錄"},
		{Name: "ucenter_title", Value: "個人中心"},
		{Name: "umodify_title", Value: "資料修改"},
		{Name: "comment_status", Value: "1"},
		{Name: "comment_check_code", Value: "1"},
		{Name: "comment_verify", Value: "1"},
		{Name: "comment_anonymous", Value: "0"},
		{Name: "home_upload_ext", Value: "jpg,jpeg,png,gif,xls,xlsx,doc,docx,ppt,pptx,rar,zip,pdf,txt"},
		// 搜尋引擎推送配置
		{Name: "baidu_zz_token", Value: ""},
		{Name: "baidu_ks_token", Value: ""},
		{Name: "bing_indexnow_key", Value: ""},
		// 標題樣式配置
		{Name: "index_title", Value: ""},
		{Name: "list_title", Value: ""},
		{Name: "content_title", Value: ""},
		{Name: "about_title", Value: ""},
		{Name: "other_title", Value: ""},
		// URL 規則配置
		{Name: "url_rule_content_path", Value: "0"},
		{Name: "url_index_404", Value: "0"},
		// 圖片水印配置
		{Name: "watermark_open", Value: "0"},
		{Name: "watermark_text", Value: ""},
		{Name: "watermark_text_size", Value: "16"},
		{Name: "watermark_text_color", Value: "100,100,100"},
		{Name: "watermark_text_font", Value: ""},
		{Name: "watermark_pic", Value: ""},
		{Name: "watermark_position", Value: "4"},
	}
	for _, c := range configs {
		model.DB.Create(&c)
	}
}

// ensureMemberConfigs 確保會員配置項存在（用於已有資料庫的版本升級）
func ensureMemberConfigs() {
	memberConfigs := []system.Config{
		{Name: "register_status", Value: "1"},
		{Name: "register_type", Value: "1"},
		{Name: "register_check_code", Value: "1"},
		{Name: "register_verify", Value: "0"},
		{Name: "register_score", Value: "0"},
		{Name: "register_gcode", Value: ""},
		{Name: "register_title", Value: "會員註冊"},
		{Name: "login_status", Value: "1"},
		{Name: "login_check_code", Value: "1"},
		{Name: "login_title", Value: "會員登錄"},
		{Name: "ucenter_title", Value: "個人中心"},
		{Name: "umodify_title", Value: "資料修改"},
		{Name: "comment_status", Value: "1"},
		{Name: "comment_check_code", Value: "1"},
		{Name: "comment_verify", Value: "1"},
		{Name: "comment_anonymous", Value: "0"},
		{Name: "home_upload_ext", Value: "jpg,jpeg,png,gif,xls,xlsx,doc,docx,ppt,pptx,rar,zip,pdf,txt"},
	}
	for _, c := range memberConfigs {
		var count int64
		model.DB.Model(&system.Config{}).Where("name = ?", c.Name).Count(&count)
		if count == 0 {
			model.DB.Create(&c)
		}
	}
}
