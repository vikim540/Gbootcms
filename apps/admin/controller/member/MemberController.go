package member

import (
	"crypto/md5"
	"fmt"
	"pbootcms-go/apps/admin/helper"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MemberController - 會員管理控制器
// 對應 PHP: apps/admin/controller/member/MemberController.php
type MemberController struct {
	common.BaseController
}

// Index - 會員列表（含新增Tab）
func (mb *MemberController) Index(c *gin.Context) {
	var members []model.Member
	model.DB.Order("register_time DESC, id DESC").Find(&members)
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
		"list":    true,
		"members": members,
		"groups":  groups,
		"C":       "member",
	})
}

// Add - 新增會員
func (mb *MemberController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		username := c.PostForm("username")
		password := c.PostForm("password")

		if username == "" {
			mb.JSONFail(c, "用戶帳號不能為空")
			return
		}
		if password == "" {
			mb.JSONFail(c, "密碼不能為空")
			return
		}

		// 檢查用戶名是否重複
		var count int64
		model.DB.Model(&model.Member{}).Where("username = ? OR useremail = ? OR usermobile = ?", username, username, username).Count(&count)
		if count > 0 {
			mb.JSONFail(c, "用戶名已經存在")
			return
		}

		// 雙MD5密碼
		firstMd5 := fmt.Sprintf("%x", md5.Sum([]byte(password)))
		encPwd := fmt.Sprintf("%x", md5.Sum([]byte(firstMd5)))

		// 生成ucode（基於最後一條記錄自增）
		var lastMember model.Member
		model.DB.Order("id DESC").First(&lastMember)
		ucode := "10001"
		if lastMember.ID > 0 {
			if n, err := strconv.Atoi(lastMember.Ucode); err == nil {
				ucode = fmt.Sprintf("%d", n+1)
			}
		}

		gid := c.PostForm("gid")
		if gid == "" {
			gid = "1"
		}
		score, _ := strconv.Atoi(c.DefaultPostForm("score", "0"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		model.DB.Create(&model.Member{
			Ucode:       ucode,
			Username:    username,
			Nickname:    c.PostForm("nickname"),
			Password:    encPwd,
			Useremail:   c.PostForm("useremail"),
			Usermobile:  c.PostForm("usermobile"),
			Headpic:     c.PostForm("headpic"),
			GID:         gid,
			Score:       score,
			Status:      status,
			Activation:  1,
			LoginCount:  0,
		})
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
				mb.JSONOKMsg(c, "啟用成功")
			} else {
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

	// 單欄位切換（狀態開關）
	field := c.Query("field")
	value := c.Query("value")
	if field != "" && value != "" {
		model.DB.Model(&model.Member{}).Where("id = ?", id).Update(field, value)
		c.Redirect(302, "/admin/member/index")
		return
	}

	if c.Request.Method == "POST" {
		username := c.PostForm("username")
		if username == "" {
			mb.JSONFail(c, "用戶帳號不能為空")
			return
		}

		// 檢查用戶名是否重複（排除自身）
		var count int64
		model.DB.Model(&model.Member{}).Where("(username = ? OR useremail = ? OR usermobile = ?) AND id <> ?", username, username, username, id).Count(&count)
		if count > 0 {
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
			pwdMd5 := fmt.Sprintf("%x", md5.Sum([]byte(password)))
			updates["password"] = fmt.Sprintf("%x", md5.Sum([]byte(pwdMd5)))
		}
		model.DB.Model(&model.Member{}).Where("id = ?", id).Updates(updates)
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

// Del - 刪除會員（支援批量刪除）
func (mb *MemberController) Del(c *gin.Context) {
	// 批量刪除
	if c.Request.Method == "POST" {
		list := c.PostFormArray("list[]")
		if len(list) > 0 {
			model.DB.Where("id IN ?", list).Delete(&model.Member{})
			mb.JSONOKMsg(c, common.NoticeDelete)
			return
		}
	}

	// 單個刪除
	idStr := c.Query("id")
	if idStr == "" {
		idStr = c.PostForm("id")
	}
	model.DB.Delete(&model.Member{}, idStr)
	mb.JSONOKMsg(c, common.NoticeDelete)
}
