package common

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"gbootcms/apps/admin/model"
)

// === Cloudflare Turnstile 驗證模組 ===
// 與 VerifyCaptcha 同構：configName 控制開關，關閉則直接放行。
// 開啟時從表單讀取 cf-turnstile-response token，向 Cloudflare siteverify 校驗。
// 官方文檔：https://developers.cloudflare.com/turnstile/get-started/server-side-validation/

const turnstileSiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// turnstileResponse Cloudflare siteverify 回應結構
type turnstileResponse struct {
	Success     bool     `json:"success"`
	ErrorCodes  []string `json:"error-codes"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	Action      string   `json:"action"`
	Cdata       string   `json:"cdata"`
}

// VerifyTurnstile 校驗 Turnstile token
// configName 為配置項名稱（如 message_turnstile、form_turnstile）
// defaultVal 為配置項不存在時的默認值（"0"=關閉，"1"=啟用）
// 通過返回 true（含已關閉的情況），失敗返回 false 並已寫入 JSON 響應
func VerifyTurnstile(c *gin.Context, configName, defaultVal string) bool {
	if model.GetConfigValue(configName, defaultVal) == "0" {
		return true // Turnstile 已關閉
	}

	token := c.PostForm("cf-turnstile-response")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "請完成人機驗證"})
		return false
	}

	secret := model.GetConfigValue("turnstile_secret", "")
	if secret == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Turnstile 密鑰未配置"})
		return false
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(turnstileSiteverifyURL, url.Values{
		"secret":   {secret},
		"response": {token},
		"remoteip": {c.ClientIP()},
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "驗證服務暫時不可用"})
		return false
	}
	defer resp.Body.Close()

	var result turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "驗證回應解析失敗"})
		return false
	}

	if !result.Success {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "人機驗證失敗，請重試"})
		return false
	}

	return true
}
