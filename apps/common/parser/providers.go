package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/admin/model/content"
	"gbootcms/core/acodeplugin"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Context struct {
	Sort         *model.ContentSort
	Content      *model.Content
	ContentExt   map[string]interface{}   // 當前內容的擴展字段快取（詳情頁避免重複查詢）
	sortPathCache map[string]string        // 欄目路徑快取（scode → sortPath），避免重複查詢 ContentSort
	ifContext    map[string]interface{}   // {gboot:if} 條件上下文快取，避免每個 if 標籤重複建構 + 8 次 GetConfigValue
	Site         *model.Site
	Company      *model.Company
	Page         map[string]interface{}
	Member       *model.Member
	Gcode        int    // 當前訪客的會員等級編號（0=未登入或無等級）
	Ucode        string // 當前訪客的會員編號
	Keyword      string
	CurrentPage  int
	Filters      map[string]string // ext_ 篩選參數 (ext_type=基礎版 等)
	Ctx          context.Context   // 請求級 context，用於區域數據隔離
	CurrentPath  string            // 當前頁面路徑（已剝離 acode 前綴），用於語言切換保持當前頁
	HttpURL      string            // 動態請求 URL（http(s)://Host），對齊 PbootCMS get_http_url()
}

// safeFieldRe 擴展字段名白名單正則：只允許 ext_ 前綴 + 字母數字下底線
// 防止 SQL 注入：字段名直接拼接進 SQL 查詢，必須嚴格驗證
var safeFieldRe = regexp.MustCompile(`^ext_[a-zA-Z0-9_]+$`)

// 預編譯熱路徑正則（避免每次渲染重複編譯）
var (
	safeTableRe    = regexp.MustCompile(`^[\w]+$`)
	commentSubRe   = regexp.MustCompile(`(?s)\{gboot:commentsub(?:\s+([^}]+))?\}(.*?)\{/gboot:commentsub\}`)
	innerIfRe      = regexp.MustCompile(`(?s)\{gboot:if\(([^)]+)\)\}(.*?)(?:\{else\}(.*?))?\{/gboot:if\}`)
)

// IsSafeFieldName 驗證字段名是否安全（僅允許 ext_ 前綴的合法欄位名）
func IsSafeFieldName(name string) bool {
	return safeFieldRe.MatchString(name)
}

// addLeadingSlash 為本地路徑添加前導 /
func addLeadingSlash(path string) string {
	if path != "" && !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "http") {
		return "/" + path
	}
	return path
}

// i18nUIDict 內建多語言 UI 靜態文字詞典（支持中簡、中繁、英）
var i18nUIDict = map[string]map[string]string{
	"home": {
		"tc": "首頁",
		"sc": "首页",
		"en": "Homepage",
		"default": "首頁",
	},
	"ucenter": {
		"tc": "個人中心",
		"sc": "个人中心",
		"en": "My Account",
		"default": "個人中心",
	},
	"register": {
		"tc": "註冊",
		"sc": "注册",
		"en": "Register",
		"default": "註冊",
	},
	"login": {
		"tc": "登入",
		"sc": "登录",
		"en": "Login",
		"default": "登入",
	},
	"contact_us": {
		"tc": "聯繫我們",
		"sc": "联系我们",
		"en": "Contact Us",
		"default": "聯繫我們",
	},
	"phone": {
		"tc": "電話",
		"sc": "电话",
		"en": "Phone",
		"default": "電話",
	},
	"email": {
		"tc": "郵箱",
		"sc": "邮箱",
		"en": "Email",
		"default": "郵箱",
	},
	"address": {
		"tc": "公司地址",
		"sc": "公司地址",
		"en": "Address",
		"default": "公司地址",
	},
	"license": {
		"tc": "營業執照",
		"sc": "营业执照",
		"en": "Business License",
		"default": "營業執照",
	},
	"icp": {
		"tc": "備案號",
		"sc": "备案号",
		"en": "Filing No.",
		"default": "備案號",
	},
	"scan_mobile": {
		"tc": "掃碼手機端訪問",
		"sc": "扫码手机端访问",
		"en": "Scan for mobile access",
		"default": "掃碼手機端訪問",
	},
	"friendly_links": {
		"tc": "友情連結",
		"sc": "友情链接",
		"en": "Links",
		"default": "友情連結",
	},
	"online_service": {
		"tc": "在線諮詢",
		"sc": "在线咨询",
		"en": "Online Consultation",
		"default": "在線諮詢",
	},
	"free_call": {
		"tc": "免費通話",
		"sc": "免费通话",
		"en": "Free Call",
		"default": "免費通話",
	},
	"back_to_top": {
		"tc": "返回頂部",
		"sc": "返回顶部",
		"en": "Back to Top",
		"default": "返回頂部",
	},
	"wechat_contact": {
		"tc": "微信聯繫",
		"sc": "微信联系",
		"en": "WeChat Contact",
		"default": "微信聯繫",
	},
	"search": {
		"tc": "搜索",
		"sc": "搜索",
		"en": "Search",
		"default": "搜索",
	},
	"home_page": {
		"tc": "首頁",
		"sc": "首页",
		"en": "First",
		"default": "首頁",
	},
	"prev_page": {
		"tc": "上一頁",
		"sc": "上一页",
		"en": "Previous",
		"default": "上一頁",
	},
	"next_page": {
		"tc": "下一頁",
		"sc": "下一页",
		"en": "Next",
		"default": "下一頁",
	},
	"last_page": {
		"tc": "尾頁",
		"sc": "尾页",
		"en": "Last",
		"default": "尾頁",
	},
	"no_data": {
		"tc": "未查詢到任何資料！",
		"sc": "未查询到任何数据！",
		"en": "No data found!",
		"default": "未查詢到任何資料！",
	},
	"current_position": {
		"tc": "當前位置",
		"sc": "当前位置",
		"en": "Location",
		"default": "當前位置",
	},
	"all": {
		"tc": "全部",
		"sc": "全部",
		"en": "All",
		"default": "全部",
	},
	"date_label": {
		"tc": "時間",
		"sc": "时间",
		"en": "Date",
		"default": "時間",
	},
	"author_label": {
		"tc": "作者",
		"sc": "作者",
		"en": "Author",
		"default": "作者",
	},
	"visits_label": {
		"tc": "瀏覽",
		"sc": "浏览",
		"en": "Views",
		"default": "瀏覽",
	},
	"editor_label": {
		"tc": "編輯人",
		"sc": "编辑人",
		"en": "Editor",
		"default": "編輯人",
	},
	"media_platform": {
		"tc": "媒體平台",
		"sc": "媒体平台",
		"en": "Media Platform",
		"default": "媒體平台",
	},
	"prev_content": {
		"tc": "上一篇",
		"sc": "上一篇",
		"en": "Previous",
		"default": "上一篇",
	},
	"next_content": {
		"tc": "下一篇",
		"sc": "下一篇",
		"en": "Next",
		"default": "下一篇",
	},
	"post_comment": {
		"tc": "發表評論",
		"sc": "发表评论",
		"en": "Post Comment",
		"default": "發表評論",
	},
	"enter_comment": {
		"tc": "請輸入評論內容",
		"sc": "请输入评论内容",
		"en": "Enter your comment",
		"default": "請輸入評論內容",
	},
	"enter_checkcode": {
		"tc": "請輸入驗證碼",
		"sc": "请输入验证码",
		"en": "Enter verification code",
		"default": "請輸入驗證碼",
	},
	"click_refresh": {
		"tc": "點擊刷新",
		"sc": "点击刷新",
		"en": "Click to refresh",
		"default": "點擊刷新",
	},
	"submit_comment": {
		"tc": "提交評論",
		"sc": "提交评论",
		"en": "Submit Comment",
		"default": "提交評論",
	},
	"comment_list": {
		"tc": "評論記錄",
		"sc": "评论记录",
		"en": "Comments",
		"default": "評論記錄",
	},
	"reply": {
		"tc": "回覆",
		"sc": "回复",
		"en": "Reply",
		"default": "回覆",
	},
	"no_comments": {
		"tc": "暫無評論！",
		"sc": "暂无评论！",
		"en": "No comments yet!",
		"default": "暫無評論！",
	},
	"send_reply": {
		"tc": "發送回覆",
		"sc": "发送回复",
		"en": "Send Reply",
		"default": "發送回覆",
	},
	"enter_reply": {
		"tc": "請輸入回覆內容",
		"sc": "请输入回复内容",
		"en": "Enter your reply",
		"default": "請輸入回覆內容",
	},
	"submit_reply": {
		"tc": "提交回覆",
		"sc": "提交回复",
		"en": "Submit Reply",
		"default": "提交回覆",
	},
	"cancel": {
		"tc": "取消",
		"sc": "取消",
		"en": "Cancel",
		"default": "取消",
	},
	"message_record": {
		"tc": "留言記錄",
		"sc": "留言记录",
		"en": "Messages",
		"default": "留言記錄",
	},
	"admin_reply": {
		"tc": "管理員回覆",
		"sc": "管理员回复",
		"en": "Admin Reply",
		"default": "管理員回覆",
	},
	"do_not_fill": {
		"tc": "請勿填寫此欄位",
		"sc": "请勿填写此字段",
		"en": "Please leave blank",
		"default": "請勿填寫此欄位",
	},
	"contacts": {
		"tc": "聯繫人",
		"sc": "联系人",
		"en": "Contact Person",
		"default": "聯繫人",
	},
	"enter_contacts": {
		"tc": "請輸入聯繫人",
		"sc": "请输入联系人",
		"en": "Enter your name",
		"default": "請輸入聯繫人",
	},
	"mobile": {
		"tc": "手機",
		"sc": "手机",
		"en": "Mobile",
		"default": "手機",
	},
	"enter_mobile": {
		"tc": "請輸入聯繫人手機",
		"sc": "请输入联系人手机",
		"en": "Enter mobile number",
		"default": "請輸入聯繫人手機",
	},
	"content_label": {
		"tc": "內容",
		"sc": "内容",
		"en": "Content",
		"default": "內容",
	},
	"enter_content": {
		"tc": "請輸入留言內容",
		"sc": "请输入留言内容",
		"en": "Enter message content",
		"default": "請輸入留言內容",
	},
	"checkcode": {
		"tc": "驗證碼",
		"sc": "验证码",
		"en": "Captcha",
		"default": "驗證碼",
	},
	"captcha": {
		"tc": "人機驗證",
		"sc": "人机验证",
		"en": "Security Check",
		"default": "人機驗證",
	},
	"submit_message": {
		"tc": "提交留言",
		"sc": "提交留言",
		"en": "Submit Message",
		"default": "提交留言",
	},
	"search_results": {
		"tc": "搜索結果",
		"sc": "搜索结果",
		"en": "Search Results",
		"default": "搜索結果",
	},
	"keyword_label": {
		"tc": "關鍵字",
		"sc": "关键字",
		"en": "Keyword",
		"default": "關鍵字",
	},
	"enter_keyword": {
		"tc": "請輸入關鍵字",
		"sc": "请输入关键字",
		"en": "Enter keyword...",
		"default": "請輸入關鍵字",
	},
	"release_date": {
		"tc": "上架時間",
		"sc": "上架时间",
		"en": "Release Date",
		"default": "上架時間",
	},
	"product_type": {
		"tc": "產品類型",
		"sc": "产品类型",
		"en": "Product Type",
		"default": "產品類型",
	},
	"product_color": {
		"tc": "產品顏色",
		"sc": "产品颜色",
		"en": "Color",
		"default": "產品顏色",
	},
	"product_price": {
		"tc": "產品價格",
		"sc": "产品价格",
		"en": "Price",
		"default": "產品價格",
	},
	"inquire_now": {
		"tc": "馬上諮詢",
		"sc": "马上咨询",
		"en": "Inquire Now",
		"default": "馬上諮詢",
	},
	"product_details": {
		"tc": "產品詳情",
		"sc": "产品详情",
		"en": "Product Details",
		"default": "產品詳情",
	},
	"type_label": {
		"tc": "類型",
		"sc": "类型",
		"en": "Type",
		"default": "類型",
	},
	"color_label": {
		"tc": "顏色",
		"sc": "颜色",
		"en": "Color",
		"default": "顏色",
	},
	"account_info": {
		"tc": "帳號資訊",
		"sc": "账号信息",
		"en": "Account Info",
		"default": "帳號資訊",
	},
	"my_comments": {
		"tc": "我的評論",
		"sc": "我的评论",
		"en": "My Comments",
		"default": "我的評論",
	},
	"logout": {
		"tc": "退出登入",
		"sc": "退出登录",
		"en": "Logout",
		"default": "退出登入",
	},
	"username": {
		"tc": "帳號",
		"sc": "账号",
		"en": "Username",
		"default": "帳號",
	},
	"enter_username": {
		"tc": "請輸入登入用戶名/郵箱/手機號碼",
		"sc": "请输入登录用户名/邮箱/手机号码",
		"en": "Enter username / email / phone",
		"default": "請輸入登入用戶名/郵箱/手機號碼",
	},
	"password": {
		"tc": "密碼",
		"sc": "密码",
		"en": "Password",
		"default": "密碼",
	},
	"enter_password": {
		"tc": "請輸入登入密碼",
		"sc": "请输入登录密码",
		"en": "Enter password",
		"default": "請輸入登入密碼",
	},
	"login_now": {
		"tc": "立即登入",
		"sc": "立即登录",
		"en": "Login",
		"default": "立即登入",
	},
	"register_account": {
		"tc": "註冊帳號",
		"sc": "注册账号",
		"en": "Register Account",
		"default": "註冊帳號",
	},
	"forgot_password": {
		"tc": "找回密碼",
		"sc": "找回密码",
		"en": "Forgot Password",
		"default": "找回密碼",
	},
	"enter_reg_username": {
		"tc": "請輸入登入帳號",
		"sc": "请输入登录账号",
		"en": "Enter login account",
		"default": "請輸入登入帳號",
	},
	"nickname": {
		"tc": "暱稱",
		"sc": "昵称",
		"en": "Nickname",
		"default": "暱稱",
	},
	"enter_nickname": {
		"tc": "請輸入帳號暱稱",
		"sc": "请输入账号昵称",
		"en": "Enter nickname",
		"default": "請輸入帳號暱稱",
	},
	"confirm_password": {
		"tc": "確認密碼",
		"sc": "确认密码",
		"en": "Confirm Password",
		"default": "確認密碼",
	},
	"enter_confirm_password": {
		"tc": "請再次輸入登入密碼",
		"sc": "请再次输入登录密码",
		"en": "Re-enter password",
		"default": "請再次輸入登入密碼",
	},
	"register_now": {
		"tc": "立即註冊",
		"sc": "立即注册",
		"en": "Register Now",
		"default": "立即註冊",
	},
	"have_account": {
		"tc": "已有帳號？",
		"sc": "已有账号？",
		"en": "Have an account?",
		"default": "已有帳號？",
	},
	"enter_reg_account": {
		"tc": "請輸入已註冊帳號",
		"sc": "请输入已注册账号",
		"en": "Enter registered account",
		"default": "請輸入已註冊帳號",
	},
	"enter_reg_email": {
		"tc": "請輸入已註冊郵箱",
		"sc": "请输入已注册邮箱",
		"en": "Enter registered email",
		"default": "請輸入已註冊郵箱",
	},
	"email_code": {
		"tc": "郵箱驗證碼",
		"sc": "邮箱验证码",
		"en": "Email Code",
		"default": "郵箱驗證碼",
	},
	"send": {
		"tc": "發送",
		"sc": "发送",
		"en": "Send",
		"default": "發送",
	},
	"new_password": {
		"tc": "新密碼",
		"sc": "新密码",
		"en": "New Password",
		"default": "新密碼",
	},
	"enter_new_password": {
		"tc": "請輸入新的密碼",
		"sc": "请输入新的密码",
		"en": "Enter new password",
		"default": "請輸入新的密碼",
	},
	"confirm_reset": {
		"tc": "確認重置",
		"sc": "确认重置",
		"en": "Confirm Reset",
		"default": "確認重置",
	},
	"back_to_login": {
		"tc": "返回登入",
		"sc": "返回登录",
		"en": "Back to Login",
		"default": "返回登入",
	},
	"modify": {
		"tc": "修改",
		"sc": "修改",
		"en": "Edit",
		"default": "修改",
	},
}

func RegisterAllProviders(p *TagParser, ctx *Context) {
	registerSingleProviders(p, ctx)
	registerPairProviders(p, ctx)
	registerIfProvider(p, ctx)
	registerJSONLDProvider(p, ctx)
	p.SetCtx(ctx) // 用於 checkLabelLevel 權限檢查
}

func registerSingleProviders(p *TagParser, ctx *Context) {
	p.Register("site", func(tagName string, params map[string]string, inner string) string {
		if ctx.Site == nil {
			return ""
		}
		field := params["_field"]
		switch field {
		case "title":
			return ctx.Site.Title
		case "subtitle":
			return ctx.Site.Subtitle
		case "keywords":
			return ctx.Site.Keywords
		case "description":
			return ctx.Site.Description
		case "logo":
			logo := ctx.Site.Logo
			if logo != "" && !strings.HasPrefix(logo, "/") && !strings.HasPrefix(logo, "http") {
				logo = "/" + logo
			}
			return logo
		case "icp":
			return ctx.Site.ICP
		case "copyright":
			return ctx.Site.Copyright
		case "statistical":
			return ctx.Site.Statistical
		case "tplpath":
			theme := ctx.Site.Theme
			if theme == "" {
				theme = "default"
			}
			return "/template/" + theme + "/static"
		case "index":
			return "/"
		case "path":
			return "/"
		default:
			return ""
		}
	})

	// GBoot 兼容: {gboot:sitepath}, {gboot:sitetitle}, {gboot:sitetplpath} 等
	p.Register("gboot", func(tagName string, params map[string]string, inner string) string {
		if ctx.Site == nil {
			return ""
		}
		field := params["_field"]
		switch field {
		case "sitetitle":
			return ctx.Site.Title
		case "sitesubtitle":
			return ctx.Site.Subtitle
		case "sitekeywords":
			return ctx.Site.Keywords
		case "sitedescription":
			return ctx.Site.Description
		case "sitelogo":
			logo := ctx.Site.Logo
			if logo != "" && !strings.HasPrefix(logo, "/") && !strings.HasPrefix(logo, "http") {
				logo = "/" + logo
			}
			return logo
		case "siteicp":
			return ctx.Site.ICP
		case "sitecopyright":
			return ctx.Site.Copyright
		case "sitestatistical":
			return ctx.Site.Statistical
		case "sitetplpath":
			// 靜態資源（CSS/JS/圖片）所有語言共用 default 目錄，避免重複和同步問題
			return "/template/default/static"
		case "sitepath":
			return currentHomePath(ctx)
		case "pagetitle":
			return resolvePageTitle(ctx)
		case "pagekeywords":
			return resolvePageKeywords(ctx)
		case "pagedescription":
			return resolvePageDescription(ctx)
		// 會員相關標籤
		case "loginstatus":
			return model.GetConfigValue("login_status", "1")
		case "login":
			return "/login"
		case "registerstatus":
			return model.GetConfigValue("register_status", "1")
		case "register":
			return "/register"
		case "ucenter":
			return "/ucenter"
		case "umodify":
			return "/umodify"
		case "logout":
			return "/logout"
		case "retrieve":
			return "/retrieve"
		case "islogin":
			if ctx.Member != nil {
				return "1"
			}
			return "0"
		case "mustlogin":
			// 標籤本身回傳空字串（權限檢查在 controller 層的 checkMustLogin 處理）
			return ""
		case "commentstatus":
			return model.GetConfigValue("comment_status", "1")
		case "commentcodestatus":
			return model.GetConfigValue("comment_check_code", "1")
		case "msgcodestatus":
			return model.GetConfigValue("message_check_code", "1")
		case "msgturnstilestatus":
		return model.GetConfigValue("message_turnstile", "0")
	case "turnstile_sitekey":
		return model.GetConfigValue("turnstile_sitekey", "")
	case "httpurl":
		// 對齊 PbootCMS get_http_url(): 動態返回請求 URL，非配置值
		if ctx != nil && ctx.HttpURL != "" {
			return ctx.HttpURL
		}
		return model.GetConfigValue("httpurl", "/")
		// Company 字段路由: {gboot:companyname} → company.name
		case "companyname":
			if ctx.Company != nil {
				return ctx.Company.Name
			}
			return ""
		case "companyaddress":
			if ctx.Company != nil {
				return ctx.Company.Address
			}
			return ""
		case "companypostcode":
			if ctx.Company != nil {
				return ctx.Company.Postcode
			}
			return ""
		case "companycontact":
			if ctx.Company != nil {
				return ctx.Company.Contact
			}
			return ""
		case "companyphone":
			if ctx.Company != nil {
				return ctx.Company.Phone
			}
			return ""
		case "companymobile":
			if ctx.Company != nil {
				return ctx.Company.Mobile
			}
			return ""
		case "companyfax":
			if ctx.Company != nil {
				return ctx.Company.Fax
			}
			return ""
		case "companyemail":
			if ctx.Company != nil {
				return ctx.Company.Email
			}
			return ""
		case "companyqq":
			if ctx.Company != nil {
				return ctx.Company.Qq
			}
			return ""
		case "companyweixin":
			if ctx.Company != nil {
				return addLeadingSlash(ctx.Company.Weixin)
			}
			return ""
		case "companyicp":
			if ctx.Company != nil {
				return ctx.Company.ICP
			}
			return ""
		case "companyblicense":
			if ctx.Company != nil {
				return ctx.Company.Blicense
			}
			return ""
		case "companylegal":
			if ctx.Company != nil {
				return ctx.Company.Legal
			}
			return ""
		case "companybusiness":
			if ctx.Company != nil {
				return ctx.Company.Business
			}
			return ""
		case "companyother":
			if ctx.Company != nil {
				return ctx.Company.Other
			}
			return ""
		// Site info alias: {gboot:siteurl}
		case "siteurl":
			return "/"
		default:
			return ""
		}
	})

	p.Register("company", func(tagName string, params map[string]string, inner string) string {
		if ctx.Company == nil {
			return ""
		}
		field := params["_field"]
		switch field {
		case "name":
			return ctx.Company.Name
		case "address":
			return ctx.Company.Address
		case "postcode":
			return ctx.Company.Postcode
		case "contact":
			return ctx.Company.Contact
		case "phone":
			return ctx.Company.Phone
		case "mobile":
			return ctx.Company.Mobile
		case "fax":
			return ctx.Company.Fax
		case "email":
			return ctx.Company.Email
		case "qq":
			return ctx.Company.Qq
		case "weixin":
			return ctx.Company.Weixin
		case "icp":
			return ctx.Company.ICP
		case "blicense":
			return ctx.Company.Blicense
		case "legal":
			return ctx.Company.Legal
		case "business":
			return ctx.Company.Business
		case "other":
			return ctx.Company.Other
		default:
			return ""
		}
	})

	// QRCode 標籤: {gboot:qrcode string=xxx}
	p.Register("qrcode", func(tagName string, params map[string]string, inner string) string {
		str := params["string"]
		if str == "" {
			str = params["_field"]
		}
		if str == "" {
			return ""
		}
		// 返回一個簡單的 QR code 圖片（使用在線 API）
		// URL 編碼 str 防止 XSS（避免雙引號截斷 src 屬性）
		return fmt.Sprintf("<img src=\"https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=%s\" alt=\"QR Code\">", url.QueryEscape(str))
	})

	p.Register("sort", func(tagName string, params map[string]string, inner string) string {
		if ctx.Sort == nil {
			return ""
		}
		field := params["_field"]
		return getSortField(ctx.Ctx, ctx.Sort, field)
	})

	p.Register("content", func(tagName string, params map[string]string, inner string) string {
		if ctx.Content == nil {
			return ""
		}
		field := params["_field"]
		return getContentField(ctx, field, params)
	})

	p.Register("page", func(tagName string, params map[string]string, inner string) string {
		if ctx.Page == nil {
			return ""
		}
		field := params["_field"]
		if field == "numbar" {
			// 生成數字分頁條: [1] 2 3 4 5
			current := 1
			totalPages := 1
			total := 0
			if v, ok := ctx.Page["current"]; ok {
				current, _ = v.(int)
			} else if v, ok := ctx.Page["current_page"]; ok {
				current, _ = v.(int)
			}
			if v, ok := ctx.Page["count"]; ok {
				totalPages, _ = v.(int)
			} else if v, ok := ctx.Page["total_pages"]; ok {
				totalPages, _ = v.(int)
			}
			if v, ok := ctx.Page["rows"]; ok {
				total, _ = v.(int)
			} else if v, ok := ctx.Page["total"]; ok {
				total, _ = v.(int)
			}
			if total == 0 {
			return ""
		}
		// 讀取分頁數字條數量配置（對齊 PHP Paging::pageNumBar，預設 5）
		pageNum := parseIntConfig("pagenum", 5)
		var sb strings.Builder
		// 計算顯示範圍（對齊 PHP pageNumBar 邏輯：當前頁前後各顯示一半）
		halfL := pageNum / 2
		halfU := (pageNum + 1) / 2
		start := 1
		end := totalPages
		if totalPages > pageNum {
			if current > halfU {
				start = current - halfL
			}
			if current+halfL <= totalPages {
				end = current + halfU
			} else {
				end = totalPages
				start = totalPages - pageNum + 1
			}
			if start < 1 {
				start = 1
			}
			if end > totalPages {
				end = totalPages
			}
		}
		for i := start; i <= end; i++ {
			link := fmt.Sprintf("?page=%d", i)
			if baseP, ok := ctx.Page["basePath"]; ok {
				if bp, ok2 := baseP.(string); ok2 && bp != "" {
					link = bp + "page=" + strconv.Itoa(i)
				}
			}
			if i == current {
				sb.WriteString(fmt.Sprintf("<a class=\"page-num page-num-current\" href=\"%s\">%d</a>", link, i))
			} else {
				sb.WriteString(fmt.Sprintf("<a class=\"page-num\" href=\"%s\">%d</a>", link, i))
			}
		}
		return sb.String()
		}
		if val, ok := ctx.Page[field]; ok {
			return ValToStr(val)
		}
		return ""
	})

	p.Register("user", func(tagName string, params map[string]string, inner string) string {
		if ctx.Member == nil {
			return ""
		}
		field := params["_field"]
		switch field {
		case "username":
			return ctx.Member.Username
		case "nickname":
			return ctx.Member.Nickname
		case "headpic":
			return ctx.Member.Headpic
		case "email":
			return ctx.Member.Email
		case "useremail":
			return ctx.Member.Useremail
		case "usermobile":
			return ctx.Member.Usermobile
		case "ucode":
			return ctx.Member.Ucode
		case "uid", "id":
			return strconv.FormatUint(uint64(ctx.Member.ID), 10)
		case "gid":
			return ctx.Member.GID
		case "gcode":
			// 透過 gid 查 ay_member_group 取 gcode（對應 PHP getUser() JOIN）
			if ctx.Member.GID != "" && ctx.Member.GID != "0" {
				var group model.MemberGroup
				model.DB.WithContext(ctx.Ctx).Where("id = ?", ctx.Member.GID).First(&group)
				return group.Gcode
			}
			return ""
		case "score":
			return strconv.Itoa(ctx.Member.Score)
		case "logincount":
			return strconv.Itoa(ctx.Member.LoginCount)
		case "register_time":
			t := ctx.Member.RegisterTime
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02 15:04:05")
		case "last_login_time":
			return ctx.Member.LastLoginTime
		case "last_login_ip":
			return ctx.Member.LastLoginIP
		case "sex":
			return ctx.Member.Sex
		case "birthday":
			return ctx.Member.Birthday
		case "telephone":
			return ctx.Member.Telephone
		case "qq":
			return ctx.Member.QQ
		case "gname":
			if ctx.Member != nil && ctx.Member.GID != "" {
				var group model.MemberGroup
				model.DB.WithContext(ctx.Ctx).Where("id = ?", ctx.Member.GID).First(&group)
				return group.Gname
			}
			return ""
		case "registertime":
			t := ctx.Member.RegisterTime
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02 15:04:05")
		case "lastloginip":
			return ctx.Member.LastLoginIP
		case "lastlogintime":
			return ctx.Member.LastLoginTime
		default:
			return ""
		}
	})

	p.Register("label", func(tagName string, params map[string]string, inner string) string {
		field := params["_field"]
		var labels []model.Label
		model.DB.WithContext(ctx.Ctx).Where("name = ?", field).Find(&labels)
		if len(labels) > 0 {
			return labels[0].Value
		}
		return ""
	})

	p.Register("i18n", func(tagName string, params map[string]string, inner string) string {
		field := params["_field"]
		if field == "" {
			field = params["field"]
		}
		if field == "" {
			return ""
		}
		acode := acodeplugin.GetAcode(ctx.Ctx)
		if acode == "" {
			acode = "default"
		}
		if dict, ok := i18nUIDict[field]; ok {
			if val, vok := dict[acode]; vok && val != "" {
				return val
			}
			if val, vok := dict["default"]; vok && val != "" {
				return val
			}
			if val, vok := dict["tc"]; vok && val != "" {
				return val
			}
		}
		return field
	})

	// buildSortChain 從當前欄目向上遞歸到根欄目，返回從根到當前的有序鏈
	buildSortChain := func(scode string) []model.ContentSort {
		var chain []model.ContentSort
		current := scode
		for current != "" && current != "0" {
			var s model.ContentSort
			if err := model.DB.WithContext(ctx.Ctx).Where("scode = ?", current).First(&s).Error; err != nil {
				break
			}
			chain = append(chain, s)
			current = s.Pcode
		}
		for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
			chain[i], chain[j] = chain[j], chain[i]
		}
		return chain
	}

	// sortLink 生成欄目鏈接：filename > urlname > /sort/scode
	sortLink := func(s *model.ContentSort) string {
		if s.Filename != "" {
			return "/" + s.Filename
		}
		if s.URLName != "" {
			return "/" + s.URLName
		}
		return fmt.Sprintf("/sort/%s", s.Scode)
	}

	p.Register("position", func(tagName string, params map[string]string, inner string) string {
		sep := params["separator"]
		if sep == "" {
			sep = "/"
		}
		idxText := params["indextext"]
		if idxText == "" {
			idxText = "首頁"
		}
		parts := []string{fmt.Sprintf(`<a href="/">%s</a>`, idxText)}
		if ctx.Sort != nil && ctx.Sort.Name != "" {
			// 遍歷父級欄目鏈（當前欄目 → pcode 向上 → 根欄目）
			chain := buildSortChain(ctx.Sort.Scode)
			// 去重：根欄目可能和當前欄目相同
			seen := map[string]bool{}
			for _, s := range chain {
				if seen[s.Scode] {
					continue
				}
				seen[s.Scode] = true
				parts = append(parts, fmt.Sprintf(`<a href="%s">%s</a>`, sortLink(&s), s.Name))
			}
		}
		return strings.Join(parts, sep)
	})

	p.Register("pagetitle", func(tagName string, params map[string]string, inner string) string {
		return resolvePageTitle(ctx)
	})

	p.Register("pagekeywords", func(tagName string, params map[string]string, inner string) string {
		return resolvePageKeywords(ctx)
	})

	p.Register("pagedescription", func(tagName string, params map[string]string, inner string) string {
		return resolvePageDescription(ctx)
	})

	p.Register("httpurl", func(tagName string, params map[string]string, inner string) string {
		// 對齊 PbootCMS get_http_url(): 動態返回請求 URL
		if ctx != nil && ctx.HttpURL != "" {
			return ctx.HttpURL
		}
		return model.GetConfigValue("httpurl", "/")
	})

	p.Register("pageurl", func(tagName string, params map[string]string, inner string) string {
		if ctx.Content != nil && ctx.Content.URLName != "" {
			return "/" + ctx.Content.URLName + ".html"
		}
		return "/"
	})

	p.Register("islogin", func(tagName string, params map[string]string, inner string) string {
		if ctx.Member != nil {
			return "1"
		}
		return "0"
	})

	p.Register("checkcode", func(tagName string, params map[string]string, inner string) string {
		return "/api/checkcode"
	})

	p.Register("msgaction", func(tagName string, params map[string]string, inner string) string {
		return "/message"
	})

	p.Register("scaction", func(tagName string, params map[string]string, inner string) string {
		return "/search"
	})

	// 時間戳：用於前端時間陷阱反垃圾
	p.Register("timestamp", func(tagName string, params map[string]string, inner string) string {
		return fmt.Sprintf("%d", time.Now().Unix())
	})

	// Form provider: {gboot:form fcode=X} → 表單提交 URL
	p.Register("form", func(tagName string, params map[string]string, inner string) string {
		fcode := params["fcode"]
		if fcode != "" {
			return "/message?fcode=" + fcode
		}
		return "/message"
	})

	p.Register("keyword", func(tagName string, params map[string]string, inner string) string {
		return html.EscapeString(ctx.Keyword)
	})

	p.Register("commentaction", func(tagName string, params map[string]string, inner string) string {
		// 帶上 contentid 參數（與 PHP 原版一致）
		if ctx.Content != nil {
			return "/comment/add?contentid=" + strconv.FormatUint(uint64(ctx.Content.ID), 10)
		}
		return "/comment/add"
	})

	p.Register("lgpath", func(tagName string, params map[string]string, inner string) string {
		return "/home/index/area"
	})

	// acode — 當前語言區域代碼（用於模板條件判斷）
	p.Register("acode", func(tagName string, params map[string]string, inner string) string {
		return acodeplugin.GetAcode(ctx.Ctx)
	})

	// homename — 導航菜單首項文字（本地化「首頁」），站點標題請用 {gboot:sitetitle}
	p.Register("homename", func(tagName string, params map[string]string, inner string) string {
		acode := acodeplugin.GetAcode(ctx.Ctx)
		switch acode {
		case "sc":
			return "首页"
		case "en":
			return "Home"
		default:
			return "首頁"
		}
	})

	// morename — 當前語言的「查看更多」文字
	p.Register("morename", func(tagName string, params map[string]string, inner string) string {
		acode := acodeplugin.GetAcode(ctx.Ctx)
		switch acode {
		case "sc":
			return "查看更多"
		case "en":
			return "View More"
		default:
			return "查看更多"
		}
	})

	// 會員相關 URL 標籤
	p.Register("login", func(tagName string, params map[string]string, inner string) string {
		return "/login"
	})
	p.Register("register", func(tagName string, params map[string]string, inner string) string {
		return "/register"
	})
	p.Register("logout", func(tagName string, params map[string]string, inner string) string {
		return "/logout"
	})
	p.Register("ucenter", func(tagName string, params map[string]string, inner string) string {
		return "/ucenter"
	})
	p.Register("umodify", func(tagName string, params map[string]string, inner string) string {
		return "/umodify"
	})
	p.Register("retrieve", func(tagName string, params map[string]string, inner string) string {
		return "/retrieve"
	})

	// 驗證碼狀態標籤（返回 "1" 或 "0"）
	p.Register("logincodestatus", func(tagName string, params map[string]string, inner string) string {
		if model.GetConfigValue("login_check_code", "1") != "0" {
			return "1"
		}
		return "0"
	})
	p.Register("registercodestatus", func(tagName string, params map[string]string, inner string) string {
		if model.GetConfigValue("register_check_code", "1") != "0" {
			return "1"
		}
		return "0"
	})
}

func registerPairProviders(p *TagParser, ctx *Context) {
	p.Register("list", func(tagName string, params map[string]string, inner string) string {
		scode := params["scode"]
		if scode == "" && ctx.Sort != nil {
			scode = ctx.Sort.Scode
		}
		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		order := params["order"]
		if order == "" {
			order = "date"
		}

		query := model.DB.WithContext(ctx.Ctx).Where("status = 1 AND date <= ?", time.Now())
		if scode != "" {
			// 遞歸查找當前欄目及其所有子欄目的 scode
			childScodes := findAllChildScodes(ctx.Ctx, scode)
			query = query.Where("scode IN ?", childScodes)
		}
		// 排序優先級（對齊 PbootCMS ParserController 默認排序邏輯）
		// istop > isrecommend > isheadline > sorting > date > id
		switch order {
		case "date":
			query = query.Order("istop DESC, date DESC, isrecommend DESC, isheadline DESC, sorting ASC, id DESC")
		case "sorting":
			query = query.Order("sorting ASC, istop DESC, isrecommend DESC, isheadline DESC, date DESC, id DESC")
		case "istop":
			query = query.Order("istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "isrecommend":
			query = query.Order("isrecommend DESC, istop DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "isheadline":
			query = query.Order("isrecommend DESC, istop DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "visits":
			query = query.Order(order+" DESC, istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "id":
			query = query.Order("id DESC, istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC")
		case "random":
			query = query.Order("RANDOM()")
		default:
			// 自定義欄位排序
			query = query.Order(order)
		}

		// Pagination support
		pageEnabled := params["page"] == "1"
		var total int64
		currentPage := 1

		// ext_ 篩選：使用 JOIN 方式過濾（比子查詢在 GORM 中更可靠）
		if len(ctx.Filters) > 0 {
			query = query.Joins("JOIN ay_content_ext ON ay_content.id = ay_content_ext.contentid")
			for field, value := range ctx.Filters {
				if !IsSafeFieldName(field) {
					continue // 跳過不安全的字段名，防止 SQL 注入
				}
				query = query.Where("ay_content_ext."+field+" LIKE ?", "%"+value+"%")
			}
		}

		// 先取數，再獨立取總數（避免 GORM Count 污染查詢狀態）
		var contents []model.Content
		if pageEnabled {
			if ctx.CurrentPage > 0 {
				currentPage = ctx.CurrentPage
			}
			offset := (currentPage - 1) * num
			query.Offset(offset).Limit(num).Find(&contents)

			// 獨立查詢取總記錄數（含 scode 過濾 + ext_ 篩選）
			countQuery := model.DB.WithContext(ctx.Ctx).Model(&model.Content{}).
				Where("status = 1 AND date <= ?", time.Now())
			if acode := acodeplugin.GetAcode(ctx.Ctx); acode != "" {
				countQuery = countQuery.Where("acode = ?", acode)
			}
			if scode != "" {
				countQuery = countQuery.Where("scode IN (?)", findAllChildScodes(ctx.Ctx, scode))
			}
			if len(ctx.Filters) > 0 {
			countQuery = countQuery.Joins("JOIN ay_content_ext ON ay_content.id = ay_content_ext.contentid")
			for field, value := range ctx.Filters {
				if !IsSafeFieldName(field) {
					continue
				}
				countQuery = countQuery.Where("ay_content_ext."+field+" LIKE ?", "%"+value+"%")
			}
		}
			countQuery.Count(&total)
		} else {
			query.Limit(num).Find(&contents)
			// 獨立查詢取總記錄數（含 scode 過濾 + ext_ 篩選）
			countQuery := model.DB.WithContext(ctx.Ctx).Model(&model.Content{}).
				Where("status = 1 AND date <= ?", time.Now())
			if acode := acodeplugin.GetAcode(ctx.Ctx); acode != "" {
				countQuery = countQuery.Where("acode = ?", acode)
			}
			if scode != "" {
				countQuery = countQuery.Where("scode IN (?)", findAllChildScodes(ctx.Ctx, scode))
			}
			if len(ctx.Filters) > 0 {
				countQuery = countQuery.Joins("JOIN ay_content_ext ON ay_content.id = ay_content_ext.contentid")
				for field, value := range ctx.Filters {
					if !IsSafeFieldName(field) {
						continue
					}
					countQuery = countQuery.Where("ay_content_ext."+field+" LIKE ?", "%"+value+"%")
				}
			}
			countQuery.Count(&total)
		}

		// 始終設置分頁資訊（page:rows / page:count 等），供 {gboot:if({page:rows}>0)} 判斷
		totalPages := int(total) / num
		if int(total)%num > 0 {
			totalPages++
		}
		if totalPages < 1 {
			totalPages = 1
		}

		basePath := "?"
		if ctx.Sort != nil {
			if ctx.Sort.Filename != "" {
				basePath = "/" + ctx.Sort.Filename + "/?"
			} else if ctx.Sort.URLName != "" {
				basePath = "/" + ctx.Sort.URLName + "/?"
			}
		}
		// 保留 ext_ 篩選參數，使分頁鏈接攜帶當前篩選條件（驗證字段名安全）
		for k, v := range ctx.Filters {
			if !IsSafeFieldName(k) {
				continue
			}
			basePath += k + "=" + urlEncode(v) + "&"
		}

		ctx.Page["current"] = currentPage
		ctx.Page["count"] = totalPages
		ctx.Page["rows"] = int(total)
		ctx.Page["basePath"] = basePath
		ctx.Page["index"] = basePath + "page=1"
		if currentPage > 1 {
			ctx.Page["pre"] = fmt.Sprintf("%spage=%d", basePath, currentPage-1)
		} else {
			ctx.Page["pre"] = ""
		}
		if currentPage < totalPages {
			ctx.Page["next"] = fmt.Sprintf("%spage=%d", basePath, currentPage+1)
		} else {
			ctx.Page["next"] = ""
		}
		ctx.Page["last"] = fmt.Sprintf("%spage=%d", basePath, totalPages)

		var sb strings.Builder
		// 批量預載入擴展字段，避免 N+1 查詢
		contentIDs := make([]uint, len(contents))
		for i, c := range contents {
			contentIDs[i] = c.ID
		}
		extMap := content.GetContentExtByContentIDs(contentIDs)
		// 批量預載入欄目路徑，避免 contentURL 的 N+1 查詢
		preloadSortPaths(ctx, contents)
		for i, c := range contents {
			data := contentToMap(ctx, &c, i, extMap)
			row := ReplaceInnerTags(inner, "list", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("nav", func(tagName string, params map[string]string, inner string) string {
		parent := params["parent"]
		num := 0
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}

		var sorts []model.ContentSort
		query := model.DB.WithContext(ctx.Ctx).Where("status = 1")
		if parent != "" {
			query = query.Where("pcode = ?", parent)
		} else {
			query = query.Where("pcode = '' OR pcode = '0'")
		}
		query = query.Order("sorting ASC, id ASC")
		if num > 0 {
			query = query.Limit(num)
		}
		query.Find(&sorts)

		// 批量預載入欄目內容數量，避免 N+1 查詢
		countMap := buildSortCountMap(ctx.Ctx)
		var sb strings.Builder
		for i, s := range sorts {
			data := sortToMap(ctx.Ctx, &s, i, countMap)
			row := ReplaceInnerTags(inner, "nav", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("sort_loop", func(tagName string, params map[string]string, inner string) string {
		scode := params["scode"]
		if scode == "" {
			return ""
		}
		var sorts []model.ContentSort
		model.DB.WithContext(ctx.Ctx).Where("scode IN (?)", strings.Split(scode, ",")).Order("sorting ASC").Find(&sorts)

		// 批量預載入欄目內容數量，避免 N+1 查詢
		countMap := buildSortCountMap(ctx.Ctx)
		var sb strings.Builder
		for i, s := range sorts {
			data := sortToMap(ctx.Ctx, &s, i, countMap)
			row := ReplaceInnerTags(inner, "sort", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("content_loop", func(tagName string, params map[string]string, inner string) string {
		id := params["id"]
		scode := params["scode"]
		if id == "" && scode == "" {
			return ""
		}
		var c model.Content
		query := model.DB.WithContext(ctx.Ctx).Where("status = 1 AND date <= ?", time.Now())
		if id != "" {
			query = query.Where("id = ?", id)
		}
		if scode != "" {
			query = query.Where("scode = ?", scode)
		}
		if query.First(&c).Error != nil {
			return ""
		}

		data := contentToMap(ctx, &c, 0, nil)
		return ReplaceInnerTags(inner, "content", data)
	})

	p.Register("slide", func(tagName string, params map[string]string, inner string) string {
		gid := params["gid"]
		num := 5
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		var slides []model.Slide
		query := model.DB.WithContext(ctx.Ctx).Order("gid ASC, sorting ASC, id ASC")
		if gid != "" {
			query = query.Where("gid = ?", gid)
		}
		query.Limit(num).Find(&slides)

		var sb strings.Builder
		for i, s := range slides {
			// 圖片路徑確保以 / 開頭
			src := s.Pic
			if src != "" && !strings.HasPrefix(src, "/") && !strings.HasPrefix(src, "http") {
				src = "/" + src
			}
			pic := s.Pic
			if pic != "" && !strings.HasPrefix(pic, "/") && !strings.HasPrefix(pic, "http") {
				pic = "/" + pic
			}
			picmobile := s.PicMobile
			if picmobile != "" && !strings.HasPrefix(picmobile, "/") && !strings.HasPrefix(picmobile, "http") {
				picmobile = "/" + picmobile
			}
			data := map[string]interface{}{
				"n":           i,
				"i":           i + 1,
				"src":         src,
				"pic":         pic,
				"picmobile":   picmobile,
				"link":        s.Link,
				"title":       s.Title,
				"subtitle":    s.Subtitle,
				"button_text": s.ButtonText,
			}
			row := ReplaceInnerTags(inner, "slide", data)
			// 處理內嵌的 {gboot:if(...)} 標籤（在 slide 數據上下文中）
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("link", func(tagName string, params map[string]string, inner string) string {
		gid := params["gid"]
		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		var links []model.Link
		query := model.DB.WithContext(ctx.Ctx).Order("sorting ASC")
		if gid != "" {
			query = query.Where("gid = ?", gid)
		}
		query.Limit(num).Find(&links)

		var sb strings.Builder
		for i, l := range links {
			logo := l.Logo
			if logo != "" && !strings.HasPrefix(logo, "/") && !strings.HasPrefix(logo, "http") {
				logo = "/" + logo
			}
			data := map[string]interface{}{
				"n":   i,
				"i":   i + 1,
				"logo": logo,
				"link": l.Link,
				"title": l.Name,
			}
			row := ReplaceInnerTags(inner, "link", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	// language — 多語言切換標籤
	// 用法：{gboot:language}[language:name]{/gboot:language}
	// 切換時保持當前頁面路徑（如 /sc/article → /tc/article → /en/article）
	p.Register("language", func(tagName string, params map[string]string, inner string) string {
		areas := model.GetCachedAreas()
		if len(areas) <= 1 {
			return ""
		}

		// 當前 acode
		currentAcode := acodeplugin.GetAcode(ctx.Ctx)

		// 找默認區域
		defaultAcode := ""
		for _, a := range areas {
			if a.IsDefault == "1" {
				defaultAcode = a.Acode
				break
			}
		}
		if defaultAcode == "" && len(areas) > 0 {
			defaultAcode = areas[0].Acode
		}

		// 當前頁面路徑（已剝離 acode 前綴），確保首頁為 /
		currentPath := ctx.CurrentPath
		if currentPath == "" {
			currentPath = "/"
		}
		if !strings.HasPrefix(currentPath, "/") {
			currentPath = "/" + currentPath
		}

		var sb strings.Builder
		for i, a := range areas {
			// 構建保持當前頁面的語言切換連結
			// 默認區域: /{currentPath}，非默認: /{acode}{currentPath}
			link := currentPath
			if a.Acode != defaultAcode {
				link = "/" + a.Acode + currentPath
			}
			data := map[string]interface{}{
				"i":      i + 1,
				"n":      i,
				"acode":  a.Acode,
				"name":   a.Name,
				"link":   link,
				"active": a.Acode == currentAcode,
			}
			row := ReplaceInnerTags(inner, "language", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("loop", func(tagName string, params map[string]string, inner string) string {
		start, _ := strconv.Atoi(params["start"])
		end, _ := strconv.Atoi(params["end"])
		if end <= start {
			return ""
		}

		var sb strings.Builder
		for i := start; i <= end; i++ {
			data := map[string]interface{}{
				"n":     i - start,
				"i":     i - start + 1,
				"index": i,
			}
			row := ReplaceInnerTags(inner, "loop", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("tags", func(tagName string, params map[string]string, inner string) string {
		num := 0
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		var contents []model.Content
		// 限制最多載入 2000 條記錄，防止大型站點 OOM
		model.DB.WithContext(ctx.Ctx).Where("status = 1 AND keywords != '' AND date <= ?", time.Now()).Order("id DESC").Limit(2000).Find(&contents)
		tagSet := map[string]bool{}
		var tagList []string
		for _, c := range contents {
			for _, kw := range strings.Split(c.Keywords, ",") {
				kw = strings.TrimSpace(kw)
				if kw != "" && !tagSet[kw] {
					tagSet[kw] = true
					tagList = append(tagList, kw)
				}
			}
		}
		if num > 0 && num < len(tagList) {
			tagList = tagList[:num]
		}
		var sb strings.Builder
		for i, tag := range tagList {
			data := map[string]interface{}{
				"n":    i,
				"i":    i + 1,
				"tag":  tag,
				"text": tag,
				"link": "/search?keyword=" + tag,
			}
			row := ReplaceInnerTags(inner, "tags", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	// hreflang — SEO 多語言替代連結標籤（自引用 + 雙向對稱 + x-default）
	// 用法：在 <head> 中放置 {gboot:hreflang}
	// 生成：<link rel="alternate" hreflang="zh-Hant" href="..." />
	p.Register("hreflang", func(tagName string, params map[string]string, inner string) string {
		return buildHreflang(ctx)
	})

	// canonical — SEO 標準連結標籤，指向當前頁面的標準 URL
	// 用法：在 <head> 中放置 {gboot:canonical}
	p.Register("canonical", func(tagName string, params map[string]string, inner string) string {
		return buildCanonical(ctx)
	})

	// htmllang — 當前語言的 HTML lang 屬性值（zh-Hans / zh-Hant / en）
	// 用法：<html lang="{gboot:htmllang}">
	p.Register("htmllang", func(tagName string, params map[string]string, inner string) string {
		return acodeToHreflang(acodeplugin.GetAcode(ctx.Ctx))
	})

	// og — Open Graph 社交分享 meta 標籤
	// 用法：在 <head> 中放置 {gboot:og}
	p.Register("og", func(tagName string, params map[string]string, inner string) string {
		return buildOpenGraph(ctx)
	})

	p.Register("pics", func(tagName string, params map[string]string, inner string) string {
		if ctx.Content == nil || ctx.Content.Pics == "" {
			return ""
		}
		picList := strings.Split(ctx.Content.Pics, ",")
		var sb strings.Builder
		for i, pic := range picList {
			pic = strings.TrimSpace(pic)
			if pic == "" {
				continue
			}
			// 確保路徑以 / 開頭
			if !strings.HasPrefix(pic, "/") {
				pic = "/" + pic
			}
			data := map[string]interface{}{
				"n":   i,
				"i":   i + 1,
				"src": pic,
			}
			row := ReplaceInnerTags(inner, "pics", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("checkbox", func(tagName string, params map[string]string, inner string) string {
		name := params["name"]
		value := params["value"]
		if name == "" || value == "" {
			return ""
		}
		checked := ""
		if ctx.Content != nil {
			// Get field value by name from content
			fieldVal := getContentField(ctx, name, nil)
			if fieldVal != "" {
				for _, v := range strings.Split(fieldVal, ",") {
					if strings.TrimSpace(v) == value {
						checked = " checked"
						break
					}
				}
			}
		}
		return fmt.Sprintf(`<input type="checkbox" name="%s" value="%s"%s>`, name, value, checked)
	})

	p.Register("message", func(tagName string, params map[string]string, inner string) string {
		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		var messages []model.Message
		model.DB.WithContext(ctx.Ctx).Where("status = 1").Order("id DESC").Limit(num).Find(&messages)

		// 批量查會員信息（LEFT JOIN ay_member），匹配 PHP 原版 getMessage()
		uidSet := map[int]bool{}
		for _, m := range messages {
			if m.UID > 0 {
				uidSet[m.UID] = true
			}
		}
		type memberInfo struct {
			Nickname string
			HeadPic  string
		}
		memberMap := map[int]memberInfo{}
		if len(uidSet) > 0 {
			uids := make([]int, 0, len(uidSet))
			for uid := range uidSet {
				uids = append(uids, uid)
			}
			var members []model.Member
			model.DB.WithContext(ctx.Ctx).Where("id IN ?", uids).Find(&members)
			for _, mem := range members {
				memberMap[int(mem.ID)] = memberInfo{Nickname: mem.Nickname, HeadPic: mem.Headpic}
			}
		}

		var sb strings.Builder
		for i, m := range messages {
			// nickname: 有會員暱稱用暱稱，無會員顯示"匿名用戶"（PHP 原版邏輯）
			nickname := "匿名用戶"
			headpic := "/static/admin/images/logo.png"
			if m.UID > 0 {
				if mi, ok := memberMap[m.UID]; ok {
					if mi.Nickname != "" {
						nickname = mi.Nickname
					}
					if mi.HeadPic != "" {
						headpic = mi.HeadPic
					}
				}
			}
			// replydate: 零值時間顯示為空（避免顯示 0001-01-01）
			replyDate := ""
			if !m.UpdateTime.IsZero() {
				replyDate = m.UpdateTime.Format("2006-01-02")
			}
			data := map[string]interface{}{
				"n":          i,
				"i":          i + 1,
				"contacts":   m.Contacts,
				"mobile":     m.Mobile,
				"content":    m.Content,
				"create_time": m.CreateTime.Format("2006-01-02"),
				"askdate":    m.CreateTime.Format("2006-01-02"),
				"status":     m.Status,
				"recontent":  m.ReContent,
				"update_time": m.UpdateTime.Format("2006-01-02"),
				"replydate":  replyDate,
				"os":         m.OS,
				"bs":         m.Browser,
				"ip":         m.IP,
				"nickname":   nickname,
				"headpic":    headpic,
			}
			row := ReplaceInnerTags(inner, "message", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("formlist", func(tagName string, params map[string]string, inner string) string {
		fcode := params["fcode"]
		if fcode == "" {
			return ""
		}
		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		// 查 ay_form 獲取 table_name
		tableName := content.GetFormTableByCode(fcode)
		if tableName == "" {
			return ""
		}
		// 查動態表數據
		var rows []map[string]interface{}
		// SQL 注入防護：驗證表名（表名從 DB 讀取，但額外驗證以防資料被篡改）
		if !safeTableRe.MatchString(tableName) {
			return ""
		}
		model.DB.WithContext(ctx.Ctx).Raw("SELECT * FROM `" + tableName + "` ORDER BY id DESC LIMIT " + strconv.Itoa(num)).Scan(&rows)
		if len(rows) == 0 {
			return ""
		}
		var sb strings.Builder
		for i, row := range rows {
			data := make(map[string]interface{})
			for k, v := range row {
				data[k] = fmt.Sprintf("%v", v)
			}
			data["i"] = strconv.Itoa(i + 1)
			data["date"] = data["create_time"]
			rowHTML := ReplaceInnerTags(inner, "form", data)
			rowHTML = processInnerIfTags(rowHTML)
			sb.WriteString(rowHTML)
		}
		return sb.String()
	})

	p.Register("search", func(tagName string, params map[string]string, inner string) string {
		keyword := ctx.Keyword
		if keyword == "" {
			return ""
		}
		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}
		like := "%" + keyword + "%"
		query := model.DB.WithContext(ctx.Ctx).Where("status = 1 AND date <= ? AND (title LIKE ? OR keywords LIKE ? OR description LIKE ?)", time.Now(), like, like, like)

		// scode 過濾（可選）
		scode := params["scode"]
		if scode != "" {
			query = query.Where("scode IN ?", findAllChildScodes(ctx.Ctx, scode))
		}

		order := params["order"]
		if order == "" {
			order = "date"
		}
		// 排序優先級（對齊 PbootCMS ParserController 默認排序邏輯）
		switch order {
		case "date":
			query = query.Order("istop DESC, date DESC, isrecommend DESC, isheadline DESC, sorting ASC, id DESC")
		case "sorting":
			query = query.Order("sorting ASC, istop DESC, isrecommend DESC, isheadline DESC, date DESC, id DESC")
		case "istop":
			query = query.Order("istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "isrecommend":
			query = query.Order("isrecommend DESC, istop DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "isheadline":
			query = query.Order("isrecommend DESC, istop DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "visits":
			query = query.Order(order+" DESC, istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC, id DESC")
		case "id":
			query = query.Order("id DESC, istop DESC, isrecommend DESC, isheadline DESC, sorting ASC, date DESC")
		default:
			query = query.Order("istop DESC, date DESC, isrecommend DESC, isheadline DESC, sorting ASC, id DESC")
		}

		// 分頁支援
		pageEnabled := params["page"] == "1"
		var total int64
		currentPage := 1

		var contents []model.Content
		if pageEnabled {
			if ctx.CurrentPage > 0 {
				currentPage = ctx.CurrentPage
			}
			offset := (currentPage - 1) * num
			query.Offset(offset).Limit(num).Find(&contents)

			countQuery := model.DB.WithContext(ctx.Ctx).Model(&model.Content{}).
			Where("status = 1 AND date <= ? AND (title LIKE ? OR keywords LIKE ? OR description LIKE ?)", time.Now(), like, like, like)
		if acode := acodeplugin.GetAcode(ctx.Ctx); acode != "" {
			countQuery = countQuery.Where("acode = ?", acode)
		}
		if scode != "" {
			countQuery = countQuery.Where("scode IN ?", findAllChildScodes(ctx.Ctx, scode))
		}
		countQuery.Count(&total)
	} else {
		query.Limit(num).Find(&contents)
		countQuery := model.DB.WithContext(ctx.Ctx).Model(&model.Content{}).
			Where("status = 1 AND date <= ? AND (title LIKE ? OR keywords LIKE ? OR description LIKE ?)", time.Now(), like, like, like)
		if acode := acodeplugin.GetAcode(ctx.Ctx); acode != "" {
			countQuery = countQuery.Where("acode = ?", acode)
		}
			if scode != "" {
				countQuery = countQuery.Where("scode IN ?", findAllChildScodes(ctx.Ctx, scode))
			}
			countQuery.Count(&total)
		}

		// 設置分頁資訊
		totalPages := int(total) / num
		if int(total)%num > 0 {
			totalPages++
		}
		if totalPages < 1 {
			totalPages = 1
		}

		basePath := "/search?keyword=" + urlEncode(keyword) + "&"
		ctx.Page["current"] = currentPage
		ctx.Page["count"] = totalPages
		ctx.Page["rows"] = int(total)
		ctx.Page["basePath"] = basePath
		ctx.Page["index"] = basePath + "page=1"
		if currentPage > 1 {
			ctx.Page["pre"] = fmt.Sprintf("%spage=%d", basePath, currentPage-1)
		} else {
			ctx.Page["pre"] = ""
		}
		if currentPage < totalPages {
			ctx.Page["next"] = fmt.Sprintf("%spage=%d", basePath, currentPage+1)
		} else {
			ctx.Page["next"] = ""
		}
		ctx.Page["last"] = fmt.Sprintf("%spage=%d", basePath, totalPages)

		var sb strings.Builder
		// 批量預載入擴展字段，避免 N+1 查詢
		contentIDs := make([]uint, len(contents))
		for i, c := range contents {
			contentIDs[i] = c.ID
		}
		extMap := content.GetContentExtByContentIDs(contentIDs)
		// 批量預載入欄目路徑，避免 contentURL 的 N+1 查詢
		preloadSortPaths(ctx, contents)
		for i, c := range contents {
			data := contentToMap(ctx, &c, i, extMap)
			row := ReplaceInnerTags(inner, "search", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("comment", func(tagName string, params map[string]string, inner string) string {
		// 取得 contentid：優先用 params，其次用 ctx.Content
		contentid := params["contentid"]
		cid, _ := strconv.Atoi(contentid)
		if cid == 0 && ctx.Content != nil {
			cid = int(ctx.Content.ID)
		}
		if cid == 0 {
			return ""
		}

		num := 20
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}

		// 查詢主評論（pid=0, status=1）
		var comments []model.CommentView
		model.DB.WithContext(ctx.Ctx).Table("ay_member_comment a").
			Select("a.*, b.username, b.nickname, b.headpic, c.username as pusername, c.nickname as pnickname, c.headpic as pheadpic").
			Joins("LEFT JOIN ay_member b ON a.uid=b.id").
			Joins("LEFT JOIN ay_member c ON a.puid=c.id").
			Where("a.contentid = ? AND a.pid = 0 AND a.status = 1", cid).
			Order("a.id DESC").
			Limit(num).
			Find(&comments)

		// 提取 commentsub 塊的內容
		subMatch := commentSubRe.FindStringSubmatch(inner)
		subInner := ""
		if len(subMatch) >= 3 {
			subInner = subMatch[2]
		}
		// 移除 inner 中的 commentsub 塊（避免被外層重複渲染）
		innerWithoutSub := commentSubRe.ReplaceAllString(inner, "{__COMMENTSUB__}")

		var sb strings.Builder

		// 批量預載入所有子評論，避免 N+1 查詢（20 條主評論 = 原本 21 次 DB 查詢 → 現在 2 次）
		subMap := map[uint][]model.CommentView{}
		if subInner != "" && len(comments) > 0 {
			parentIDs := make([]uint, len(comments))
			for i, cm := range comments {
				parentIDs[i] = cm.ID
			}
			var allSubs []model.CommentView
			model.DB.WithContext(ctx.Ctx).Table("ay_member_comment a").
				Select("a.*, b.username, b.nickname, b.headpic, c.username as pusername, c.nickname as pnickname, c.headpic as pheadpic").
				Joins("LEFT JOIN ay_member b ON a.uid=b.id").
				Joins("LEFT JOIN ay_member c ON a.puid=c.id").
				Where("a.contentid = ? AND a.pid IN ? AND a.status = 1", cid, parentIDs).
				Order("a.pid ASC, a.id ASC").
				Limit(100*len(comments)).
				Find(&allSubs)
			for i := range allSubs {
				subMap[allSubs[i].Pid] = append(subMap[allSubs[i].Pid], allSubs[i])
			}
		}

		for i, cm := range comments {
			data := commentToMap(&cm, i+1)
			row := ReplaceInnerTags(innerWithoutSub, "comment", data)
			// 處理 commentsub 塊（從預載入的 subMap 中取，無需再查 DB）
			if subInner != "" {
				subs := subMap[cm.ID]

				var subSB strings.Builder
				for j, sc := range subs {
					subData := commentToMap(&sc, j+1)
					subRow := ReplaceInnerTags(subInner, "commentsub", subData)
					subRow = processInnerIfTags(subRow)
					subSB.WriteString(subRow)
				}
				row = strings.Replace(row, "{__COMMENTSUB__}", subSB.String(), 1)
			} else {
				row = strings.Replace(row, "{__COMMENTSUB__}", "", 1)
			}
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	p.Register("commentsub", func(tagName string, params map[string]string, inner string) string {
		// commentsub 由 comment provider 內部處理，此處為兜底返回空
		return ""
	})

	p.Register("mycomment", func(tagName string, params map[string]string, inner string) string {
		// 需要從 session 取得 uid，但 provider 無法直接存取 gin.Context
		// 透過 ctx.Member 取得（renderMemberPage 時已設定）
		if ctx.Member == nil {
			return ""
		}
		uid := ctx.Member.ID
		if uid == 0 {
			return ""
		}

		num := 10
		if n, err := strconv.Atoi(params["num"]); err == nil && n > 0 {
			num = n
		}

		var comments []model.CommentView
		model.DB.WithContext(ctx.Ctx).Table("ay_member_comment a").
			Select("a.*, b.username, b.nickname, b.headpic, c.username as pusername, c.nickname as pnickname, c.headpic as pheadpic, d.title").
			Joins("LEFT JOIN ay_member b ON a.uid=b.id").
			Joins("LEFT JOIN ay_member c ON a.puid=c.id").
			Joins("LEFT JOIN ay_content d ON a.contentid=d.id").
			Where("a.uid = ?", uid).
			Order("a.id DESC").
			Limit(num).
			Find(&comments)

		var sb strings.Builder
		for i, cm := range comments {
			data := commentToMap(&cm, i+1)
			data["delaction"] = "/comment/del?id=" + strconv.FormatUint(uint64(cm.ID), 10)
			data["title"] = cm.Title
			data["status"] = strconv.Itoa(cm.Status)
			if !cm.CreateTime.IsZero() {
				data["date"] = cm.CreateTime.Format("2006-01-02 15:04:05")
			} else {
				data["date"] = ""
			}
			row := ReplaceInnerTags(inner, "mycomment", data)
			row = processInnerIfTags(row)
			sb.WriteString(row)
		}
		return sb.String()
	})

	// selectall: 生成「全部」連結（清除該欄位篩選）
	p.Register("selectall", func(tagName string, params map[string]string, inner string) string {
		field := params["field"]
		if field == "" {
			return ""
		}
		text := params["text"]
		if text == "" {
			text = "全部"
		}
		class := params["class"]
		active := params["active"]
		if active == "" {
			active = class
		}
		// 當前是否有該欄位的篩選
		currentVal := ctx.Filters[field]
		cssClass := class
		if currentVal == "" {
			cssClass = active
		}
		// 生成清除該欄位的 URL（保留其他篩選參數）
		link := buildFilterURL(ctx, field, "")
		return fmt.Sprintf(`<a href="%s" class="%s">%s</a>`, link, cssClass, text)
	})

	// select: 遍歷擴展欄位的選項，渲染篩選按鈕
	p.Register("select", func(tagName string, params map[string]string, inner string) string {
		field := params["field"]
		if field == "" || inner == "" {
			return ""
		}
		// 查 ay_extfield 取選項（表名 ay_extfield，非 GORM 默認推斷）
		var ef content.ExtField
		if err := model.DB.WithContext(ctx.Ctx).Raw("SELECT * FROM ay_extfield WHERE field = ? LIMIT 1", field).Scan(&ef).Error; err != nil {
			return ""
		}
		if ef.Value == "" {
			return ""
		}
		options := strings.Split(ef.Value, ",")
		currentVal := ctx.Filters[field]
		var buf strings.Builder
		for _, opt := range options {
			opt = strings.TrimSpace(opt)
			if opt == "" {
				continue
			}
			link := buildFilterURL(ctx, field, opt)
			// 替換 inner 中的 [select:link] [select:value] [select:current]
			row := inner
			row = strings.ReplaceAll(row, "[select:link]", link)
			row = strings.ReplaceAll(row, "[select:value]", opt)
			row = strings.ReplaceAll(row, "[select:current]", currentVal)
			buf.WriteString(row)
		}
		return buf.String()
	})
}

// processInnerIfTags 處理循環體內的 {gboot:if(...)} 標籤
// 使用正則匹配，避免 processIfTags 的字符串索引 bug
func processInnerIfTags(content string) string {
	for innerIfRe.MatchString(content) {
		content = innerIfRe.ReplaceAllStringFunc(content, func(match string) string {
			subs := innerIfRe.FindStringSubmatch(match)
			if len(subs) < 3 {
				return ""
			}
			cond := strings.TrimSpace(subs[1])
			trueBody := subs[2]
			falseBody := ""
			if len(subs) > 3 {
				falseBody = subs[3]
			}
			// 簡單條件求值
			if evalInnerCondition(cond) {
				return trueBody
			}
			return falseBody
		})
	}
	return content
}

// evalInnerCondition 在循環上下文中求值簡單條件
// 對齊 PbootCMS PHP symbol() 函數，支援 == != > >= < <= 運算符
func evalInnerCondition(cond string) bool {
	cond = strings.TrimSpace(cond)

	// 處理 && (AND) 邏輯運算符
	if idx := strings.Index(cond, "&&"); idx > 0 {
		left := strings.TrimSpace(cond[:idx])
		right := strings.TrimSpace(cond[idx+2:])
		return evalInnerCondition(left) && evalInnerCondition(right)
	}
	// 處理 || (OR) 邏輯運算符
	if idx := strings.Index(cond, "||"); idx > 0 {
		left := strings.TrimSpace(cond[:idx])
		right := strings.TrimSpace(cond[idx+2:])
		return evalInnerCondition(left) || evalInnerCondition(right)
	}

	// 比較運算符（按長度降序匹配，避免 >= 被 > 截獲）
	for _, op := range []string{">=", "<=", "!=", "==", ">", "<"} {
		if idx := strings.Index(cond, op); idx > 0 {
			left := strings.TrimSpace(strings.Trim(cond[:idx], "'\" "))
			right := strings.TrimSpace(strings.Trim(cond[idx+len(op):], "'\" "))

			// 嘗試數值比較
			leftNum, leftErr := strconv.Atoi(left)
			rightNum, rightErr := strconv.Atoi(right)

			if leftErr == nil && rightErr == nil {
				// 數值比較
				switch op {
				case "==":
					return leftNum == rightNum
				case "!=":
					return leftNum != rightNum
				case ">":
					return leftNum > rightNum
				case ">=":
					return leftNum >= rightNum
				case "<":
					return leftNum < rightNum
				case "<=":
					return leftNum <= rightNum
				}
			} else {
				// 字串比較
				switch op {
				case "==":
					return left == right
				case "!=":
					return left != right
				case ">":
					return left > right
				case ">=":
					return left >= right
				case "<":
					return left < right
				case "<=":
					return left <= right
				}
			}
		}
	}

	// 處理 % 取模運算（如 [pics:n]%2==0 判斷偶數）
	// 無操作符：非空即真
	return cond != "" && cond != "0" && cond != "false"
}

func registerIfProvider(p *TagParser, ctx *Context) {
	p.Register("if", func(tagName string, params map[string]string, inner string) string {
		cond := params["condition"]
		trueBranch := params["true"]
		falseBranch := params["false"]

		data := buildIfContext(ctx)
		if EvalIfCondition(cond, data) {
			return trueBranch
		}
		return falseBranch
	})
}

func buildIfContext(ctx *Context) map[string]interface{} {
	// 快取：同一頁面渲染期間 config 值不變，首次建構後直接返回
	if ctx.ifContext != nil {
		return ctx.ifContext
	}
	data := make(map[string]interface{})
	if ctx.Content != nil {
		data["title"] = ctx.Content.Title
		data["id"] = ctx.Content.ID
		data["visits"] = ctx.Content.Visits
		data["istop"] = ctx.Content.IsTop
		data["isrecommend"] = ctx.Content.IsRecommend
		data["isheadline"] = ctx.Content.IsHeadline
	}
	if ctx.Sort != nil {
		data["scode"] = ctx.Sort.Scode
		data["sortname"] = ctx.Sort.Name
	}
	if ctx.Site != nil {
		data["sitetitle"] = ctx.Site.Title
	}
	if ctx.Member != nil {
		data["islogin"] = 1
	} else {
		data["islogin"] = 0
	}
	// 會員驗證碼狀態（供 {gboot:if(logincodestatus)} 等條件判斷使用）
	if model.GetConfigValue("login_check_code", "1") != "0" {
		data["logincodestatus"] = 1
	} else {
		data["logincodestatus"] = 0
	}
	if model.GetConfigValue("register_check_code", "1") != "0" {
		data["registercodestatus"] = 1
	} else {
		data["registercodestatus"] = 0
	}
	// 評論功能開關（與 commentstatus provider 一致，空值默認啟用）
	if model.GetConfigValue("comment_status", "1") != "0" {
		data["commentstatus"] = 1
	} else {
		data["commentstatus"] = 0
	}
	// 評論驗證碼開關
	if model.GetConfigValue("comment_check_code", "1") != "0" {
		data["commentcodestatus"] = 1
	} else {
		data["commentcodestatus"] = 0
	}
	// 會員登入開關
	if model.GetConfigValue("login_status", "1") != "0" {
		data["loginstatus"] = 1
	} else {
		data["loginstatus"] = 0
	}
	// 會員註冊開關
	if model.GetConfigValue("register_status", "1") != "0" {
		data["registerstatus"] = 1
	} else {
		data["registerstatus"] = 0
	}
	// 留言驗證碼開關
	if model.GetConfigValue("message_check_code", "1") != "0" {
		data["msgcodestatus"] = 1
	} else {
		data["msgcodestatus"] = 0
	}
	ctx.ifContext = data // 快取供後續 {gboot:if} 標籤復用
	return data
}

func getSortField(ctx context.Context, s *model.ContentSort, field string) string {
	switch field {
	case "name":
		return s.Name
	case "scode":
		return s.Scode
	case "pcode":
		return s.Pcode
	case "tcode":
		// 頂級父欄目代碼：沿 pcode 鏈向上查找
		code := s.Scode
		pcode := s.Pcode
		for pcode != "" && pcode != "0" {
			code = pcode
			var parent model.ContentSort
			if err := model.DB.WithContext(ctx).Where("scode = ?", pcode).First(&parent).Error; err != nil {
				break
			}
			pcode = parent.Pcode
		}
		return code
	case "toplink":
		// 頂級父欄目的鏈接
		tcode := getSortField(ctx, s, "tcode")
		var topSort model.ContentSort
		if err := model.DB.WithContext(ctx).Where("scode = ?", tcode).First(&topSort).Error; err == nil {
			if topSort.Outlink != "" {
				return topSort.Outlink
			}
			if topSort.Filename != "" {
				return "/" + topSort.Filename + "/"
			}
			if topSort.URLName != "" {
				return "/" + topSort.URLName + "/"
			}
		}
		return "/" + tcode + "/"
	case "toprows":
		// 頂級父欄目及其所有子欄目的內容總數
		tcode := getSortField(ctx, s, "tcode")
		allScodes := findAllChildScodes(ctx, tcode)
		var cnt int64
		topRowsQ := model.DB.WithContext(ctx).Model(&model.Content{}).
			Where("scode IN ? AND status = 1 AND date <= ?", allScodes, time.Now())
		if acode := acodeplugin.GetAcode(ctx); acode != "" {
			topRowsQ = topRowsQ.Where("acode = ?", acode)
		}
		topRowsQ.Count(&cnt)
		return strconv.FormatInt(cnt, 10)
	case "link":
		if s.Outlink != "" {
			return s.Outlink
		}
		// 優先用欄目自定義 URL 名稱 (filename)
		if s.Filename != "" {
			return "/" + s.Filename + ".html"
		}
		// fallback 用 model 的 urlname
		if s.URLName != "" {
			return "/" + s.URLName + ".html"
		}
		// 最後 fallback 到動態 scode 路由
		return "/sort/" + s.Scode + ".html"
	case "outlink":
		return s.Outlink
	case "urlname":
		return s.URLName
	case "listtpl":
		return s.ListTpl
	case "contenttpl":
		return s.ContentTpl
	case "ico":
		v := s.Ico
		if v != "" && !strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "http") {
			v = "/" + v
		}
		return v
	case "pic":
		v := s.Pic
		if v != "" && !strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "http") {
			v = "/" + v
		}
		return v
	case "keywords":
		return s.Keywords
	case "description":
		return s.Description
	case "subname":
		return s.Subname
	case "isnav":
		return "1" // 默認導航可見
	case "isblank":
		return "0" // 默認不新窗口
	case "sorttype":
		return fmt.Sprintf("%d", s.Type)
	default:
		return ""
	}
}

// preloadSortPaths 批量預載入欄目路徑到 ctx.sortPathCache
// 避免 contentURL 在列表循環中逐個查 DB（N+1 → 1 次查詢）
func preloadSortPaths(ctx *Context, contents []model.Content) {
	if len(contents) == 0 {
		return
	}
	if ctx.sortPathCache == nil {
		ctx.sortPathCache = make(map[string]string)
	}
	// 收集未快取的 scode
	var needed []string
	seen := map[string]bool{}
	for _, c := range contents {
		if c.Scode != "" && !seen[c.Scode] {
			seen[c.Scode] = true
			if _, ok := ctx.sortPathCache[c.Scode]; !ok {
				needed = append(needed, c.Scode)
			}
		}
	}
	if len(needed) == 0 {
		return
	}
	// 單次批量查詢
	var sorts []model.ContentSort
	model.DB.WithContext(ctx.Ctx).
		Where("scode IN ?", needed).
		Find(&sorts)
	for _, s := range sorts {
		if s.Filename != "" {
			ctx.sortPathCache[s.Scode] = s.Filename
		} else if s.URLName != "" {
			ctx.sortPathCache[s.Scode] = s.URLName
		} else {
			ctx.sortPathCache[s.Scode] = ""
		}
	}
}

// contentURL 生成內容鏈接（Google SEO 標準：無 .html 副檔名）
// URL 規則：
//   1. 多段 slug（含 /，如 test/a/b）→ /test/a/b（自定義完整路徑）
//   2. 單段 slug（如 my-article）→ /{sortPath}/{slug}（欄目路徑 + slug）
//   3. 無 slug，欄目有 pathname → /{sortPath}/{id}
//   4. 無 slug，欄目無 pathname → /content/{id}（兜底）
// 使用 ctx.sortPathCache 快取欄目路徑，避免同一頁面重複查詢 ContentSort
func contentURL(ctx *Context, c *model.Content) string {
	if c.Outlink != "" {
		return c.Outlink
	}
	// 查詢欄目 pathname（使用快取避免重複查詢）
	var sortPath string
	if c.Scode != "" {
		if ctx.sortPathCache == nil {
			ctx.sortPathCache = make(map[string]string)
		}
		if cached, ok := ctx.sortPathCache[c.Scode]; ok {
			sortPath = cached
		} else {
			var s model.ContentSort
			if model.DB.WithContext(ctx.Ctx).Where("scode = ?", c.Scode).First(&s).Error == nil {
				if s.Filename != "" {
					sortPath = s.Filename
				} else if s.URLName != "" {
					sortPath = s.URLName
				}
			}
			ctx.sortPathCache[c.Scode] = sortPath
		}
	}
	// 多段 slug（含 /）→ 直接作為完整路徑
	if c.Filename != "" {
		if strings.Contains(c.Filename, "/") {
			return "/" + c.Filename
		}
		// 單段 slug → 欄目路徑 + slug
		if sortPath != "" {
			return "/" + sortPath + "/" + c.Filename
		}
		return "/" + c.Filename
	}
	if c.URLName != "" {
		if strings.Contains(c.URLName, "/") {
			return "/" + c.URLName
		}
		if sortPath != "" {
			return "/" + sortPath + "/" + c.URLName
		}
		return "/" + c.URLName
	}
	// 無 slug，欄目有 pathname → /{sortPath}/{id}
	if sortPath != "" {
		return "/" + sortPath + "/" + strconv.Itoa(int(c.ID))
	}
	// 兜底
	return "/content/" + strconv.Itoa(int(c.ID))
}

// buildFilterURL 生成帶篩選參數的 URL（保留其他欄位的篩選）
func buildFilterURL(ctx *Context, field, value string) string {
	// 基礎路徑：當前欄目頁
	base := "/"
	if ctx.Sort != nil {
		if ctx.Sort.Filename != "" {
			base = "/" + ctx.Sort.Filename
		} else if ctx.Sort.URLName != "" {
			base = "/" + ctx.Sort.URLName
		} else {
			base = "/sort/" + ctx.Sort.Scode
		}
	}
	// 構建查詢參數（驗證字段名安全，防止 URL 注入）
	var params []string
	for k, v := range ctx.Filters {
		if k == field {
			continue // 跳過當前欄位（由 value 決定是否重新加入）
		}
		if !IsSafeFieldName(k) {
			continue
		}
		params = append(params, k+"="+urlEncode(v))
	}
	if value != "" && IsSafeFieldName(field) {
		params = append(params, field+"="+urlEncode(value))
	}
	if len(params) > 0 {
		base += "?" + strings.Join(params, "&")
	}
	return base
}

// urlEncode 簡單 URL 編碼
func urlEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func getContentField(ctx *Context, field string, params map[string]string) string {
	c := ctx.Content
	if c == nil {
		return ""
	}
	switch field {
	case "title":
		return AdjustValue(c.Title, params)
	case "titlecolor":
		return c.TitleColor
	case "subtitle":
		return c.Subtitle
	case "keywords":
		return c.Keywords
	case "description":
		return c.Description
	case "content":
		// 內鏈替換 + 外鏈 nofollow + 敏感詞過濾
		raw := c.Content
		raw = replaceContentTags(raw, ctx.Ctx)
		raw = replaceKeyword(raw)
		raw = addNofollowToExternalLinks(raw)
		return raw
	case "ico":
		v := c.Ico
		if v != "" && !strings.HasPrefix(v, "/") && !strings.HasPrefix(v, "http") {
			v = "/" + v
		}
		return v
	case "source":
		return c.Source
	case "author":
		return c.Author
	case "visits":
		val := c.Visits
		if operate, ok := params["operate"]; ok {
			operate = strings.TrimSpace(operate)
			if strings.HasPrefix(operate, "+") {
				if n, err := strconv.Atoi(strings.TrimPrefix(operate, "+")); err == nil {
					val += n
				}
			} else if strings.HasPrefix(operate, "-") {
				if n, err := strconv.Atoi(strings.TrimPrefix(operate, "-")); err == nil {
					val -= n
				}
			} else if n, err := strconv.Atoi(operate); err == nil {
				val = n
			}
		}
		return strconv.Itoa(val)
	case "date":
		if style, ok := params["style"]; ok {
			return c.Date.Format(phpToGoFormat(style))
		}
		return c.Date.Format("2006-01-02")
	case "id":
		return strconv.Itoa(int(c.ID))
	case "link":
		return contentURL(ctx, c)
	case "istop":
		return strconv.Itoa(c.IsTop)
	case "isrecommend":
		return strconv.Itoa(c.IsRecommend)
	case "isheadline":
		return strconv.Itoa(c.IsHeadline)
	case "enclosure":
		return c.Enclosure
	case "precontent":
		// 上一篇：同欄目下 ID 小於當前的最大記錄
		var prev model.Content
		if err := model.DB.WithContext(ctx.Ctx).Where("scode = ? AND id < ? AND status = 1 AND date <= ?", c.Scode, c.ID, time.Now()).Order("id desc").First(&prev).Error; err == nil {
			return fmt.Sprintf("<a href=\"%s\">%s</a>", contentURL(ctx, &prev), prev.Title)
		}
		return "沒有了"
	case "nextcontent":
		// 下一篇：同欄目下 ID 大於當前的最小記錄
		var next model.Content
		if err := model.DB.WithContext(ctx.Ctx).Where("scode = ? AND id > ? AND status = 1 AND date <= ?", c.Scode, c.ID, time.Now()).Order("id asc").First(&next).Error; err == nil {
			return fmt.Sprintf("<a href=\"%s\">%s</a>", contentURL(ctx, &next), next.Title)
		}
		return "沒有了"
	default:
		if strings.HasPrefix(field, "ext_") {
			// 從 ay_content_ext 表讀取擴展字段（對齊 PHP: $data->ext_xxx 動態屬性）
			// 使用請求級快取避免同一頁面多次存取 ext_ 欄位時重複查詢
			if ctx.ContentExt == nil {
				ctx.ContentExt = content.GetContentExtByContentID(c.ID)
				if ctx.ContentExt == nil {
					ctx.ContentExt = map[string]interface{}{} // 標記為已載入但無數據
				}
			}
			if v, ok := ctx.ContentExt[field]; ok {
				return fmt.Sprintf("%v", v)
			}
			return ""
		}
		return ""
	}
}

// parseExtraJSON 解析 content.Extra JSON 字段為 map
func parseExtraJSON(extra string) map[string]string {
	result := make(map[string]string)
	if extra == "" {
		return result
	}
	// 嘗試解析為 JSON
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(extra), &m); err == nil {
		for k, v := range m {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// parseIntConfig 從 DB 讀取整數配置（簡化 GetConfigValue+Atoi 組合）
func parseIntConfig(name string, defaultVal int) int {
	v, err := strconv.Atoi(model.GetConfigValue(name, ""))
	if err != nil {
		return defaultVal
	}
	return v
}

// commentToMap 將 CommentView 轉為模板欄位 map
func commentToMap(c *model.CommentView, index int) map[string]interface{} {
	nickname := c.Nickname
	if nickname == "" {
		nickname = c.Username
	}
	if nickname == "" {
		nickname = "遊客"
	}
	pnickname := c.Pnickname
	if pnickname == "" {
		pnickname = c.Pusername
	}
	if pnickname == "" {
		pnickname = "遊客"
	}
	headpic := c.Headpic
	if headpic == "" {
		headpic = "/static/admin/images/logo.png"
	}
	dateStr := ""
	if !c.CreateTime.IsZero() {
		dateStr = c.CreateTime.Format("2006-01-02 15:04:05")
	}
	// replyaction: comment/add?contentid=X&pid=Y&puid=Z
	pid := c.Pid
	if pid == 0 {
		pid = c.ID
	}
	replyaction := fmt.Sprintf("/comment/add?contentid=%d&pid=%d&puid=%d", c.Contentid, pid, c.Uid)

	return map[string]interface{}{
		"i":           strconv.Itoa(index),
		"n":           strconv.Itoa(index - 1),
		"id":          strconv.FormatUint(uint64(c.ID), 10),
		"comment":     c.Comment,
		"contentid":   strconv.FormatUint(uint64(c.Contentid), 10),
		"nickname":    nickname,
		"pnickname":   pnickname,
		"username":    c.Username,
		"headpic":     headpic,
		"date":        dateStr,
		"ip":          c.UserIP,
		"os":          c.UserOS,
		"bs":          c.UserBS,
		"replyaction": replyaction,
	}
}

// contentToMap 將 Content 模型轉為模板可用的 map。
// extMap 為批量預載入的擴展字段（避免 N+1），nil 時回退到單條查詢。
func contentToMap(ctx *Context, c *model.Content, index int, extMap map[uint]map[string]interface{}) map[string]interface{} {
	// URL 生成統一使用 contentURL（與前台路由邏輯一致）
	link := contentURL(ctx, c)
	// 圖片路徑標準化：確保以 / 開頭
	ico := c.Ico
	if ico != "" && !strings.HasPrefix(ico, "/") && !strings.HasPrefix(ico, "http") {
		ico = "/" + ico
	}
	// 注意：Content 結構用的是 Pics（複數）不是 Pic
	pics := c.Pics
	if pics != "" && !strings.HasPrefix(pics, "/") && !strings.HasPrefix(pics, "http") {
		pics = "/" + pics
	}
	m := map[string]interface{}{
		"n":           index,
		"i":           index + 1,
		"id":          c.ID,
		"scode":       c.Scode,
		"title":       c.Title,
		"titlecolor":  c.TitleColor,
		"subtitle":    c.Subtitle,
		"keywords":    c.Keywords,
		"description": c.Description,
		"content":     c.Content,
		"ico":         ico,
		"pics":        pics,
		"source":      c.Source,
		"author":      c.Author,
		"visits":      c.Visits,
		"date":        c.Date.Format("2006-01-02"),
		"istop":       c.IsTop,
		"isrecommend": c.IsRecommend,
		"isheadline":  c.IsHeadline,
		"link":        link,
	}

	// 注入擴展字段（ext_前綴），供列表頁 [list:ext_xxx] 使用
	// 優先使用批量預載入的 extMap，避免 N+1 查詢
	var ext map[string]interface{}
	if extMap != nil {
		ext = extMap[c.ID]
	} else {
		ext = content.GetContentExtByContentID(c.ID)
	}
	if ext != nil {
		for k, v := range ext {
			if strings.HasPrefix(k, "ext_") {
				m[k] = fmt.Sprintf("%v", v)
			}
		}
	}
	return m
}

func sortToMap(ctx context.Context, s *model.ContentSort, index int, countMap map[string]int) map[string]interface{} {
	// URL 生成規則（對齊 PbootCMS PHP）：
	//   1. 優先用欄目自定義 URL 名稱 (s.Filename) —— 後台「URL名稱」欄位
	//   2. 退而求其次用模型的 urlname (s.URLName) —— 通常等同於 scode
	//   3. 最後 fallback 到動態 scode 路由
	// 注意：區分 urlname（model 表字段）和 filename（欄目表字段），
	// PbootCMS PHP 中也用 filename 對應「URL名稱」。
	link := "/" + s.Filename + "/"
	if link == "//" {
		link = "/" + s.URLName + "/"
	}
	if link == "//" {
		link = fmt.Sprintf("/sort/%s", s.Scode)
	}
	// 圖片路徑標準化：確保以 / 開頭
	ico := s.Ico
	if ico != "" && !strings.HasPrefix(ico, "/") && !strings.HasPrefix(ico, "http") {
		ico = "/" + ico
	}
	pic := s.Pic
	if pic != "" && !strings.HasPrefix(pic, "/") && !strings.HasPrefix(pic, "http") {
		pic = "/" + pic
	}
	// 計算該欄目及其子欄目的內容數量
	// 優先使用批量預載入的 countMap，避免 N+1 查詢
	var rowCount int
	if countMap != nil {
		rowCount = countMap[s.Scode]
	} else {
		childScodes := findAllChildScodes(ctx, s.Scode)
		var c int64
		fallbackQ := model.DB.WithContext(ctx).Model(&model.Content{}).
			Where("scode IN ? AND status = 1 AND date <= ?", childScodes, time.Now())
		if acode := acodeplugin.GetAcode(ctx); acode != "" {
			fallbackQ = fallbackQ.Where("acode = ?", acode)
		}
		fallbackQ.Count(&c)
		rowCount = int(c)
	}
	return map[string]interface{}{
		"n":       index,
		"i":       index + 1,
		"scode":   s.Scode,
		"name":    s.Name,
		"subname": s.Subname,
		"ico":     ico,
		"pic":     pic,
		"link":    link,
		"rows":    rowCount,
	}
}

// buildSortCountMap 批量預載入所有欄目及其子欄目的內容數量。
// 一次性載入全部 ContentSort 和內容計數，在記憶體中建構樹狀結構，
// 將 N 次遞迴查詢降為 2 欝查詢（全部欄目 + 全部內容計數）。
func buildSortCountMap(ctx context.Context) map[string]int {
	// 1. 載入所有啟用的欄目
	var allSorts []model.ContentSort
	q := model.DB.WithContext(ctx).Where("status = 1")
	if acode := acodeplugin.GetAcode(ctx); acode != "" {
		q = q.Where("acode = ?", acode)
	}
	q.Find(&allSorts)

	// 2. 建構 parent → children 記憶體映射
	childrenMap := make(map[string][]string)
	for _, s := range allSorts {
		childrenMap[s.Pcode] = append(childrenMap[s.Pcode], s.Scode)
	}

	// 3. 批量查詢每個 scode 的內容數量（單次 GROUP BY）
	type scodeCount struct {
		Scode string
		Cnt   int64
	}
	var counts []scodeCount
	countQ := model.DB.WithContext(ctx).Model(&model.Content{}).
		Select("scode, count(*) as cnt").
		Where("status = 1 AND date <= ?", time.Now())
	if acode := acodeplugin.GetAcode(ctx); acode != "" {
		countQ = countQ.Where("acode = ?", acode)
	}
	countQ.Group("scode").Scan(&counts)
	directCount := make(map[string]int)
	for _, c := range counts {
		directCount[c.Scode] = int(c.Cnt)
	}

	// 4. 遞迴計算每個欄目的總數量（自身 + 所有子孫）
	result := make(map[string]int, len(allSorts))
	var computeTotal func(scode string) int
	computeTotal = func(scode string) int {
		total := directCount[scode]
		for _, child := range childrenMap[scode] {
			total += computeTotal(child)
		}
		return total
	}
	for _, s := range allSorts {
		result[s.Scode] = computeTotal(s.Scode)
	}
	return result
}

// findAllChildScodes 遞歸查找指定 scode 及其所有子欄目的 scode 列表
// 使用 SQLite 遞迴 CTE 單次查詢取代 N 次遞迴 DB 查詢（配合 idx_sort_pcode 索引）
func findAllChildScodes(ctx context.Context, parentScode string) []string {
	acode := acodeplugin.GetAcode(ctx)

	query := `WITH RECURSIVE descendants AS (
		SELECT scode FROM ay_content_sort WHERE scode = ? AND status = 1`
	if acode != "" {
		query += ` AND acode = ?`
	}
	query += `
		UNION ALL
		SELECT s.scode FROM ay_content_sort s
		INNER JOIN descendants d ON s.pcode = d.scode
		WHERE s.status = 1`
	if acode != "" {
		query += ` AND s.acode = ?`
	}
	query += `
	)
	SELECT scode FROM descendants`

	var rows []struct {
		Scode string
	}
	if acode != "" {
		model.DB.WithContext(ctx).Raw(query, parentScode, acode, acode).Scan(&rows)
	} else {
		model.DB.WithContext(ctx).Raw(query, parentScode).Scan(&rows)
	}

	if len(rows) == 0 {
		return []string{parentScode}
	}
	scodes := make([]string, len(rows))
	for i, r := range rows {
		scodes[i] = r.Scode
	}
	return scodes
}

// ─── 標題樣式解析（對齊 PHP index_title/list_title/content_title/about_title/other_title 配置） ───

// currentHomePath 返回當前語言區域的首頁路徑
// 默認區域返回 /，非默認返回 /{acode}/
func currentHomePath(ctx *Context) string {
	acode := acodeplugin.GetAcode(ctx.Ctx)
	if acode == "" {
		return "/"
	}
	// 查默認區域
	areas := model.GetCachedAreas()
	for _, a := range areas {
		if a.IsDefault == "1" && a.Acode == acode {
			return "/"
		}
	}
	return "/" + acode + "/"
}

// acodeToHreflang 將 acode 映射為標準 hreflang 代碼（ISO 639-1 + ISO 3166-1）
func acodeToHreflang(acode string) string {
	switch acode {
	case "sc":
		return "zh-Hans"
	case "tc":
		return "zh-Hant"
	case "en":
		return "en"
	default:
		return acode
	}
}

// getEffectiveDomain 獲取當前頁面生效的域名（包含協議）
func getEffectiveDomain(ctx *Context) string {
	// 如果是測試環境（如 gocms.icun.eu.org、localhost 等），強制使用當前請求的動態 Host，絕不跨域名污染生產站
	if ctx != nil && ctx.HttpURL != "" {
		host := strings.ToLower(ctx.HttpURL)
		if strings.Contains(host, ".eu.org") || strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") ||
			strings.Contains(host, ".test") || strings.Contains(host, ".local") || strings.Contains(host, ".nip.io") {
			return strings.TrimRight(ctx.HttpURL, "/")
		}
	}

	// 生產環境：優先使用站點配置的正式域名
	if ctx != nil && ctx.Site != nil && ctx.Site.Domain != "" {
		domain := ctx.Site.Domain
		if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			domain = "https://" + domain
		}
		return strings.TrimRight(domain, "/")
	}

	if ctx != nil && ctx.HttpURL != "" {
		return strings.TrimRight(ctx.HttpURL, "/")
	}
	return "http://localhost:8080"
}

// buildDetailURL 構建內容頁語言相對路徑
func buildDetailURL(acode, defaultAcode string, c *model.Content) string {
	var path string
	if c.Filename != "" {
		path = "/" + c.Filename + ".html"
	} else if c.URLName != "" {
		path = "/" + c.URLName + ".html"
	} else {
		var sortPath string
		if c.Scode != "" {
			var s model.ContentSort
			if model.DB.WithContext(acodeplugin.SkipAcode(context.Background())).Where("scode = ? AND acode = ?", c.Scode, acode).First(&s).Error == nil {
				if s.Filename != "" {
					sortPath = s.Filename
				} else if s.URLName != "" {
					sortPath = s.URLName
				}
			}
		}
		if sortPath != "" {
			path = fmt.Sprintf("/%s/%d", sortPath, c.ID)
		} else {
			path = fmt.Sprintf("/content/%d", c.ID)
		}
	}

	if acode == defaultAcode {
		return path
	}
	return "/" + acode + path
}

// buildSortURLWithLang 構建欄目頁語言相對路徑
func buildSortURLWithLang(acode, defaultAcode string, s *model.ContentSort) string {
	var path string
	if s.Filename != "" {
		path = "/" + s.Filename
	} else if s.URLName != "" {
		path = "/" + s.URLName
	} else if s.Scode != "" {
		path = fmt.Sprintf("/sort/%s", s.Scode)
	} else {
		path = "/"
	}

	if acode == defaultAcode {
		return path
	}
	return "/" + acode + path
}

// findTranslatedContent 在目標語言中查找與當前文章對應的翻譯版本（精準匹配，杜絕 404）
func findTranslatedContent(reqCtx context.Context, cur *model.Content, targetAcode string) *model.Content {
	if cur == nil || targetAcode == "" {
		return nil
	}
	skipCtx := acodeplugin.SkipAcode(reqCtx)

	// 1. 若當前文章有關聯 ID (rel_id > 0)
	if cur.RelID > 0 {
		var target model.Content
		if err := model.DB.WithContext(skipCtx).Where("(rel_id = ? OR id = ?) AND acode = ? AND status = 1", cur.RelID, cur.RelID, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	} else {
		// 當前文章 rel_id == 0，嘗試查是否有其他文章將 rel_id 指向當前文章 ID
		var target model.Content
		if err := model.DB.WithContext(skipCtx).Where("rel_id = ? AND acode = ? AND status = 1", cur.ID, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	}

	// 2. 自定義 URL 別名（urlname / filename）精準匹配
	if cur.URLName != "" {
		var target model.Content
		if err := model.DB.WithContext(skipCtx).Where("urlname = ? AND acode = ? AND status = 1", cur.URLName, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	}
	if cur.Filename != "" {
		var target model.Content
		if err := model.DB.WithContext(skipCtx).Where("filename = ? AND acode = ? AND status = 1", cur.Filename, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	}

	return nil
}

// findTranslatedSort 在目標語言中查找對應欄目
func findTranslatedSort(reqCtx context.Context, cur *model.ContentSort, targetAcode string) *model.ContentSort {
	if cur == nil || targetAcode == "" {
		return nil
	}
	skipCtx := acodeplugin.SkipAcode(reqCtx)

	var target model.ContentSort
	if cur.Scode != "" {
		if err := model.DB.WithContext(skipCtx).Where("scode = ? AND acode = ? AND status = 1", cur.Scode, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	}
	if cur.Filename != "" {
		if err := model.DB.WithContext(skipCtx).Where("filename = ? AND acode = ? AND status = 1", cur.Filename, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	}
	return nil
}

// buildHreflang 生成 hreflang 標籤組（自引用 + 雙向對稱 + x-default）
func buildHreflang(ctx *Context) string {
	areas := model.GetCachedAreas()
	if len(areas) <= 1 {
		return ""
	}

	domain := getEffectiveDomain(ctx)

	// 找默認區域
	defaultAcode := ""
	for _, a := range areas {
		if a.IsDefault == "1" {
			defaultAcode = a.Acode
			break
		}
	}
	if defaultAcode == "" && len(areas) > 0 {
		defaultAcode = areas[0].Acode
	}

	var sb strings.Builder
	var defaultHref string

	// 場景 1：內容詳情頁 (ctx.Content != nil)
	if ctx.Content != nil {
		curContent := ctx.Content
		curAcode := curContent.Acode

		for _, a := range areas {
			hreflangCode := acodeToHreflang(a.Acode)

			// 若是當前語言，使用當前文章自己的真實 URL
			if a.Acode == curAcode {
				link := buildDetailURL(a.Acode, defaultAcode, curContent)
				href := domain + link
				sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s" />`, hreflangCode, href))
				sb.WriteString("\n")
				if a.Acode == defaultAcode {
					defaultHref = href
				}
				continue
			}

			// 跨語言尋找真正對應的翻譯文章（避免 404 與偽實現）
			targetContent := findTranslatedContent(ctx.Ctx, curContent, a.Acode)
			if targetContent == nil {
				// 該語言無對應翻譯文章，依 Google 規範嚴格不輸出該語言 hreflang，避免 404 報錯
				continue
			}

			link := buildDetailURL(a.Acode, defaultAcode, targetContent)
			href := domain + link
			sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s" />`, hreflangCode, href))
			sb.WriteString("\n")
			if a.Acode == defaultAcode {
				defaultHref = href
			}
		}

		if defaultHref != "" {
			sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="x-default" href="%s" />`, defaultHref))
		}
		return sb.String()
	}

	// 場景 2：欄目列表頁 (ctx.Sort != nil && ctx.Content == nil)
	if ctx.Sort != nil {
		curSort := ctx.Sort
		curAcode := curSort.Acode

		for _, a := range areas {
			hreflangCode := acodeToHreflang(a.Acode)
			if a.Acode == curAcode {
				link := buildSortURLWithLang(a.Acode, defaultAcode, curSort)
				href := domain + link
				sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s" />`, hreflangCode, href))
				sb.WriteString("\n")
				if a.Acode == defaultAcode {
					defaultHref = href
				}
				continue
			}

			targetSort := findTranslatedSort(ctx.Ctx, curSort, a.Acode)
			if targetSort == nil {
				continue
			}
			link := buildSortURLWithLang(a.Acode, defaultAcode, targetSort)
			href := domain + link
			sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s" />`, hreflangCode, href))
			sb.WriteString("\n")
			if a.Acode == defaultAcode {
				defaultHref = href
			}
		}

		if defaultHref != "" {
			sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="x-default" href="%s" />`, defaultHref))
		}
		return sb.String()
	}

	// 場景 3：首頁 / 靜態頁
	currentPath := ctx.CurrentPath
	if currentPath == "" || currentPath == "/" {
		currentPath = "/"
	} else if !strings.HasPrefix(currentPath, "/") {
		currentPath = "/" + currentPath
	}

	for _, a := range areas {
		link := currentPath
		if a.Acode != defaultAcode {
			if currentPath == "/" {
				link = "/" + a.Acode + "/"
			} else {
				link = "/" + a.Acode + currentPath
			}
		}
		href := domain + link
		hreflangCode := acodeToHreflang(a.Acode)
		sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s" />`, hreflangCode, href))
		sb.WriteString("\n")
		if a.Acode == defaultAcode {
			defaultHref = href
		}
	}

	if defaultHref != "" {
		sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="x-default" href="%s" />`, defaultHref))
	}

	return sb.String()
}

// GetCanonicalURL 獲取當前頁面的標準 Canonical URL（共享函數，保證與 JSON-LD @id 100% 一致）
func GetCanonicalURL(ctx *Context) string {
	acode := acodeplugin.GetAcode(ctx.Ctx)

	currentPath := ctx.CurrentPath
	if currentPath == "" {
		currentPath = "/"
	}
	if !strings.HasPrefix(currentPath, "/") {
		currentPath = "/" + currentPath
	}

	// 默認區域不需要前綴
	areas := model.GetCachedAreas()
	isDefault := false
	for _, a := range areas {
		if a.Acode == acode && a.IsDefault == "1" {
			isDefault = true
			break
		}
	}

	canonicalPath := currentPath
	if !isDefault && acode != "" && !strings.HasPrefix(canonicalPath, "/"+acode+"/") && canonicalPath != "/"+acode {
		canonicalPath = "/" + acode + canonicalPath
	}

	domain := getEffectiveDomain(ctx)
	return domain + canonicalPath
}

// buildCanonical 生成 canonical 標籤，指向當前頁面的標準 URL
func buildCanonical(ctx *Context) string {
	canonicalURL := GetCanonicalURL(ctx)
	return fmt.Sprintf(`<link rel="canonical" href="%s" />`, canonicalURL)
}

// acodeToLocale 將 acode 轉換為 Open Graph locale 格式（zh_CN / zh_HK / en_US）
func acodeToLocale(acode string) string {
	switch acode {
	case "sc":
		return "zh_CN"
	case "tc":
		return "zh_HK"
	case "en":
		return "en_US"
	default:
		return acode
	}
}

// buildOpenGraph 生成 Open Graph meta 標籤（社交分享用）
func buildOpenGraph(ctx *Context) string {
	acode := acodeplugin.GetAcode(ctx.Ctx)

	// 站點域名（環境感知：測試站使用當前 Host，生產站使用配置域名）
	domain := getEffectiveDomain(ctx)
	siteName := ""
	if ctx.Site != nil {
		siteName = ctx.Site.Title
	}

	// 當前頁面 URL（與 canonical 相同）
	currentPath := ctx.CurrentPath
	if currentPath == "" {
		currentPath = "/"
	}
	if !strings.HasPrefix(currentPath, "/") {
		currentPath = "/" + currentPath
	}

	link := currentPath
	if acode != "" {
		areas := model.GetCachedAreas()
		isDefault := false
		for _, a := range areas {
			if a.Acode == acode && a.IsDefault == "1" {
				isDefault = true
				break
			}
		}
		if !isDefault {
			link = "/" + acode + currentPath
		}
	}

	canonicalURL := link
	if domain != "" {
		canonicalURL = domain + link
	}

	// 頁面標題和描述
	title := resolvePageTitle(ctx)
	description := resolvePageDescription(ctx)

	// 頁面類型
	ogType := "website"
	if ctx.Content != nil {
		ogType = "article"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<meta property="og:title" content="%s" />`, html.EscapeString(title)))
	sb.WriteString(fmt.Sprintf(`<meta property="og:description" content="%s" />`, html.EscapeString(description)))
	sb.WriteString(fmt.Sprintf(`<meta property="og:url" content="%s" />`, canonicalURL))
	sb.WriteString(fmt.Sprintf(`<meta property="og:type" content="%s" />`, ogType))
	if siteName != "" {
		sb.WriteString(fmt.Sprintf(`<meta property="og:site_name" content="%s" />`, html.EscapeString(siteName)))
	}
	sb.WriteString(fmt.Sprintf(`<meta property="og:locale" content="%s" />`, acodeToLocale(acode)))

	// og:locale:alternate — 其他語言
	for _, a := range model.GetCachedAreas() {
		if a.Acode != acode {
			sb.WriteString(fmt.Sprintf(`<meta property="og:locale:alternate" content="%s" />`, acodeToLocale(a.Acode)))
		}
	}

	return sb.String()
}

// resolvePageTitle 根據頁面類型讀取對應的 *_title 配置，解析巢狀標籤後返回最終標題
func resolvePageTitle(ctx *Context) string {
	var tmpl string
	if ctx.Content != nil && ctx.Sort != nil {
		// 內容頁（有欄目和內容）
		tmpl = model.GetConfigValue("content_title", "")
	} else if ctx.Sort != nil {
		// 列表/欄目頁
		tmpl = model.GetConfigValue("list_title", "")
	} else if ctx.Content != nil {
		// 單頁（有內容但無欄目）
		tmpl = model.GetConfigValue("about_title", "")
	} else {
		// 首頁
		tmpl = model.GetConfigValue("index_title", "")
	}

	if tmpl == "" {
		// 配置為空時使用預設值（對齊 PHP 各方法的預設格式）
		if ctx.Content != nil && ctx.Sort != nil {
			tmpl = "{content:title}-{sort:name}-{gboot:sitetitle}-{gboot:sitesubtitle}"
		} else if ctx.Sort != nil {
			tmpl = "{sort:name}-{gboot:sitetitle}-{gboot:sitesubtitle}"
		} else if ctx.Content != nil {
			tmpl = "{content:title}-{gboot:sitetitle}-{gboot:sitesubtitle}"
		} else {
			tmpl = "{gboot:sitetitle}-{gboot:sitesubtitle}"
		}
	}

	return resolveTitleTags(tmpl, ctx)
}

// resolvePageKeywords 根據頁面類型返回關鍵詞
func resolvePageKeywords(ctx *Context) string {
	if ctx.Content != nil && ctx.Content.Keywords != "" {
		return ctx.Content.Keywords
	}
	if ctx.Sort != nil && ctx.Sort.Keywords != "" {
		return ctx.Sort.Keywords
	}
	if ctx.Site != nil {
		return ctx.Site.Keywords
	}
	return ""
}

// resolvePageDescription 根據頁面類型返回描述
func resolvePageDescription(ctx *Context) string {
	if ctx.Content != nil && ctx.Content.Description != "" {
		return ctx.Content.Description
	}
	if ctx.Sort != nil && ctx.Sort.Description != "" {
		return ctx.Sort.Description
	}
	if ctx.Site != nil {
		return ctx.Site.Description
	}
	return ""
}

// resolveTitleTags 解析標題模板中的巢狀標籤
// 支援：{gboot:sitetitle} {gboot:sitesubtitle} {content:title} {sort:name} {sort:title} 等
func resolveTitleTags(tmpl string, ctx *Context) string {
	result := tmpl

	// 站點標籤
	if ctx.Site != nil {
		result = strings.ReplaceAll(result, "{gboot:sitetitle}", ctx.Site.Title)
		result = strings.ReplaceAll(result, "{gboot:sitesubtitle}", ctx.Site.Subtitle)
		result = strings.ReplaceAll(result, "{gboot:sitekeywords}", ctx.Site.Keywords)
		result = strings.ReplaceAll(result, "{gboot:sitedescription}", ctx.Site.Description)
		result = strings.ReplaceAll(result, "{pboot:sitetitle}", ctx.Site.Title)
		result = strings.ReplaceAll(result, "{pboot:sitesubtitle}", ctx.Site.Subtitle)
		result = strings.ReplaceAll(result, "{pboot:sitekeywords}", ctx.Site.Keywords)
		result = strings.ReplaceAll(result, "{pboot:sitedescription}", ctx.Site.Description)
	}

	// 內容標籤
	if ctx.Content != nil {
		result = strings.ReplaceAll(result, "{content:title}", ctx.Content.Title)
		result = strings.ReplaceAll(result, "{content:keywords}", ctx.Content.Keywords)
		result = strings.ReplaceAll(result, "{content:description}", ctx.Content.Description)
	}

	// 欄目標籤
	if ctx.Sort != nil {
		sortName := ctx.Sort.Name
		sortTitle := ctx.Sort.Title
		if sortTitle == "" {
			sortTitle = sortName
		}
		result = strings.ReplaceAll(result, "{sort:name}", sortName)
		result = strings.ReplaceAll(result, "{sort:title}", sortTitle)
		result = strings.ReplaceAll(result, "{sort:keywords}", ctx.Sort.Keywords)
		result = strings.ReplaceAll(result, "{sort:description}", ctx.Sort.Description)
	}

	return result
}
