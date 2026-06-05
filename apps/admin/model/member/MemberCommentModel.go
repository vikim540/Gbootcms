package member

import "time"

// MemberComment 會員評論模型
type MemberComment struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ContentID  uint      `gorm:"column:contentid" json:"contentid"`
	MemberID   uint      `gorm:"column:memberid" json:"memberid"`
	Nickname   string    `gorm:"column:nickname" json:"nickname"`
	Content    string    `gorm:"column:content" json:"content"`
	IP         string    `gorm:"column:ip" json:"ip"`
	IsCheck    int       `gorm:"column:ischeck" json:"ischeck"`
	CreateTime time.Time `gorm:"column:createtime" json:"createtime"`
	Pid        uint      `gorm:"column:pid" json:"pid"`
}

// TableName 返回評論表名
func (MemberComment) TableName() string {
	return "ay_comment"
}
