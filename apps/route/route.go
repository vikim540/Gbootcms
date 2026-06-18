package route

import (
	admin "pbootcms-go/apps/admin/controller"
	content "pbootcms-go/apps/admin/controller/content"
	member "pbootcms-go/apps/admin/controller/member"
	system "pbootcms-go/apps/admin/controller/system"
	"pbootcms-go/apps/common/middleware"

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
		adminGroup.POST("/index/clearCache", ic.ClearCache)
		adminGroup.POST("/index/area", ic.Area)
		adminGroup.GET("/index/checkCode", ic.CheckCode)
		adminGroup.POST("/index/upload", ic.Upload)
		adminGroup.POST("/index/upload/watermark/:flag", ic.Upload)

		cc := &content.ContentController{}
		adminGroup.GET("/content/index", cc.Index)
		adminGroup.GET("/content/index/*action", cc.IndexCatchAll)
		adminGroup.GET("/content/add", cc.Add)
		adminGroup.GET("/content/add/*action", cc.AddCatchAll)
		adminGroup.POST("/content/add", cc.Add)
		adminGroup.POST("/content/add/*action", cc.AddCatchAll)
		adminGroup.Any("/content/mod/*action", cc.Mod)
		adminGroup.POST("/content/del", cc.Del)
		adminGroup.Any("/content/del/*action", cc.DelCatchAll)

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
		adminGroup.GET("/content/message/mod/:id", ms.Mod)
		adminGroup.POST("/content/message/mod/:id", ms.Mod)
		adminGroup.POST("/content/message/del", ms.Del)
		adminGroup.POST("/content/message/clear", ms.Clear)

		tg := &content.TagsController{}
		adminGroup.GET("/content/tags/index", tg.Index)
		adminGroup.GET("/content/tags/add", tg.Add)
		adminGroup.POST("/content/tags/add", tg.Add)
		adminGroup.POST("/content/tags/del", tg.Del)
		adminGroup.GET("/content/tags/mod/:id", tg.Mod)
		adminGroup.POST("/content/tags/mod/:id", tg.Mod)

		lb := &content.LabelController{}
		adminGroup.GET("/content/label/index", lb.Index)
		adminGroup.GET("/content/label/add", lb.Add)
		adminGroup.POST("/content/label/add", lb.Add)
		adminGroup.GET("/content/label/mod/:id", lb.Mod)
		adminGroup.POST("/content/label/mod/:id", lb.Mod)
		adminGroup.POST("/content/label/del", lb.Del)

		dc := &content.DeleCacheController{}
		adminGroup.GET("/content/deleCache/index", dc.Index)
		adminGroup.POST("/content/deleCache/index", dc.Index)

		fm := &content.FormController{}
		adminGroup.GET("/content/form/index", fm.Index)
		adminGroup.POST("/content/form/add", fm.Add)
		adminGroup.POST("/content/form/del", fm.Del)
		adminGroup.POST("/content/form/mod", fm.Mod)
		adminGroup.POST("/content/form/clear", fm.Clear)

		md := &content.ModelController{}
		adminGroup.GET("/content/model/index", md.Index)
		adminGroup.Any("/content/model/add", md.Add)
		adminGroup.Any("/content/model/mod/*action", md.Mod)
		adminGroup.Any("/content/model/del/*action", md.Del)

		ef := &content.ExtFieldController{}
		adminGroup.GET("/content/extField/index", ef.Index)
		adminGroup.GET("/content/extField/add", ef.Add)
		adminGroup.POST("/content/extField/add", ef.Add)
		adminGroup.GET("/content/extField/mod/:id", ef.Mod)
		adminGroup.POST("/content/extField/mod/:id", ef.Mod)
		adminGroup.POST("/content/extField/del", ef.Del)

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
		adminGroup.GET("/system/user/mod/:id", uc.Mod)
		adminGroup.POST("/system/user/mod/:id", uc.Mod)
		adminGroup.POST("/system/user/del", uc.Del)

		rc := &system.RoleController{}
		adminGroup.GET("/system/role/index", rc.Index)
		adminGroup.GET("/system/role/add", rc.Add)
		adminGroup.POST("/system/role/add", rc.Add)
		adminGroup.GET("/system/role/mod/:id", rc.Mod)
		adminGroup.POST("/system/role/mod/:id", rc.Mod)
		adminGroup.POST("/system/role/del", rc.Del)

		cf := &system.ConfigController{}
		adminGroup.GET("/system/config/index", cf.Index)
		adminGroup.POST("/system/config/index", cf.Index)

		db := &system.DatabaseController{}
		adminGroup.GET("/system/database/index", db.Index)
		adminGroup.POST("/system/database/mod", db.Mod)

		ar := &system.AreaController{}
		adminGroup.GET("/system/area/index", ar.Index)
		adminGroup.GET("/system/area/add", ar.Add)
		adminGroup.POST("/system/area/add", ar.Add)
		adminGroup.GET("/system/area/mod/:id", ar.Mod)
		adminGroup.POST("/system/area/mod/:id", ar.Mod)
		adminGroup.POST("/system/area/del", ar.Del)

		slc := &system.SyslogController{}
		adminGroup.GET("/system/syslog/index", slc.Index)
		adminGroup.POST("/system/syslog/clear", slc.Clear)

		tp := &system.TypeController{}
		adminGroup.GET("/system/type/index", tp.Index)
		adminGroup.GET("/system/type/add", tp.Add)
		adminGroup.POST("/system/type/add", tp.Add)
		adminGroup.GET("/system/type/mod/:id", tp.Mod)
		adminGroup.POST("/system/type/mod/:id", tp.Mod)
		adminGroup.POST("/system/type/del", tp.Del)

		up := &system.UpgradeController{}
		adminGroup.GET("/system/upgrade/index", up.Index)

		ie := &system.ImageExtController{}
		adminGroup.GET("/system/imageExt/index", ie.Index)
		adminGroup.GET("/system/imageExt/checkDataFile", ie.CheckDataFile)
		adminGroup.POST("/system/imageExt/doExt", ie.DoExt)

		mb := &member.MemberController{}
		adminGroup.GET("/member/index", mb.Index)
		adminGroup.GET("/member/add", mb.Add)
		adminGroup.POST("/member/add", mb.Add)
		adminGroup.GET("/member/mod/:id", mb.Mod)
		adminGroup.POST("/member/mod/:id", mb.Mod)
		adminGroup.POST("/member/del", mb.Del)

		mg := &member.MemberGroupController{}
		adminGroup.GET("/member/group/index", mg.Index)
		adminGroup.GET("/member/group/add", mg.Add)
		adminGroup.POST("/member/group/add", mg.Add)
		adminGroup.GET("/member/group/mod/:id", mg.Mod)
		adminGroup.POST("/member/group/mod/:id", mg.Mod)
		adminGroup.POST("/member/group/del", mg.Del)

		mf := &member.MemberFieldController{}
		adminGroup.GET("/member/field/index", mf.Index)
		adminGroup.GET("/member/field/add", mf.Add)
		adminGroup.POST("/member/field/add", mf.Add)
		adminGroup.GET("/member/field/mod/:id", mf.Mod)
		adminGroup.POST("/member/field/mod/:id", mf.Mod)
		adminGroup.POST("/member/field/del", mf.Del)

		mcc := &member.MemberCommentController{}
		adminGroup.GET("/member/comment/index", mcc.Index)
		adminGroup.GET("/member/comment/mod/:id", mcc.Mod)
		adminGroup.POST("/member/comment/mod/:id", mcc.Mod)
		adminGroup.POST("/member/comment/del", mcc.Del)
	}
}