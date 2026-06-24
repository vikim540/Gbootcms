package content

import (
	"pbootcms-go/core/db"
)

// EnsureContentExtTable 確保 ay_content_ext 基礎表存在（冪等操作）。
// 僅含基礎列 extid + contentid，動態列由新增字段時按需添加。
func EnsureContentExtTable() {
	db.DB.Exec(`CREATE TABLE IF NOT EXISTS ay_content_ext (
		extid INTEGER PRIMARY KEY AUTOINCREMENT,
		contentid INTEGER NOT NULL
	)`)
	db.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_content_ext_contentid ON ay_content_ext(contentid)`)
}

// ColumnExistsInContentExt 檢查 ay_content_ext 表中是否已有指定列
func ColumnExistsInContentExt(colName string) bool {
	type colInfo struct {
		Cid  int
		Name string
	}
	var cols []colInfo
	db.DB.Raw("PRAGMA table_info(ay_content_ext)").Scan(&cols)
	for _, c := range cols {
		if c.Name == colName {
			return true
		}
	}
	return false
}

// AddColumnToContentExt 動態添加列到 ay_content_ext 表
func AddColumnToContentExt(colName, colType string) error {
	return db.DB.Exec("ALTER TABLE ay_content_ext ADD COLUMN " + colName + " " + colType).Error
}

// SqliteColumnTypeForExtType 根據字段類型返回 SQLite 列類型（與 PHP 版一致）
func SqliteColumnTypeForExtType(typ string) string {
	switch typ {
	case "2": // 多行文本
		return "TEXT(1000)"
	case "7": // 日期時間
		return "TEXT"
	case "8": // 編輯器
		return "TEXT(10000)"
	case "10": // 多圖
		return "TEXT(1000)"
	default: // 單行文本/單選/多選/圖片/文件/下拉
		return "TEXT(200)"
	}
}

type ExtField struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Mcode       string `gorm:"column:mcode" json:"mcode"`
	Name        string `gorm:"column:name" json:"name"`
	Field       string `gorm:"column:field" json:"field"`
	Type        string `gorm:"column:type" json:"type"`
	Description string `gorm:"column:description;default:''" json:"description"`
	Value       string `gorm:"column:value;default:''" json:"value"`
	Required    int    `gorm:"column:required" json:"required"`
	Sorting     int    `gorm:"column:sorting" json:"sorting"`
	Status      int    `gorm:"column:status" json:"status"`
}

func GetAllExtFields() []ExtField {
	var list []ExtField
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(mcode,'') AS mcode, COALESCE(name,'') AS name, COALESCE(field,'') AS field, COALESCE(type,'') AS type, COALESCE(description,'') AS description, COALESCE(value,'') AS value, COALESCE(required,0) AS required, COALESCE(sorting,0) AS sorting, COALESCE(status,1) AS status FROM ay_extfield ORDER BY sorting ASC, id ASC").Scan(&list)
	// 向後兼容：舊記錄的 field 列可能為空，此時 name 即為 DB 列名
	for i := range list {
		if list[i].Field == "" && list[i].Name != "" {
			list[i].Field = list[i].Name
		}
	}
	return list
}

func GetExtFieldById(id int) ExtField {
	var ef ExtField
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(mcode,'') AS mcode, COALESCE(name,'') AS name, COALESCE(field,'') AS field, COALESCE(type,'') AS type, COALESCE(description,'') AS description, COALESCE(value,'') AS value, COALESCE(required,0) AS required, COALESCE(sorting,0) AS sorting, COALESCE(status,1) AS status FROM ay_extfield WHERE id = ?", id).Scan(&ef)
	// 向後兼容
	if ef.Field == "" && ef.Name != "" {
		ef.Field = ef.Name
	}
	return ef
}

func GetExtFieldsByModelCode(mcode string) []ExtField {
	var list []ExtField
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(mcode,'') AS mcode, COALESCE(name,'') AS name, COALESCE(field,'') AS field, COALESCE(type,'') AS type, COALESCE(description,'') AS description, COALESCE(value,'') AS value, COALESCE(required,0) AS required, COALESCE(sorting,0) AS sorting, COALESCE(status,1) AS status FROM ay_extfield WHERE mcode = ? AND COALESCE(status,1) = 1 ORDER BY sorting ASC, id ASC", mcode).Scan(&list)
	// 向後兼容
	for i := range list {
		if list[i].Field == "" && list[i].Name != "" {
			list[i].Field = list[i].Name
		}
	}
	return list
}

func AddExtField(mcode, name, field, typ string, required, sorting int) error {
	return db.DB.Exec("INSERT INTO ay_extfield (mcode, name, field, type, description, required, sorting, status) VALUES (?, ?, ?, ?, ?, ?, ?, 1)", mcode, name, field, typ, name, required, sorting).Error
}

func UpdateExtField(id int, mcode, name, field, typ string, required, sorting int) error {
	return db.DB.Exec("UPDATE ay_extfield SET mcode=?, name=?, field=?, type=?, description=?, required=?, sorting=? WHERE id=?", mcode, name, field, typ, name, required, sorting, id).Error
}

func UpdateExtFieldSingleField(id int, field, value string) error {
	return db.DB.Exec("UPDATE ay_extfield SET "+field+" = ? WHERE id=?", value, id).Error
}

func DeleteExtField(id int) error {
	// ⚠️ 熔断：严禁执行 ALTER TABLE ay_content_ext DROP COLUMN
	// 仅删除 ay_extfield 表中的字段定义记录，物理表结构不做任何修改
	return db.DB.Exec("DELETE FROM ay_extfield WHERE id=?", id).Error
}

// EnsureExtColumnExists 確保擴展字段的物理列存在於 ay_content_ext 表中。
// 在新增字段時調用，如果列已存在則跳過。
func EnsureExtColumnExists(fieldName, extType string) error {
	EnsureContentExtTable()
	if !ColumnExistsInContentExt(fieldName) {
		colType := SqliteColumnTypeForExtType(extType)
		return AddColumnToContentExt(fieldName, colType)
	}
	return nil
}
