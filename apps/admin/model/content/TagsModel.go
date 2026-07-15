package content

// Tags 標籤模型（對應 ay_tags 表）
type Tags struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Acode      string `gorm:"column:acode" json:"acode"`
	Name       string `gorm:"column:name" json:"name"`
	Link       string `gorm:"column:link" json:"link"`
	Sorting    int    `gorm:"column:sorting" json:"sorting"`
	CreateUser string `gorm:"column:create_user" json:"create_user"`
	UpdateUser string `gorm:"column:update_user" json:"update_user"`
	CreateTime string `gorm:"column:create_time" json:"create_time"`
	UpdateTime string `gorm:"column:update_time" json:"update_time"`
}
