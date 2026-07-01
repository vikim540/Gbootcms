package content

import "time"

// Message 留言模型
// 對應 PbootCMS ay_message 表，欄位名與原版完全一致
type Message struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Acode      string    `gorm:"column:acode" json:"acode"`
	Contacts   string    `gorm:"column:contacts" json:"contacts"`
	Mobile     string    `gorm:"column:mobile" json:"mobile"`
	Content    string    `gorm:"column:content" json:"content"`
	IP         string    `gorm:"column:user_ip" json:"user_ip"`
	OS         string    `gorm:"column:user_os" json:"user_os"`
	Browser    string    `gorm:"column:user_bs" json:"user_bs"`
	ReContent  string    `gorm:"column:recontent" json:"recontent"`
	Status     int       `gorm:"column:status" json:"status"`
	CreateUser string    `gorm:"column:create_user" json:"create_user"`
	UpdateUser string    `gorm:"column:update_user" json:"update_user"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`
	UID        int       `gorm:"column:uid" json:"uid"`
}
