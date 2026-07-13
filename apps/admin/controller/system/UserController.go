package system

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"github.com/gin-gonic/gin"
)

// UserController 用戶管理控制器
// 對應 PHP: apps/admin/controller/system/UserController.php
type UserController struct {
	common.BaseController
}

// roleOption 角色下拉選項（含選中狀態）
type roleOption struct {
	Rcode    string `json:"rcode"`
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

// Index 用戶列表
func (uc *UserController) Index(c *gin.Context) {
	page, pageSize, offset := uc.Paginate(c)
	var total int64
	model.DB.Model(&model.AdminUser{}).Count(&total)
	var users []model.AdminUser
	model.DB.Order("id ASC").Offset(offset).Limit(pageSize).Find(&users)

	// 查詢所有角色，構建 rcode → name 映射
	var roles []model.Role
	model.DB.Find(&roles)
	roleMap := make(map[string]string)
	for _, r := range roles {
		roleMap[r.Rcode] = r.Name
	}

	// 填充角色名稱和預格式化時間
	for i := range users {
		if users[i].Ucode == "10001" {
			users[i].Rolename = "創始人"
		} else if users[i].Rcodes != "" {
			rcodes := strings.Split(users[i].Rcodes, ",")
			var names []string
			for _, rc := range rcodes {
				rc = strings.TrimSpace(rc)
				if name, ok := roleMap[rc]; ok {
					names = append(names, name)
				}
			}
			users[i].Rolename = strings.Join(names, ",")
		}
		if !users[i].CreateTime.IsZero() {
			users[i].CreateTimeStr = users[i].CreateTime.Format("2006-01-02 15:04:05")
		}
		if !users[i].UpdateTime.IsZero() {
			users[i].UpdateTimeStr = users[i].UpdateTime.Format("2006-01-02 15:04:05")
		}
		if !users[i].LastLoginTime.IsZero() {
			users[i].LastLoginTimeStr = users[i].LastLoginTime.Format("2006-01-02 15:04:05")
		}
	}

	// 構建角色選項（給新增表單用）
	opts := make([]roleOption, 0, len(roles))
	for _, r := range roles {
		opts = append(opts, roleOption{Rcode: r.Rcode, Name: r.Name})
	}

	baseURL := "/admin/system/user/index"
	common.Render(c, "system/user.html", gin.H{
		"list":     true,
		"users":    users,
		"roles":    opts,
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
	})
}

// Add 新增用戶
func (uc *UserController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		username := c.PostForm("username")
		if username == "" {
			uc.JSONFail(c, "用戶名不能為空！")
			return
		}

		// 檢查用戶名是否已存在
		var count int64
		model.DB.Model(&model.AdminUser{}).Where("username = ?", username).Count(&count)
		if count > 0 {
			uc.JSONFail(c, "用戶名已存在！")
			return
		}

		password := c.PostForm("password")
		if password == "" {
			uc.JSONFail(c, "密碼不能為空！")
			return
		}

		// 使用 bcrypt 雜湊密碼（與登入系統一致）
		encPwd, err := common.HashPassword(password)
		if err != nil {
			uc.JSONFail(c, "密碼加密失敗！")
			return
		}

		// 讀取角色（支援 roles[] 陣列或 rcodes 字串）
		rcodes := c.PostFormArray("roles[]")
		rcodesStr := strings.Join(rcodes, ",")

		// 生成 ucode
		var lastUser model.AdminUser
		model.DB.Order("id DESC").First(&lastUser)
		newUcode := autoUcode(lastUser.Ucode)

		now := time.Now()
		user := &model.AdminUser{
			Ucode:      newUcode,
			Username:   username,
			Password:   encPwd,
			Realname:   c.PostForm("realname"),
			Rcodes:     rcodesStr,
			Status:     1,
			CreateUser: uc.GetAdminUsername(c),
			UpdateUser: uc.GetAdminUsername(c),
			CreateTime: now,
			UpdateTime: now,
		}
		if err := model.DB.Create(user).Error; err != nil {
			uc.JSONFail(c, "新增失敗！")
			return
		}

		uc.LogAction(c, "新增用戶"+username+"成功")
		c.JSON(200, gin.H{"code": 1, "data": common.NoticeAdd, "msg": common.NoticeAdd, "tourl": "/admin/User/index"})
		return
	}

	// GET: 顯示新增表單
	var roles []model.Role
	model.DB.Find(&roles)
	opts := make([]roleOption, 0, len(roles))
	for _, r := range roles {
		opts = append(opts, roleOption{Rcode: r.Rcode, Name: r.Name})
	}
	common.Render(c, "system/user.html", gin.H{
		"add":   true,
		"roles": opts,
	})
}

// Mod 修改用戶
func (uc *UserController) Mod(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))
	ucode := params["ucode"]
	if ucode == "" {
		ucode = params["id"]
	}
	if ucode == "" {
		ucode = c.Query("ucode")
	}

	if ucode == "" {
		uc.JSONFail(c, "傳遞的參數值錯誤！")
		return
	}

	// 內置管理員保護
	if ucode == "10001" {
		field := params["field"]
		if field != "" {
			uc.JSONFail(c, "不允許此操作！")
			return
		}
	}

	// 單欄位切換（狀態開關）
	field := params["field"]
	value := params["value"]
	if field != "" && value != "" {
		updates := map[string]interface{}{
			field:        value,
			"update_user": uc.GetAdminUsername(c),
			"update_time": time.Now().Format("2006-01-02 15:04:05"),
		}
		if err := model.DB.Model(&model.AdminUser{}).Where("ucode = ?", ucode).Updates(updates).Error; err != nil {
			uc.JSONFail(c, "修改失敗！")
			return
		}
		uc.JSONOKMsg(c, common.NoticeModify)
		return
	}

	if c.Request.Method == "POST" {
		// 內置管理員保護
		if ucode == "10001" {
			uc.JSONFail(c, "不允許此操作！")
			return
		}

		updates := map[string]interface{}{
			"realname":    c.PostForm("realname"),
			"update_user": uc.GetAdminUsername(c),
			"update_time": time.Now().Format("2006-01-02 15:04:05"),
		}

		// 角色關聯
		rcodes := c.PostFormArray("roles[]")
		if len(rcodes) > 0 {
			updates["rcodes"] = strings.Join(rcodes, ",")
		}

		// 密碼（可選）
		password := c.PostForm("password")
		if password != "" {
			encPwd, err := common.HashPassword(password)
			if err != nil {
				uc.JSONFail(c, "密碼加密失敗！")
				return
			}
			updates["password"] = encPwd
		}

		if err := model.DB.Model(&model.AdminUser{}).Where("ucode = ?", ucode).Updates(updates).Error; err != nil {
			uc.JSONFail(c, "修改失敗！")
			return
		}

		uc.LogAction(c, "修改用戶"+ucode+"成功")
		c.JSON(200, gin.H{"code": 1, "data": common.NoticeModify, "msg": common.NoticeModify, "tourl": "/admin/User/index"})
		return
	}

	// GET: 顯示修改表單
	var user model.AdminUser
	model.DB.Where("ucode = ?", ucode).First(&user)
	if user.ID == 0 {
		uc.JSONFail(c, "編輯的內容已經不存在！")
		return
	}

	// 預格式化時間
	if !user.CreateTime.IsZero() {
		user.CreateTimeStr = user.CreateTime.Format("2006-01-02 15:04:05")
	}
	if !user.UpdateTime.IsZero() {
		user.UpdateTimeStr = user.UpdateTime.Format("2006-01-02 15:04:05")
	}
	if !user.LastLoginTime.IsZero() {
		user.LastLoginTimeStr = user.LastLoginTime.Format("2006-01-02 15:04:05")
	}

	// 構建角色選項（含選中狀態）
	var roles []model.Role
	model.DB.Find(&roles)
	userRcodes := make(map[string]bool)
	for _, rc := range strings.Split(user.Rcodes, ",") {
		rc = strings.TrimSpace(rc)
		if rc != "" {
			userRcodes[rc] = true
		}
	}
	opts := make([]roleOption, 0, len(roles))
	for _, r := range roles {
		opts = append(opts, roleOption{
			Rcode:    r.Rcode,
			Name:     r.Name,
			Selected: userRcodes[r.Rcode],
		})
	}

	common.Render(c, "system/user.html", gin.H{
		"mod":   true,
		"user":  user,
		"roles": opts,
	})
}

// Del 刪除用戶
func (uc *UserController) Del(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))
	ucode := params["id"]
	if ucode == "" {
		ucode = params["ucode"]
	}
	if ucode == "" {
		ucode = c.Query("ucode")
	}

	if ucode == "" {
		uc.JSONFail(c, "傳遞的參數值錯誤！")
		return
	}

	// 內置管理員保護
	if ucode == "10001" {
		uc.JSONFail(c, "不允許刪除！")
		return
	}

	result := model.DB.Where("ucode = ?", ucode).Delete(&model.AdminUser{})
	if result.RowsAffected == 0 {
		uc.JSONFail(c, "刪除失敗，用戶不存在！")
		return
	}

	uc.LogAction(c, "刪除用戶"+ucode+"成功")
	c.JSON(200, gin.H{"code": 1, "data": common.NoticeDelete, "msg": common.NoticeDelete, "tourl": "/admin/User/index"})
}

// autoUcode 自動生成用戶編碼（10001, 10002, ...）
func autoUcode(lastCode string) string {
	if lastCode == "" {
		return "10001"
	}
	num, err := strconv.Atoi(lastCode)
	if err != nil {
		return "10001"
	}
	return fmt.Sprintf("%05d", num+1)
}
