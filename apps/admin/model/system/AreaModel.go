package system

import (
	"context"
	"time"

	"gbootcms/core/acodeplugin"
	"gbootcms/core/db"
	"gorm.io/gorm"
)

// Area 數據區域模型
// DB table: ay_area
// 業務主鍵: acode（區域編碼），非 id
type Area struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Acode      string    `gorm:"column:acode" json:"acode"`
	Pcode      string    `gorm:"column:pcode" json:"pcode"`
	Name       string    `gorm:"column:name" json:"name"`
	Domain     string    `gorm:"column:domain" json:"domain"`
	IsDefault  string    `gorm:"column:is_default" json:"is_default"`
	CreateUser string    `gorm:"column:create_user" json:"create_user"`
	UpdateUser string    `gorm:"column:update_user" json:"update_user"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`
}

// AreaTreeNode 帶子節點的樹形結構
type AreaTreeNode struct {
	Area
	Son []AreaTreeNode `json:"son"`
}

// skipCtx 返回跳過 acode 隔離的 context（區域管理本身需跨區查詢）
var skipCtx = acodeplugin.SkipAcode(context.Background())

// GetAreaList 獲取所有區域並構建樹形結構
func GetAreaList() []AreaTreeNode {
	var areas []Area
	db.DB.WithContext(skipCtx).Order("pcode, acode").Find(&areas)
	return buildAreaTree(areas, "0")
}

// GetAreaSelect 獲取區域下拉選擇樹（僅 pcode/acode/name）
func GetAreaSelect() []AreaTreeNode {
	var areas []Area
	db.DB.WithContext(skipCtx).Select("pcode, acode, name").Order("pcode, acode").Find(&areas)
	return buildAreaTree(areas, "0")
}

// buildAreaTree 遞迴構建樹形結構
func buildAreaTree(areas []Area, parentCode string) []AreaTreeNode {
	var nodes []AreaTreeNode
	for _, a := range areas {
		if a.Pcode == parentCode {
			node := AreaTreeNode{Area: a}
			node.Son = buildAreaTree(areas, a.Acode)
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// GetAreaByAcode 根據 acode 獲取單個區域
func GetAreaByAcode(acode string) *Area {
	var area Area
	if err := db.DB.WithContext(skipCtx).Where("acode = ?", acode).First(&area).Error; err != nil {
		return nil
	}
	return &area
}

// CheckAreaAcodeExists 檢查區域編碼是否已存在（excludeAcode 用於修改時排除自身）
func CheckAreaAcodeExists(acode, excludeAcode string) bool {
	var count int64
	query := db.DB.WithContext(skipCtx).Model(&Area{}).Where("acode = ?", acode)
	if excludeAcode != "" {
		query = query.Where("acode <> ?", excludeAcode)
	}
	query.Count(&count)
	return count > 0
}

// CheckAreaDomainExists 檢查域名是否已綁定（excludeAcode 用於修改時排除自身）
func CheckAreaDomainExists(domain, excludeAcode string) bool {
	var count int64
	query := db.DB.WithContext(skipCtx).Model(&Area{}).Where("domain = ? AND domain != ''", domain)
	if excludeAcode != "" {
		query = query.Where("acode <> ?", excludeAcode)
	}
	query.Count(&count)
	return count > 0
}

// AddArea 新增區域
func AddArea(area *Area) error {
	area.CreateTime = time.Now()
	area.UpdateTime = time.Now()

	if err := db.DB.WithContext(skipCtx).Create(area).Error; err != nil {
		return err
	}
	if area.IsDefault == "1" {
		UnsetDefault(area.Acode)
	}
	return nil
}

// ModArea 修改區域
func ModArea(acode string, updates map[string]interface{}) error {
	result := db.DB.WithContext(skipCtx).Model(&Area{}).Where("acode = ?", acode).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	// 如果設為默認，取消其他區域的默認
	if isDefault, ok := updates["is_default"]; ok && isDefault == "1" {
		newAcode, _ := updates["acode"].(string)
		if newAcode == "" {
			newAcode = acode
		}
		UnsetDefault(newAcode)
	}
	// 如果 acode 變更，更新子區域的 pcode
	if newAcode, ok := updates["acode"].(string); ok && newAcode != "" && newAcode != acode {
		db.DB.WithContext(skipCtx).Model(&Area{}).Where("pcode = ?", acode).Update("pcode", newAcode)
	}
	return nil
}

// DelArea 刪除區域（含子區域），但不允許刪除默認區域
func DelArea(acode string) error {
	result := db.DB.WithContext(skipCtx).Where("acode = ? OR pcode = ?", acode, acode).
		Where("is_default = 0").
		Delete(&Area{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UnsetDefault 取消指定 acode 以外所有區域的默認狀態
func UnsetDefault(acode string) {
	db.DB.WithContext(skipCtx).Model(&Area{}).Where("acode <> ?", acode).Update("is_default", "0")
}
