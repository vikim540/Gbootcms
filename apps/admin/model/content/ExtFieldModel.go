package content

import (
	"gbootcms/core/db"
	"strings"
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

// EnsureExtFieldScodeColumn 確保 ay_extfield 表有 scode 列（用於存儲適用欄目，多選逗號分隔）
func EnsureExtFieldScodeColumn() {
	var count int64
	db.DB.Raw("SELECT COUNT(*) FROM pragma_table_info('ay_extfield') WHERE name='scode'").Scan(&count)
	if count == 0 {
		db.DB.Exec("ALTER TABLE ay_extfield ADD COLUMN scode TEXT DEFAULT ''")
	}
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
	Value       string `gorm:"column:value;default:''" json:"value"`   // 選項值（單選/多選/下拉的選項列表）
	Scode       string `gorm:"column:scode;default:''" json:"scode"`   // 適用欄目，逗號分隔（空=全展示）
	Required    int    `gorm:"column:required" json:"required"`
	Sorting     int    `gorm:"column:sorting" json:"sorting"`
	Status      int    `gorm:"column:status" json:"status"`
}

// extFieldSelectColumns 標準查詢欄位列表（含 scode）
const extFieldSelectColumns = "COALESCE(id,0) AS id, COALESCE(mcode,'') AS mcode, COALESCE(name,'') AS name, COALESCE(field,'') AS field, COALESCE(type,'') AS type, COALESCE(description,'') AS description, COALESCE(value,'') AS value, COALESCE(scode,'') AS scode, COALESCE(required,0) AS required, COALESCE(sorting,0) AS sorting, COALESCE(status,1) AS status"

// NormalizeOptions 將選項文字中的換行符替換為逗號，並清理多餘的逗號和空白。
// 支援使用者以回車或逗號分隔選項，統一存儲為逗號分隔格式。
func NormalizeOptions(options string) string {
	options = strings.ReplaceAll(options, "\r\n", ",")
	options = strings.ReplaceAll(options, "\r", ",")
	options = strings.ReplaceAll(options, "\n", ",")
	parts := strings.Split(options, ",")
	var clean []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			clean = append(clean, p)
		}
	}
	return strings.Join(clean, ",")
}

// ScodeMatches 檢查欄位適用的 scode 列表中是否包含指定欄目。
// fieldScode 為空字串表示全展示（匹配所有欄目）。
// fieldScode 支援逗號分隔的多選格式，如 "3,4"。
func ScodeMatches(fieldScode, targetScode string) bool {
	if fieldScode == "" {
		return true
	}
	for _, s := range strings.Split(fieldScode, ",") {
		if strings.TrimSpace(s) == targetScode {
			return true
		}
	}
	return false
}

// MigrateScodeFromValue 將舊的 || 格式數據遷移到 scode 列。
// 對每條記錄：如果 value 含 "||"，將前綴移到 scode 列，value 保留純 options。
// 冪等操作：已遷移的記錄（scode 已有值或 value 無 ||）不受影響。
func MigrateScodeFromValue() {
	var rows []struct {
		ID    int    `gorm:"column:id"`
		Value string `gorm:"column:value"`
		Scode string `gorm:"column:scode"`
	}
	db.DB.Raw("SELECT id, value, COALESCE(scode,'') AS scode FROM ay_extfield").Scan(&rows)
	for _, r := range rows {
		if r.Scode != "" {
			continue // 已有 scode 值，跳過
		}
		if idx := strings.Index(r.Value, "||"); idx >= 0 {
			scode := r.Value[:idx]
			cleanValue := r.Value[idx+2:]
			db.DB.Exec("UPDATE ay_extfield SET scode=?, value=? WHERE id=?", scode, cleanValue, r.ID)
		}
	}
}

func GetAllExtFields() []ExtField {
	EnsureExtFieldScodeColumn()
	var list []ExtField
	db.DB.Raw("SELECT "+extFieldSelectColumns+" FROM ay_extfield ORDER BY sorting ASC, id ASC").Scan(&list)
	for i := range list {
		if list[i].Field == "" && list[i].Name != "" {
			list[i].Field = list[i].Name
		}
	}
	return list
}

func GetExtFieldById(id int) ExtField {
	EnsureExtFieldScodeColumn()
	var ef ExtField
	db.DB.Raw("SELECT "+extFieldSelectColumns+" FROM ay_extfield WHERE id = ?", id).Scan(&ef)
	if ef.Field == "" && ef.Name != "" {
		ef.Field = ef.Name
	}
	return ef
}

func GetExtFieldsByModelCode(mcode string) []ExtField {
	EnsureExtFieldScodeColumn()
	var list []ExtField
	db.DB.Raw("SELECT "+extFieldSelectColumns+" FROM ay_extfield WHERE mcode = ? AND COALESCE(status,1) = 1 ORDER BY sorting ASC, id ASC", mcode).Scan(&list)
	for i := range list {
		if list[i].Field == "" && list[i].Name != "" {
			list[i].Field = list[i].Name
		}
	}
	return list
}

// GetExtFieldsByModelCodeAndScode 按模型代碼和欄目代碼查詢擴展字段。
// 返回：scode 為空（全展示）的 + scode 列表中包含指定欄目的（支援多選逗號分隔）。
func GetExtFieldsByModelCodeAndScode(mcode, scode string) []ExtField {
	all := GetExtFieldsByModelCode(mcode)
	var result []ExtField
	for _, ef := range all {
		if ScodeMatches(ef.Scode, scode) {
			result = append(result, ef)
		}
	}
	if result == nil {
		result = []ExtField{}
	}
	return result
}

// CheckFieldUnique 檢查同一模型下 field 名稱是否唯一
// excludeID 用於修改時排除自身（新增時傳 0）
func CheckFieldUnique(mcode, field string, excludeID int) bool {
	var count int64
	db.DB.Table("ay_extfield").Where("mcode = ? AND field = ? AND id != ?", mcode, field, excludeID).Count(&count)
	return count > 0
}

func AddExtField(mcode, name, field, typ, value, scode string, required, sorting int) error {
	return db.DB.Exec("INSERT INTO ay_extfield (mcode, name, field, type, value, scode, description, required, sorting, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1)", mcode, name, field, typ, value, scode, name, required, sorting).Error
}

func UpdateExtField(id int, mcode, name, field, typ, value, scode string, required, sorting int) error {
	return db.DB.Exec("UPDATE ay_extfield SET mcode=?, name=?, field=?, type=?, value=?, scode=?, description=?, required=?, sorting=? WHERE id=?", mcode, name, field, typ, value, scode, name, required, sorting, id).Error
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
