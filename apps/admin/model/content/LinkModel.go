package content

// Link 友情鏈接模型
type Link struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	GID     int    `gorm:"column:gid" json:"gid"`
	Logo    string `gorm:"column:logo" json:"logo"`
	Link    string `gorm:"column:link" json:"link"`
	Title   string `gorm:"column:title" json:"title"`
	Sorting int    `gorm:"column:sorting" json:"sorting"`
}
