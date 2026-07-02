package controller

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"pbootcms-go/apps/common/parser"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// doubleMD5 雙重 MD5 加密，與 PbootCMS 用戶數據兼容
func doubleMD5(s string) string {
	first := fmt.Sprintf("%x", md5.Sum([]byte(s)))
	return fmt.Sprintf("%x", md5.Sum([]byte(first)))
}

// generateUcode 生成會員編碼（基於最後一條記錄自增）
func generateUcode() string {
	var lastMember model.Member
	model.DB.Order("id DESC").First(&lastMember)
	if lastMember.ID == 0 {
		return "10001"
	}
	n, _ := strconv.Atoi(lastMember.Ucode)
	if n == 0 {
		return "10001"
	}
	return fmt.Sprintf("%d", n+1)
}

// renderMemberPage 渲染會員頁面（與前台頁面使用相同的 parser 流程）
func (fc *FrontController) renderMemberPage(c *gin.Context, tpl string) {
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render(tpl)
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

// Login 會員登錄
func (fc *FrontController) Login(c *gin.Context) {
	// 已登錄則跳轉用戶中心
	if common.GetSessionInt(c, "pboot_uid") > 0 {
		c.Redirect(http.StatusFound, "/ucenter")
		return
	}

	if c.Request.Method == "POST" {
		// 檢查登錄功能是否開啟
		if model.GetConfigValue("login_status", "1") == "0" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "系統已關閉登錄功能"})
			return
		}

		// 驗證碼檢查（VerifyCaptcha 會自動發送錯誤 JSON）
		if !common.VerifyCaptcha(c, "login_check_code", "1") {
			return
		}

		username := c.PostForm("username")
		password := c.PostForm("password")

		if username == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "帳號不能為空"})
			return
		}
		if password == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "密碼不能為空"})
			return
		}

		// 三合一查詢：用戶名 / 郵箱 / 手機
		var member model.Member
		if err := model.DB.Where(
			"username = ? OR useremail = ? OR usermobile = ?",
			username, username, username,
		).First(&member).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "帳號不存在"})
			return
		}

		// 驗證密碼（雙重 MD5）
		if member.Password != doubleMD5(password) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "帳號密碼錯誤"})
			return
		}

		// 檢查帳號狀態
		if member.Status == 0 {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "您的帳號待審核，請聯繫管理員"})
			return
		}

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
			model.DB.Where("id = ?", gidInt).First(&group)
		}
		common.SetSession(c, "pboot_gcode", group.Gcode)
		common.SetSession(c, "pboot_gname", group.Gname)

		// 更新登錄統計
		model.DB.Model(&member).Updates(map[string]interface{}{
			"login_count":     member.LoginCount + 1,
			"last_login_ip":   c.ClientIP(),
			"last_login_time": time.Now().Format("2006-01-02 15:04:05"),
		})

		// 返回跳轉地址
		tourl := c.Query("backurl")
		if tourl == "" {
			tourl = "/ucenter"
		}
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "登錄成功", "tourl": tourl})
		return
	}

	fc.renderMemberPage(c, "member/login.html")
}

// Register 會員註冊
func (fc *FrontController) Register(c *gin.Context) {
	// 已登錄則跳轉用戶中心
	if common.GetSessionInt(c, "pboot_uid") > 0 {
		c.Redirect(http.StatusFound, "/ucenter")
		return
	}

	if c.Request.Method == "POST" {
		// 檢查註冊功能是否開啟
		if model.GetConfigValue("register_status", "1") == "0" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "系統已關閉註冊功能"})
			return
		}

		// 10 秒防刷
		lastReg := common.GetSessionInt(c, "lastreg")
		if lastReg > 0 && time.Now().Unix()-int64(lastReg) < 10 {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "您註冊太頻繁了，請稍後再試"})
			return
		}

		// 驗證碼檢查
		if !common.VerifyCaptcha(c, "register_check_code", "1") {
			return
		}

		username := c.PostForm("username")
		password := c.PostForm("password")
		rpassword := c.PostForm("rpassword")
		nickname := c.PostForm("nickname")

		registerType := model.GetConfigValue("register_type", "1")
		var useremail, usermobile string

		switch registerType {
		case "2": // 郵箱註冊
			useremail = username
			if useremail == "" {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "請輸入註冊的郵箱帳號"})
				return
			}
			if !regexpMatch(`^[\w]+@[\w\.]+\.[a-zA-Z]+$`, useremail) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "郵箱格式不正確"})
				return
			}
			if memberExists("useremail = ? OR username = ?", useremail, useremail) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "您輸入的郵箱已被註冊"})
				return
			}
		case "3": // 手機註冊
			usermobile = username
			if usermobile == "" {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "請輸入註冊的手機號碼"})
				return
			}
			if !regexpMatch(`^1[0-9]{10}$`, usermobile) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "手機號碼格式不正確"})
				return
			}
			if memberExists("usermobile = ? OR username = ?", usermobile, usermobile) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "您輸入的手機號碼已被註冊"})
				return
			}
		default: // 帳號註冊
			if username == "" {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "用戶名不能為空"})
				return
			}
			if !regexpMatch(`^[\w\@\.]+$`, username) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "用戶帳號含有不允許的特殊字符"})
				return
			}
			if memberExists("username = ? OR useremail = ? OR usermobile = ?", username, username, username) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "您輸入的帳號已被註冊"})
				return
			}
		}

		if password != rpassword {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "確認密碼不正確"})
			return
		}
		if password == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "密碼不能為空"})
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
			model.DB.Where("gcode = ?", registerGcode).First(&group)
		}
		if group.ID == 0 {
			model.DB.Where("status = 1").Order("id ASC").First(&group)
		}
		gid := fmt.Sprintf("%d", group.ID)

		// 創建會員
		newMember := model.Member{
			Ucode:        generateUcode(),
			Username:     username,
			Useremail:    useremail,
			Usermobile:   usermobile,
			Nickname:     nickname,
			Password:     doubleMD5(password),
			Status:       status,
			GID:          gid,
			Activation:   1,
			Score:        score,
			RegisterTime: time.Now(),
			LoginCount:   0,
		}

		if err := model.DB.Create(&newMember).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "註冊失敗"})
			return
		}

		// 記錄註冊時間（防刷）
		common.SetSession(c, "lastreg", int(time.Now().Unix()))

		msg := "註冊成功"
		if status == 0 {
			msg = "註冊成功，請等待管理員審核"
		}
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": msg, "tourl": "/login"})
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
	c.Redirect(http.StatusFound, "/login")
}

// Ucenter 會員中心
func (fc *FrontController) Ucenter(c *gin.Context) {
	if common.GetSessionInt(c, "pboot_uid") == 0 {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	fc.renderMemberPage(c, "member/ucenter.html")
}

// Umodify 修改資料
func (fc *FrontController) Umodify(c *gin.Context) {
	uid := common.GetSessionInt(c, "pboot_uid")
	if uid == 0 {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if c.Request.Method == "POST" {
		// 驗證當前密碼
		opassword := c.PostForm("opassword")
		if opassword == "" {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "請輸入當前密碼"})
			return
		}

		var member model.Member
		model.DB.First(&member, uid)
		if member.Password != doubleMD5(opassword) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "當前密碼不正確"})
			return
		}

		// 構建更新數據
		updates := map[string]interface{}{
			"nickname":   c.PostForm("nickname"),
			"useremail":  c.PostForm("useremail"),
			"usermobile": c.PostForm("usermobile"),
			"headpic":    c.PostForm("headpic"),
			"sex":        c.PostForm("sex"),
			"birthday":   c.PostForm("birthday"),
			"qq":         c.PostForm("qq"),
			"telephone":  c.PostForm("telephone"),
		}

		// 修改密碼（可選）
		password := c.PostForm("password")
		rpassword := c.PostForm("rpassword")
		if password != "" {
			if password != rpassword {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "確認密碼不正確"})
				return
			}
			updates["password"] = doubleMD5(password)
		}

		model.DB.Model(&model.Member{}).Where("id = ?", uid).Updates(updates)

		// 同步 Session 中的暱稱和郵箱
		common.SetSession(c, "pboot_useremail", c.PostForm("useremail"))
		common.SetSession(c, "pboot_usermobile", c.PostForm("usermobile"))

		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "修改成功", "tourl": "/umodify"})
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
func memberExists(where string, args ...interface{}) bool {
	var count int64
	model.DB.Model(&model.Member{}).Where(where, args...).Count(&count)
	return count > 0
}
