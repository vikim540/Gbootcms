package seed

import (
	"crypto/md5"
	"fmt"
	"log/slog"
	"time"

	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/member"
	"gbootcms/apps/admin/model/system"
	"gbootcms/core/acodeplugin"
	"gbootcms/core/db"

	"context"
	"gorm.io/gorm"
)

func Init() {
	SeedData()
}

func SeedData() {
	// 確保 ay_content_ext 基礎表存在（冪等操作，每次啟動都執行）
	content.EnsureContentExtTable()

	var user system.AdminUser
	if db.DB.Where("1 = 1").First(&user).Error != gorm.ErrRecordNotFound {
		// 用戶表非空：跳過首次種子，但仍要確保選單是最新版本
		ensureArea()
		ensureMenuVersion()
		ensureMemberConfigs()
		return
	}

	now := time.Now()

	seedAdminUser(now)
	seedArea()
	seedSite()
	seedCompany()
	seedMenus(now)
	seedRoles(now)
	seedMemberGroups()
	seedContentModels()
	seedConfigs()
	seedDemoSortsAndArticles(now)
}

// ensureArea 確保多語言區域存在
func ensureArea() {
	var count int64
	db.DB.Model(&system.Area{}).Count(&count)
	if count > 0 {
		return
	}
	seedArea()
}

func seedArea() {
	areas := []system.Area{
		{Acode: "tc", Name: "繁體中文", IsDefault: "1"},
		{Acode: "sc", Name: "簡體中文", IsDefault: "0"},
		{Acode: "en", Name: "English", IsDefault: "0"},
	}
	for _, a := range areas {
		db.DB.Create(&a)
	}
}

// ensureMenuVersion 檢查選單資料是否與最新 seedMenus 版本一致
func ensureMenuVersion() {
	var count int64
	db.DB.Model(&system.Menu{}).Where("mcode = ?", "M1010").Count(&count)
	if count > 0 {
		return
	}
	if err := db.DB.Exec("DELETE FROM ay_menu").Error; err != nil {
		slog.Error("清空舊選單失敗", "error", err)
		return
	}
	seedMenus(time.Now())
	// 寫入版本標記（隱藏選單，不作導航用）
	db.DB.Create(&system.Menu{
		Mcode:      "M1010",
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
	db.DB.Create(&system.AdminUser{
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
	sites := []content.Site{
		{
			Acode:       "tc",
			Name:        "企業內容管理系統",
			Title:       "Gbootcms 官方網站",
			Subtitle:    "高併發、原生輕量的 Go 語言現代化內容管理系統",
			Keywords:    "Gbootcms,Go CMS,企業官網,內容管理系統",
			Description: "Gbootcms 是一款基於 Go 語言的高性能企業級開源內容管理系統，支援多語言、SQLite WAL 與自動圖片優化。",
			Theme:       "default",
		},
		{
			Acode:       "sc",
			Name:        "企业内容管理系统",
			Title:       "Gbootcms 官方网站",
			Subtitle:    "高并发、原生轻量的 Go 语言现代化内容管理系统",
			Keywords:    "Gbootcms,Go CMS,企业官网,内容管理系统",
			Description: "Gbootcms 是一款基于 Go 语言的高性能企业级开源内容管理系统，支持多语言、SQLite WAL 与自动图片优化。",
			Theme:       "default",
		},
		{
			Acode:       "en",
			Name:        "Enterprise CMS",
			Title:       "Gbootcms Official Website",
			Subtitle:    "High-Performance, Lightweight Modern CMS in Go",
			Keywords:    "Gbootcms,Go CMS,Enterprise Website,Content Management System",
			Description: "Gbootcms is a high-performance modern CMS built in pure Go, featuring native multilingual support and SQLite WAL concurrency.",
			Theme:       "default",
		},
	}
	skipCtx := acodeplugin.SkipAcode(context.Background())
	for _, s := range sites {
		db.DB.WithContext(skipCtx).Create(&s)
	}
}

func seedCompany() {
	companies := []content.Company{
		{
			Acode:    "tc",
			Name:     "Gbootcms 網絡科技",
			Address:  "香港九龍旺角商務中心",
			Phone:    "+852 2345 6789",
			Email:    "service@gbootcms.org",
			Business: "企業數位化解決方案與現代化網站建設",
		},
		{
			Acode:    "sc",
			Name:     "Gbootcms 网络科技",
			Address:  "上海市浦东新区张江高科技园区",
			Phone:    "+86 021 5678 1234",
			Email:    "service@gbootcms.org",
			Business: "企业数字化解决方案与现代化网站建设",
		},
		{
			Acode:    "en",
			Name:     "Gbootcms Technology Co., Ltd.",
			Address:  "Business Center, Nathan Road, Kowloon, Hong Kong",
			Phone:    "+852 2345 6789",
			Email:    "service@gbootcms.org",
			Business: "Enterprise Digital Solutions and Modern Web Construction",
		},
	}
	skipCtx := acodeplugin.SkipAcode(context.Background())
	for _, c := range companies {
		db.DB.WithContext(skipCtx).Create(&c)
	}
}

func seedMenus(now time.Time) {
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
		{Mcode: "M133", Pcode: "M130", Name: "文檔回收站", URL: "/admin/content/trash", Ico: "fa-recycle", Sorting: 403, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M134", Pcode: "M130", Name: "301重定向", URL: "/admin/content/redirect/index", Ico: "fa-reply", Sorting: 404, Status: 1, Shortcut: 0, Type: 1},

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
		db.DB.Create(&m)
	}
}

func seedRoles(now time.Time) {
	db.DB.Create(&system.Role{
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
		db.DB.Create(&g)
	}
}

func seedContentModels() {
	models := []content.Model{
		{Mcode: "3D1", Name: "文章模型", URLName: "list", Status: 1, Type: 2, Issystem: 0},
		{Mcode: "3D2", Name: "單頁模型", URLName: "about", Status: 1, Type: 1, Issystem: 0},
	}
	for _, m := range models {
		db.DB.Create(&m)
	}
}

func seedConfigs() {
	configs := []system.Config{
		{Name: "close_site", Value: "0"},
		{Name: "open_wap", Value: "0"},
		{Name: "admin_check_code", Value: "0"},
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
		// 圖片壓縮配置
		{Name: "image_compress_enable", Value: "1"},
		{Name: "image_quality", Value: "80"},
		{Name: "image_max_width", Value: "1920"},
		{Name: "image_warn_size", Value: "1024"},
		// MeiliSearch 全文搜索配置
		{Name: "meilisearch_url", Value: ""},
		{Name: "meilisearch_key", Value: ""},
		// Cloudflare R2 雲存儲配置
		{Name: "r2_enabled", Value: "0"},
		{Name: "r2_endpoint", Value: ""},
		{Name: "r2_access_key", Value: ""},
		{Name: "r2_secret_key", Value: ""},
		{Name: "r2_bucket", Value: ""},
		{Name: "r2_public_url", Value: ""},
		{Name: "r2_cache_ttl", Value: "300"},
		// 外鏈 nofollow 配置
		{Name: "nofollow_external", Value: "1"},
		// 敏感詞過濾配置
		{Name: "sensitive_words", Value: ""},
		// JSON-LD 結構化資料配置
		{Name: "opening_hours", Value: ""},
		{Name: "geo_latitude", Value: ""},
		{Name: "geo_longitude", Value: ""},
		{Name: "price_currency", Value: "CNY"},
	}
	for _, c := range configs {
		db.DB.Create(&c)
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
		db.DB.Model(&system.Config{}).Where("name = ?", c.Name).Count(&count)
		if count == 0 {
			db.DB.Create(&c)
		}
	}
}

// seedDemoSortsAndArticles 初始化現代企業演示欄目與文章（自帶精確三語 rel_id 關聯）
func seedDemoSortsAndArticles(now time.Time) {
	skipCtx := acodeplugin.SkipAcode(context.Background())

	// 欄目定義（三語：tc, sc, en）
	sorts := []content.ContentSort{
		// 繁體
		{Acode: "tc", Scode: "1", Pcode: "0", Name: "關於我們", Mcode: "3D2", ListTpl: "about.html", ContentTpl: "about.html", Status: 1, Filename: "about"},
		{Acode: "tc", Scode: "2", Pcode: "0", Name: "新聞資訊", Mcode: "3D1", ListTpl: "newslist.html", ContentTpl: "news.html", Status: 1, Filename: "news"},
		{Acode: "tc", Scode: "3", Pcode: "0", Name: "產品中心", Mcode: "3D1", ListTpl: "productlist.html", ContentTpl: "product.html", Status: 1, Filename: "product"},
		{Acode: "tc", Scode: "4", Pcode: "0", Name: "聯絡我們", Mcode: "3D2", ListTpl: "about.html", ContentTpl: "about.html", Status: 1, Filename: "contact"},

		// 簡體
		{Acode: "sc", Scode: "1", Pcode: "0", Name: "关于我们", Mcode: "3D2", ListTpl: "about.html", ContentTpl: "about.html", Status: 1, Filename: "about"},
		{Acode: "sc", Scode: "2", Pcode: "0", Name: "新闻资讯", Mcode: "3D1", ListTpl: "newslist.html", ContentTpl: "news.html", Status: 1, Filename: "news"},
		{Acode: "sc", Scode: "3", Pcode: "0", Name: "产品中心", Mcode: "3D1", ListTpl: "productlist.html", ContentTpl: "product.html", Status: 1, Filename: "product"},
		{Acode: "sc", Scode: "4", Pcode: "0", Name: "联系我们", Mcode: "3D2", ListTpl: "about.html", ContentTpl: "about.html", Status: 1, Filename: "contact"},

		// 英文
		{Acode: "en", Scode: "1", Pcode: "0", Name: "About Us", Mcode: "3D2", ListTpl: "about.html", ContentTpl: "about.html", Status: 1, Filename: "about"},
		{Acode: "en", Scode: "2", Pcode: "0", Name: "News", Mcode: "3D1", ListTpl: "newslist.html", ContentTpl: "news.html", Status: 1, Filename: "news"},
		{Acode: "en", Scode: "3", Pcode: "0", Name: "Products", Mcode: "3D1", ListTpl: "productlist.html", ContentTpl: "product.html", Status: 1, Filename: "product"},
		{Acode: "en", Scode: "4", Pcode: "0", Name: "Contact Us", Mcode: "3D2", ListTpl: "about.html", ContentTpl: "about.html", Status: 1, Filename: "contact"},
	}

	for _, s := range sorts {
		db.DB.WithContext(skipCtx).Create(&s)
	}

	// 演示文章（三語互相對齊，RelID 保持精確綁定）
	articles := []content.Content{
		// 繁體主文章 (RelID: 1)
		{
			Acode:      "tc",
			Scode:      "2",
			Title:      "企業數位化轉型全面解決方案",
			Content:    "<p>隨著現代互聯網技術的高速發展，企業數位化轉型已成為提升運營效率與市場競爭力的關鍵路徑。GbootCMS 提供高併發、極致輕量與全鏈路 SEO 解決方案。</p>",
			Status:     1,
			RelID:      1,
			Date:       now,
			CreateTime: now,
			UpdateTime: now,
		},
		// 簡體對應文章 (RelID: 1)
		{
			Acode:      "sc",
			Scode:      "2",
			Title:      "企业数字化转型全面解决方案",
			Content:    "<p>随着现代互联网技术的高速发展，企业数字化转型已成为提升运营效率与市场竞争力的关键路径。GbootCMS 提供高并发、极致轻量与全链路 SEO 解决方案。</p>",
			Status:     1,
			RelID:      1,
			Date:       now,
			CreateTime: now,
			UpdateTime: now,
		},
		// 英文對應文章 (RelID: 1)
		{
			Acode:      "en",
			Scode:      "2",
			Title:      "Comprehensive Solutions for Enterprise Digital Transformation",
			Content:    "<p>With the rapid development of modern Internet technology, enterprise digital transformation has become a critical path to improve operational efficiency and market competitiveness. GbootCMS provides high concurrency, lightweight architecture, and full-funnel SEO solutions.</p>",
			Status:     1,
			RelID:      1,
			Date:       now,
			CreateTime: now,
			UpdateTime: now,
		},
	}

	for _, a := range articles {
		db.DB.WithContext(skipCtx).Create(&a)
	}
}
