package system

import "time"

// Type - Dictionary Type Model
type Type struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Code    string `gorm:"column:code" json:"code"`
	Name    string `gorm:"column:name" json:"name"`
	Sorting int    `gorm:"column:sorting" json:"sorting"`
	Status  int    `gorm:"column:status" json:"status"`
}

// DictType - Dictionary Type Model
type DictType struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DictID     int       `gorm:"column:dict_id" json:"dict_id"`
	Name       string    `gorm:"column:name" json:"name"`
	Value      string    `gorm:"column:value" json:"value"`
	Code       string    `gorm:"column:code" json:"code"`
	Sorting    int       `gorm:"column:sorting" json:"sorting"`
	Status     int       `gorm:"column:status" json:"status"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
}
