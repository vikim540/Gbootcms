package model

import (
	"log/slog"
	"time"
)

// NotifyLog 通知日誌（複用 ay_syslog 表，透過 level 欄位區分類型）
// 沿用 PbootCMS 原始表結構的 NOT NULL 欄位：level, event, user_ip, user_os, user_bs, create_user, create_time
type NotifyLog struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Level      string `gorm:"column:level" json:"level"`
	Event      string `gorm:"column:event" json:"event"`
	UserIP     string `gorm:"column:user_ip" json:"user_ip"`
	UserOS     string `gorm:"column:user_os" json:"user_os"`
	UserBs     string `gorm:"column:user_bs" json:"user_bs"`
	CreateUser string `gorm:"column:create_user" json:"create_user"`
	CreateTime string `gorm:"column:create_time" json:"create_time"`
}

// TableName 指定複用 ay_syslog 表
func (NotifyLog) TableName() string {
	return "ay_syslog"
}

// LogNotify 記錄通知日誌
// category: "mail" 或 "webhook"
// status: "success" 或 "error"
// message: 詳細訊息
func LogNotify(category, status, message string) {
	// event 欄位按位元組截斷（VARCHAR(200)），避免超長導致寫入失敗
	truncated := truncateEventBytes(message, 200)
	now := time.Now().Format("2006-01-02 15:04:05")
	if err := DB.Create(&NotifyLog{
		Level:      category + "_" + status,
		Event:      truncated,
		UserIP:     "system",
		UserOS:     "-",
		UserBs:     "-",
		CreateUser: "system",
		CreateTime: now,
	}).Error; err != nil {
		// DB 寫入失敗時降級到終端日誌，避免靜默丟失
		slog.Error("[NotifyLog] DB 寫入失敗", "err", err, "message", truncated)
	}
}

// GetNotifyLogs 取得通知日誌（level 以 mail_ 或 webhook_ 開頭）
func GetNotifyLogs(limit int) []NotifyLog {
	var logs []NotifyLog
	if err := DB.Where("level LIKE 'mail_%' OR level LIKE 'webhook_%'").
		Order("id DESC").Limit(limit).Find(&logs).Error; err != nil {
		slog.Error("[NotifyLog] 查詢失敗", "err", err)
	}
	return logs
}

// GetNotifyLogsPaged 取得通知日誌（分頁）
func GetNotifyLogsPaged(limit, offset int) []NotifyLog {
	var logs []NotifyLog
	if err := DB.Where("level LIKE 'mail_%' OR level LIKE 'webhook_%'").
		Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		slog.Error("[NotifyLog] 查詢失敗", "err", err)
	}
	return logs
}

// ClearNotifyLogs 清空通知日誌（僅刪除 mail_/webhook_ 記錄，不影響系統日誌）
func ClearNotifyLogs() {
	if err := DB.Where("level LIKE 'mail_%' OR level LIKE 'webhook_%'").
		Delete(&NotifyLog{}).Error; err != nil {
		slog.Error("[NotifyLog] 清理失敗", "err", err)
	}
}

// truncateEventBytes 按位元組截斷訊息，確保不超過 DB 欄位長度限制
// 在 UTF-8 多位元組邊界安全截斷，不產生半個字元
func truncateEventBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	// 從 maxBytes 往前找完整的 UTF-8 邊界
	end := maxBytes
	for end > 0 && (s[end]&0xC0) == 0x80 {
		end--
	}
	return s[:end]
}
