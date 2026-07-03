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

// Send 異步推送 Webhook 通知到配置的 URL
// 自動偵測釘釘/企業微信/通用格式
func Send(formName, ip, os, browser string, fields []map[string]string) {
	webhookURL := model.GetConfigValue("webhook_url", "")
	if webhookURL == "" {
		return
	}

	// 異步發送，不阻塞用戶請求
	go func() {
		var jsonData []byte
		var err error

		switch {
		case strings.Contains(webhookURL, "oapi.dingtalk.com"):
			// 釘釘機器人
			jsonData, err = buildDingTalkPayload(formName, ip, os, browser, fields)
		case strings.Contains(webhookURL, "qyapi.weixin.qq.com"):
			// 企業微信機器人
			jsonData, err = buildWeComPayload(formName, ip, os, browser, fields)
		default:
			// 通用格式
			payload := Payload{
				FormName:  formName,
				Timestamp: time.Now().Format("2006-01-02 15:04:05"),
				IP:        ip,
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

		// 釘釘/企業微信恆回 HTTP 200，需檢查 JSON body 的 errcode
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

// SendIf 帶分項開關檢查的推送（category: message/form/comment）
func SendIf(category, formName, ip, os, browser string, fields []map[string]string) {
	if model.GetConfigValue("webhook_"+category, "1") != "1" {
		return
	}
	Send(formName, ip, os, browser, fields)
}

// buildDingTalkPayload 構建釘釘機器人 Markdown 訊息
func buildDingTalkPayload(formName, ip, os, browser string, fields []map[string]string) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("#### %s\n\n", formName))
	sb.WriteString(fmt.Sprintf("> **時間**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	if ip != "" {
		sb.WriteString(fmt.Sprintf("> **IP**: %s\n", ip))
	}
	if os != "" {
		sb.WriteString(fmt.Sprintf("> **系統**: %s\n", os))
	}
	if browser != "" {
		sb.WriteString(fmt.Sprintf("> **瀏覽器**: %s\n", browser))
	}
	sb.WriteString("\n")
	for _, f := range fields {
		sb.WriteString(fmt.Sprintf("> **%s**: %s\n", f["label"], f["value"]))
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": formName,
			"text":  sb.String(),
		},
		"at": map[string]interface{}{
			"isAtAll": false,
		},
	}
	return json.Marshal(payload)
}

// buildWeComPayload 構建企業微信機器人 Markdown 訊息
func buildWeComPayload(formName, ip, os, browser string, fields []map[string]string) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### %s\n\n", formName))
	sb.WriteString(fmt.Sprintf("**時間**: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	if ip != "" {
		sb.WriteString(fmt.Sprintf("**IP**: %s\n", ip))
	}
	if os != "" {
		sb.WriteString(fmt.Sprintf("**系統**: %s\n", os))
	}
	if browser != "" {
		sb.WriteString(fmt.Sprintf("**瀏覽器**: %s\n", browser))
	}
	sb.WriteString("\n")
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
