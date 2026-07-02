package member

import "time"

// MemberComment 對應 PbootCMS ay_member_comment 表
type MemberComment struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`
	Pid        uint      `gorm:"column:pid" json:"pid"`
	Contentid  uint      `gorm:"column:contentid" json:"contentid"`
	Comment    string    `gorm:"column:comment" json:"comment"`
	Uid        uint      `gorm:"column:uid" json:"uid"`
	Puid       uint      `gorm:"column:puid" json:"puid"`
	Likes      int       `gorm:"column:likes" json:"likes"`
	Oppose     int       `gorm:"column:oppose" json:"oppose"`
	Status     int       `gorm:"column:status" json:"status"`
	UserIP     string    `gorm:"column:user_ip" json:"user_ip"`
	UserOS     string    `gorm:"column:user_os" json:"user_os"`
	UserBS     string    `gorm:"column:user_bs" json:"user_bs"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`
}

func (MemberComment) TableName() string { return "ay_member_comment" }
