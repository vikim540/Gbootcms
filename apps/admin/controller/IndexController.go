package admin

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"gbootcms/apps/common/watermark"
	"gbootcms/config"
	basic "gbootcms/core/basic"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gbootcms/core/acodeplugin"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 統一驗證碼已移至 apps/common/captcha.go，此處不再維護獨立存儲


type IndexController struct {
	common.BaseController
}

func (ic *IndexController) Index(c *gin.Context) {
	if common.GetSessionInt(c, "admin_uid") > 0 {
		c.Redirect(http.StatusFound, "/admin/index/home")
		return
	}
	common.Render(c, "index.html", nil)
}

func (ic *IndexController) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	// 統一驗證碼校驗（讀取 admin_check_code 配置，默認啟用）
	if !common.VerifyCaptcha(c, "admin_check_code", "1") {
		return
	}

	if remainTime := ic.checkLoginBlack(c); remainTime > 0 {
		ic.JSONFail(c, fmt.Sprintf("登錄失敗次數過多，請%d秒後重試！", remainTime))
		return
	}

	if username == "" {
		ic.JSONFail(c, "用戶名不能為空！")
		return
	}
	if password == "" {
		ic.JSONFail(c, "密碼不能為空！")
		return
	}

	var user model.AdminUser
	firstHash := md5.Sum([]byte(password))
	secondHash := md5.Sum([]byte(fmt.Sprintf("%x", firstHash)))
	encPwd := fmt.Sprintf("%x", secondHash)

	// 登入查詢必須跳過 acode 隔離（用戶可能屬於任何區域）
	loginCtx := acodeplugin.SkipAcode(c.Request.Context())
	if err := model.DB.WithContext(loginCtx).Where("username = ? AND password = ? AND status = 1", username, encPwd).First(&user).Error; err != nil {
		ic.setLoginBlack(c)
		ic.log(c, "登录失败!")
		ic.JSONFail(c, "用户名或密码错误！")
		return
	}

	ic.clearLoginBlack(c)

	firstHashSid := md5.Sum([]byte(fmt.Sprintf("%d%d", time.Now().UnixNano(), user.ID)))
	secondHashSid := md5.Sum([]byte(fmt.Sprintf("%x", firstHashSid)))
	sid := fmt.Sprintf("%x", secondHashSid)

	pwsecurity := encPwd != "c3284d0f94606de1fd2af172aba15bf3"

	acodes := strings.Split(user.Acodes, ",")
	if user.Acodes == "" {
		acodes = []string{}
	}

	var levels []string
	if user.Rcodes != "" {
		rcodeList := strings.Split(user.Rcodes, ",")
		var roleLevels []model.RoleLevel
		model.DB.WithContext(loginCtx).Where("rcode IN ?", rcodeList).Find(&roleLevels)
		for _, rl := range roleLevels {
			levels = append(levels, rl.URL)
		}
	}

	var areas []model.Area
	model.DB.WithContext(loginCtx).Find(&areas)
	areaMap := make(map[string]string)
	for _, a := range areas {
		areaMap[a.Acode] = a.Name
	}

	// 初始 acode：優先選擇用戶有權限的默認區域（is_default=1），否則取第一個
	acode := ""
	if len(acodes) > 0 {
		acode = acodes[0]
		for _, a := range areas {
			if a.IsDefault == "1" {
				for _, uac := range acodes {
					if uac == a.Acode {
						acode = a.Acode
						break
					}
				}
				break
			}
		}
	}

	newSessionID := ic.generateSessionID()
	common.SetSessionData(c, newSessionID, map[string]interface{}{
		"sid":              sid,
		"admin_uid":         user.ID,
		"admin_username":   user.Username,
		"admin_realname":   user.Realname,
		"admin_ucode":      user.Ucode,
		"admin_rcodes":     user.Rcodes,
		"pwsecurity":       pwsecurity,
		"acodes":           acodes,
		"user_acodes":      strings.Join(acodes, ","),
		"acode":            acode,
		"levels":           levels,
		"area_map":         areaMap,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "PbootGo",
		Value:    newSessionID,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   false,
	})

	model.DB.WithContext(loginCtx).Model(&user).Updates(map[string]interface{}{
		"login_count":    gorm.Expr("login_count + 1"),
		"last_login_ip":  c.ClientIP(),
		"lastlogintime":  time.Now(),
	})

	ic.log(c, "登录成功!")
	ic.JSONOK(c, "/admin/index/home")
}

func (ic *IndexController) LoginOut(c *gin.Context) {
	sessionID := ic.getCookie(c, "PbootGo")
	if sessionID != "" {
		common.DeleteSessionData(sessionID)
	}
	ic.setCookie(c, "PbootGo", "", -1)
	c.Redirect(http.StatusFound, "/admin/")
}

func (ic *IndexController) Home(c *gin.Context) {
	if c.Query("action") == "moddb" {
		if ic.modDB(c) {
			ic.log(c, "自动修改数据库名成功！")
		}
	}

	if deldb, ok := common.GetSession(c, "deldb").(string); ok && deldb != "" {
		os.Remove(deldb)
		common.DeleteSessionKey(c, "deldb")
	}

	dbsecurity := true
	dbType := model.GetConfigValue("database.type", "sqlite")
	if dbType == "" || dbType == "sqlite" {
		dbPath := model.GetDBName()
		if strings.Contains(dbPath, "pbootcms") {
			clientIP := c.ClientIP()
			if clientIP != "127.0.0.1" && clientIP != "::1" {
				if ic.modDB(c) {
					dbsecurity = true
				} else {
					dbsecurity = false
				}
			}
		} else {
			defaultDB := "data/pbootcms.db"
			if _, err := os.Stat(defaultDB); err == nil {
				newName := fmt.Sprintf("data/%s.db", fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))))
				os.Rename(defaultDB, newName)
			}
		}
	} else {
		defaultDB := "data/pbootcms.db"
		if _, err := os.Stat(defaultDB); err == nil {
			newName := fmt.Sprintf("data/%s.db", fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))))
			os.Rename(defaultDB, newName)
		}
	}

	ucode, _ := common.GetSession(c, "admin_ucode").(string)
	var user model.AdminUser
	// 用戶查詢跳過 acode 隔離（當前用戶可能屬於任何區域）
	model.DB.WithContext(acodeplugin.SkipAcode(c.Request.Context())).Where("ucode = ?", ucode).First(&user)

	// 留言計數：按當前區域過濾
	var msgCount int64
	model.DB.WithContext(c.Request.Context()).Model(&model.Message{}).Where("status = 0").Count(&msgCount)

	var models []model.ContentModel
	model.DB.Where("status = 1").Order("sorting ASC").Find(&models)

	serverInfo := gin.H{
		"PhpOs":             runtime.GOOS,
		"HttpHost":          c.Request.Host,
		"ServerName":        c.Request.Host,
		"ServerPort":        c.Request.URL.Port(),
		"ServerAddr":        c.Request.Host,
		"ServerSoftware":    "Go/Gin",
		"PhpVersion":        "Go " + runtime.Version(),
		"DbDriver":          strings.ToUpper(dbType),
		"UploadMaxFilesize": model.GetConfigValue("upload_max_size", "50") + "M",
		"PostMaxSize":       model.GetConfigValue("upload_max_size", "50") + "M",
	}

	branch := model.GetConfigValue("upgrade_branch", "3.X")
	if branch == "3.X.dev" {
		branch = "3.X.dev"
	} else {
		branch = "3.X"
	}
	revise := model.GetConfigValue("revise_version", "0")
	if revise == "" {
		revise = "0"
	}
	snuser := model.GetConfigValue("sn_user", "0")
	if snuser == "" {
		snuser = "0"
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	site := scheme + "://" + c.Request.Host

	// 區域切換數據已由 common.Render 全局注入（Areas/CurrentAcode/OneArea）

	data := gin.H{
		"C":              "Index",
		"URL":            "/admin/index/home",
		"PrimaryMenuURL": "/admin/index/home",
		"DBSecurity":     dbsecurity,
		"User":           user,
		"UserInfo":       user,
		"SumMsg":         msgCount,
		"Server":         serverInfo,
		"Branch":         branch,
		"Revise":         revise,
		"Snuser":         snuser,
		"Site":           site,
		"ModelMsg":       buildModelMsg(c, models),
		"ModelCounts":    buildModelCounts(c),
	}

	common.Render(c, "system/home.html", data)
}

func (ic *IndexController) Ucenter(c *gin.Context) {
	uid := common.GetSessionInt(c, "admin_uid")
	var user model.AdminUser
	model.DB.WithContext(acodeplugin.SkipAcode(c.Request.Context())).First(&user, uid)
	common.Render(c, "system/ucenter.html", gin.H{"user": user})
}

func (ic *IndexController) UcenterMod(c *gin.Context) {
	uid := common.GetSessionInt(c, "admin_uid")

	oldPwd := c.PostForm("oldpassword")
	newPwd := c.PostForm("newpassword")
	rePwd := c.PostForm("repassword")

	var user model.AdminUser
	model.DB.WithContext(acodeplugin.SkipAcode(c.Request.Context())).First(&user, uid)

	encOld1 := fmt.Sprintf("%x", md5.Sum([]byte(oldPwd)))
	encOld := fmt.Sprintf("%x", md5.Sum([]byte(encOld1)))
	if user.Password != encOld {
		ic.JSONFail(c, "原密码错误！")
		return
	}
	if newPwd != rePwd {
		ic.JSONFail(c, "两次密码输入不一致！")
		return
	}
	if len(newPwd) < 6 {
		ic.JSONFail(c, "密码长度不能少于6位！")
		return
	}

	encNew1 := fmt.Sprintf("%x", md5.Sum([]byte(newPwd)))
	encNew := fmt.Sprintf("%x", md5.Sum([]byte(encNew1)))
	model.DB.WithContext(acodeplugin.SkipAcode(c.Request.Context())).Model(&user).Update("password", encNew)
	ic.JSONOKMsg(c, common.NoticePassword)
}

// ClearCache 清理模板快取（內存 + debug 文件）
func (ic *IndexController) ClearCache(c *gin.Context) {
	// 清除內存中的模板快取（核心操作）
	basic.ClearTemplateCache()

	// 清除 runtime 目錄下的 pongo2 debug 文件
	debugFiles, _ := filepath.Glob("runtime/pongo2_debug_*.html")
	for _, f := range debugFiles {
		os.Remove(f)
	}

	// ajaxlink JS 讀 response.data，用 JSONOK 返回 data 欄位
	ic.JSONOK(c, common.NoticeCacheCleaned)
}

// ClearOnlySysCache 清理系統快取（與 ClearCache 語義相同，Go 版模板快取全在內存）
func (ic *IndexController) ClearOnlySysCache(c *gin.Context) {
	basic.ClearTemplateCache()
	ic.JSONOK(c, common.NoticeSystemCacheCleaned)
}

// ClearSession 清理所有會話（排除當前管理員）
func (ic *IndexController) ClearSession(c *gin.Context) {
	count := common.ClearAllSessions(c)
	ic.JSONOK(c, common.NoticeSessionCleaned(count))
}

func (ic *IndexController) Area(c *gin.Context) {
	code := c.PostForm("acode")

	// 權限驗證：檢查用戶是否有權切換到此區域
	// 對齊 PHP: IndexController::area() 中 in_array($acode, session('acodes'))
	userAcodes := common.GetSessionString(c, "user_acodes")
	if userAcodes != "" {
		allowed := false
		for _, a := range strings.Split(userAcodes, ",") {
			if strings.TrimSpace(a) == code {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(403, gin.H{"code": 0, "data": "", "msg": "無權限切換到此區域", "tourl": ""})
			return
		}
	}

	common.SetSession(c, "area_code", code)
	common.SetSession(c, "acode", code)
	c.JSON(200, gin.H{"code": 1, "data": common.NoticeSwitch, "msg": common.NoticeSwitch, "tourl": "/admin/Index/home"})
}

func (ic *IndexController) CheckCode(c *gin.Context) {
	common.GenerateCaptcha(c)
}

func (ic *IndexController) log(c *gin.Context, msg string) {
	// 嘗試從 session 取用戶名；登錄時 session 剛寫入可能讀不到，降級用表單值
	username, _ := common.GetSession(c, "admin_username").(string)
	if username == "" {
		username = c.PostForm("username")
	}
	if username == "" {
		username = "unknown"
	}

	ua := c.Request.UserAgent()
	chPlatformVer := c.GetHeader("Sec-CH-UA-Platform-Version")
	osName, browser := common.ParseUserAgent(ua, chPlatformVer)
	now := time.Now()

	entry := model.Syslog{
		// 原始 PbootCMS NOT NULL 欄位
		Level:      "admin",
		Event:      msg,
		UserIP:     c.ClientIP(),
		UserOS:     osName,
		UserBs:     browser,
		CreateUser: username,
		CreateTime: now.Format("2006-01-02 15:04:05"),
		// GORM 擴展欄位（模板顯示用）
		Username: username,
		URL:      c.Request.URL.Path,
		Content:  msg,
		IP:       c.ClientIP(),
		LogTime:  now,
	}
	if err := model.DB.Create(&entry).Error; err != nil {
		slog.Error("[Syslog] 寫入失敗", "err", err, "msg", msg)
	}
}

func (ic *IndexController) getCookie(c *gin.Context, name string) string {
	cookie, err := c.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie
}

func (ic *IndexController) setCookie(c *gin.Context, name, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   false,
	})
}

func (ic *IndexController) generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (ic *IndexController) checkLoginBlack(c *gin.Context) int {
	lockCount, _ := strconv.Atoi(model.GetConfigValue("lock_count", "5"))
	lockTime, _ := strconv.Atoi(model.GetConfigValue("lock_time", "900"))

	blackFile := filepath.Join("runtime", "data", fmt.Sprintf("%x.php", md5.Sum([]byte("login_black"))))
	if _, err := os.Stat(blackFile); os.IsNotExist(err) {
		return 0
	}

	data, err := os.ReadFile(blackFile)
	if err != nil {
		return 0
	}

	userIP := c.ClientIP()
	content := string(data)
	ipKey := fmt.Sprintf("'%s'", userIP)

	if strings.Contains(content, ipKey) {
		lines := strings.Split(content, "\n")
		inIPBlock := false
		var count, timestamp int64
		for _, line := range lines {
			if strings.Contains(line, ipKey) {
				inIPBlock = true
			}
			if inIPBlock {
				if strings.Contains(line, "'time'") {
					parts := strings.Split(line, "=>")
					if len(parts) == 2 {
						fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &timestamp)
					}
				}
				if strings.Contains(line, "'count'") {
					parts := strings.Split(line, "=>")
					if len(parts) == 2 {
						fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &count)
					}
				}
				if strings.Contains(line, "),") || strings.Contains(line, ")),") {
					break
				}
			}
		}

		if count >= int64(lockCount) {
			elapsed := time.Now().Unix() - timestamp
			remain := lockTime - int(elapsed)
			if remain > 0 {
				return remain
			}
		}
	}
	return 0
}

func (ic *IndexController) setLoginBlack(c *gin.Context) {
	lockCount, _ := strconv.Atoi(model.GetConfigValue("lock_count", "5"))
	lockTime, _ := strconv.Atoi(model.GetConfigValue("lock_time", "900"))

	blackFile := filepath.Join("runtime", "data", fmt.Sprintf("%x.php", md5.Sum([]byte("login_black"))))
	os.MkdirAll(filepath.Dir(blackFile), 0755)

	userIP := c.ClientIP()
	now := time.Now().Unix()

	existingData := make(map[string]map[string]int64)
	if data, err := os.ReadFile(blackFile); err == nil {
		content := string(data)
		lines := strings.Split(content, "\n")
		var currentIP string
		for _, line := range lines {
			if strings.Contains(line, "=>") && strings.Contains(line, "array(") {
				ipMatch := regexp.MustCompile(`'([^']+)'\s*=>\s*array\(`).FindStringSubmatch(line)
				if len(ipMatch) > 1 {
					currentIP = ipMatch[1]
					existingData[currentIP] = make(map[string]int64)
				}
			}
			if currentIP != "" {
				if strings.Contains(line, "'time'") {
					parts := strings.Split(line, "=>")
					if len(parts) == 2 {
						var t int64
						fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &t)
						existingData[currentIP]["time"] = t
					}
				}
				if strings.Contains(line, "'count'") {
					parts := strings.Split(line, "=>")
					if len(parts) == 2 {
						var cnt int64
						fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &cnt)
						existingData[currentIP]["count"] = cnt
					}
				}
			}
		}
	}

	if entry, ok := existingData[userIP]; ok {
		if entry["count"] < int64(lockCount) && now-entry["time"] < int64(lockTime) {
			entry["count"]++
			entry["time"] = now
		} else {
			entry["count"] = 1
			entry["time"] = now
		}
	} else {
		existingData[userIP] = map[string]int64{"time": now, "count": 1}
	}

	var buf strings.Builder
	buf.WriteString("<?php\nreturn array(\n")
	for ip, data := range existingData {
		buf.WriteString(fmt.Sprintf("    '%s' => array(\n", ip))
		buf.WriteString(fmt.Sprintf("        'time' => %d,\n", data["time"]))
		buf.WriteString(fmt.Sprintf("        'count' => %d,\n", data["count"]))
		buf.WriteString("    ),\n")
	}
	buf.WriteString(");")
	os.WriteFile(blackFile, []byte(buf.String()), 0644)
}

func (ic *IndexController) clearLoginBlack(c *gin.Context) {
	blackFile := filepath.Join("runtime", "data", fmt.Sprintf("%x.php", md5.Sum([]byte("login_black"))))
	if _, err := os.Stat(blackFile); os.IsNotExist(err) {
		return
	}

	userIP := c.ClientIP()
	data, err := os.ReadFile(blackFile)
	if err != nil {
		return
	}

	content := string(data)
	ipPattern := regexp.MustCompile(fmt.Sprintf(`\s*'%s'\s*=>\s*array\([^)]+\),?\n`, regexp.QuoteMeta(userIP)))
	content = ipPattern.ReplaceAllString(content, "")

	if strings.Contains(content, "return array(\n);") || strings.Contains(content, "return array();") {
		os.Remove(blackFile)
		return
	}

	os.WriteFile(blackFile, []byte(content), 0644)
}

func (ic *IndexController) modDB(c *gin.Context) bool {
	cfg := config.Get()
	sname := cfg.Database.DBName
	dname := fmt.Sprintf("data/%s.db", fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))))

	configFile := "config/config.json"
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return false
	}

	newConfigData := strings.ReplaceAll(string(configData), sname, dname)

	if err := os.WriteFile(configFile, []byte(newConfigData), 0644); err != nil {
		return false
	}

	if err := copyFile(sname, dname); err != nil {
		os.WriteFile(configFile, configData, 0644)
		return false
	}

	common.SetSession(c, "deldb", sname)

	cfg.Database.DBName = dname

	return true
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = srcFile.Seek(0, 0)
	if err != nil {
		return err
	}

	_, err = dstFile.ReadFrom(srcFile)
	return err
}

func buildModelMsg(c *gin.Context, models []model.ContentModel) []interface{} {
	var result []interface{}
	acode := acodeplugin.GetAcode(c.Request.Context())
	for _, m := range models {
		var count int64
		// 對齊 PHP: getModelCount() — JOIN ay_content_sort ON scode，WHERE mcode = ?
		// Raw SQL 繞過 AcodePlugin，需手動加 acode 條件（content 和 content_sort 都要過濾）
		model.DB.WithContext(c.Request.Context()).
			Raw("SELECT COUNT(*) FROM ay_content a LEFT JOIN ay_content_sort b ON a.scode = b.scode AND b.acode = a.acode WHERE b.mcode = ? AND a.acode = ?", m.Mcode, acode).
			Scan(&count)
		result = append(result, gin.H{
			"Mcode": m.Mcode,
			"Name":  m.Name,
			"Type":  m.Type,
			"Count": count,
		})
	}
	return result
}

func buildModelCounts(c *gin.Context) gin.H {
	var total int64
	model.DB.WithContext(c.Request.Context()).Model(&model.Content{}).Count(&total)
	return gin.H{
		"Content": total,
	}
}

// Upload - 檔案上傳端點（供 layui 上傳元件使用）
func (ic *IndexController) Upload(c *gin.Context) {
	file, err := c.FormFile("upload")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "未接收到上傳檔案"})
		return
	}

	// 驗證副檔名（從資料庫配置 home_upload_ext 讀取允許的副檔名清單）
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExtStr := model.GetConfigValue("home_upload_ext", "jpg,jpeg,png,gif,bmp,webp,avif,ico,doc,docx,pdf,xls,xlsx,ppt,pptx,rar,zip,7z,mp3,mp4,avi,flv,txt,csv")
	allowedExts := make(map[string]bool)
	for _, e := range strings.Split(allowedExtStr, ",") {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		allowedExts[e] = true
	}
	if !allowedExts[ext] {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "不支援的檔案類型：" + ext})
		return
	}

	// 驗證檔案大小（從資料庫配置 upload_max_size 讀取，單位 MB，預設 50MB）
	maxSizeMB, err := strconv.Atoi(model.GetConfigValue("upload_max_size", "50"))
	if err != nil || maxSizeMB <= 0 {
		maxSizeMB = 50
	}
	if file.Size > int64(maxSizeMB)*1024*1024 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": fmt.Sprintf("檔案大小超過%dMB限制", maxSizeMB)})
		return
	}

	// 建立上傳目錄：static/upload/YYYYMM/
	dateDir := time.Now().Format("200601")
	uploadDir := filepath.Join("static", "upload", dateDir)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "建立上傳目錄失敗"})
		return
	}

	// 產生唯一檔名
	ts := time.Now().Format("20060102150405")
	randStr := fmt.Sprintf("%04d", rand.Intn(10000))
	newFilename := ts + "_" + randStr + ext
	savePath := filepath.Join(uploadDir, newFilename)

	// 儲存檔案
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "檔案儲存失敗：" + err.Error()})
		return
	}

	// 對圖片應用水印（對齊 PHP watermark_img，僅圖片格式且水印開啟時生效）
	if isImageExt(ext) {
		if err := watermark.ApplyWatermark(savePath); err != nil {
			// 水印失敗不影響上傳，僅記錄日誌
			slog.Warn("水印處理失敗", "err", err)
		}
	}

	// 回傳相對於專案根目錄的路徑（layui 期望此格式）
	relPath := filepath.ToSlash(savePath)
	c.JSON(http.StatusOK, gin.H{"code": 1, "data": []string{relPath}})
}

// isImageExt 判斷是否為圖片副檔名
func isImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp":
		return true
	}
	return false
}
