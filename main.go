package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	home "gbootcms/apps/home/controller"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/member"
	"gbootcms/apps/admin/model/system"
	"gbootcms/apps/admin/seed"
	"gbootcms/apps/api"
	"gbootcms/apps/common"
	"gbootcms/apps/common/middleware"
	"gbootcms/apps/common/parser"
	"gbootcms/apps/common/storage"
	"gbootcms/apps/route"
	"gbootcms/config"
	"gbootcms/core/acodeplugin"
	"gbootcms/core/basic"
	gdb "gbootcms/core/db"

	"github.com/gin-gonic/gin"
)

// validAcodes 快取合法的 acode 列表（從 DB 動態載入，管理員新增區域後即時生效）
var (
	validAcodes   map[string]bool
	validAcodesMu sync.RWMutex
)

func loadValidAcodes() {
	skipCtx := acodeplugin.SkipAcode(context.Background())
	var areas []model.Area
	model.DB.WithContext(skipCtx).Find(&areas)
	m := make(map[string]bool, len(areas))
	for _, a := range areas {
		m[a.Acode] = true
	}
	validAcodesMu.Lock()
	validAcodes = m
	validAcodesMu.Unlock()
}

func isValidAcode(acode string) bool {
	validAcodesMu.RLock()
	if validAcodes == nil {
		validAcodesMu.RUnlock()
		loadValidAcodes()
		validAcodesMu.RLock()
	}
	result := validAcodes[acode]
	validAcodesMu.RUnlock()
	return result
}

// RefreshValidAcodes 刷新合法 acode 快取（管理員新增/修改/刪除區域後呼叫）
func RefreshValidAcodes() {
	loadValidAcodes()
}

func main() {
	cfg := config.Load("config/config.json")

	if err := model.InitDB(cfg); err != nil {
		log.Fatalf("Database init failed: %v", err)
	}
	defer model.CloseDB()

	// 註冊數據變更回調：基於 Cache Tag 精準失效
	// 不同表的變更觸發不同層級的快取失效：
	//   - content:{id}  → 僅失效該文章詳情頁
	//   - content:list  → 失效所有列表頁 + 首頁
	//   - global        → 失效全部頁面（配置/站點/導航等全局數據變更）
	gdb.OnDataChange = func(tableName string, id int) {
		slog.Info("OnDataChange triggered", "table", tableName, "id", id)

		// === HTML 頁面快取 tag 失效 ===
		switch tableName {
		case "content":
			if id > 0 {
				middleware.InvalidateTag(fmt.Sprintf("content:%d", id))
			}
			middleware.InvalidateTag("content:list")
		case "content_ext":
			// content_ext 主鍵即為 content_id（1:1 關聯）
			if id > 0 {
				middleware.InvalidateTag(fmt.Sprintf("content:%d", id))
			}
		case "content_sort":
			if id > 0 {
				middleware.InvalidateTag(fmt.Sprintf("content_sort:%d", id))
			}
			middleware.InvalidateTag("content:list")
		default:
			// site, company, config, menu, slide, link, tags, label, single 等
			// 這些表的數據出現在多個頁面，安全起見失效全部
			slog.Warn("OnDataChange default branch → global invalidation", "table", tableName, "id", id)
			middleware.InvalidateTag("global")
		}

		// === 非 HTML 快取按表精準清除（避免不必要的快取失效） ===
		switch tableName {
		case "config":
			model.ClearConfigCache()
		case "area":
			model.ClearAreasCache()
			RefreshValidAcodes()
		case "site":
			home.ClearDataCache()
			parser.ClearSiteDomainsCache()
		case "company":
			home.ClearDataCache()
		case "tags":
			parser.ClearTagsCache()
		}
	}

	// AutoMigrate all models: system + content + member
	system.AutoMigrate()
	content.AutoMigrate()
	member.AutoMigrate()

	// Seed initial data (admin user, menus, configs, etc.)
	seed.Init()

	basic.InitViewEngine(cfg.App.TemplateDir, cfg.App.AdminTemplateDir)

	tagParser := parser.New()
	store, err := parser.NewTemplateStore(cfg.App.TemplateDir, tagParser)
	if err != nil {
		log.Fatalf("Template engine init failed: %v", err)
	}
	defer store.Close()

	// 模板熱重載時觸發 HTML 快取失效（全域 tag + 檔案快取清除）
	parser.OnTemplateChange = func() {
		middleware.ClearHTMLCache()
		slog.Info("模板變更，已清除全部 HTML 快取")
	}

	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// URL 規範化：剝離 .html 後綴 + 語言前綴解析（/sc /tc /en）
	// gin 路由匹配先於中間件，需用 HandleContext 重新路由
	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path

		// 跳過後台、靜態資源、API、SEO 文件路徑（不解析語言前綴）
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/static") ||
			strings.HasPrefix(path, "/template") || strings.HasPrefix(path, "/api") ||
			strings.HasPrefix(path, "/sitemap") || strings.HasPrefix(path, "/robots") ||
			strings.HasPrefix(path, "/llms") {
			c.Next()
			return
		}

		// SEO：301 重定向 .html URL 到無後綴的乾淨 URL（Google SEO 標準）
		// 放在語言前綴檢測之前，確保重定向保留語言前綴（如 /tc/industry/36.html → /tc/industry/36）
		if strings.HasSuffix(path, ".html") {
			c.Redirect(http.StatusMovedPermanently, strings.TrimSuffix(path, ".html"))
			c.Abort()
			return
		}

		// 檢測語言前綴（/sc/... /tc/... /en/...）
		trimmed := strings.TrimPrefix(path, "/")
		segments := strings.SplitN(trimmed, "/", 2)
		if len(segments) > 0 && isValidAcode(segments[0]) {
			acode := segments[0]
			// 剝離前綴後的重路徑
			if len(segments) > 1 && segments[1] != "" {
				c.Request.URL.Path = "/" + segments[1]
			} else {
				c.Request.URL.Path = "/"
			}
			// 將 acode 存入 request context 供 InjectAcode 讀取
			ctx := middleware.SetURLAcode(c.Request.Context(), acode)
			c.Request = c.Request.WithContext(ctx)
			r.HandleContext(c)
			c.Abort()
			return
		}

		c.Next()
	})

	// 關站檢查（對齊 PHP HomeController.__construct，最高優先級，前台生效）
	r.Use(middleware.SiteStatus())

	// 301 重定向（應用層 URL 重寫規則，補充 Nginx 靜態規則）
	r.Use(middleware.Redirect301())

	// HTTPS 跳轉和主域名跳轉（對齊 PHP HomeController.__construct，前台生效）
	r.Use(middleware.SiteRedirect())

	// IP 黑白名單過濾（對齊 PHP HomeController.__construct，前台生效）
	r.Use(middleware.IPFilter())

	// 手機版切換（對齊 PHP HomeController.__construct 的 open_wap 邏輯）
	r.Use(middleware.MobileSwitch())

	// 蜘蛛訪問記錄（對齊 PHP SpiderController，異步寫入日誌）
	r.Use(middleware.SpiderLog())

	// 頁面壓縮：Brotli 優先，Gzip 回退（對齊 PHP Controller::gzip）
	// 必須在 HTMLCache 之前：壓縮先包裹 ResponseWriter，緩存再包裹壓縮層
	// 這樣緩存存的是未壓縮 HTML，命中時經壓縮層正常壓縮返回
	r.Use(middleware.Compress())

	// 動態頁面緩存（記憶體層永遠開啟，檔案層由 tpl_html_cache 控制）
	// 在 Compress 之後：快取未壓縮的原始 HTML，命中時自動經壓縮層返回
	r.Use(middleware.HTMLCache())

	// 區域隔離：後台從 session 讀取 acode，前台依域名匹配
	// 注入到 request context，供 GORM AcodePlugin 自動過濾
	r.Use(middleware.AcodeMiddleware())

	r.Static("/static", cfg.App.StaticDir)
	// PbootCMS 兼容: 前台模板靜態資源（CSS/JS/圖片）
	r.Static("/template/default/static", filepath.Join(cfg.App.TemplateDir, "static"))
	// 多語言模板靜態資源
	r.Static("/template/sc/static", filepath.Join(filepath.Dir(cfg.App.TemplateDir), "sc", "static"))
	r.Static("/template/en/static", filepath.Join(filepath.Dir(cfg.App.TemplateDir), "en", "static"))

	route.SetupAdminRoutes(r)
	route.SetupAPIRoutes(r)

	// 初始化 MeiliSearch 全文搜索（如果已配置）
	go api.InitMeiliSearch()

	// API JWT 密鑰檢查：未配置時輸出警告（不阻止啟動，但登入會回傳 500）
	if !api.IsJWTConfigured() {
		slog.Warn("API JWT 密鑰未配置（api_jwt_secret），API 登入將不可用，請在後台系統配置中設定")
	}

	// 載入敏感詞列表
	common.LoadSensitiveWords()

	// 初始化雲存儲（如果配置了 R2）
	storage.InitStorage()

	fc := home.NewFrontController(store)

	r.GET("/", fc.Index)
	r.GET("/search", fc.Search)
	r.GET("/tags", fc.Tags)
	r.GET("/message", fc.Message)
	r.POST("/message", fc.Message)
	r.GET("/api/visits", middleware.APIRateLimit(), fc.Visits)
	r.POST("/api/likes", middleware.VoteRateLimit(), fc.Likes)
	r.POST("/api/oppose", middleware.VoteRateLimit(), fc.Oppose)
	r.GET("/api/checkcode", fc.CheckCode)

	// SEO: sitemap 索引 + robots.txt
	// 每語言獨立 sitemap（/sitemap-{acode}.xml）在 NoRoute 處理器中處理
	// 因為 Gin 的 :param 不支援路徑段中間匹配（如 /sitemap-:acode.xml）
	r.GET("/sitemap.xml", fc.Sitemap)
	r.GET("/robots.txt", fc.Robots)
	r.GET("/llms.txt", fc.LLMS)

	// 會員系統路由
	r.GET("/login", fc.Login)
	r.POST("/login", fc.Login)
	r.GET("/register", fc.Register)
	r.POST("/register", fc.Register)
	r.GET("/logout", fc.Logout)
	r.GET("/ucenter", fc.Ucenter)
	r.GET("/umodify", fc.Umodify)
	r.POST("/umodify", fc.Umodify)

	// 會員找回密碼
	r.GET("/retrieve", fc.Retrieve)
	r.POST("/retrieve", fc.Retrieve)
	r.POST("/member/sendemail", fc.SendMemberEmail)

	// IndexNow 密鑰驗證文件（Bing/Yandex 等搜索引擎推送所需）
	// 在 NoRoute 處理器中處理，避免參數路由影響前台路由匹配

	// 前台評論系統路由
	cc := &home.CommentController{FrontController: fc}
	r.POST("/comment/add", cc.Add)
	r.GET("/comment/my", cc.My)
	r.POST("/comment/del", cc.Del) // CSRF 防護：刪除操作僅允許 POST，防止 GET 請求偽造

	// Dynamic content routes — generated by sortToMap/contentToMap
	// when urlname is empty. PbootCMS convention:
	//   /sort/{scode}    → list page of the sort (scode 從不改變)
	//   /content/{id}     → content detail page (id is monotonic)
	r.GET("/sort/:scode", fc.SortByScode)
	r.GET("/content/:id", fc.ContentByID)

	r.NoRoute(func(c *gin.Context) {
		// 前台 NoRoute 路由也需要區域隔離（全局中間件不覆蓋 NoRoute）
		middleware.InjectAcode(c)

		original := c.Request.URL.Path

		// SEO: 每語言獨立 sitemap 處理（/sitemap-{acode}.xml）
		if strings.HasPrefix(original, "/sitemap-") && strings.HasSuffix(original, ".xml") {
			acode := strings.TrimSuffix(strings.TrimPrefix(original, "/sitemap-"), ".xml")
			fc.SitemapLang(c, acode)
			return
		}

		// IndexNow 密鑰驗證文件處理（/{key}.txt）
		if strings.HasSuffix(original, ".txt") {
			keyfile := strings.TrimPrefix(original, "/")
			keyfile = strings.TrimSuffix(keyfile, ".txt")
			if key := model.GetConfigValue("bing_indexnow_key", ""); key != "" && keyfile == key {
				c.String(http.StatusOK, key)
				return
			}
		}

		// PbootCMS 模板生成的 URL 大小寫可能與 Go 路由不一致
		// (e.g. /admin/Content/index vs /admin/content/index)
		// RewriteAdminPath 內部已做大小寫不敏感的前綴匹配，直接傳原始路徑
		// 以保留路徑參數的原始大小寫（如 rcode=R102）
		newPath := middleware.RewriteAdminPath(original)
		if newPath != original {
			c.Request.Header.Set("X-Original-Path", original)
			c.Request.URL.Path = newPath
			r.HandleContext(c)
			return
		}
		// 不匹配 admin 路径，走前台内容页
		fc.ContentPage(c)
	})

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	slog.Info("Gbootcms 已啟動", "url", "http://localhost"+addr)
	slog.Info("後台管理", "url", "http://localhost"+addr+"/admin")

	// 啟動時預熱快取：對首頁發一次內部 HTTP 請求，填充記憶體快取
	// 這樣第一個真實訪客也不會遇到冷緩存的高延遲
	go func() {
		warmURL := fmt.Sprintf("http://localhost%s/", addr)
		// 重試 3 次，每次間隔遞增（伺服器可能需要時間完全就緒）
		for i := 0; i < 3; i++ {
			time.Sleep(time.Duration(500+i*500) * time.Millisecond)
			resp, err := http.Get(warmURL)
			if err != nil {
				slog.Warn("快取預熱失敗", "attempt", i+1, "error", err)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == 200 {
				slog.Info("快取預熱完成", "url", warmURL, "status", resp.StatusCode)
				return
			}
			slog.Warn("快取預熱狀態碼異常", "attempt", i+1, "status", resp.StatusCode)
		}
		slog.Warn("快取預熱未成功，第一個訪客將遇到冷緩存")
	}()

	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
