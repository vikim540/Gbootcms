package parser

import "strings"

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
	"nodata": {
		"tc": "暫無內容",
		"sc": "暂无内容",
		"en": "No data available",
		"default": "暫無內容",
	},
	"nextpage": {
		"tc": "下一頁",
		"sc": "下一页",
		"en": "Next",
		"default": "下一頁",
	},
	"prevpage": {
		"tc": "上一頁",
		"sc": "上一页",
		"en": "Previous",
		"default": "上一頁",
	},
	"indexpage": {
		"tc": "首頁",
		"sc": "首页",
		"en": "First Page",
		"default": "首頁",
	},
	"lastpage": {
		"tc": "尾頁",
		"sc": "尾页",
		"en": "Last",
		"default": "尾頁",
	},
	"jobposition": {
		"tc": "招聘職位",
		"sc": "招聘职位",
		"en": "Job Position",
		"default": "招聘職位",
	},
	"recommend": {
		"tc": "推薦",
		"sc": "推荐",
		"en": "Recommended",
		"default": "推薦",
	},
	"upload": {
		"tc": "上傳",
		"sc": "上传",
		"en": "Upload",
		"default": "上傳",
	},
	"entercurrentpassword": {
		"tc": "請輸入原密碼",
		"sc": "请输入原密码",
		"en": "Enter current password",
		"default": "請輸入原密碼",
	},
	"avatar": {
		"tc": "頭像",
		"sc": "头像",
		"en": "Avatar",
		"default": "頭像",
	},
	"confirmpassword": {
		"tc": "確認密碼",
		"sc": "确认密码",
		"en": "Confirm Password",
		"default": "確認密碼",
	},
	"enterconfirmpassword": {
		"tc": "請確認密碼",
		"sc": "请确认密码",
		"en": "Enter confirm password",
		"default": "請確認密碼",
	},
	"male": {
		"tc": "男",
		"sc": "男",
		"en": "Male",
		"default": "男",
	},
	"female": {
		"tc": "女",
		"sc": "女",
		"en": "Female",
		"default": "女",
	},
	"gender": {
		"tc": "性別",
		"sc": "性别",
		"en": "Gender",
		"default": "性別",
	},
	"publishdate": {
		"tc": "發布時間",
		"sc": "发布时间",
		"en": "Publish Date",
		"default": "發布時間",
	},
	"whatsapp": {
		"tc": "WhatsApp",
		"sc": "WhatsApp",
		"en": "WhatsApp",
		"default": "WhatsApp",
	},
	"selectimagetoupload": {
		"tc": "選擇圖片上傳",
		"sc": "选择图片上传",
		"en": "Select image to upload",
		"default": "選擇圖片上傳",
	},
	"selectfiletoupload": {
		"tc": "選擇文件上傳",
		"sc": "选择文件上传",
		"en": "Select file to upload",
		"default": "選擇文件上傳",
	},
	"viewdetail": {
		"tc": "查看詳情",
		"sc": "查看详情",
		"en": "View Details",
		"default": "查看詳情",
	},
	"headline": {
		"tc": "頭條",
		"sc": "头条",
		"en": "Headlines",
		"default": "頭條",
	},
	"enterphone": {
		"tc": "請輸入電話",
		"sc": "请输入电话",
		"en": "Enter phone number",
		"default": "請輸入電話",
	},
	"enteremail": {
		"tc": "請輸入郵箱",
		"sc": "请输入邮箱",
		"en": "Enter email",
		"default": "請輸入郵箱",
	},
	"enteravatarurl": {
		"tc": "請輸入頭像網址",
		"sc": "请输入头像网址",
		"en": "Enter avatar URL",
		"default": "請輸入頭像網址",
	},
	"birthday": {
		"tc": "生日",
		"sc": "生日",
		"en": "Birthday",
		"default": "生日",
	},
	"close": {
		"tc": "關閉",
		"sc": "关闭",
		"en": "Close",
		"default": "關閉",
	},
	"enterwhatsapp": {
		"tc": "請輸入WhatsApp",
		"sc": "请输入WhatsApp",
		"en": "Enter WhatsApp",
		"default": "請輸入WhatsApp",
	},
	"currentpassword": {
		"tc": "原密碼",
		"sc": "原密码",
		"en": "Current Password",
		"default": "原密碼",
	},
	"newpassword": {
		"tc": "新密碼",
		"sc": "新密码",
		"en": "New Password",
		"default": "新密碼",
	},
	"back": {
		"tc": "返回",
		"sc": "返回",
		"en": "Back",
		"default": "返回",
	},
	"fileupload": {
		"tc": "文件上傳",
		"sc": "文件上传",
		"en": "File Upload",
		"default": "文件上傳",
	},
	"leaveblankifnotmodify": {
		"tc": "不修改請留空",
		"sc": "不修改请留空",
		"en": "Leave blank if not changing",
		"default": "不修改請留空",
	},
	"enternickname": {
		"tc": "請輸入暱稱",
		"sc": "请输入昵称",
		"en": "Enter nickname",
		"default": "請輸入暱稱",
	},
	"save": {
		"tc": "保存",
		"sc": "保存",
		"en": "Save",
		"default": "保存",
	},
	"top": {
		"tc": "置頂",
		"sc": "置顶",
		"en": "Top",
		"default": "置頂",
	},
	"entermobile": {
		"tc": "請輸入手機號碼",
		"sc": "请输入手机号码",
		"en": "Enter mobile number",
		"default": "請輸入手機號碼",
	},
}

// ResolveI18nText 根據字典返回對應語言的詞條
func ResolveI18nText(field, acode string) string {
	if acode == "" {
		acode = "default"
	}
	lookupKey := strings.ToLower(field)
	dict, ok := i18nUIDict[lookupKey]
	if !ok {
		dict, ok = i18nUIDict[field]
	}
	if ok {
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
}
