package content

import (
	"gbootcms/core/db"
	"time"
)

// Form 表單模型（對齊 PHP ay_form 表結構）
type Form struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Fcode      string    `gorm:"column:fcode" json:"fcode"`
	FormName   string    `gorm:"column:form_name" json:"form_name"`
	TableName  string    `gorm:"column:table_name" json:"table_name"`
	CreateUser string    `gorm:"column:create_user" json:"create_user"`
	UpdateUser string    `gorm:"column:update_user" json:"update_user"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`
}

// FormField 表單字段模型（對齊 PHP ay_form_field 表結構）
type FormField struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Fcode       string    `gorm:"column:fcode" json:"fcode"`
	Name        string    `gorm:"column:name" json:"name"`
	Length      int       `gorm:"column:length" json:"length"`
	Required    int       `gorm:"column:required" json:"required"`
	Description string    `gorm:"column:description" json:"description"`
	Sorting     int       `gorm:"column:sorting" json:"sorting"`
	CreateUser  string    `gorm:"column:create_user" json:"create_user"`
	UpdateUser  string    `gorm:"column:update_user" json:"update_user"`
	CreateTime  time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time" json:"update_time"`
}

// GetFormFieldByCode 按 fcode 查詢表單字段定義（按 sorting 排序）
func GetFormFieldByCode(fcode string) []FormField {
	var list []FormField
	db.DB.Raw("SELECT * FROM ay_form_field WHERE fcode = ? ORDER BY sorting ASC, id ASC", fcode).Scan(&list)
	return list
}

// GetFormByCode 按 fcode 查詢表單定義
func GetFormByCode(fcode string) *Form {
	var form Form
	if err := db.DB.Raw("SELECT * FROM ay_form WHERE fcode = ? LIMIT 1", fcode).Scan(&form).Error; err != nil {
		return nil
	}
	return &form
}

// GetFormTableByCode 按 fcode 獲取數據表名
func GetFormTableByCode(fcode string) string {
	var tableName string
	db.DB.Raw("SELECT table_name FROM ay_form WHERE fcode = ? LIMIT 1", fcode).Scan(&tableName)
	return tableName
}
