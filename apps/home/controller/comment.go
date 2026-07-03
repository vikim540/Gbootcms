package controller

import (
	"net/http"
	"net/url"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"pbootcms-go/apps/common/mail"
	"pbootcms-go/apps/common/webhook"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CommentController 前台評論控制器
// 對應 PHP: apps/home/controller/CommentController.php
type CommentController struct {
	FrontController
}

// Add 提交評論（AJAX POST）
func (cc *CommentController) Add(c *gin.Context) {
	// 檢查評論功能是否開啟
	if model.GetConfigValue("comment_status", "1") == "0" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "評論功能已關閉"})
		return
	}

	// 10秒防刷
	lastsub := common.GetSessionInt(c, "lastsub")
	now := int(time.Now().Unix())
	if lastsub > 0 && now-lastsub < 10 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "操作太頻繁，請10秒後再試"})
		return
	}

	uid := common.GetSessionInt(c, "pboot_uid")

	// 登入檢查（未開啟匿名評論時必須登入）
	if uid == 0 && model.GetConfigValue("comment_anonymous", "0") == "0" {
		// 帶 backurl 讓登入後返回來源頁面
		referer := c.Request.Referer()
		loginURL := "/login"
		if referer != "" {
			loginURL = "/login?backurl=" + url.QueryEscape(referer)
		}
		c.JSON(http.StatusOK, gin.H{
			"code":  0,
			"data":  "請先登入後再評論",
			"tourl": loginURL,
		})
		return
	}

	// 驗證碼檢查
	if !common.VerifyCaptcha(c, "comment_check_code", "1") {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "驗證碼錯誤"})
		return
	}

	contentid := c.PostForm("contentid")
	if contentid == "" {
		contentid = c.Query("contentid")
	}
	comment := c.PostForm("comment")
	if comment == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "評論內容不能為空"})
		return
	}

	pid, _ := strconv.Atoi(c.DefaultPostForm("pid", "0"))
	puid, _ := strconv.Atoi(c.DefaultPostForm("puid", "0"))

	// 狀態判定：comment_verify=0 表示免審核
	status := 0
	if model.GetConfigValue("comment_verify", "1") == "0" {
		status = 1
	}

	// 寫入評論
	nowTime := time.Now()
	username := common.GetSessionString(c, "pboot_username")
	if username == "" {
		username = "guest"
	}
	mc := model.MemberComment{
		Pid:        uint(pid),
		Contentid:  uint(atoiSafe(contentid)),
		Comment:    comment,
		Uid:        uint(uid),
		Puid:       uint(puid),
		Likes:      0,
		Oppose:     0,
		Status:     status,
		UserIP:     c.ClientIP(),
		UserOS:     parseUserOS(c.Request.UserAgent()),
		UserBS:     parseUserBrowser(c.Request.UserAgent()),
		CreateTime: nowTime,
		UpdateUser: username,
		UpdateTime: nowTime,
	}
	if err := model.DB.Create(&mc).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "評論失敗，請稍後再試"})
		return
	}

	// 記錄提交時間（防刷）
	common.SetSession(c, "lastsub", now)

	// 評論郵件通知 + Webhook 推送
	notifyFields := []map[string]string{
		{"label": "評論內容", "value": comment},
		{"label": "來源IP", "value": c.ClientIP()},
		{"label": "作業系統", "value": parseUserOS(c.Request.UserAgent())},
		{"label": "瀏覽器", "value": parseUserBrowser(c.Request.UserAgent())},
	}
	if model.GetConfigValue("comment_send_mail", "0") == "1" {
		mail.SendNotifyMail("新評論通知", notifyFields)
	}
	webhook.SendIf("comment", "新評論", c.ClientIP(), parseUserOS(c.Request.UserAgent()), parseUserBrowser(c.Request.UserAgent()), notifyFields)

	if status == 1 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "data": "評論成功"})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": 1, "data": "評論已提交，待審核後顯示"})
	}
}

// My 我的評論列表頁
func (cc *CommentController) My(c *gin.Context) {
	uid := common.GetSessionInt(c, "pboot_uid")
	if uid == 0 {
		currentURL := c.Request.URL.String()
		c.Redirect(http.StatusFound, "/login?backurl="+url.QueryEscape(currentURL))
		return
	}
	cc.renderMemberPage(c, "member/mycomment.html")
}

// Del 刪除評論（AJAX GET）
func (cc *CommentController) Del(c *gin.Context) {
	uid := common.GetSessionInt(c, "pboot_uid")
	if uid == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "請先登入"})
		return
	}

	idStr := c.Query("id")
	if idStr == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "缺少參數"})
		return
	}

	// 安全刪除：只能刪除自己的評論
	result := model.DB.Where("id = ? AND uid = ?", idStr, uid).Delete(&model.MemberComment{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "刪除失敗，評論不存在或無權限"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 1, "data": "刪除成功"})
}

// atoiSafe 安全字串轉整數
func atoiSafe(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
