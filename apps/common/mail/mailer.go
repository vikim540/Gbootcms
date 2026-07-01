package mail

import (
	"fmt"
	"pbootcms-go/apps/admin/model"
	"strconv"
	"strings"

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

// SendTestMail 發送測試郵件
func SendTestMail(to string) error {
	subject := "【GbootCMS】測試郵件"
	body := `<html><body>
<h2>測試郵件</h2>
<p>歡迎您使用 GbootCMS 網站開發管理系統！</p>
<p>如果您收到了這封郵件，說明 SMTP 配置正確。</p>
</body></html>`
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
