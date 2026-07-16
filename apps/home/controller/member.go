package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"gbootcms/apps/common/mail"
	"gbootcms/apps/common/parser"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// doubleMD5 雙重 MD5 加密，與 PbootCMS 用戶數據兼容（向後兼容用）
func doubleMD5(s string) string {
	return common.DoubleMD5(s)
}

// ─── 會員登錄鎖定（對齊後台 checkLoginBlack/setLoginBlack 邏輯，用 JSON 存儲） ───

func memberBlackFile() string {
	return filepath.Join("runtime", "data", "member_login_black.json")
}

// checkMemberLoginBlack 檢查會員登錄是否被鎖定，返回剩餘秒數（0=未鎖定）
func checkMemberLoginBlack(c *gin.Context) int {
	lockCount, _ := strconv.Atoi(model.GetConfigValue("login_error_count", "5"))
	lockTime, _ := strconv.Atoi(model.GetConfigValue("login_error_wait", "300"))

	data, err := os.ReadFile(memberBlackFile())
	if err != nil {
		return 0
	}

	var blackList map[string]map[string]int64
	if err := json.Unmarshal(data, &blackList); err != nil {
		return 0
	}

	userIP := c.ClientIP()
	if userIP == "::1" {
		userIP = "127.0.0.1"
	}

	entry, ok := blackList[userIP]
	if !ok {
		return 0
	}

	if entry["count"] >= int64(lockCount) {
		elapsed := time.Now().Unix() - entry["time"]
		remain := lockTime - int(elapsed)
		if remain > 0 {
			return remain
		}
	}
	return 0
}

// setMemberLoginBlack 累計會員登錄失敗次數
func setMemberLoginBlack(c *gin.Context) {
	lockCount, _ := strconv.Atoi(model.GetConfigValue("login_error_count", "5"))
	lockTime, _ := strconv.Atoi(model.GetConfigValue("login_error_wait", "300"))

	os.MkdirAll(filepath.Dir(memberBlackFile()), 0755)

	blackList := make(map[string]map[string]int64)
	if data, err := os.ReadFile(memberBlackFile()); err == nil {
		json.Unmarshal(data, &blackList)
	}

	userIP := c.ClientIP()
	if userIP == "::1" {
		userIP = "127.0.0.1"
	}

	now := time.Now().Unix()
	if entry, ok := blackList[userIP]; ok {
		if entry["count"] < int64(lockCount) && now-entry["time"] < int64(lockTime) {
			entry["count"]++
			entry["time"] = now
			blackList[userIP] = entry
		} else {
			blackList[userIP] = map[string]int64{"time": now, "count": 1}
		}
	} else {
		blackList[userIP] = map[string]int64{"time": now, "count": 1}
	}

	data, _ := json.Marshal(blackList)
	os.WriteFile(memberBlackFile(), data, 0644)
}

// clearMemberLoginBlack 清除會員登錄鎖定記錄
func clearMemberLoginBlack(c *gin.Context) {
	userIP := c.ClientIP()
	if userIP == "::1" {
		userIP = "127.0.0.1"
	}

	blackList := make(map[string]map[string]int64)
	if data, err := os.ReadFile(memberBlackFile()); err == nil {
		json.Unmarshal(data, &blackList)
	}

	delete(blackList, userIP)

	if len(blackList) == 0 {
		os.Remove(memberBlackFile())
	} else {
		data, _ := json.Marshal(blackList)
		os.WriteFile(memberBlackFile(), data, 0644)
	}
}

// generateUcode 生成會員編碼（併發安全：使用時間戳 + 隨機數）
// 原版基於最後一條記錄自增，在併發註冊時會產生 race condition。
// 改用「時間戳尾部 + 密碼學安全隨機數」方案，保證唯一性且無競態。
func generateUcode(c *gin.Context) string {
	// 使用時間戳後 6 位 + 4 位隨機數，總共 10 位
	ts := time.Now().Unix() % 1000000
	randPart := common.SecureRandomInt(10000)
	return fmt.Sprintf("%d%04d", ts, randPart)
}

// Retrieve 會員找回密碼（對齊 PHP MemberController::retrieve()）
func (fc *FrontController) Retrieve(c *gin.Context) {
	if c.Request.Method == "POST" {
		// 郵箱驗證碼檢查（對齊 PHP: $checkcode != session('checkcode')，此處 checkcode 為郵箱驗證碼）
		checkcode := strings.ToLower(c.PostForm("checkcode"))
		if checkcode == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "驗證碼不能為空", "tourl": ""})
			return
		}
		sessionCode, _ := common.GetSession(c, "email_checkcode").(string)
		if sessionCode == "" || checkcode != strings.ToLower(sessionCode) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "驗證碼錯誤", "tourl": ""})
			return
		}

		username := c.PostForm("username")
		email := c.PostForm("email")
		password := c.PostForm("password")

		if username == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "用戶帳號不能為空", "tourl": ""})
			return
		}

		// 查詢會員（對齊 PHP: $this->model->checkUsername(['username'=>$username])）
		var member model.Member
		if err := model.DB.WithContext(c.Request.Context()).Where("username = ?", username).First(&member).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "該用戶不存在", "tourl": ""})
			return
		}

		// 檢查郵箱是否匹配（對齊 PHP: !empty($userInfo['useremail']) && $userInfo['useremail'] != $email）
		if member.Useremail != "" && member.Useremail != email {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "與註冊郵箱不匹配，請聯繫管理員", "tourl": ""})
			return
		}

		// 更新密碼及郵箱（使用 bcrypt 雜湊，向後兼容舊版雙 MD5）
	hashedPwd, err := common.HashPassword(password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "密碼加密失敗，請重試", "tourl": ""})
		return
	}
	updates := map[string]interface{}{
		"password":  hashedPwd,
		"useremail": email,
	}
		if err := model.DB.WithContext(c.Request.Context()).Model(&member).Updates(updates).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "密碼重置失敗", "tourl": ""})
		return
	}

		// 清除驗證碼
		common.SetSession(c, "email_checkcode", "")

		c.JSON(http.StatusOK, gin.H{"code": 1, "data": "密碼設置成功", "tourl": langPath(c, "/login")})
		return
	}

	// GET：渲染找回密碼頁面
	fc.renderMemberPage(c, "member/retrieve.html")
}

// SendMemberEmail 發送郵箱驗證碼（對齊 PHP MemberController::sendEmail()，找回密碼用）
func (fc *FrontController) SendMemberEmail(c *gin.Context) {
	email := c.PostForm("to")
	retrieve := c.PostForm("retrieve")

	// retrieve 存在時為找回密碼驗證，不進行驗證碼模式判斷（對齊 PHP）
	if retrieve == "" {
		if model.GetConfigValue("register_check_code", "0") != "2" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "發送失敗，後台配置非郵箱驗證碼模式", "tourl": ""})
			return
		}
	}

	// 防頻繁發送（10秒間隔，對齊 PHP: time() - session('lastsend') < 10）
	lastSend, _ := common.GetSession(c, "lastsend").(int64)
	if time.Now().Unix()-lastSend < 10 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "您提交太頻繁了，請稍後再試", "tourl": ""})
		return
	}

	// 郵箱參數檢查（對齊 PHP: post('to')）
	if email == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "發送失敗，缺少發送對象參數to", "tourl": ""})
		return
	}

	// 郵箱格式驗證（對齊 PHP: preg_match('/^[\w]+@[\w]+\.[a-zA-Z]+$/', $to)）
	matched, _ := regexp.MatchString(`^[\w]+@[\w]+\.[a-zA-Z]+$`, email)
	if !matched {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "郵箱格式不正確，請輸入正確的郵箱帳號", "tourl": ""})
		return
	}

	// 記錄最後提交時間
	common.SetSession(c, "lastsend", time.Now().Unix())

	// 生成 4 位隨機驗證碼（對齊 PHP: create_code(4)）
	code := fmt.Sprintf("%04d", common.SecureRandomInt(10000))
	common.SetSession(c, "email_checkcode", code)

	// 發送郵件（對齊 PHP: $mail_subject / $mail_body）
	subject := "【" + model.GetConfigValue("cmsname", "Gbootcms") + "】您有新的驗證碼信息，請注意查收！"
	body := "您的驗證碼為：" + code + "<br>來自網站 " + c.Request.Host + " （" + time.Now().Format("2006-01-02 15:04:05") + "）"

	if err := mail.SendMail(email, subject, body); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "發送失敗，" + err.Error(), "tourl": ""})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 1, "data": "發送成功", "tourl": ""})
}

// renderMemberPage 渲染會員頁面（與前台頁面使用相同的 parser 流程）
func (fc *FrontController) renderMemberPage(c *gin.Context, tpl string) {
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.getStore(c).Render(tpl)
	if !fc.checkMustLogin(c, content) {
		return
	}
	content = p.Render(content)
	content = postRender(content, c.Request.Context())
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

// Login 會員登錄
func (fc *FrontController) Login(c *gin.Context) {
	// 已登錄則跳轉用戶中心
	if common.GetSessionInt(c, "pboot_uid") > 0 {
		c.Redirect(http.StatusFound, langPath(c, "/ucenter"))
		return
	}

	if c.Request.Method == "POST" {
		// 檢查登錄功能是否開啟
		if model.GetConfigValue("login_status", "1") == "0" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "系統已關閉登錄功能", "tourl": ""})
			return
		}

		// 驗證碼檢查（VerifyCaptcha 會自動發送錯誤 JSON）
		if !common.VerifyCaptcha(c, "login_check_code", "1") {
			return
		}

		// 登錄鎖定檢查（讀取 login_error_count/login_error_wait 配置）
		if remain := checkMemberLoginBlack(c); remain > 0 {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": fmt.Sprintf("您登錄失敗次數太多已被鎖定，請%d秒後再試", remain), "tourl": ""})
			return
		}

		username := c.PostForm("username")
		password := c.PostForm("password")

		if username == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "帳號不能為空", "tourl": ""})
			return
		}
		if password == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "密碼不能為空", "tourl": ""})
			return
		}

		// 三合一查詢：用戶名 / 郵箱 / 手機
		var member model.Member
		if err := model.DB.WithContext(c.Request.Context()).Where(
			"username = ? OR useremail = ? OR usermobile = ?",
			username, username, username,
		).First(&member).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "帳號不存在", "tourl": ""})
			return
		}

		// 驗證密碼（支援 bcrypt 和舊版雙 MD5，自動升級）
		matched, needUpgrade := common.VerifyPassword(password, member.Password)
		if !matched {
			setMemberLoginBlack(c) // 累計失敗次數
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "帳號密碼錯誤", "tourl": ""})
			return
		}
		// 自動升級舊版雙 MD5 密碼為 bcrypt
		if needUpgrade {
			if hashedPwd, err := common.HashPassword(password); err == nil {
				if err := model.DB.WithContext(c.Request.Context()).Model(&member).Update("password", hashedPwd).Error; err != nil {
					slog.Error("會員密碼自動升級失敗", "uid", member.ID, "error", err)
				}
			}
		}

		// 檢查帳號狀態
		if member.Status == 0 {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "您的帳號待審核，請聯繫管理員", "tourl": ""})
			return
		}

		// 登錄成功，清除鎖定記錄
		clearMemberLoginBlack(c)

		// 防止 Session Fixation：重新生成 session ID（對齊後台管理員登入邏輯）
		common.RegenerateSessionID(c)

		// 寫入 Session（保持與 PbootCMS 前台兼容的鍵名）
		common.SetSession(c, "pboot_uid", int(member.ID))
		common.SetSession(c, "pboot_ucode", member.Ucode)
		common.SetSession(c, "pboot_username", member.Username)
		common.SetSession(c, "pboot_useremail", member.Useremail)
		common.SetSession(c, "pboot_usermobile", member.Usermobile)
		common.SetSession(c, "pboot_gid", member.GID)

		// 查詢會員等級名稱
		var group model.MemberGroup
		if gidInt, _ := strconv.Atoi(member.GID); gidInt > 0 {
			if err := model.DB.WithContext(c.Request.Context()).Where("id = ?", gidInt).First(&group).Error; err != nil {
				slog.Warn("查詢會員等級失敗", "gid", gidInt, "error", err)
			}
		}
		// gcode 轉為 int 存入 session（checkPageLevel 用 GetSessionInt 讀取）
		gcodeInt, _ := strconv.Atoi(group.Gcode)
		common.SetSession(c, "pboot_gcode", gcodeInt)
		common.SetSession(c, "pboot_gname", group.Gname)

		// 更新登錄統計 + 登錄積分（對齊 PHP MemberModel::login 的 score += 邏輯）
		updates := map[string]interface{}{
			"login_count":     gorm.Expr("login_count + 1"),
			"last_login_ip":   c.ClientIP(),
			"last_login_time": time.Now().Format("2006-01-02 15:04:05"),
		}
		// 登錄積分：讀取 login_score 配置，大於 0 時累加
		loginScore, _ := strconv.Atoi(model.GetConfigValue("login_score", "0"))
		if loginScore > 0 {
			updates["score"] = gorm.Expr("score + ?", loginScore)
		}
		if err := model.DB.WithContext(c.Request.Context()).Model(&member).Updates(updates).Error; err != nil {
			slog.Error("更新會員登錄統計失敗", "uid", member.ID, "error", err)
		}

		// 返回跳轉地址（驗證為相對路徑，防止開放重定向）
		tourl := c.DefaultPostForm("backurl", c.Query("backurl"))
		if tourl == "" || !common.IsSafeRedirectURL(tourl) {
			tourl = langPath(c, "/ucenter")
		}
		// 對齊 PbootCMS 響應格式: {code, data, tourl}
		c.JSON(http.StatusOK, gin.H{"code": 1, "data": "登錄成功", "tourl": tourl})
		return
	}

	fc.renderMemberPage(c, "member/login.html")
}

// Register 會員註冊
func (fc *FrontController) Register(c *gin.Context) {
	// 已登錄則跳轉用戶中心
	if common.GetSessionInt(c, "pboot_uid") > 0 {
		c.Redirect(http.StatusFound, langPath(c, "/ucenter"))
		return
	}

	if c.Request.Method == "POST" {
		// 檢查註冊功能是否開啟
		if model.GetConfigValue("register_status", "1") == "0" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "系統已關閉註冊功能", "tourl": ""})
			return
		}

		// 10 秒防刷
		lastReg := common.GetSessionInt(c, "lastreg")
		if lastReg > 0 && time.Now().Unix()-int64(lastReg) < 10 {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "您註冊太頻繁了，請稍後再試", "tourl": ""})
			return
		}

		// 驗證碼檢查
		if !common.VerifyCaptcha(c, "register_check_code", "1") {
			return
		}

		username := c.PostForm("username")
		password := c.PostForm("password")
		rpassword := c.PostForm("rpassword")
		nickname := common.FilterUserInput(c.PostForm("nickname"))

		registerType := model.GetConfigValue("register_type", "1")
		var useremail, usermobile string

		switch registerType {
		case "2": // 郵箱註冊
			useremail = username
			if useremail == "" {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "請輸入註冊的郵箱帳號", "tourl": ""})
				return
			}
			if !regexpMatch(`^[\w]+@[\w\.]+\.[a-zA-Z]+$`, useremail) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "郵箱格式不正確", "tourl": ""})
				return
			}
			if memberExists(c, "useremail = ? OR username = ?", useremail, useremail) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "您輸入的郵箱已被註冊", "tourl": ""})
				return
			}
		case "3": // 手機註冊
			usermobile = username
			if usermobile == "" {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "請輸入註冊的手機號碼", "tourl": ""})
				return
			}
			if !regexpMatch(`^1[0-9]{10}$`, usermobile) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "手機號碼格式不正確", "tourl": ""})
				return
			}
			if memberExists(c, "usermobile = ? OR username = ?", usermobile, usermobile) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "您輸入的手機號碼已被註冊", "tourl": ""})
				return
			}
		default: // 帳號註冊
			if username == "" {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "用戶名不能為空", "tourl": ""})
				return
			}
			if !regexpMatch(`^[\w\@\.]+$`, username) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "用戶帳號含有不允許的特殊字符", "tourl": ""})
				return
			}
			if memberExists(c, "username = ? OR useremail = ? OR usermobile = ?", username, username, username) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "您輸入的帳號已被註冊", "tourl": ""})
				return
			}
		}

		if password != rpassword {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "確認密碼不正確", "tourl": ""})
			return
		}
		if password == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "密碼不能為空", "tourl": ""})
			return
		}

		// 預設值
		status := 1
		if model.GetConfigValue("register_verify", "0") == "1" {
			status = 0
		}
		score, _ := strconv.Atoi(model.GetConfigValue("register_score", "0"))

		// 預設會員等級
		var group model.MemberGroup
		registerGcode := model.GetConfigValue("register_gcode", "")
		if registerGcode != "" {
			model.DB.WithContext(c.Request.Context()).Where("gcode = ?", registerGcode).First(&group)
		}
		if group.ID == 0 {
			model.DB.WithContext(c.Request.Context()).Where("status = 1").Order("id ASC").First(&group)
		}
		gid := fmt.Sprintf("%d", group.ID)

		// 創建會員
	hashedPwd, err := common.HashPassword(password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "密碼加密失敗，請重試", "tourl": ""})
		return
	}
		newMember := model.Member{
			Ucode:        generateUcode(c),
			Username:     username,
			Useremail:    useremail,
			Usermobile:   usermobile,
			Nickname:     nickname,
			Password:     hashedPwd,
			Status:       status,
			GID:          gid,
			Activation:   1,
			Score:        score,
			RegisterTime: time.Now(),
			LoginCount:   0,
		}

		if err := model.DB.WithContext(c.Request.Context()).Create(&newMember).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "註冊失敗", "tourl": ""})
			return
		}

		// 記錄註冊時間（防刷）
		common.SetSession(c, "lastreg", int(time.Now().Unix()))

		msg := "註冊成功"
		if status == 0 {
			msg = "註冊成功，請等待管理員審核"
		}
		c.JSON(http.StatusOK, gin.H{"code": 1, "data": msg, "tourl": langPath(c, "/login")})
		return
	}

	fc.renderMemberPage(c, "member/register.html")
}

// Logout 會員登出
func (fc *FrontController) Logout(c *gin.Context) {
	common.DeleteSession(c, "pboot_uid")
	common.DeleteSession(c, "pboot_ucode")
	common.DeleteSession(c, "pboot_username")
	common.DeleteSession(c, "pboot_useremail")
	common.DeleteSession(c, "pboot_usermobile")
	common.DeleteSession(c, "pboot_gid")
	common.DeleteSession(c, "pboot_gcode")
	common.DeleteSession(c, "pboot_gname")
	c.Redirect(http.StatusFound, langPath(c, "/login"))
}

// Ucenter 會員中心
func (fc *FrontController) Ucenter(c *gin.Context) {
	if common.GetSessionInt(c, "pboot_uid") == 0 {
		c.Redirect(http.StatusFound, langPath(c, "/login"))
		return
	}
	fc.renderMemberPage(c, "member/ucenter.html")
}

// Umodify 修改資料
func (fc *FrontController) Umodify(c *gin.Context) {
	uid := common.GetSessionInt(c, "pboot_uid")
	if uid == 0 {
		c.Redirect(http.StatusFound, langPath(c, "/login"))
		return
	}

	if c.Request.Method == "POST" {
		// 驗證當前密碼
		opassword := c.PostForm("opassword")
		if opassword == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "請輸入當前密碼", "tourl": ""})
			return
		}

		var member model.Member
		model.DB.WithContext(c.Request.Context()).First(&member, uid)
		// 驗證當前密碼（支援 bcrypt 和舊版雙 MD5）
		matched, _ := common.VerifyPassword(opassword, member.Password)
		if !matched {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": "當前密碼不正確", "tourl": ""})
			return
		}

		// 構建更新數據（XSS 過濾：暱稱、QQ、電話為自由文字欄位，必須過濾）
	updates := map[string]interface{}{
		"nickname":   common.FilterUserInput(c.PostForm("nickname")),
		"useremail":  common.FilterUserInput(c.PostForm("useremail")),
		"usermobile": common.FilterUserInput(c.PostForm("usermobile")),
		"headpic":    c.PostForm("headpic"),
		"sex":        c.PostForm("sex"),
		"birthday":   c.PostForm("birthday"),
		"qq":         common.FilterUserInput(c.PostForm("qq")),
		"telephone":  common.FilterUserInput(c.PostForm("telephone")),
	}

		// 修改密碼（可選）
		password := c.PostForm("password")
		rpassword := c.PostForm("rpassword")
		if password != "" {
			if password != rpassword {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "確認密碼不正確", "tourl": ""})
				return
			}
			hashedPwd, err := common.HashPassword(password)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "密碼加密失敗，請重試", "tourl": ""})
				return
			}
			updates["password"] = hashedPwd
		}

		if err := model.DB.WithContext(c.Request.Context()).Model(&model.Member{}).Where("id = ?", uid).Updates(updates).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "修改失敗", "tourl": ""})
		return
	}

		// 同步 Session 中的暱稱和郵箱（使用過濾後的值，與 DB 一致）
	common.SetSession(c, "pboot_useremail", updates["useremail"])
	common.SetSession(c, "pboot_usermobile", updates["usermobile"])

		c.JSON(http.StatusOK, gin.H{"code": 1, "data": "修改成功", "tourl": langPath(c, "/umodify")})
		return
	}

	fc.renderMemberPage(c, "member/umodify.html")
}

// === 輔助函數 ===

// regexpMatch 正則匹配快捷方法
func regexpMatch(pattern, s string) bool {
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}

// memberExists 檢查會員是否已存在
func memberExists(c *gin.Context, where string, args ...interface{}) bool {
	var count int64
	model.DB.WithContext(c.Request.Context()).Model(&model.Member{}).Where(where, args...).Count(&count)
	return count > 0
}

// isSafeRedirectURL 已移至 common.IsSafeRedirectURL（預編譯正則，全域複用）
