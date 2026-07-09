package member

import (
	"gbootcms/core/db"
	"strconv"
	"time"
)

// MemberField 對應 PbootCMS ay_member_field 表
type MemberField struct {
	ID          uint      `gorm:"primaryKey;column:id" json:"id"`
	Name        string    `gorm:"column:name" json:"name"`
	Length      int       `gorm:"column:length" json:"length"`
	Required    int       `gorm:"column:required" json:"required"`
	Description string    `gorm:"column:description" json:"description"`
	Sorting     int       `gorm:"column:sorting" json:"sorting"`
	Status      int       `gorm:"column:status" json:"status"`
	CreateUser  string    `gorm:"column:create_user" json:"create_user"`
	UpdateUser  string    `gorm:"column:update_user" json:"update_user"`
	CreateTime  time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time" json:"update_time"`
}

func (MemberField) TableName() string { return "ay_member_field" }

// ColumnExistsInMember 檢查 ay_member 表中是否已有指定列（對齊 PHP isExistField）
func ColumnExistsInMember(colName string) bool {
	type colInfo struct {
		Cid  int
		Name string
	}
	var cols []colInfo
	db.DB.Raw("PRAGMA table_info(ay_member)").Scan(&cols)
	for _, c := range cols {
		if c.Name == colName {
			return true
		}
	}
	return false
}

// IsFieldRegistered 檢查欄位名是否已登記在 ay_member_field 表中（對齊 PHP checkField）
func IsFieldRegistered(name string) bool {
	var count int64
	db.DB.Model(&MemberField{}).Where("name = ?", name).Count(&count)
	return count > 0
}

// AddColumnToMember 動態添加列到 ay_member 表
func AddColumnToMember(colName string, length int) error {
	return db.DB.Exec("ALTER TABLE ay_member ADD COLUMN " + colName + " TEXT(" + strconv.Itoa(length) + ")").Error
}

// DropColumnFromMember 從 ay_member 表刪除列（僅 MySQL 支援，SQLite 由控制器判斷跳過）
func DropColumnFromMember(colName string) error {
	return db.DB.Exec("ALTER TABLE ay_member DROP COLUMN " + colName).Error
}

// GetFieldNameByID 根據 ID 取得欄位名稱（刪除時用於確定要 DROP 的列名）
func GetFieldNameByID(id string) string {
	var field MemberField
	if err := db.DB.First(&field, id).Error; err != nil {
		return ""
	}
	return field.Name
}
