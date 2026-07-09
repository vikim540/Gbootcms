package content

import (
	"gbootcms/core/db"
)

// ContentExt 操作 ay_content_ext 表（動態列，使用 map 而非 struct）

// GetContentExtByContentID 返回指定內容的擴展字段數據
func GetContentExtByContentID(contentID uint) map[string]interface{} {
	var rows []map[string]interface{}
	db.DB.Raw("SELECT * FROM ay_content_ext WHERE contentid = ?", contentID).Scan(&rows)
	if len(rows) == 0 {
		return nil
	}
	return rows[0]
}

// InsertContentExt 插入擴展數據行（data 必須包含 "contentid"）
func InsertContentExt(data map[string]interface{}) error {
	return db.DB.Table("ay_content_ext").Create(data).Error
}

// UpdateContentExt 更新擴展數據行
func UpdateContentExt(contentID uint, data map[string]interface{}) error {
	return db.DB.Table("ay_content_ext").Where("contentid = ?", contentID).Updates(data).Error
}

// UpsertContentExt 插入或更新擴展數據（PHP 原版邏輯：有行則 UPDATE，無行則 INSERT）
func UpsertContentExt(contentID uint, data map[string]interface{}) error {
	existing := GetContentExtByContentID(contentID)
	if existing != nil {
		return UpdateContentExt(contentID, data)
	}
	data["contentid"] = contentID
	return InsertContentExt(data)
}

// DeleteContentExt 刪除內容的擴展數據
func DeleteContentExt(contentID uint) error {
	return db.DB.Exec("DELETE FROM ay_content_ext WHERE contentid = ?", contentID).Error
}
