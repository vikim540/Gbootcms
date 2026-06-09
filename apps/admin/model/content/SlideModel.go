package content

// Slide - Slide Model
type Slide struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	GID     int    `gorm:"column:gid" json:"gid"`
	Pic     string `gorm:"column:pic" json:"pic"`
	Link    string `gorm:"column:link" json:"link"`
	Title   string `gorm:"column:title" json:"title"`
	Sorting int    `gorm:"column:sorting" json:"sorting"`
}
