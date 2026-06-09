package content

import "time"

// Message 留言模型
type Message struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Contacts     string    `gorm:"column:contacts" json:"contacts"`
	Mobile       string    `gorm:"column:mobile" json:"mobile"`
	Content      string    `gorm:"column:content" json:"content"`
	IP           string    `gorm:"column:ip" json:"ip"`
	OS           string    `gorm:"column:os" json:"os"`
	Browser      string    `gorm:"column:bs" json:"bs"`
	AskDate      time.Time `gorm:"column:askdate" json:"askdate"`
	ReplyDate    time.Time `gorm:"column:replydate" json:"replydate"`
	ReplyContent string    `gorm:"column:replycontent" json:"replycontent"`
	Status       int       `gorm:"column:status" json:"status"`
	Nickname     string    `gorm:"column:nickname" json:"nickname"`
	HeadPic      string    `gorm:"column:headpic" json:"headpic"`
}
