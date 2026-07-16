package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"gbootcms/apps/admin/model"
	"gbootcms/apps/admin/model/system"
	"gbootcms/apps/common"
	"gbootcms/core/acodeplugin"

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
		apiFail(c, http.StatusBadRequest, "請求參數錯誤")
		return
	}

	// 檢查 JWT 密鑰是否已配置（提前檢查，避免無謂的密碼校驗）
	if !IsJWTConfigured() {
		apiFail(c, http.StatusInternalServerError, "API 未正確配置，請聯繫管理員設定 api_jwt_secret")
		return
	}

	// 登入鎖定檢查
	if remain := checkLoginLock(c); remain > 0 {
		apiFail(c, http.StatusTooManyRequests, fmt.Sprintf("登入嘗試過多，請 %d 秒後再試", remain))
		return
	}

	// 查詢管理員（跳過 acode 隔離，與後台登入邏輯一致）
	loginCtx := acodeplugin.SkipAcode(c.Request.Context())
	var user system.AdminUser
	if err := model.DB.WithContext(loginCtx).Where("username = ? AND status = 1", req.Username).First(&user).Error; err != nil {
		recordLoginFailure(c)
		apiFail(c, http.StatusUnauthorized, "用戶名或密碼錯誤")
		return
	}

	// 密碼校驗：支援 bcrypt 和舊版雙 MD5（自動升級），與後台登入完全一致
	matched, needUpgrade := common.VerifyPassword(req.Password, user.Password)
	if !matched {
		recordLoginFailure(c)
		apiFail(c, http.StatusUnauthorized, "用戶名或密碼錯誤")
		return
	}

	// 自動升級舊版雙 MD5 密碼為 bcrypt
	if needUpgrade {
		if hashedPwd, err := common.HashPassword(req.Password); err == nil {
			if err := model.DB.WithContext(loginCtx).Model(&user).Update("password", hashedPwd).Error; err != nil {
				slog.Error("API 密碼自動升級失敗", "uid", user.ID, "error", err)
			}
		}
	}

	// 登入成功，清除失敗記錄
	clearLoginFailure(c)

	// 生成 JWT Token
	token, err := GenerateToken(int(user.ID), user.Username)
	if err != nil {
		apiFail(c, http.StatusInternalServerError, "Token 生成失敗")
		return
	}

	apiOKWithMsg(c, "登入成功", gin.H{
		"token":      token,
		"expires_in": 259200, // 72h in seconds
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"realname": user.Realname,
		},
	})
}

// RefreshToken 刷新 JWT Token
// POST /api/v1/auth/refresh
func RefreshToken(c *gin.Context) {
	uid, exists := c.Get("api_uid")
	if !exists {
		apiFail(c, http.StatusUnauthorized, "未認證")
		return
	}
	username, _ := c.Get("api_username")
	token, err := GenerateToken(uid.(int), username.(string))
	if err != nil {
		apiFail(c, http.StatusInternalServerError, "Token 生成失敗")
		return
	}
	apiOKWithMsg(c, "刷新成功", gin.H{
		"token":      token,
		"expires_in": 259200,
	})
}

// apiResponse 統一 API 回應格式
type apiResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
	Meta *apiMeta    `json:"meta,omitempty"`
}

type apiMeta struct {
	Page       int   `json:"page"`
	Pagesize   int   `json:"pagesize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func apiOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, apiResponse{Code: 1, Msg: "success", Data: data})
}

func apiOKWithMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, apiResponse{Code: 1, Msg: msg, Data: data})
}

func apiOKWithMeta(c *gin.Context, data interface{}, meta *apiMeta) {
	// 計算總頁數
	if meta.Pagesize > 0 {
		meta.TotalPages = int((meta.Total + int64(meta.Pagesize) - 1) / int64(meta.Pagesize))
	}
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
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("pagesize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pagesize = v
		}
	}
	return page, pagesize
}
