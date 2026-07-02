package member

// MemberGroup 對應 PbootCMS ay_member_group 表
type MemberGroup struct {
	ID          uint   `gorm:"primaryKey;column:id" json:"id"`
	Gcode       string `gorm:"column:gcode" json:"gcode"`
	Gname       string `gorm:"column:gname" json:"gname"`
	Description string `gorm:"column:description" json:"description"`
	Lscore      int    `gorm:"column:lscore" json:"lscore"`
	Uscore      int    `gorm:"column:uscore" json:"uscore"`
	Status      int    `gorm:"column:status" json:"status"`
}

func (MemberGroup) TableName() string { return "ay_member_group" }
