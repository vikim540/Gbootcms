package system

import (
	"strings"
	"time"

	"gbootcms/core/db"
)

// Role - Role Model (aligned with PbootCMS ay_role schema)
type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"column:code" json:"code"`
	Rcode       string    `gorm:"column:rcode" json:"rcode"`
	Name        string    `gorm:"column:name" json:"name"`
	Description string    `gorm:"column:description;default:''" json:"description"`
	Levels      string    `gorm:"column:levels" json:"levels"`
	Status      int       `gorm:"column:status" json:"status"`
	CreateUser  string    `gorm:"column:create_user;default:''" json:"create_user"`
	UpdateUser  string    `gorm:"column:update_user;default:''" json:"update_user"`
	CreateTime  time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time" json:"update_time"`

	// 非資料庫欄位：預格式化時間字串（pongo2 無 date 過濾器）
	CreateTimeStr string `gorm:"-" json:"create_time_str"`
	UpdateTimeStr string `gorm:"-" json:"update_time_str"`

	// 非資料庫欄位：角色已有的權限列表（修改頁面用）
	LevelList []string `gorm:"-" json:"level_list"`
	AreaList  []string `gorm:"-" json:"area_list"`
}

// RoleLevel - Role Permission Level Model
// 注意：原版 PbootCMS 資料表欄位為 `level`（非 `url`），這裡保持與原版一致
type RoleLevel struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Rcode string `gorm:"column:rcode" json:"rcode"`
	URL   string `gorm:"column:level" json:"url"`
}

// RoleArea - Role Area Association Model
type RoleArea struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Rcode string `gorm:"column:rcode" json:"rcode"`
	Acode string `gorm:"column:acode" json:"acode"`
}

// GetRoleLevels 獲取角色的權限 URL 列表
func GetRoleLevels(rcode string) []string {
	var levels []RoleLevel
	db.DB.Where("rcode = ?", rcode).Find(&levels)
	result := make([]string, 0, len(levels))
	for _, l := range levels {
		result = append(result, l.URL)
	}
	return result
}

// AddRoleLevels 批量插入角色權限
func AddRoleLevels(rcode string, levels []string) {
	for _, level := range levels {
		level = strings.TrimSpace(level)
		if level == "" {
			continue
		}
		db.DB.Create(&RoleLevel{Rcode: rcode, URL: level})
	}
}

// DelRoleLevels 刪除角色的所有權限
func DelRoleLevels(rcode string) {
	db.DB.Where("rcode = ?", rcode).Delete(&RoleLevel{})
}

// GetRoleAreas 獲取角色的區域列表
func GetRoleAreas(rcode string) []string {
	var areas []RoleArea
	db.DB.Where("rcode = ?", rcode).Find(&areas)
	result := make([]string, 0, len(areas))
	for _, a := range areas {
		result = append(result, a.Acode)
	}
	return result
}

// AddRoleAreas 批量插入角色區域
func AddRoleAreas(rcode string, acodes []string) {
	for _, acode := range acodes {
		acode = strings.TrimSpace(acode)
		if acode == "" {
			continue
		}
		db.DB.Create(&RoleArea{Rcode: rcode, Acode: acode})
	}
}

// DelRoleAreas 刪除角色的所有區域
func DelRoleAreas(rcode string) {
	db.DB.Where("rcode = ?", rcode).Delete(&RoleArea{})
}

// GetLastRcode 獲取最後一個角色編碼（用於自動生成新編碼）
func GetLastRcode() string {
	var role Role
	db.DB.Order("id DESC").First(&role)
	return role.Rcode
}

// CheckRcodeExists 檢查角色編碼是否已存在
func CheckRcodeExists(rcode string) bool {
	var count int64
	db.DB.Model(&Role{}).Where("rcode = ?", rcode).Count(&count)
	return count > 0
}

// GetRoleByRcode 根據 rcode 獲取角色（含權限和區域列表）
func GetRoleByRcode(rcode string) *Role {
	var role Role
	if db.DB.Where("rcode = ?", rcode).First(&role).Error != nil {
		return nil
	}
	role.LevelList = GetRoleLevels(rcode)
	role.AreaList = GetRoleAreas(rcode)
	return &role
}

// DelRoleByRcode 刪除角色（含關聯的權限和區域）
func DelRoleByRcode(rcode string) error {
	if err := db.DB.Where("rcode = ?", rcode).Delete(&Role{}).Error; err != nil {
		return err
	}
	DelRoleLevels(rcode)
	DelRoleAreas(rcode)
	return nil
}
