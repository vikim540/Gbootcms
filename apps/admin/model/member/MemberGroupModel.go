package member

// MemberGroup 對應 PbootCMS ay_member_group 表
type MemberGroup struct {
	ID          uint   `gorm:"primaryKey;column:id" json:"id"`
	Acode       string `gorm:"column:acode" json:"acode"`
	Gcode       string `gorm:"column:gcode" json:"gcode"`
	Gname       string `gorm:"column:gname" json:"gname"`
	Description string `gorm:"column:description" json:"description"`
	Lscore      int    `gorm:"column:lscore" json:"lscore"`
	Uscore      int    `gorm:"column:uscore" json:"uscore"`
	Status      int    `gorm:"column:status" json:"status"`
	CreateUser  string `gorm:"column:create_user" json:"create_user"`
	UpdateUser  string `gorm:"column:update_user" json:"update_user"`
	CreateTime  string `gorm:"column:create_time" json:"create_time"`
	UpdateTime  string `gorm:"column:update_time" json:"update_time"`
}

func (MemberGroup) TableName() string { return "ay_member_group" }
