package member

// MemberGroup - Member Group Model
type MemberGroup struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Code   string `gorm:"column:code" json:"code"`
	Name   string `gorm:"column:name" json:"name"`
	Status int    `gorm:"column:status" json:"status"`
}
