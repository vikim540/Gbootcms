package content

import (
	"pbootcms-go/core/db"
	"time"
)

// Form 表單模型
type Form struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Fcode      string    `gorm:"column:fcode" json:"fcode"`
	FormName   string    `gorm:"column:form_name" json:"form_name"`
	Table      string    `gorm:"column:table_name" json:"table_name"`
	Status     int       `gorm:"column:status" json:"status"`
	CreateTime time.Time `gorm:"column:createtime" json:"createtime"`
	UpdateTime time.Time `gorm:"column:updatetime" json:"updatetime"`
}

// FormField 表單字段模型
type FormField struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Fcode      string    `gorm:"column:fcode" json:"fcode"`
	Name       string    `gorm:"column:name" json:"name"`
	Field      string    `gorm:"column:field" json:"field"`
	Type       string    `gorm:"column:type" json:"type"`
	Required   int       `gorm:"column:required" json:"required"`
	Sorting    int       `gorm:"column:sorting" json:"sorting"`
	Status     int       `gorm:"column:status" json:"status"`
	CreateTime time.Time `gorm:"column:createtime" json:"createtime"`
	UpdateTime time.Time `gorm:"column:updatetime" json:"updatetime"`
}

// GetFormFieldByCode 按 fcode 查詢表單字段定義（按 sorting 排序）
func GetFormFieldByCode(fcode string) []FormField {
	var list []FormField
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(fcode,'') AS fcode, COALESCE(name,'') AS name, COALESCE(field,'') AS field, COALESCE(type,'') AS type, COALESCE(required,0) AS required, COALESCE(sorting,255) AS sorting, COALESCE(status,1) AS status, COALESCE(createtime,'') AS createtime, COALESCE(updatetime,'') AS updatetime FROM ay_form_field WHERE fcode = ? ORDER BY sorting ASC, id ASC", fcode).Scan(&list)
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
