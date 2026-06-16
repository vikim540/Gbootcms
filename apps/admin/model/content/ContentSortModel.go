package content

import (
	"fmt"
	"math/rand"
	"pbootcms-go/core/db"
	"regexp"
	"time"
)

// FilenamePattern: 與 PbootCMS PHP 一致，只允許字母、數字、橫線、斜線
var filenamePattern = regexp.MustCompile(`^[a-zA-Z0-9\-\/]+$`)

// IsValidFilename 校驗 URL 名稱格式
func IsValidFilename(filename string) bool {
	if filename == "" {
		return true
	}
	return filenamePattern.MatchString(filename)
}

// CheckUrlname 檢查 URL 名稱是否與 ay_model 的 urlname 衝突
//
// PbootCMS 原文：
//   if ($filename && $this->model->checkUrlname($filename)) {
//       alert_back('URL名稱與模型URL名稱衝突，請換一個名稱！');
//   }
func CheckUrlname(filename string) bool {
	if filename == "" {
		return false
	}
	var count int64
	db.DB.Table("ay_model").Where("urlname = ?", filename).Count(&count)
	return count > 0
}

// CheckFilename 檢查 URL 名稱是否已被其他欄目使用
// extraWhere 為額外的 WHERE 條件（例如 "scode<>'123'" 排除自己）
func CheckFilename(filename string, extraWhere ...string) bool {
	if filename == "" {
		return false
	}
	q := db.DB.Table("ay_content_sort").Where("filename = ?", filename)
	for _, w := range extraWhere {
		if w != "" {
			q = q.Where(w)
		}
	}
	var count int64
	q.Count(&count)
	return count > 0
}

// GenerateUniqueFilename 若 filename 已被佔用，自動加 _rand(1,20) 後綴
// 與 PbootCMS 原文：
//   while ($this->model->checkFilename($filename)) {
//       $filename = $filename . '_' . mt_rand(1, 20);
//   }
func GenerateUniqueFilename(filename string, extraWhere ...string) string {
	if filename == "" {
		return ""
	}
	for i := 0; i < 100; i++ { // 最多嘗試 100 次，防止死循環
		if !CheckFilename(filename, extraWhere...) {
			return filename
		}
		filename = fmt.Sprintf("%s_%d", filename, 1+rand.Intn(20))
	}
	return filename
}

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
