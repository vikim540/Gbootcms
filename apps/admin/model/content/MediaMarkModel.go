package content

import "time"

// MediaMark 標記保護的媒體文件
type MediaMark struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Path       string    `gorm:"column:path;uniqueIndex" json:"path"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
}

// TableName 指定表名（帶 ay_ 前綴）
func (MediaMark) TableName() string {
	return "ay_media_mark"
}
