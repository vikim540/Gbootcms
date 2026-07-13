package api

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	"gbootcms/apps/admin/model"
	"gbootcms/apps/admin/model/system"

	"github.com/gin-gonic/gin"
)

// Login API 登入，返回 JWT Token
// POST /api/v1/auth/login
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "msg": "請求參數錯誤"})
		return
	}

	// 查詢管理員
	var user system.AdminUser
	if err := model.DB.Where("username = ? AND status = 1", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "msg": "用戶名或密碼錯誤"})
		return
	}

	// 驗證密碼（雙 MD5 向後相容）
	firstMd5 := fmt.Sprintf("%x", md5.Sum([]byte(req.Password)))
	encPwd := fmt.Sprintf("%x", md5.Sum([]byte(firstMd5)))
	if user.Password != encPwd && user.Password != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "msg": "用戶名或密碼錯誤"})
		return
	}

	// 生成 JWT Token
	token, err := GenerateToken(int(user.ID), user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "msg": "Token 生成失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"msg":  "登入成功",
		"data": gin.H{
			"token":      token,
			"expires_in": 259200, // 72h in seconds
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"realname": user.Realname,
			},
		},
	})
}

// RefreshToken 刷新 JWT Token
// POST /api/v1/auth/refresh
func RefreshToken(c *gin.Context) {
	uid, exists := c.Get("api_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "msg": "未認證"})
		return
	}
	username, _ := c.Get("api_username")
	token, err := GenerateToken(uid.(int), username.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "msg": "Token 生成失敗"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"msg":  "刷新成功",
		"data": gin.H{
			"token":      token,
			"expires_in": 259200,
		},
	})
}

// apiResponse 統一 API 回應格式
type apiResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
	Meta *apiMeta    `json:"meta,omitempty"`
}

type apiMeta struct {
	Page     int   `json:"page"`
	Pagesize int   `json:"pagesize"`
	Total    int64 `json:"total"`
}

func apiOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, apiResponse{Code: 1, Msg: "success", Data: data})
}

func apiOKWithMeta(c *gin.Context, data interface{}, meta *apiMeta) {
	c.JSON(http.StatusOK, apiResponse{Code: 1, Msg: "success", Data: data, Meta: meta})
}

func apiFail(c *gin.Context, code int, msg string) {
	c.JSON(code, apiResponse{Code: 0, Msg: msg})
}

// parsePagination 解析分頁參數
func parsePagination(c *gin.Context) (int, int) {
	page := 1
	pagesize := 15
	if p := c.Query("page"); p != "" {
		if v, err := fmtAtoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("pagesize"); ps != "" {
		if v, err := fmtAtoi(ps); err == nil && v > 0 && v <= 100 {
			pagesize = v
		}
	}
	return page, pagesize
}

func fmtAtoi(s string) (int, error) {
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

// getCurrentTime 返回當前時間（用於格式化）
func getCurrentTime() time.Time {
	return time.Now()
}
