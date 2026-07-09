package content

// Link 友情鏈接模型
// DB table: ay_link
type Link struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Acode      string `gorm:"column:acode" json:"acode"`
	GID        int    `gorm:"column:gid" json:"gid"`
	Name       string `gorm:"column:name" json:"name"`
	Link       string `gorm:"column:link" json:"link"`
	Logo       string `gorm:"column:logo" json:"logo"`
	Sorting    int    `gorm:"column:sorting" json:"sorting"`
	CreateUser string `gorm:"column:create_user" json:"create_user"`
	UpdateUser string `gorm:"column:update_user" json:"update_user"`
	CreateTime string `gorm:"column:create_time" json:"create_time"`
	UpdateTime string `gorm:"column:update_time" json:"update_time"`
}
