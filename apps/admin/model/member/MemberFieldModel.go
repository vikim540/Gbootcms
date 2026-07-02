package member

import "time"

// MemberField 對應 PbootCMS ay_member_field 表
type MemberField struct {
	ID          uint      `gorm:"primaryKey;column:id" json:"id"`
	Name        string    `gorm:"column:name" json:"name"`
	Length      int       `gorm:"column:length" json:"length"`
	Required    int       `gorm:"column:required" json:"required"`
	Description string    `gorm:"column:description" json:"description"`
	Sorting     int       `gorm:"column:sorting" json:"sorting"`
	Status      int       `gorm:"column:status" json:"status"`
	CreateUser  string    `gorm:"column:create_user" json:"create_user"`
	UpdateUser  string    `gorm:"column:update_user" json:"update_user"`
	CreateTime  time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time" json:"update_time"`
}

func (MemberField) TableName() string { return "ay_member_field" }
