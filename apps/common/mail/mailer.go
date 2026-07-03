package mail

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"pbootcms-go/apps/admin/model"
	"strconv"
	"strings"
	"time"
)

// SendMail 發送郵件（使用標準庫 net/smtp + crypto/tls）
// 自動根據 smtp_ssl 配置選擇隱式 SSL（465）或 STARTTLS（587/25）
func SendMail(to, subject, body string) error {
	smtpServer := model.GetConfigValue("smtp_server", "")
	smtpPortStr := model.GetConfigValue("smtp_port", "25")
	smtpUsername := model.GetConfigValue("smtp_username", "")
	smtpPassword := model.GetConfigValue("smtp_password", "")
	smtpSSL := model.GetConfigValue("smtp_ssl", "0")

	if smtpServer == "" || smtpUsername == "" {
		return fmt.Errorf("SMTP 配置不完整")
	}

	smtpPort, _ := strconv.Atoi(smtpPortStr)
	if smtpPort == 0 {
		smtpPort = 25
	}

	// 支援逗號分隔多收件人
	recipients := strings.Split(to, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}

	from := smtpUsername
	headers := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", "GbootCMS", from),
		"To":           to,
		"Subject":      fmt.Sprintf("=?UTF-8?B?%s?=", encodeBase64(subject)),
		"Content-Type": "text/html; charset=UTF-8",
		"MIME-Version": "1.0",
		"Date":         time.Now().Format(time.RFC1123Z),
	}

	// 構建郵件正文
	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	addr := fmt.Sprintf("%s:%d", smtpServer, smtpPort)
	auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpServer)

	if smtpSSL == "1" {
		// 隱式 SSL/TLS（port 465）
		return sendMailSSL(addr, smtpServer, auth, from, recipients, msg.String())
	}
	// STARTTLS 或明文（port 587/25）
	return sendMailSTARTTLS(addr, smtpServer, smtpPort, auth, from, recipients, msg.String())
}

// sendMailSSL 使用隱式 TLS 連線發送郵件（port 465）
func sendMailSSL(addr, host string, auth smtp.Auth, from string, to []string, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	if err != nil {
		return fmt.Errorf("TLS 連線失敗: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("建立 SMTP 客戶端失敗: %w", err)
	}
	defer c.Close()

	if err := c.Hello("localhost"); err != nil {
		return fmt.Errorf("EHLO 失敗: %w", err)
	}

	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("認證失敗: %w", err)
	}

	if err := c.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM 失敗: %w", err)
	}

	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return fmt.Errorf("RCPT TO 失敗 (%s): %w", addr, err)
		}
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA 失敗: %w", err)
	}
	if _, err := fmt.Fprint(w, msg); err != nil {
		return fmt.Errorf("寫入郵件內容失敗: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("關閉 DATA 失敗: %w", err)
	}

	return c.Quit()
}

// sendMailSTARTTLS 使用 STARTTLS 或明文連線發送郵件（port 587/25）
func sendMailSTARTTLS(addr, host string, port int, auth smtp.Auth, from string, to []string, msg string) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("連線失敗: %w", err)
	}
	defer c.Close()

	if err := c.Hello("localhost"); err != nil {
		return fmt.Errorf("EHLO 失敗: %w", err)
	}

	// 嘗試 STARTTLS
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("STARTTLS 失敗: %w", err)
		}
	}

	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("認證失敗: %w", err)
	}

	if err := c.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM 失敗: %w", err)
	}

	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return fmt.Errorf("RCPT TO 失敗 (%s): %w", addr, err)
		}
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA 失敗: %w", err)
	}
	if _, err := fmt.Fprint(w, msg); err != nil {
		return fmt.Errorf("寫入郵件內容失敗: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("關閉 DATA 失敗: %w", err)
	}

	return c.Quit()
}

// SendNotifyMail 發送通知郵件（留言/表單/評論）
func SendNotifyMail(formName string, fields []map[string]string) error {
	sendTo := model.GetConfigValue("message_send_to", "")
	if sendTo == "" {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<html><body style="margin:0;padding:0;background:#f4f6f9;">
<div style="max-width:600px;margin:20px auto;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
  <div style="background:linear-gradient(135deg,#4e73df 0%%,#224abe 100%%);padding:30px 40px;">
    <h1 style="margin:0;color:#fff;font-size:22px;font-weight:600;">%s</h1>
    <p style="margin:5px 0 0;color:rgba(255,255,255,0.8);font-size:13px;">系統通知</p>
  </div>
  <div style="padding:30px 40px;">`, formName))

	for _, f := range fields {
		sb.WriteString(fmt.Sprintf(`<p style="color:#555;font-size:14px;line-height:1.8;margin:0 0 10px;"><strong>%s:</strong> %s</p>`, f["label"], f["value"]))
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	sb.WriteString(fmt.Sprintf(`</div>
  <div style="background:#f8f9fc;padding:20px 40px;text-align:center;">
    <p style="margin:0;color:#999;font-size:12px;">發送時間：%s</p>
    <p style="margin:5px 0 0;color:#999;font-size:12px;">此郵件由系統自動發送，請勿回覆</p>
  </div>
</div>
</body></html>`, now))

	return SendMail(sendTo, "新通知："+formName, sb.String())
}

// SendTestMail 發送測試郵件（美觀 HTML 模板）
func SendTestMail(to string) error {
	siteName := model.GetConfigValue("cmsname", "GbootCMS")
	now := time.Now().Format("2006-01-02 15:04:05")
	subject := "【" + siteName + "】測試郵件"
	body := fmt.Sprintf(`<html><body style="margin:0;padding:0;background:#f4f6f9;">
<div style="max-width:600px;margin:20px auto;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
  <div style="background:linear-gradient(135deg,#4e73df 0%%,#224abe 100%%);padding:30px 40px;">
    <h1 style="margin:0;color:#fff;font-size:22px;font-weight:600;">%s</h1>
    <p style="margin:5px 0 0;color:rgba(255,255,255,0.8);font-size:13px;">測試郵件通知</p>
  </div>
  <div style="padding:30px 40px;">
    <h2 style="color:#333;font-size:18px;margin:0 0 15px;">SMTP 配置驗證成功</h2>
    <p style="color:#555;font-size:14px;line-height:1.8;margin:0 0 15px;">您好！這是一封來自 <strong>%s</strong> 系統的測試郵件。</p>
    <p style="color:#555;font-size:14px;line-height:1.8;margin:0 0 15px;">如果您收到了這封郵件，說明您的 SMTP 郵件配置已正確生效，系統可以正常發送通知郵件。</p>
    <div style="background:#f8f9fc;border-left:4px solid #4e73df;padding:15px 20px;margin:20px 0;border-radius:0 4px 4px 0;">
      <p style="margin:0;color:#666;font-size:13px;">發送時間：%s</p>
      <p style="margin:5px 0 0;color:#666;font-size:13px;">收件地址：%s</p>
    </div>
  </div>
  <div style="background:#f8f9fc;padding:20px 40px;text-align:center;">
    <p style="margin:0;color:#999;font-size:12px;">此郵件由系統自動發送，請勿回覆</p>
    <p style="margin:5px 0 0;color:#999;font-size:12px;">&copy; %d %s</p>
  </div>
</div>
</body></html>`, siteName, siteName, now, to, time.Now().Year(), siteName)
	return SendMail(to, subject, body)
}

// encodeBase64 Base64 編碼（用於郵件主旨）
func encodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
