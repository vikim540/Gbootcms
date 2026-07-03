package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"pbootcms-go/apps/admin/model"
	"strings"
	"time"
)

// Payload 通用 Webhook 推送的 JSON 結構（自訂格式）
type Payload struct {
	FormName  string              `json:"form_name"`
	Timestamp string              `json:"timestamp"`
	IP        string              `json:"ip"`
	OS        string              `json:"os"`
	Browser   string              `json:"browser"`
	Fields    []map[string]string `json:"fields"`
}

// robotResponse 釘釘/企業微信的回應結構
type robotResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// Send 異步推送 Webhook 通知到配置的 URL（無查看詳情連結）
// 自動偵測釘釘/企業微信/通用格式
func Send(formName, ip, os, browser string, fields []map[string]string) {
	SendWithURL(formName, ip, os, browser, fields, "")
}

// SendIf 帶分項開關檢查的推送（category: message/form/comment）
// 自動根據 category 構造後台管理 URL 作為查看詳情連結
func SendIf(category, formName, ip, os, browser string, fields []map[string]string) {
	if model.GetConfigValue("webhook_"+category, "1") != "1" {
		return
	}
	SendWithURL(formName, ip, os, browser, fields, adminURLFor(category))
}

// SendWithURL 帶查看詳情 URL 的推送
func SendWithURL(formName, ip, os, browser string, fields []map[string]string, detailURL string) {
	webhookURL := model.GetConfigValue("webhook_url", "")
	if webhookURL == "" {
		return
	}

	go func() {
		var jsonData []byte
		var err error

		switch {
		case strings.Contains(webhookURL, "oapi.dingtalk.com"):
			jsonData, err = buildDingTalkActionCard(formName, ip, os, browser, fields, detailURL)
		case strings.Contains(webhookURL, "qyapi.weixin.qq.com"):
			jsonData, err = buildWeComPayload(formName, ip, os, browser, fields)
		default:
			payload := Payload{
				FormName:  formName,
				Timestamp: time.Now().Format("2006-01-02 15:04:05"),
				IP:        normalizeIP(ip),
				OS:        os,
				Browser:   browser,
				Fields:    fields,
			}
			jsonData, err = json.Marshal(payload)
		}

		if err != nil {
			fmt.Printf("[Webhook] JSON 序列化失敗: %v\n", err)
			return
		}

		req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("[Webhook] 建立請求失敗: %v\n", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "GbootCMS-Webhook/1.0")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[Webhook] 推送失敗: %v\n", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if strings.Contains(webhookURL, "oapi.dingtalk.com") || strings.Contains(webhookURL, "qyapi.weixin.qq.com") {
			var rr robotResponse
			if json.Unmarshal(body, &rr) == nil && rr.ErrCode != 0 {
				fmt.Printf("[Webhook] 機器人返回錯誤: errcode=%d, errmsg=%s\n", rr.ErrCode, rr.ErrMsg)
			}
		} else if resp.StatusCode >= 400 {
			fmt.Printf("[Webhook] 推送返回異常狀態碼: %d\n", resp.StatusCode)
		}
	}()
}

// adminURLFor 根據 category 返回後台管理頁面 URL
func adminURLFor(category string) string {
	baseURL := model.GetConfigValue("httpurl", "")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	switch category {
	case "comment":
		return baseURL + "/admin/member/comment/index"
	case "message":
		return baseURL + "/admin/content/message/index"
	case "form":
		return baseURL + "/admin/content/form/index"
	default:
		return baseURL + "/admin"
	}
}

// normalizeIP 將 ::1（IPv6 localhost）轉為 127.0.0.1
func normalizeIP(ip string) string {
	if ip == "::1" {
		return "127.0.0.1"
	}
	return ip
}

// buildDingTalkActionCard 構建釘釘 ActionCard 訊息（美觀卡片）
func buildDingTalkActionCard(formName, ip, os, browser string, fields []map[string]string, detailURL string) ([]byte, error) {
	ip = normalizeIP(ip)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("#### %s\n\n", formName))
	sb.WriteString(fmt.Sprintf("> **時間**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("> **IP**: %s  |  **系統**: %s  |  **瀏覽器**: %s\n\n---\n", ip, os, browser))
	for _, f := range fields {
		sb.WriteString(fmt.Sprintf("**%s**: %s\n\n", f["label"], f["value"]))
	}

	if detailURL == "" {
		detailURL = "#"
	}

	payload := map[string]interface{}{
		"msgtype": "actionCard",
		"actionCard": map[string]string{
			"title":       formName,
			"text":        sb.String(),
			"singleTitle": "查看詳情",
			"singleURL":   detailURL,
			"hideAvatar":  "0",
		},
	}
	return json.Marshal(payload)
}

// buildWeComPayload 構建企業微信機器人 Markdown 訊息
func buildWeComPayload(formName, ip, os, browser string, fields []map[string]string) ([]byte, error) {
	ip = normalizeIP(ip)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### %s\n\n", formName))
	sb.WriteString(fmt.Sprintf("**時間**: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**IP**: %s  |  **系統**: %s  |  **瀏覽器**: %s\n\n", ip, os, browser))
	for _, f := range fields {
		sb.WriteString(fmt.Sprintf("**%s**: %s\n", f["label"], f["value"]))
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": sb.String(),
		},
	}
	return json.Marshal(payload)
}
