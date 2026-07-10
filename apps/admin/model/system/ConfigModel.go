package system

// Config - System Configuration Model (映射 ay_config 表，與 PbootCMS 原版欄位 1:1 對齊)
type Config struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"column:name" json:"name"`
	Value       string `gorm:"column:value" json:"value"`
	Type        int    `gorm:"column:type" json:"type"`
	Sorting     int    `gorm:"column:sorting" json:"sorting"`
	Description string `gorm:"column:description" json:"description"`
}
