package member

import (
	"fmt"
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 會員格式驗證正則（對齊 PHP preg_match 規則）
var (
	memberUsernameRe = regexp.MustCompile(`^[\w@\.]+$`)
	memberEmailRe    = regexp.MustCompile(`^[\w]+@[\w\.]+\.[a-zA-Z]+$`)
	memberMobileRe   = regexp.MustCompile(`^1[0-9]{10}$`)
)

// MemberController - 會員管理控制器
// 對應 PHP: apps/admin/controller/member/MemberController.php
type MemberController struct {
	common.BaseController
}

// Index - 會員列表（含新增Tab）
func (mb *MemberController) Index(c *gin.Context) {
	// 分頁處理
	page, pageSize, offset := mb.Paginate(c)
	baseURL := "/admin/member/index"

	// 統計總記錄數
	var total int64
	model.DB.Model(&model.Member{}).Count(&total)

	var members []model.Member
	model.DB.Order("register_time DESC, id DESC").Offset(offset).Limit(pageSize).Find(&members)
	var groups []model.MemberGroup
	model.DB.Where("status = 1").Find(&groups)
	// 填充等級名稱
	groupMap := make(map[string]string)
	for _, g := range groups {
		groupMap[fmt.Sprintf("%d", g.ID)] = g.Gname
	}
	for i := range members {
		if gname, ok := groupMap[members[i].GID]; ok {
			members[i].Gname = gname
		}
		if !members[i].RegisterTime.IsZero() {
			members[i].RegisterTimeStr = members[i].RegisterTime.Format("2006-01-02 15:04:05")
		}
	}
	common.Render(c, "member/member.html", gin.H{
		"list":     true,
		"members":  members,
		"groups":   groups,
		"C":        "member",
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
	})
}

// Add - 新增會員
func (mb *MemberController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		username := c.PostForm("username")
		password := c.PostForm("password")

		if username == "" {
			mb.LogAction(c, "新增會員失敗")
			mb.JSONFail(c, "用戶帳號不能為空")
			return
		}
		// 用戶名格式驗證（對齊 PHP: preg_match('/^[\w\@\.]+$/', $username)）
		if !memberUsernameRe.MatchString(username) {
			mb.LogAction(c, "新增會員失敗")
			mb.JSONFail(c, "用戶帳號只能包含字母、數字、底線、@和句點")
			return
		}
		// 郵箱格式驗證（對齊 PHP: preg_match('/^[\w]+@[\w\.]+\.[a-zA-Z]+$/')）
		useremail := c.PostForm("useremail")
		if useremail != "" && !memberEmailRe.MatchString(useremail) {
			mb.LogAction(c, "新增會員失敗")
			mb.JSONFail(c, "郵箱格式不正確")
			return
		}
		// 手機號格式驗證（對齊 PHP: preg_match('/^1[0-9]{10}$/', $usermobile)）
		usermobile := c.PostForm("usermobile")
		if usermobile != "" && !memberMobileRe.MatchString(usermobile) {
			mb.LogAction(c, "新增會員失敗")
			mb.JSONFail(c, "手機號格式不正確")
			return
		}
		if password == "" {
			mb.LogAction(c, "新增會員失敗")
			mb.JSONFail(c, "密碼不能為空")
			return
		}

		// 檢查用戶名是否重複
		var count int64
		model.DB.Model(&model.Member{}).Where("username = ? OR useremail = ? OR usermobile = ?", username, username, username).Count(&count)
		if count > 0 {
			mb.LogAction(c, "新增會員失敗")
			mb.JSONFail(c, "用戶名已經存在")
			return
		}

		// 使用 bcrypt 雜湊密碼（向後兼容舊版雙 MD5）
	hashedPwd, _ := common.HashPassword(password)

	// 生成 ucode（時間戳 + 隨機數，併發安全）
	ucode := fmt.Sprintf("%d%04d", time.Now().Unix()%1000000, common.SecureRandomInt(10000))

		gid := c.PostForm("gid")
		if gid == "" {
			gid = "1"
		}
		score, _ := strconv.Atoi(c.DefaultPostForm("score", "0"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		if err := model.DB.Create(&model.Member{
		Ucode:        ucode,
		Username:     username,
		Nickname:     c.PostForm("nickname"),
		Password:     hashedPwd,
		Useremail:    c.PostForm("useremail"),
		Usermobile:   c.PostForm("usermobile"),
		Headpic:      c.PostForm("headpic"),
		GID:          gid,
		Score:        score,
		Status:       status,
		Activation:   1,
		LoginCount:   0,
		RegisterTime: time.Now(),
	}).Error; err != nil {
		mb.LogAction(c, "新增會員失敗")
		mb.JSONFail(c, "新增失敗："+err.Error())
		return
	}
		mb.LogAction(c, "新增會員成功")
		mb.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	// GET 請求重定向到列表頁
	c.Redirect(302, "/admin/member/index")
}

// Mod - 修改會員（支援狀態切換 + 批量操作 + 完整修改）
// 路由: /admin/member/mod/:id 或 /admin/member/mod/*action
func (mb *MemberController) Mod(c *gin.Context) {
	// 批量操作（POST submit=verify1/verify0）
	if c.Request.Method == "POST" {
		submit := c.PostForm("submit")
		if submit == "verify1" || submit == "verify0" {
			list := c.PostFormArray("list[]")
			if len(list) == 0 {
				mb.JSONFail(c, "請選擇要操作的會員")
				return
			}
			status := 1
			if submit == "verify0" {
				status = 0
			}
			model.DB.Model(&model.Member{}).Where("id IN ?", list).Update("status", status)
			if status == 1 {
				mb.LogAction(c, "會員批量啟用成功")
				mb.JSONOKMsg(c, "啟用成功")
			} else {
				mb.LogAction(c, "會員批量禁用成功")
				mb.JSONOKMsg(c, "禁用成功")
			}
			return
		}
	}

	// 取得ID（支援 :id 參數和 *action 通配符）
	idStr := c.Param("id")
	if idStr == "" {
		params := helper.ParseWildcardAction(c.Param("action"))
		idStr = params["id"]
	}
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	// 單欄位切換（狀態開關）— 使用 JSON 回應（對齊 class="switch" AJAX 攔截）
	field := c.Query("field")
	value := c.Query("value")
	if field != "" && value != "" {
		if err := model.DB.Model(&model.Member{}).Where("id = ?", id).Update(field, value).Error; err != nil {
			mb.JSONFail(c, "操作失敗")
			return
		}
		mb.JSONOKMsg(c, common.NoticeModify)
		return
	}

	if c.Request.Method == "POST" {
		username := c.PostForm("username")
		if username == "" {
			mb.LogAction(c, "修改會員失敗")
			mb.JSONFail(c, "用戶帳號不能為空")
			return
		}
		// 用戶名格式驗證（對齊 PHP: preg_match('/^[\w\@\.]+$/', $username)）
		if !memberUsernameRe.MatchString(username) {
			mb.LogAction(c, "修改會員失敗")
			mb.JSONFail(c, "用戶帳號只能包含字母、數字、底線、@和句點")
			return
		}
		// 郵箱格式驗證
		useremail := c.PostForm("useremail")
		if useremail != "" && !memberEmailRe.MatchString(useremail) {
			mb.LogAction(c, "修改會員失敗")
			mb.JSONFail(c, "郵箱格式不正確")
			return
		}
		// 手機號格式驗證
		usermobile := c.PostForm("usermobile")
		if usermobile != "" && !memberMobileRe.MatchString(usermobile) {
			mb.LogAction(c, "修改會員失敗")
			mb.JSONFail(c, "手機號格式不正確")
			return
		}

		// 檢查用戶名是否重複（排除自身）
		var count int64
		model.DB.Model(&model.Member{}).Where("(username = ? OR useremail = ? OR usermobile = ?) AND id <> ?", username, username, username, id).Count(&count)
		if count > 0 {
			mb.LogAction(c, "修改會員失敗")
			mb.JSONFail(c, "用戶名已經存在")
			return
		}

		score, _ := strconv.Atoi(c.DefaultPostForm("score", "0"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		updates := map[string]interface{}{
			"username":   c.PostForm("username"),
			"nickname":   c.PostForm("nickname"),
			"useremail":  c.PostForm("useremail"),
			"usermobile": c.PostForm("usermobile"),
			"headpic":    c.PostForm("headpic"),
			"gid":        c.PostForm("gid"),
			"score":      score,
			"status":     status,
		}
		password := c.PostForm("password")
		if password != "" {
			hashedPwd, _ := common.HashPassword(password)
			updates["password"] = hashedPwd
		}
		if err := model.DB.Model(&model.Member{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		mb.LogAction(c, "修改會員失敗")
		mb.JSONFail(c, "修改失敗："+err.Error())
		return
	}
		mb.LogAction(c, "修改會員成功")
		mb.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// GET 載入修改頁面
	var member model.Member
	model.DB.First(&member, id)
	var groups []model.MemberGroup
	model.DB.Where("status = 1").Find(&groups)
	// 填充等級名稱
	for _, g := range groups {
		if fmt.Sprintf("%d", g.ID) == member.GID {
			member.Gname = g.Gname
			break
		}
	}
	common.Render(c, "member/member.html", gin.H{
		"mod":    true,
		"member": member,
		"groups": groups,
		"C":      "member",
	})
}

// Del - 刪除會員（支援批量刪除 + 單個刪除）
func (mb *MemberController) Del(c *gin.Context) {
	// 批量刪除
	if c.Request.Method == "POST" {
		list := c.PostFormArray("list[]")
		if len(list) > 0 {
			model.DB.Where("id IN ?", list).Delete(&model.Member{})
			mb.LogAction(c, "刪除會員成功")
			mb.JSONOKMsg(c, common.NoticeDelete)
			return
		}
	}

	// 單個刪除 — 支援 *action 通配符路徑: /del/id/123
	params := helper.ParseWildcardAction(c.Param("action"))
	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	if idStr == "" {
		idStr = c.PostForm("id")
	}
	if idStr == "" {
		mb.LogAction(c, "刪除會員失敗")
		mb.JSONFail(c, "缺少刪除目標ID")
		return
	}
	if err := model.DB.Delete(&model.Member{}, idStr).Error; err != nil {
		mb.LogAction(c, "刪除會員失敗")
		mb.JSONFail(c, "刪除失敗："+err.Error())
		return
	}
	mb.LogAction(c, "刪除會員成功")
	mb.JSONOKMsg(c, common.NoticeDelete)
}
