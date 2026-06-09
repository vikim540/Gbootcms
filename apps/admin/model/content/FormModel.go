package content

import "time"

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
