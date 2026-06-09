package content

import "time"

// ContentSort 欄目分類模型 (aligned with PbootCMS ay_content_sort schema)
type ContentSort struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Acode      string    `gorm:"column:acode;default:''" json:"acode"`
	Mcode      string    `gorm:"column:mcode;default:''" json:"mcode"`
	Pcode      string    `gorm:"column:pcode" json:"pcode"`
	Scode      string    `gorm:"column:scode" json:"scode"`
	Name       string    `gorm:"column:name" json:"name"`
	Subname    string    `gorm:"column:subname" json:"subname"`
	Type       int       `gorm:"column:type" json:"type"`
	ListTpl    string    `gorm:"column:listtpl" json:"listtpl"`
	ContentTpl string    `gorm:"column:contenttpl" json:"contenttpl"`
	Ico        string    `gorm:"column:ico" json:"ico"`
	Pic        string    `gorm:"column:pic" json:"pic"`
	Title      string    `gorm:"column:title;default:''" json:"title"`
	Keywords   string    `gorm:"column:keywords" json:"keywords"`
	Description string   `gorm:"column:description" json:"description"`
	Filename   string    `gorm:"column:filename;default:''" json:"filename"`
	Sort       int       `gorm:"column:sorting" json:"sorting"`
	Status     int       `gorm:"column:status" json:"status"`
	Outlink    string    `gorm:"column:outlink" json:"outlink"`
	CreateUser string    `gorm:"column:create_user;default:''" json:"create_user"`
	UpdateUser string    `gorm:"column:update_user;default:''" json:"update_user"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`
	GType      string    `gorm:"column:gtype;default:'4'" json:"gtype"`
	Gid        string    `gorm:"column:gid;default:''" json:"gid"`
	Gnote      string    `gorm:"column:gnote;default:''" json:"gnote"`
	Def1       string    `gorm:"column:def1;default:''" json:"def1"`
	Def2       string    `gorm:"column:def2;default:''" json:"def2"`
	Def3       string    `gorm:"column:def3;default:''" json:"def3"`
	URLName    string    `gorm:"column:urlname" json:"urlname"`
}
