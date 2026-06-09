package content

import "time"

// Content 內容模型 (aligned with PbootCMS ay_content schema)
type Content struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Acode       string    `gorm:"column:acode;default:''" json:"acode"`
	Scode       string    `gorm:"column:scode" json:"scode"`
	Subscode    string    `gorm:"column:subscode" json:"subscode"`
	Title       string    `gorm:"column:title" json:"title"`
	TitleColor  string    `gorm:"column:titlecolor;default:''" json:"titlecolor"`
	Subtitle    string    `gorm:"column:subtitle" json:"subtitle"`
	Filename    string    `gorm:"column:filename;default:''" json:"filename"`
	Author      string    `gorm:"column:author" json:"author"`
	Source      string    `gorm:"column:source" json:"source"`
	Outlink     string    `gorm:"column:outlink" json:"outlink"`
	Date        time.Time `gorm:"column:date" json:"date"`
	Ico         string    `gorm:"column:ico" json:"ico"`
	Pics        string    `gorm:"column:pics" json:"pics"`
	Content     string    `gorm:"column:content" json:"content"`
	Tags        string    `gorm:"column:tags;default:''" json:"tags"`
	Enclosure   string    `gorm:"column:enclosure" json:"enclosure"`
	Keywords    string    `gorm:"column:keywords" json:"keywords"`
	Description string    `gorm:"column:description" json:"description"`
	Sorting     int       `gorm:"column:sorting" json:"sorting"`
	Status      int       `gorm:"column:status" json:"status"`
	IsTop       int       `gorm:"column:istop" json:"istop"`
	IsRecommend int       `gorm:"column:isrecommend" json:"isrecommend"`
	IsHeadline  int       `gorm:"column:isheadline" json:"isheadline"`
	Visits      int       `gorm:"column:visits" json:"visits"`
	Likes       int       `gorm:"column:likes" json:"likes"`
	Oppose      int       `gorm:"column:oppose" json:"oppose"`
	CreateUser  string    `gorm:"column:create_user;default:''" json:"create_user"`
	UpdateUser  string    `gorm:"column:update_user;default:''" json:"update_user"`
	CreateTime  time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time" json:"update_time"`
	GType       string    `gorm:"column:gtype;default:'4'" json:"gtype"`
	Gid         string    `gorm:"column:gid;default:''" json:"gid"`
	Gnote       string    `gorm:"column:gnote;default:''" json:"gnote"`
	PicsTitle   string    `gorm:"column:picstitle;default:''" json:"picstitle"`
	URLName     string    `gorm:"column:urlname" json:"urlname"`
}
