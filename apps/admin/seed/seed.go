package seed

import (
	"crypto/md5"
	"fmt"
	"time"

	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/admin/model/member"
	"pbootcms-go/apps/admin/model/system"

	"gorm.io/gorm"
)

func Init() {
	SeedData()
}

func SeedData() {
	var user system.AdminUser
	if model.DB.Where("1 = 1").First(&user).Error != gorm.ErrRecordNotFound {
		// 用户表非空：跳过首次种子，但仍要确保菜单是最新版本
		ensureMenuVersion()
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

// ensureMenuVersion 检查菜单数据是否与最新 seedMenus 版本一致。
// 检测标志：是否存在 mcode='M156'（全局配置）。若不存在则清空 ay_menu 重建。
func ensureMenuVersion() {
	var count int64
	model.DB.Model(&system.Menu{}).Where("mcode = ?", "M1006").Count(&count)
	if count > 0 {
		// 检查是否存在已废弃的M100菜单项，如果存在则删除
		var m100Count int64
		model.DB.Model(&system.Menu{}).Where("mcode = ?", "M100").Count(&m100Count)
		if m100Count > 0 {
			model.DB.Where("mcode = ?", "M100").Delete(&system.Menu{})
		}
		return
	}
	// 旧版菜单结构 → 清空重建为 PbootCMS 原版对齐版本
	model.DB.Exec("DELETE FROM ay_menu")
	seedMenus(time.Now())
}

func seedAdminUser(now time.Time) {
	model.DB.Create(&system.AdminUser{
		Ucode:      "10001",
		Username:   "admin",
		Password:   fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%x", md5.Sum([]byte("admin")))))),
		Realname:   "管理员",
		Rcodes:     "R100",
		Acodes:     "",
		Status:     1,
		CreateTime: now,
		UpdateTime: now,
	})
}

func seedSite() {
	model.DB.Create(&content.Site{
		Name:        "PbootCMS",
		Title:       "PbootCMS",
		Subtitle:    "",
		Keywords:    "PbootCMS",
		Description: "PbootCMS",
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
		Name:     "PbootCMS",
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
	// 与 PbootCMS-3.2.12 原版 ay_menu 表结构 1:1 对齐
	// 数据来源：PbootCMS-3.2.12/static/backup/sql/pbootcms_v3211.sql
	menus := []system.Menu{
		// ============ 一级菜单 ============
		{Mcode: "M156", Pcode: "", Name: "全局配置", URL: "/admin/M156/index", Ico: "fa-globe", Sorting: 200, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M110", Pcode: "", Name: "基础内容", URL: "/admin/M110/index", Ico: "fa-sliders", Sorting: 300, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M130", Pcode: "", Name: "文章内容", URL: "/admin/M130/index", Ico: "fa-file-text-o", Sorting: 400, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M157", Pcode: "", Name: "扩展内容", URL: "/admin/M157/index", Ico: "fa-arrows-alt", Sorting: 500, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1001", Pcode: "", Name: "会员中心", URL: "/admin/M1001/index", Ico: "fa-user-o", Sorting: 600, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M101", Pcode: "", Name: "系统管理", URL: "/admin/M101/index", Ico: "fa-cog", Sorting: 900, Status: 1, Shortcut: 0, Type: 1},

		// ============ 全局配置 子菜单 (M156) ============
		{Mcode: "M153", Pcode: "M156", Name: "配置参数", URL: "/admin/Config/index", Ico: "fa-sliders", Sorting: 201, Status: 1, Shortcut: 1, Type: 1},
		{Mcode: "M136", Pcode: "M156", Name: "定制标签", URL: "/admin/Label/index", Ico: "fa-wrench", Sorting: 203, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M155", Pcode: "M156", Name: "模型管理", URL: "/admin/Model/index", Ico: "fa-codepen", Sorting: 204, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M158", Pcode: "M156", Name: "模型字段", URL: "/admin/ExtField/index", Ico: "fa-external-link", Sorting: 205, Status: 1, Shortcut: 0, Type: 1},

		// ============ 基础内容 子菜单 (M110) ============
		{Mcode: "M112", Pcode: "M110", Name: "站点信息", URL: "/admin/Site/index", Ico: "fa-cog", Sorting: 301, Status: 1, Shortcut: 1, Type: 1},
		{Mcode: "M113", Pcode: "M110", Name: "公司信息", URL: "/admin/Company/index", Ico: "fa-copyright", Sorting: 302, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M129", Pcode: "M110", Name: "内容栏目", URL: "/admin/ContentSort/index", Ico: "fa-bars", Sorting: 303, Status: 1, Shortcut: 1, Type: 1},

		// ============ 文章内容 子菜单 (M130) ============
		{Mcode: "M131", Pcode: "M130", Name: "栏目", URL: "/admin/Single/index", Ico: "fa-file-o", Sorting: 401, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M132", Pcode: "M130", Name: "列表内容", URL: "/admin/Content/index", Ico: "fa-file-text-o", Sorting: 402, Status: 1, Shortcut: 1, Type: 1},

		// ============ 扩展内容 子菜单 (M157) ============
		{Mcode: "M150", Pcode: "M157", Name: "留言信息", URL: "/admin/Message/index", Ico: "fa-question-circle-o", Sorting: 501, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M151", Pcode: "M157", Name: "轮播图片", URL: "/admin/Slide/index", Ico: "fa-picture-o", Sorting: 502, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M152", Pcode: "M157", Name: "友情链接", URL: "/admin/Link/index", Ico: "fa-link", Sorting: 503, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M160", Pcode: "M157", Name: "自定义表单", URL: "/admin/Form/index", Ico: "fa-plus-square-o", Sorting: 504, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1000", Pcode: "M157", Name: "文章内链", URL: "/admin/Tags/index", Ico: "fa-random", Sorting: 505, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1006", Pcode: "M157", Name: "媒体库", URL: "/admin/Media/index", Ico: "fa-photo", Sorting: 506, Status: 1, Shortcut: 0, Type: 1},

		// ============ 会员中心 子菜单 (M1001) ============
		{Mcode: "M1002", Pcode: "M1001", Name: "会员等级", URL: "/admin/MemberGroup/index", Ico: "fa-signal", Sorting: 601, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1003", Pcode: "M1001", Name: "会员字段", URL: "/admin/MemberField/index", Ico: "fa-wpforms", Sorting: 602, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1004", Pcode: "M1001", Name: "会员管理", URL: "/admin/Member/index", Ico: "fa-users", Sorting: 603, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1005", Pcode: "M1001", Name: "文章评论", URL: "/admin/MemberComment/index", Ico: "fa-commenting-o", Sorting: 604, Status: 1, Shortcut: 0, Type: 1},

		// ============ 系统管理 子菜单 (M101) ============
		{Mcode: "M102", Pcode: "M101", Name: "数据区域", URL: "/admin/Area/index", Ico: "fa-sitemap", Sorting: 901, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M103", Pcode: "M101", Name: "系统菜单", URL: "/admin/Menu/index", Ico: "fa-bars", Sorting: 902, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M104", Pcode: "M101", Name: "系统角色", URL: "/admin/Role/index", Ico: "fa-hand-stop-o", Sorting: 903, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M105", Pcode: "M101", Name: "系统用户", URL: "/admin/User/index", Ico: "fa-users", Sorting: 904, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M106", Pcode: "M101", Name: "系统日志", URL: "/admin/Syslog/index", Ico: "fa-history", Sorting: 905, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M107", Pcode: "M101", Name: "类型管理", URL: "/admin/Type/index", Ico: "fa-tags", Sorting: 906, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M108", Pcode: "M101", Name: "数据库管理", URL: "/admin/Database/index", Ico: "fa-database", Sorting: 907, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M109", Pcode: "M101", Name: "服务器信息", URL: "/admin/Site/server", Ico: "fa-info-circle", Sorting: 908, Status: 1, Shortcut: 0, Type: 1},
		{Mcode: "M1101", Pcode: "M101", Name: "图片清理", URL: "/admin/ImageExt/index", Ico: "fa-trash", Sorting: 909, Status: 1, Shortcut: 0, Type: 1},
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
		Name:        "超级管理员",
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
		{Code: "G1", Name: "普通会员", Status: 1},
		{Code: "G2", Name: "VIP会员", Status: 1},
	}
	for _, g := range groups {
		model.DB.Create(&g)
	}
}

func seedContentModels() {
	models := []content.Model{
		{Mcode: "3D1", Name: "文章模型", URLName: "list", Status: 1, Type: 2, Issystem: 0},
		{Mcode: "3D2", Name: "单页模型", URLName: "about", Status: 1, Type: 1, Issystem: 0},
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
		{Name: "url_rule_type", Value: "2"},
		{Name: "tpl_html_cache", Value: "0"},
		{Name: "tpl_html_cache_time", Value: "900"},
		{Name: "content_sort_count", Value: "100"},
		{Name: "page_size", Value: "15"},
		{Name: "page_status", Value: "1"},
		{Name: "login_error_count", Value: "5"},
		{Name: "login_error_wait", Value: "300"},
		{Name: "session_time", Value: "1800"},
	}
	for _, c := range configs {
		model.DB.Create(&c)
	}
}
