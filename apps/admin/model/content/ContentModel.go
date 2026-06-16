package content

import (
	"fmt"
	"math/rand"
	"pbootcms-go/core/db"
	"regexp"
	"time"
)

// ContentFilenamePattern 內容 filename 校驗正則：與 PbootCMS PHP 一致，允許字母、數字、橫線、下劃線、斜線
//
// 與 ContentSort 的規則區別：
//   - ay_content_sort: ^[a-zA-Z0-9\-\/]+$    （不允許下劃線）
//   - ay_content:      ^[a-zA-Z0-9\-_\/]+$   （允許下劃線）
var contentFilenamePattern = regexp.MustCompile(`^[a-zA-Z0-9\-_\/]+$`)

// IsValidContentFilename 校驗內容 URL 名稱格式
func IsValidContentFilename(filename string) bool {
	if filename == "" {
		return true
	}
	return contentFilenamePattern.MatchString(filename)
}

// CheckContentFilename 檢查 filename 是否已被其他內容佔用
func CheckContentFilename(filename string, extraWhere ...string) bool {
	if filename == "" {
		return false
	}
	q := db.DB.Table("ay_content").Where("filename = ?", filename)
	for _, w := range extraWhere {
		if w != "" {
			q = q.Where(w)
		}
	}
	var count int64
	q.Count(&count)
	return count > 0
}

// GenerateUniqueContentFilename 內容 filename 重名時自動加 -rand(1,20) 後綴
// 與 PbootCMS PHP 原文一致（注意：內容用橫線 -，欄目用底線 _）
func GenerateUniqueContentFilename(filename string, extraWhere ...string) string {
	if filename == "" {
		return ""
	}
	for i := 0; i < 100; i++ {
		if !CheckContentFilename(filename, extraWhere...) {
			return filename
		}
		filename = fmt.Sprintf("%s-%d", filename, 1+rand.Intn(20))
	}
	return filename
}

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
