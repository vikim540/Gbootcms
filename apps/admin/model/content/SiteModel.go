package content

// Site 站點信息模型
type Site struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Acode       string `gorm:"column:acode" json:"acode"`
	Name        string `gorm:"column:name" json:"name"`
	Title       string `gorm:"column:title" json:"title"`
	Subtitle    string `gorm:"column:subtitle" json:"subtitle"`
	Domain      string `gorm:"column:domain" json:"domain"`
	Keywords    string `gorm:"column:keywords" json:"keywords"`
	Description string `gorm:"column:description" json:"description"`
	Logo        string `gorm:"column:logo" json:"logo"`
	ICP         string `gorm:"column:icp" json:"icp"`
	Copyright   string `gorm:"column:copyright" json:"copyright"`
	Statistical string `gorm:"column:statistical" json:"statistical"`
	Theme       string `gorm:"column:theme" json:"theme"`
	Lang        string `gorm:"column:lang" json:"lang"`
}
