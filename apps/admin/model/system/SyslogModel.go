package system

import "time"

// Syslog - System Log Model（對齊 PbootCMS ay_syslog 表結構）
// 原始 PbootCMS 欄位（NOT NULL）：level, event, user_ip, user_os, user_bs, create_user, create_time
// GORM 擴展欄位（可空）：username, url, content, ip, createtime
type Syslog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Level      string    `gorm:"column:level" json:"level"`
	Event      string    `gorm:"column:event" json:"event"`
	UserIP     string    `gorm:"column:user_ip" json:"user_ip"`
	UserOS     string    `gorm:"column:user_os" json:"user_os"`
	UserBs     string    `gorm:"column:user_bs" json:"user_bs"`
	CreateUser string    `gorm:"column:create_user" json:"create_user"`
	CreateTime string    `gorm:"column:create_time" json:"create_time"`
	Username   string    `gorm:"column:username" json:"username"`
	URL        string    `gorm:"column:url" json:"url"`
	Content    string    `gorm:"column:content" json:"content"`
	IP         string    `gorm:"column:ip" json:"ip"`
	LogTime    time.Time `gorm:"column:createtime" json:"createtime"`
}

// TableName 指定表名（對齊 PbootCMS ay_syslog）
func (Syslog) TableName() string { return "ay_syslog" }
