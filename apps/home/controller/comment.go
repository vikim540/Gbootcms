package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"gbootcms/apps/common/mail"
	"gbootcms/apps/common/webhook"
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
		loginURL := langPath(c, "/login")
		if referer != "" {
			loginURL = langPath(c, "/login") + "?backurl=" + url.QueryEscape(referer)
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
	commentUA := c.Request.UserAgent()
	commentOS, commentBS := common.ParseUserAgent(commentUA, c.GetHeader("Sec-CH-UA-Platform-Version"))
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
		UserOS:     commentOS,
		UserBS:     commentBS,
		CreateTime: nowTime,
		UpdateUser: username,
		UpdateTime: nowTime,
	}
	if err := model.DB.WithContext(c.Request.Context()).Create(&mc).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": "評論失敗，請稍後再試"})
		return
	}

	// 記錄提交時間（防刷）
	common.SetSession(c, "lastsub", now)

	// 評論郵件通知（comment_send_mail=1 時啟用，與 webhook 獨立判斷）
	commentIP := c.ClientIP()
	mailFields := []map[string]string{
		{"label": "評論內容", "value": comment},
		{"label": "來源IP", "value": commentIP},
		{"label": "作業系統", "value": commentOS},
		{"label": "瀏覽器", "value": commentBS},
	}
	if model.GetConfigValue("comment_send_mail", "0") == "1" {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("[Mail] 評論通知 goroutine panic: %v\n", r)
				}
			}()
			if err := mail.SendNotifyMail("新評論通知", mailFields); err != nil {
				model.LogNotify("mail", "error", "評論通知："+err.Error())
			} else {
				model.LogNotify("mail", "success", "評論通知郵件已發送")
			}
		}()
	}

	// Webhook 推送（獨立判斷，webhook_comment 開關在 SendIf 內檢查）
	webhookFields := []map[string]string{
		{"label": "評論內容", "value": comment},
	}
	webhook.SendIf("comment", "新評論", commentIP, commentOS, commentBS, webhookFields)

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
		backurl := langPath(c, currentURL)
		c.Redirect(http.StatusFound, langPath(c, "/login")+"?backurl="+url.QueryEscape(backurl))
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
	result := model.DB.WithContext(c.Request.Context()).Where("id = ? AND uid = ?", idStr, uid).Delete(&model.MemberComment{})
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
