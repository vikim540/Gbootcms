package admin

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"pbootcms-go/config"
	"regexp"
	"runtime"
	"strings"
	"time"

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

	if err := model.DB.Where("username = ? AND password = ? AND status = 1", username, encPwd).First(&user).Error; err != nil {
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
	acode := ""
	if len(acodes) > 0 {
		acode = acodes[0]
	}

	var levels []string
	if user.Rcodes != "" {
		rcodeList := strings.Split(user.Rcodes, ",")
		var roleLevels []model.RoleLevel
		model.DB.Where("rcode IN ?", rcodeList).Find(&roleLevels)
		for _, rl := range roleLevels {
			levels = append(levels, rl.URL)
		}
	}

	var areas []model.Area
	model.DB.Find(&areas)
	areaMap := make(map[string]string)
	for _, a := range areas {
		areaMap[a.Code] = a.Name
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
		"acode":            acode,
		"levels":           levels,
		"area_map":         areaMap,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "PbootGo",
		Value:    newSessionID,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: false,
		Secure:   false,
	})

	model.DB.Model(&user).Updates(map[string]interface{}{
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
	model.DB.Where("ucode = ?", ucode).First(&user)

	var msgCount int64
	model.DB.Model(&model.Message{}).Where("status = 0").Count(&msgCount)

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
		"UploadMaxFilesize": "50M",
		"PostMaxSize":       "50M",
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

	data := gin.H{
		"C":              "Index",
		"URL":            "/admin/index/home",
		"PrimaryMenuURL": "/admin/index/home",
		"OneArea":        true,
		"DBSecurity":     dbsecurity,
		"AreaHtml":       "",
		"User":           user,
		"UserInfo":       user,
		"SumMsg":         msgCount,
		"Server":         serverInfo,
		"Branch":         branch,
		"Revise":         revise,
		"Snuser":         snuser,
		"Site":           site,
		"ModelMsg":       buildModelMsg(models),
		"ModelCounts":    buildModelCounts(),
	}

	common.Render(c, "system/home.html", data)
}

func (ic *IndexController) Ucenter(c *gin.Context) {
	uid := common.GetSessionInt(c, "admin_uid")
	var user model.AdminUser
	model.DB.First(&user, uid)
	common.Render(c, "system/ucenter.html", gin.H{"user": user})
}

func (ic *IndexController) UcenterMod(c *gin.Context) {
	uid := common.GetSessionInt(c, "admin_uid")

	oldPwd := c.PostForm("oldpassword")
	newPwd := c.PostForm("newpassword")
	rePwd := c.PostForm("repassword")

	var user model.AdminUser
	model.DB.First(&user, uid)

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
	model.DB.Model(&user).Update("password", encNew)
	ic.JSONOKMsg(c, common.NoticePassword)
}

func (ic *IndexController) ClearCache(c *gin.Context) {
	os.RemoveAll("runtime/cache")
	os.RemoveAll("runtime/compile")
	os.RemoveAll("runtime/config")
	os.MkdirAll("runtime/cache", 0755)
	os.MkdirAll("runtime/compile", 0755)
	os.MkdirAll("runtime/config", 0755)
	ic.JSONOKMsg(c, common.NoticeCacheCleaned)
}

func (ic *IndexController) Area(c *gin.Context) {
	code := c.PostForm("code")
	common.SetSession(c, "area_code", code)
	ic.JSONOKMsg(c, common.NoticeSwitch)
}

func (ic *IndexController) CheckCode(c *gin.Context) {
	common.GenerateCaptcha(c)
}

func (ic *IndexController) log(c *gin.Context, msg string) {
	username, _ := common.GetSession(c, "admin_username").(string)
	entry := model.Syslog{
		Username: username,
		Content:  msg,
		IP:       c.ClientIP(),
		URL:      c.Request.URL.Path,
	}
	model.DB.Create(&entry)
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
		HttpOnly: false,
		Secure:   false,
	})
}

func (ic *IndexController) generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

const (
	LoginLockTime   = 900
	LoginLockCount  = 5
)

func (ic *IndexController) checkLoginBlack(c *gin.Context) int {
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

		if count >= LoginLockCount {
			elapsed := time.Now().Unix() - timestamp
			remain := LoginLockTime - int(elapsed)
			if remain > 0 {
				return remain
			}
		}
	}
	return 0
}

func (ic *IndexController) setLoginBlack(c *gin.Context) {
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
		if entry["count"] < LoginLockCount && now-entry["time"] < LoginLockTime {
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

func buildModelMsg(models []model.ContentModel) []interface{} {
	var result []interface{}
	for _, m := range models {
		var count int64
		model.DB.Model(&model.Content{}).Where("scode IN (SELECT scode FROM ay_content_sort WHERE type = ?)", m.Type).Count(&count)
		result = append(result, gin.H{
			"Mcode": m.Mcode,
			"Name":  m.Name,
			"Type":  m.Type,
			"Count": count,
		})
	}
	return result
}

func buildModelCounts() gin.H {
	var total int64
	model.DB.Model(&model.Content{}).Count(&total)
	return gin.H{
		"Content": total,
	}
}

// Upload - File upload endpoint for layui upload component
func (ic *IndexController) Upload(c *gin.Context) {
	file, err := c.FormFile("upload")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "未接收到上传文件"})
		return
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".webp": true, ".avif": true, ".ico": true,
		".doc": true, ".docx": true, ".pdf": true,
		".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
		".rar": true, ".zip": true, ".7z": true,
		".mp3": true, ".mp4": true, ".avi": true, ".flv": true,
		".txt": true, ".csv": true,
	}
	if !allowedExts[ext] {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "不支持的文件类型：" + ext})
		return
	}

	// Validate file size (50MB)
	if file.Size > 50*1024*1024 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "文件大小超过50MB限制"})
		return
	}

	// Create upload directory: static/upload/YYYYMM/
	dateDir := time.Now().Format("200601")
	uploadDir := filepath.Join("static", "upload", dateDir)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "创建上传目录失败"})
		return
	}

	// Generate unique filename
	ts := time.Now().Format("20060102150405")
	randStr := fmt.Sprintf("%04d", rand.Intn(10000))
	newFilename := ts + "_" + randStr + ext
	savePath := filepath.Join(uploadDir, newFilename)

	// Save file
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "文件保存失败：" + err.Error()})
		return
	}

	// Return path relative to project root (layui expects this format)
	relPath := filepath.ToSlash(savePath)
	c.JSON(http.StatusOK, gin.H{"code": 1, "data": []string{relPath}})
}
