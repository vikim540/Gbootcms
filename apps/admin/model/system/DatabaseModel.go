package system

import "time"

// Database - Database Backup Model
type Database struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Filename   string    `gorm:"column:filename" json:"filename"`
	Filesize   int64     `gorm:"column:filesize" json:"filesize"`
	CreateTime time.Time `gorm:"column:createtime" json:"createtime"`
}
