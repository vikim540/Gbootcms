package content

// Tags 標籤模型
type Tags struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Acode   string `gorm:"column:acode" json:"acode"`
	Name    string `gorm:"column:name" json:"name"`
	Link    string `gorm:"column:link" json:"link"`
	Sorting int    `gorm:"column:sorting" json:"sorting"`
}
