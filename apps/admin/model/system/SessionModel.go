package system

import (
	"time"
)

// Session 會話持久化模型（Go 版新增，非 PbootCMS 原版表）
// 用於 SQLite 持久化 session，解決重啟丟失 + 多實例共享問題
type Session struct {
	SID          string    `gorm:"primaryKey;column:sid" json:"sid"`
	Data         string    `gorm:"column:data;type:text" json:"data"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	LastActivity time.Time `gorm:"column:last_activity" json:"last_activity"`
}

// TableName 指定表名
func (Session) TableName() string { return "ay_session" }
