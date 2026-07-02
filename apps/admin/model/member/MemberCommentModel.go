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

// CommentView 用於列表/詳情查詢的視圖結構（含 JOIN ay_content + ay_member）
type CommentView struct {
	ID         uint      `gorm:"column:id" json:"id"`
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
	// JOIN ay_content b
	Title string `gorm:"column:title" json:"title"`
	// JOIN ay_member c（評論人）
	Username string `gorm:"column:username" json:"username"`
	Nickname string `gorm:"column:nickname" json:"nickname"`
	Headpic  string `gorm:"column:headpic" json:"headpic"`
	// JOIN ay_member d（被回覆人，僅詳情頁使用）
	Pusername string `gorm:"column:pusername" json:"pusername"`
	Pnickname string `gorm:"column:pnickname" json:"pnickname"`
	// 非資料庫欄位：格式化時間字串
	CreateTimeStr string `gorm:"-" json:"create_time_str"`
}
