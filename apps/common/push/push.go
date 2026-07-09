package push

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// httpClient 推送用的 HTTP 客戶端（帶超時，避免阻塞用戶請求）
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// PushResult 推送結果
type PushResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int    `json:"count"`
	Remain  int    `json:"remain"`
}

// PushBaidu 百度站長推送（對齊 PHP post_baidu 函數）
// api 格式: http://data.zz.baidu.com/urls?site=DOMAIN&token=TOKEN 或加 &type=daily
// urls 為要推送的 URL 列表，token 為百度站長 token，fast=true 時使用快速收錄
func PushBaidu(api string, urls []string) (*PushResult, error) {
	if len(urls) == 0 {
		return &PushResult{Success: false, Message: "沒有可推送的 URL"}, nil
	}

	body := strings.Join(urls, "\n")
	req, err := http.NewRequest("POST", api, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("創建請求失敗: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("推送請求失敗: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("讀取響應失敗: %w", err)
	}

	// 解析百度響應（對齊 PHP: json_decode）
	var baiduResp struct {
		Error        int    `json:"error"`
		Message      string `json:"message"`
		Success      int    `json:"success"`
		Remain       int    `json:"remain"`
		SuccessDaily int    `json:"success_daily"`
		RemainDaily  int    `json:"remain_daily"`
	}
	if err := json.Unmarshal(respBody, &baiduResp); err != nil {
		return nil, fmt.Errorf("解析響應失敗: %w", err)
	}

	if baiduResp.Error != 0 {
		return &PushResult{
			Success: false,
			Message: fmt.Sprintf("百度推送錯誤: %s", baiduResp.Message),
		}, nil
	}

	// 快速收錄使用 success_daily/remain_daily，普通收錄使用 success/remain
	count := baiduResp.Success
	remain := baiduResp.Remain
	if count == 0 && baiduResp.SuccessDaily > 0 {
		count = baiduResp.SuccessDaily
		remain = baiduResp.RemainDaily
	}

	return &PushResult{
		Success: true,
		Message: fmt.Sprintf("成功推送 %d 條，今天剩餘可推送 %d 條", count, remain),
		Count:   count,
		Remain:  remain,
	}, nil
}

// PushBing Bing IndexNow 推送（對齊 IndexNow 協議）
// key 為 IndexNow 密鑰（需在網站根目錄放置 {key}.txt 驗證文件）
// host 為站點域名，urls 為要推送的 URL 列表
func PushBing(host, key string, urls []string) (*PushResult, error) {
	if key == "" {
		return &PushResult{Success: false, Message: "未配置 Bing IndexNow 密鑰"}, nil
	}
	if len(urls) == 0 {
		return &PushResult{Success: false, Message: "沒有可推送的 URL"}, nil
	}

	// IndexNow JSON 請求體（對齊協議規範）
	payload := struct {
		Host     string   `json:"host"`
		Key      string   `json:"key"`
		KeyLocation string `json:"keyLocation,omitempty"`
		URLList  []string `json:"urlList"`
	}{
		Host:        host,
		Key:         key,
		KeyLocation: fmt.Sprintf("https://%s/%s.txt", host, key),
		URLList:     urls,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("構建請求失敗: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.indexnow.org/indexnow", strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("創建請求失敗: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("推送請求失敗: %w", err)
	}
	defer resp.Body.Close()

	// IndexNow 響應碼：200=成功, 202=待驗證, 400/403/422/429=錯誤
	switch resp.StatusCode {
	case 200:
		return &PushResult{Success: true, Message: fmt.Sprintf("Bing IndexNow 推送成功，共 %d 條", len(urls)), Count: len(urls)}, nil
	case 202:
		return &PushResult{Success: true, Message: fmt.Sprintf("Bing IndexNow 已接收，密鑰驗證中，共 %d 條", len(urls)), Count: len(urls)}, nil
	default:
		return &PushResult{
			Success: false,
			Message: fmt.Sprintf("Bing IndexNow 推送失敗，HTTP %d", resp.StatusCode),
		}, nil
	}
}

// PushGoogle Google sitemap ping（最佳努力方式）
// 注意: Google 已於 2023 年底棄用 sitemap ping 端點，但仍可發送請求作為通知
// sitemapURL 為站點 sitemap.xml 的完整 URL
func PushGoogle(sitemapURL string) (*PushResult, error) {
	if sitemapURL == "" {
		return &PushResult{Success: false, Message: "未配置 Sitemap URL"}, nil
	}

	pingURL := fmt.Sprintf("https://www.google.com/ping?sitemap=%s", sitemapURL)
	resp, err := httpClient.Get(pingURL)
	if err != nil {
		return &PushResult{Success: false, Message: "Google ping 請求失敗: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return &PushResult{Success: true, Message: "Google sitemap ping 已發送（注意:此端點可能已棄用）"}, nil
	}
	return &PushResult{
		Success: false,
		Message: fmt.Sprintf("Google ping 返回 HTTP %d（注意:此端點可能已棄用）", resp.StatusCode),
	}, nil
}

// GetSiteDomain 取得站點域名（用於構建推送 URL）
func GetSiteDomain(host string) string {
	if host == "" {
		return "localhost"
	}
	return host
}

// BuildFullURL 構建完整的推送 URL（域名 + 相對路徑）
func BuildFullURL(domain, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("https://%s%s", domain, path)
}
