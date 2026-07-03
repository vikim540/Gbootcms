package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"pbootcms-go/apps/admin/model"
	"time"
)

// Payload Webhook 推送的 JSON 結構
type Payload struct {
	FormName  string              `json:"form_name"`
	Timestamp string              `json:"timestamp"`
	IP        string              `json:"ip"`
	OS        string              `json:"os"`
	Browser   string              `json:"browser"`
	Fields    []map[string]string `json:"fields"`
}

// Send 異步推送 Webhook 通知到配置的 URL
// 讀取 ay_config 中的 webhook_url，若為空則跳過
func Send(formName, ip, os, browser string, fields []map[string]string) {
	webhookURL := model.GetConfigValue("webhook_url", "")
	if webhookURL == "" {
		return
	}

	payload := Payload{
		FormName:  formName,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		IP:        ip,
		OS:        os,
		Browser:   browser,
		Fields:    fields,
	}

	// 異步發送，不阻塞用戶請求
	go func() {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return
		}

		req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
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

		if resp.StatusCode >= 400 {
			fmt.Printf("[Webhook] 推送返回異常狀態碼: %d\n", resp.StatusCode)
		}
	}()
}

// SendIf 帶分項開關檢查的推送（category: message/form/comment）
// 對應配置項：webhook_message / webhook_form / webhook_comment（預設 "1" 啟用）
func SendIf(category, formName, ip, os, browser string, fields []map[string]string) {
	if model.GetConfigValue("webhook_"+category, "1") != "1" {
		return
	}
	Send(formName, ip, os, browser, fields)
}
