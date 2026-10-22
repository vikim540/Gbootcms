package content

import "time"

// Message 留言模型
// 對應 PbootCMS ay_message 表，欄位名與原版完全一致
type Message struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Acode       string    `gorm:"column:acode" json:"acode"`
	Contacts    string    `gorm:"column:contacts" json:"contacts"`
	Mobile      string    `gorm:"column:mobile" json:"mobile"`
	Content     string    `gorm:"column:content" json:"content"`
	IP          string    `gorm:"column:user_ip" json:"user_ip"`
	OS          string    `gorm:"column:user_os" json:"user_os"`
	Browser     string    `gorm:"column:user_bs" json:"user_bs"`
	ReContent   string    `gorm:"column:recontent" json:"recontent"`
	Status      int       `gorm:"column:status" json:"status"`
	CreateUser  string    `gorm:"column:create_user" json:"create_user"`
	UpdateUser  string    `gorm:"column:update_user" json:"update_user"`
	CreateTime  time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time" json:"update_time"`
	UID         int       `gorm:"column:uid" json:"uid"`

	// 商業營銷歸因欄位（Lead-Gen Attribution）
	UtmSource   string    `gorm:"column:utm_source;size:100;default:''" json:"utm_source"`
	UtmMedium   string    `gorm:"column:utm_medium;size:100;default:''" json:"utm_medium"`
	UtmCampaign string    `gorm:"column:utm_campaign;size:100;default:''" json:"utm_campaign"`
	UtmTerm     string    `gorm:"column:utm_term;size:100;default:''" json:"utm_term"`
	UtmContent  string    `gorm:"column:utm_content;size:100;default:''" json:"utm_content"`
	Gclid       string    `gorm:"column:gclid;size:150;default:''" json:"gclid"`
	LandingPage string    `gorm:"column:landing_page;size:255;default:''" json:"landing_page"`
	Referrer    string    `gorm:"column:referrer;size:255;default:''" json:"referrer"`
}

// TableName 指定表名（對齊 PbootCMS ay_message）
func (Message) TableName() string { return "ay_message" }
