package route

import (
	admin "gbootcms/apps/admin/controller"
	content "gbootcms/apps/admin/controller/content"
	member "gbootcms/apps/admin/controller/member"
	system "gbootcms/apps/admin/controller/system"
	"gbootcms/apps/api"
	"gbootcms/apps/common/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAdminRoutes(r *gin.Engine) {

	adminGroup := r.Group("/admin")
	adminGroup.Use(middleware.AdminAuth())
	{
		ic := &admin.IndexController{}
		adminGroup.GET("/", ic.Index)
		adminGroup.GET("/index/index", ic.Index)
		adminGroup.POST("/index/login", ic.Login)
		adminGroup.GET("/index/home", ic.Home)
		adminGroup.GET("/index/loginOut", ic.LoginOut)
		adminGroup.GET("/index/loginout", ic.LoginOut)
		adminGroup.GET("/index/ucenter", ic.Ucenter)
		adminGroup.POST("/index/ucenter", ic.UcenterMod)
		adminGroup.GET("/index/clearcache", ic.ClearCache)
		adminGroup.GET("/index/clearonlysyscache", ic.ClearOnlySysCache)
		adminGroup.GET("/index/clearsession", ic.ClearSession)
		adminGroup.POST("/index/area", ic.Area)
		adminGroup.GET("/index/checkCode", ic.CheckCode)
		adminGroup.GET("/index/upload", ic.Upload)
		adminGroup.POST("/index/upload", ic.Upload)
		adminGroup.POST("/index/upload/watermark/:flag", ic.Upload)

		cc := &content.ContentController{}
		adminGroup.GET("/content/index", cc.Index)
		adminGroup.GET("/content/index/*action", cc.IndexCatchAll)
		adminGroup.GET("/content/add", cc.Add)
		adminGroup.GET("/content/add/*action", cc.AddCatchAll)
		adminGroup.POST("/content/add", cc.Add)
		adminGroup.POST("/content/add/*action", cc.AddCatchAll)
		adminGroup.POST("/content/mod", cc.Mod)
		adminGroup.Any("/content/mod/*action", cc.Mod)
		adminGroup.POST("/content/del", cc.Del)
	adminGroup.Any("/content/del/*action", cc.DelCatchAll)
	adminGroup.GET("/content/trash", cc.Trash)
	adminGroup.POST("/content/restore", cc.Restore)
	adminGroup.POST("/content/permanentDel", cc.PermanentDel)

		csc := &content.ContentSortController{}
		adminGroup.GET("/content/sort/index", csc.Index)
		adminGroup.GET("/content/sort/add", csc.Add)
		adminGroup.POST("/content/sort/add", csc.Add)
		adminGroup.Any("/content/sort/mod/*action", csc.Mod)
		adminGroup.Any("/content/sort/del", csc.Del)

		sg := &content.SingleController{}
		adminGroup.GET("/content/single/index", sg.Index)
		adminGroup.GET("/content/single/index/*action", sg.IndexCatchAll)
		adminGroup.Any("/content/single/mod/*action", sg.Mod)
		adminGroup.POST("/content/single/del", sg.Del)

		mediaCtl := &content.MediaController{}
		adminGroup.GET("/content/media/index", mediaCtl.Index)
		adminGroup.GET("/content/media/list", mediaCtl.List)
		adminGroup.GET("/content/media/detail", mediaCtl.Detail)
		adminGroup.POST("/content/media/mark", mediaCtl.Mark)
		adminGroup.POST("/content/media/clean", mediaCtl.Clean)
		adminGroup.POST("/content/media/refresh", mediaCtl.Refresh)
		adminGroup.GET("/content/media/backup", mediaCtl.BackupList)
		adminGroup.POST("/content/media/restore", mediaCtl.Restore)
		adminGroup.POST("/content/media/backupClear", mediaCtl.BackupClear)

		co := &content.CompanyController{}
		adminGroup.GET("/content/company/index", co.Index)
		adminGroup.POST("/content/company/mod", co.Mod)

		si := &content.SiteController{}
		adminGroup.GET("/content/site/index", si.Index)
		adminGroup.POST("/content/site/mod", si.Mod)
		adminGroup.GET("/content/site/server", si.Server)

		sl := &content.SlideController{}
		adminGroup.GET("/content/slide/index", sl.Index)
		adminGroup.GET("/content/slide/add", sl.Add)
		adminGroup.POST("/content/slide/add", sl.Add)
		adminGroup.POST("/content/slide/mod", sl.Mod)
		adminGroup.Any("/content/slide/mod/*action", sl.Mod)
		adminGroup.POST("/content/slide/del", sl.Del)
		adminGroup.Any("/content/slide/del/*action", sl.Del)
		// 短路由別名（模板 {url./admin/Slide/mod} 生成 /admin/slide/mod）
		adminGroup.POST("/slide/mod", sl.Mod)
		adminGroup.GET("/slide/mod/*action", sl.Mod)
		adminGroup.POST("/slide/mod/*action", sl.Mod)
		adminGroup.POST("/slide/del", sl.Del)
		adminGroup.Any("/slide/del/*action", sl.Del)

		lk := &content.LinkController{}
		adminGroup.GET("/content/link/index", lk.Index)
		adminGroup.GET("/content/link/add", lk.Add)
		adminGroup.POST("/content/link/add", lk.Add)
		adminGroup.Any("/content/link/mod/*action", lk.Mod)
		adminGroup.POST("/content/link/del", lk.Del)
		adminGroup.Any("/content/link/del/*action", lk.Del)

		ms := &content.MessageController{}
		adminGroup.GET("/content/message/index", ms.Index)
		adminGroup.GET("/content/message", ms.Index)
		adminGroup.Any("/content/message/mod/*action", ms.Mod)
		adminGroup.GET("/content/message/del", ms.Del)
		adminGroup.POST("/content/message/del", ms.Del)
		adminGroup.Any("/content/message/del/*action", ms.Del)
		adminGroup.GET("/content/message/clear", ms.Clear)
		adminGroup.POST("/content/message/clear", ms.Clear)

		tg := &content.TagsController{}
		adminGroup.GET("/content/tags/index", tg.Index)
		adminGroup.GET("/content/tags/add", tg.Add)
		adminGroup.POST("/content/tags/add", tg.Add)
		adminGroup.POST("/content/tags/del", tg.Del)
		adminGroup.Any("/content/tags/del/*action", tg.Del)
		adminGroup.Any("/content/tags/mod/*action", tg.Mod)

		lb := &content.LabelController{}
		adminGroup.GET("/content/label/index", lb.Index)
		adminGroup.POST("/content/label/index", lb.Index)
		adminGroup.GET("/content/label/add", lb.Add)
		adminGroup.POST("/content/label/add", lb.Add)
		adminGroup.GET("/content/label/mod/:id", lb.Mod)
		adminGroup.POST("/content/label/mod/:id", lb.Mod)
		adminGroup.POST("/content/label/del", lb.Del)

		dc := &content.DeleCacheController{}
		adminGroup.GET("/content/deleCache/index", dc.Index)
		adminGroup.POST("/content/deleCache/index", dc.Index)
		// 別名路由：模板 URL 解析會轉全小寫，路由需匹配
		adminGroup.GET("/delecache/index", dc.Index)
		adminGroup.POST("/delecache/index", dc.Index)

		fm := &content.FormController{}
		adminGroup.GET("/content/form/index", fm.Index)
		adminGroup.GET("/content/form/index/*action", fm.Index)
		adminGroup.POST("/content/form/add", fm.Add)
		adminGroup.POST("/content/form/add/*action", fm.Add)
		adminGroup.GET("/content/form/del/*action", fm.Del)
		adminGroup.POST("/content/form/del/*action", fm.Del)
		adminGroup.GET("/content/form/mod/*action", fm.Mod)
		adminGroup.POST("/content/form/mod/*action", fm.Mod)
		adminGroup.GET("/content/form/clear/*action", fm.Clear)

		md := &content.ModelController{}
		adminGroup.GET("/content/model/index", md.Index)
		adminGroup.Any("/content/model/add", md.Add)
		adminGroup.Any("/content/model/mod/*action", md.Mod)
		adminGroup.Any("/content/model/del/*action", md.Del)

		ef := &content.ExtFieldController{}
	adminGroup.GET("/content/extField/index", ef.Index)
	adminGroup.GET("/content/extField/add", ef.Add)
	adminGroup.POST("/content/extField/add", ef.Add)
	adminGroup.Any("/content/extField/mod/*action", ef.Mod)
	adminGroup.POST("/content/extField/del", ef.Del)

	rdc := &content.RedirectController{}
	adminGroup.GET("/content/redirect/index", rdc.Index)
	adminGroup.GET("/content/redirect/add", rdc.Add)
	adminGroup.POST("/content/redirect/add", rdc.Add)
	adminGroup.Any("/content/redirect/mod/*action", rdc.Mod)
	adminGroup.POST("/content/redirect/del", rdc.Del)

		mc := &system.MenuController{}
		adminGroup.GET("/system/menu/index", mc.Index)
		adminGroup.GET("/system/menu/add", mc.Add)
		adminGroup.POST("/system/menu/add", mc.Add)
		adminGroup.GET("/system/menu/mod/:id", mc.Mod)
		adminGroup.POST("/system/menu/mod/:id", mc.Mod)
		adminGroup.POST("/system/menu/mod", mc.Mod)
		// 短路由別名
		adminGroup.POST("/menu/mod", mc.Mod)
		adminGroup.POST("/menu/mod/:id", mc.Mod)
		adminGroup.POST("/system/menu/del", mc.Del)

		uc := &system.UserController{}
		adminGroup.GET("/system/user/index", uc.Index)
		adminGroup.GET("/system/user/add", uc.Add)
		adminGroup.POST("/system/user/add", uc.Add)
		adminGroup.Any("/system/user/mod/*action", uc.Mod)
		adminGroup.Any("/system/user/del/*action", uc.Del)

		rc := &system.RoleController{}
		adminGroup.GET("/system/role/index", rc.Index)
		adminGroup.GET("/system/role/add", rc.Add)
		adminGroup.POST("/system/role/add", rc.Add)
		adminGroup.Any("/system/role/mod/*action", rc.Mod)
		adminGroup.Any("/system/role/del/*action", rc.Del)

		cf := &system.ConfigController{}
		adminGroup.GET("/system/config/index", cf.Index)
		adminGroup.POST("/system/config/index", cf.Index)

		db := &system.DatabaseController{}
		adminGroup.GET("/system/database/index", db.Index)
		adminGroup.POST("/system/database/mod", db.Mod)
		adminGroup.GET("/system/database/download", db.DownloadBackup)
		adminGroup.POST("/system/database/restore", db.RestoreSQLite)

		ar := &system.AreaController{}
		adminGroup.GET("/system/area/index", ar.Index)
		adminGroup.GET("/system/area/add", ar.Add)
		adminGroup.POST("/system/area/add", ar.Add)
		adminGroup.GET("/system/area/mod/*action", ar.Mod)
		adminGroup.POST("/system/area/mod/*action", ar.Mod)
		adminGroup.GET("/system/area/del/*action", ar.Del)
		adminGroup.POST("/system/area/del/*action", ar.Del)

		slc := &system.SyslogController{}
		adminGroup.GET("/system/syslog/index", slc.Index)
		adminGroup.GET("/system/syslog/index/*action", slc.IndexCatchAll)
		adminGroup.POST("/system/syslog/clear", slc.Clear)
		adminGroup.POST("/system/syslog/clearspider", slc.ClearSpider)
		adminGroup.POST("/system/syslog/clearnotify", slc.ClearNotify)

		mb := &member.MemberController{}
	adminGroup.GET("/member/index", mb.Index)
	adminGroup.GET("/member/add", mb.Add)
	adminGroup.POST("/member/add", mb.Add)
	adminGroup.Any("/member/mod/*action", mb.Mod)
	adminGroup.POST("/member/del", mb.Del)
	adminGroup.Any("/member/del/*action", mb.Del)

	mg := &member.MemberGroupController{}
	adminGroup.GET("/member/group/index", mg.Index)
	adminGroup.GET("/member/group/add", mg.Add)
	adminGroup.POST("/member/group/add", mg.Add)
	adminGroup.Any("/member/group/mod/*action", mg.Mod)
	adminGroup.POST("/member/group/del", mg.Del)
	adminGroup.Any("/member/group/del/*action", mg.Del)

	mf := &member.MemberFieldController{}
	adminGroup.GET("/member/field/index", mf.Index)
	adminGroup.GET("/member/field/add", mf.Add)
	adminGroup.POST("/member/field/add", mf.Add)
	adminGroup.Any("/member/field/mod/*action", mf.Mod)
	adminGroup.POST("/member/field/del", mf.Del)
	adminGroup.Any("/member/field/del/*action", mf.Del)

	mcc := &member.MemberCommentController{}
	adminGroup.GET("/member/comment/index", mcc.Index)
	adminGroup.Any("/member/comment/mod/*action", mcc.Mod)
	adminGroup.POST("/member/comment/del", mcc.Del)
	adminGroup.Any("/member/comment/del/*action", mcc.Del)
	}
}

// SetupAPIRoutes 註冊 RESTful API 路由
func SetupAPIRoutes(r *gin.Engine) {
	// CORS 必須全域掛載，否則 Gin 在路由匹配前就返回 404 給 OPTIONS
	r.Use(api.CORS())

	apiGroup := r.Group("/api/v1")
	apiGroup.Use(api.APIAuth())
	{
		// 認證
		apiGroup.POST("/auth/login", api.Login)
		apiGroup.POST("/auth/refresh", api.RefreshToken)

		// 站點資訊
		apiGroup.GET("/site", api.GetSite)
		apiGroup.GET("/company", api.GetCompany)

		// 欄目
		apiGroup.GET("/sorts", api.ListSorts)
		apiGroup.GET("/sorts/:scode", api.GetSort)
		apiGroup.GET("/nav", api.ListNav)

		// 內容
		apiGroup.GET("/contents", api.ListContents)
		apiGroup.GET("/contents/:id", api.GetContent)
		apiGroup.GET("/contents/:id/images", api.GetContentImages)

		// 搜索
		apiGroup.GET("/search", api.SearchContent)

		// 留言
		apiGroup.POST("/messages", api.CreateMessage)
		apiGroup.GET("/messages", api.ListMessages) // 需認證

		// 自定義表單（需認證）
		apiGroup.GET("/forms/:fcode/fields", api.ListFormFields)
		apiGroup.GET("/forms/:fcode/data", api.ListFormData)

		// 幻燈片
		apiGroup.GET("/slides", api.ListSlides)

		// 友情連結
		apiGroup.GET("/links", api.ListLinks)

		// 標籤
		apiGroup.GET("/tags", api.ListTags)
	}
}