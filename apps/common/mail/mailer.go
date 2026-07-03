package mail

import (
	"fmt"
	"pbootcms-go/apps/admin/model"
	"strconv"
	"strings"
	"time"

	mail "github.com/wneessen/go-mail"
)

// SendMail 發送 HTML 郵件（對應 PHP 版 sendmail() 函數）
// 讀取 ay_config 中的 SMTP 配置，支持 SSL/TLS
func SendMail(to, subject, body string) error {
	// 讀取 SMTP 配置
	smtpServer := model.GetConfigValue("smtp_server", "")
	smtpUsername := model.GetConfigValue("smtp_username", "")
	smtpPassword := model.GetConfigValue("smtp_password", "")
	smtpPortStr := model.GetConfigValue("smtp_port", "465")
	smtpSSL := model.GetConfigValue("smtp_ssl", "1")

	if smtpServer == "" || smtpUsername == "" {
		return fmt.Errorf("SMTP 配置不完整")
	}

	port, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		port = 465
	}

	// 創建郵件客戶端
	var opts []mail.Option
	opts = append(opts,
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(smtpUsername),
		mail.WithPassword(smtpPassword),
	)
	if smtpSSL == "1" {
		opts = append(opts, mail.WithSSLPort(false)) // 隱式 SSL/TLS (port 465)
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory)) // STARTTLS
	}

	m, err := mail.NewClient(smtpServer, opts...)
	if err != nil {
		return fmt.Errorf("創建郵件客戶端失敗: %w", err)
	}

	// 創建郵件
	msg := mail.NewMsg()
	if err := msg.From(smtpUsername); err != nil {
		return fmt.Errorf("設置發件人失敗: %w", err)
	}

	// 支持逗號分隔多收件人
	recipients := strings.Split(to, ",")
	for _, r := range recipients {
		r = strings.TrimSpace(r)
		if r != "" {
			if err := msg.To(r); err != nil {
				return fmt.Errorf("設置收件人失敗: %w", err)
			}
		}
	}

	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, body)

	// 發送
	if err := m.DialAndSend(msg); err != nil {
		return fmt.Errorf("發送郵件失敗: %w", err)
	}

	return nil
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

// SendNotifyMail 發送留言/表單通知郵件
// formName: 表單名稱（如「在線留言」「搜集電話」）
// fields: 字段名→字段值 的有序列表
func SendNotifyMail(formName string, fields []map[string]string) error {
	to := model.GetConfigValue("message_send_to", "")
	if to == "" {
		return nil
	}

	subject := fmt.Sprintf("【GbootCMS】您有新的%s信息，請注意查收！", formName)

	var bodyBuilder strings.Builder
	bodyBuilder.WriteString("<html><body>")
	bodyBuilder.WriteString(fmt.Sprintf("<h2>新的%s信息</h2>", formName))
	bodyBuilder.WriteString("<table border='1' cellpadding='8' cellspacing='0' style='border-collapse:collapse;'>")
	for _, f := range fields {
		bodyBuilder.WriteString(fmt.Sprintf("<tr><td><strong>%s</strong></td><td>%s</td></tr>",
			f["label"], f["value"]))
	}
	bodyBuilder.WriteString("</table>")
	bodyBuilder.WriteString("<br><p>來自網站 GbootCMS</p>")
	bodyBuilder.WriteString("</body></html>")

	return SendMail(to, subject, bodyBuilder.String())
}
