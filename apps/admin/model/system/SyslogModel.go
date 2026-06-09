package system

import "time"

// Syslog - System Log Model
type Syslog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Username   string    `gorm:"column:username" json:"username"`
	URL        string    `gorm:"column:url" json:"url"`
	Content    string    `gorm:"column:content" json:"content"`
	IP         string    `gorm:"column:ip" json:"ip"`
	CreateTime time.Time `gorm:"column:createtime" json:"createtime"`
}
