package content

// Company 公司信息模型
type Company struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Acode    string `gorm:"column:acode" json:"acode"`
	Name     string `gorm:"column:name" json:"name"`
	Address  string `gorm:"column:address" json:"address"`
	Postcode string `gorm:"column:postcode" json:"postcode"`
	Contact  string `gorm:"column:contact" json:"contact"`
	Mobile   string `gorm:"column:mobile" json:"mobile"`
	Phone    string `gorm:"column:phone" json:"phone"`
	Fax      string `gorm:"column:fax" json:"fax"`
	Email    string `gorm:"column:email" json:"email"`
	Qq       string `gorm:"column:qq" json:"qq"`
	Weixin   string `gorm:"column:weixin" json:"weixin"`
	ICP      string `gorm:"column:icp" json:"icp"`
	Blicense string `gorm:"column:blicense" json:"blicense"`
	Other    string `gorm:"column:other" json:"other"`
	Legal    string `gorm:"column:legal" json:"legal"`
	Business string `gorm:"column:business" json:"business"`
}
