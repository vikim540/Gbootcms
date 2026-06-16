package content

// Slide - Slide Model
// DB table: ay_slide
type Slide struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Acode      string `gorm:"column:acode" json:"acode"`
	GID        int    `gorm:"column:gid" json:"gid"`
	Pic        string `gorm:"column:pic" json:"pic"`
	PicMobile  string `gorm:"column:pic_mobile" json:"pic_mobile"`
	Link       string `gorm:"column:link" json:"link"`
	Title      string `gorm:"column:title" json:"title"`
	Subtitle   string `gorm:"column:subtitle" json:"subtitle"`
	Sorting    int    `gorm:"column:sorting" json:"sorting"`
	CreateUser string `gorm:"column:create_user" json:"create_user"`
	UpdateUser string `gorm:"column:update_user" json:"update_user"`
	CreateTime string `gorm:"column:create_time" json:"create_time"`
	UpdateTime string `gorm:"column:update_time" json:"update_time"`
}
