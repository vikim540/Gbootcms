package admin

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
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

var checkCodeStore = make(map[string]string)


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
	checkcode := c.PostForm("checkcode")

	sessionID := ic.getCookie(c, "PbootGo")

	if checkcode != "" {
		savedCode := checkCodeStore[sessionID]
		if savedCode == "" || strings.ToLower(checkcode) != strings.ToLower(savedCode) {
			ic.JSONFail(c, "验证码错误！")
			return
		}
	}

	if remainTime := ic.checkLoginBlack(c); remainTime > 0 {
		ic.JSONFail(c, fmt.Sprintf("登录失败次数过多，请%d秒后重试！", remainTime))
		return
	}

	if username == "" {
		ic.JSONFail(c, "用户名不能为空！")
		return
	}
	if password == "" {
		ic.JSONFail(c, "密码不能为空！")
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
	ic.JSONOKMsg(c, "密码修改成功！")
}

func (ic *IndexController) ClearCache(c *gin.Context) {
	os.RemoveAll("runtime/cache")
	os.RemoveAll("runtime/compile")
	os.RemoveAll("runtime/config")
	os.MkdirAll("runtime/cache", 0755)
	os.MkdirAll("runtime/compile", 0755)
	os.MkdirAll("runtime/config", 0755)
	ic.JSONOKMsg(c, "缓存清理成功！")
}

func (ic *IndexController) Area(c *gin.Context) {
	code := c.PostForm("code")
	common.SetSession(c, "area_code", code)
	ic.JSONOKMsg(c, "切换成功！")
}

func (ic *IndexController) CheckCode(c *gin.Context) {
	a := randInt(9) + 1
	b := randInt(9) + 1
	op := randInt(2)
	var expr string
	var answer int
	if op == 0 {
		expr = fmt.Sprintf("%d + %d = ?", a, b)
		answer = a + b
	} else {
		if a < b {
			a, b = b, a
		}
		expr = fmt.Sprintf("%d - %d = ?", a, b)
		answer = a - b
	}

	sessionID := ic.getCookie(c, "PbootGo")
	if sessionID == "" {
		sessionID = ic.generateSessionID()
		ic.setCookie(c, "PbootGo", sessionID, 86400)
	}
	checkCodeStore[sessionID] = fmt.Sprintf("%d", answer)

	width := 200
	height := 70
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	bgColor := color.RGBA{245, 250, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	for i := 0; i < 80; i++ {
		x := randInt(width)
		y := randInt(height)
		r := uint8(180 + randInt(60))
		g := uint8(180 + randInt(60))
		b2 := uint8(180 + randInt(60))
		img.Set(x, y, color.RGBA{r, g, b2, 255})
	}

	for i := 0; i < 4; i++ {
		x1 := randInt(width)
		y1 := randInt(height)
		x2 := randInt(width)
		y2 := randInt(height)
		lineColor := color.RGBA{uint8(100 + randInt(100)), uint8(100 + randInt(100)), uint8(100 + randInt(100)), 255}
		dx := x2 - x1
		if dx == 0 {
			dx = 1
		}
		steps := dx
		if steps < 0 {
			steps = -steps
		}
		if steps == 0 {
			steps = 1
		}
		for s := 0; s <= steps; s++ {
			x := x1 + s*dx/steps
			y := y1 + (y2-y1)*s/steps
			if x >= 0 && x < width && y >= 0 && y < height {
				img.Set(x, y, lineColor)
			}
		}
	}

	digitPatterns := map[byte][][]bool{
		'0': {
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, true, true},
			{true, false, true, false, true},
			{true, true, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
		'1': {
			{false, false, true, false, false},
			{false, true, true, false, false},
			{true, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{true, true, true, true, true},
		},
		'2': {
			{false, true, true, true, false},
			{true, false, false, false, true},
			{false, false, false, false, true},
			{false, false, true, true, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, true},
		},
		'3': {
			{false, true, true, true, false},
			{true, false, false, false, true},
			{false, false, false, false, true},
			{false, false, true, true, false},
			{false, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
		'4': {
			{false, false, false, true, false},
			{false, false, true, true, false},
			{false, true, false, true, false},
			{true, false, false, true, false},
			{true, true, true, true, true},
			{false, false, false, true, false},
			{false, false, false, true, false},
		},
		'5': {
			{true, true, true, true, true},
			{true, false, false, false, false},
			{true, true, true, true, false},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
		'6': {
			{false, true, true, true, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
		'7': {
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
		},
		'8': {
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
		'9': {
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{false, true, true, true, false},
		},
		' ': {
			{false, false},
			{false, false},
			{false, false},
			{false, false},
			{false, false},
			{false, false},
			{false, false},
		},
		'+': {
			{false, false, false, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, false, false, false},
		},
		'-': {
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, true, true, true, true},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
		},
		'=': {
			{false, false, false, false, false},
			{true, true, true, true, true},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, true, true, true, true},
			{false, false, false, false, false},
			{false, false, false, false, false},
		},
		'?': {
			{false, true, true, true, false},
			{true, false, false, false, true},
			{false, false, false, false, true},
			{false, false, true, true, false},
			{false, false, true, false, false},
			{false, false, false, false, false},
			{false, false, true, false, false},
		},
	}

	pixelSize := 6
	startX := 10
	startY := 8
	charSpacing := 0

	colors := []color.RGBA{
		{220, 50, 50, 255},
		{50, 120, 220, 255},
		{40, 160, 60, 255},
		{180, 100, 20, 255},
		{130, 40, 180, 255},
		{20, 150, 150, 255},
		{200, 60, 120, 255},
	}

	for ci, ch := range []byte(expr) {
		pattern, ok := digitPatterns[ch]
		if !ok {
			continue
		}

		charColor := colors[ci%len(colors)]
		rotateOffset := randInt(5) - 2

		for py, row := range pattern {
			for px, on := range row {
				if on {
					for sx := 0; sx < pixelSize; sx++ {
						for sy := 0; sy < pixelSize; sy++ {
							px2 := startX + px*pixelSize + sx
							py2 := startY + (py+rotateOffset)*pixelSize + sy
							if px2 >= 0 && px2 < width && py2 >= 0 && py2 < height {
								img.Set(px2, py2, charColor)
							}
						}
					}
				}
			}
		}
		patWidth := 0
		for _, row := range pattern {
			if len(row) > patWidth {
				patWidth = len(row)
			}
		}
		startX += patWidth*pixelSize + charSpacing + 4
	}

	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	png.Encode(c.Writer, img)
}

func randInt(max int) int {
	if max <= 0 {
		return 0
	}
	return rand.Intn(max)
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
