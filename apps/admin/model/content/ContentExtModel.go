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

// GetContentExtByContentIDs 批量查詢多條內容的擴展字段，避免 N+1 查詢。
// 返回 map[contentID]extData，未命中 ext 的 contentID 不會出現在 map 中。
func GetContentExtByContentIDs(contentIDs []uint) map[uint]map[string]interface{} {
	if len(contentIDs) == 0 {
		return nil
	}
	var rows []map[string]interface{}
	db.DB.Raw("SELECT * FROM ay_content_ext WHERE contentid IN ?", contentIDs).Scan(&rows)
	result := make(map[uint]map[string]interface{}, len(rows))
	for _, row := range rows {
		var cid uint
		if v, ok := row["contentid"]; ok {
			switch val := v.(type) {
			case int64:
				cid = uint(val)
			case uint:
				cid = val
			case float64:
				cid = uint(val)
			}
		}
		if cid > 0 {
			result[cid] = row
		}
	}
	return result
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
