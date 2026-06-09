package system

// Area - Area Model
type Area struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Code    string `gorm:"column:code" json:"code"`
	Name    string `gorm:"column:name" json:"name"`
	Sorting int    `gorm:"column:sorting" json:"sorting"`
	Status  int    `gorm:"column:status" json:"status"`
}
