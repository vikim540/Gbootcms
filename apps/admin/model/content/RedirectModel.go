package content

import "time"

// Redirect 301 重定向規則模型（新增表，不修改原 PbootCMS 表結構）
type Redirect struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	OldURL     string    `gorm:"column:old_url;index" json:"old_url"`
	NewURL     string    `gorm:"column:new_url" json:"new_url"`
	MatchType  int       `gorm:"column:match_type;default:1" json:"match_type"` // 1=精確匹配, 2=前綴匹配
	Status     int       `gorm:"column:status;default:1" json:"status"`         // 1=啟用, 0=禁用
	Sorting    int       `gorm:"column:sorting;default:0" json:"sorting"`
	CreateUser string    `gorm:"column:create_user;default:''" json:"create_user"`
	UpdateUser string    `gorm:"column:update_user;default:''" json:"update_user"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`
}

func (Redirect) TableName() string { return "ay_301_redirect" }
