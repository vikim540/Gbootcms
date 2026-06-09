package content

import (
	"pbootcms-go/core/db"
)

type ExtField struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	ModelCode   string `gorm:"column:modelcode" json:"modelcode"`
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
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(modelcode,'') AS modelcode, COALESCE(name,'') AS name, COALESCE(field,'') AS field, COALESCE(type,'') AS type, COALESCE(description,'') AS description, COALESCE(value,'') AS value, COALESCE(required,0) AS required, COALESCE(sorting,0) AS sorting, COALESCE(status,1) AS status FROM ay_extfield ORDER BY sorting ASC, id ASC").Scan(&list)
	return list
}

func GetExtFieldById(id int) ExtField {
	var ef ExtField
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(modelcode,'') AS modelcode, COALESCE(name,'') AS name, COALESCE(field,'') AS field, COALESCE(type,'') AS type, COALESCE(description,'') AS description, COALESCE(value,'') AS value, COALESCE(required,0) AS required, COALESCE(sorting,0) AS sorting, COALESCE(status,1) AS status FROM ay_extfield WHERE id = ?", id).Scan(&ef)
	return ef
}

func GetExtFieldsByModelCode(mcode string) []ExtField {
	var list []ExtField
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(modelcode,'') AS modelcode, COALESCE(name,'') AS name, COALESCE(field,'') AS field, COALESCE(type,'') AS type, COALESCE(description,'') AS description, COALESCE(value,'') AS value, COALESCE(required,0) AS required, COALESCE(sorting,0) AS sorting, COALESCE(status,1) AS status FROM ay_extfield WHERE modelcode = ? AND status = 1 ORDER BY sorting ASC, id ASC", mcode).Scan(&list)
	return list
}

func AddExtField(modelcode, name, field, typ string, required, sorting int) error {
	return db.DB.Exec("INSERT INTO ay_extfield (modelcode, name, field, type, required, sorting, status) VALUES (?, ?, ?, ?, ?, ?, 1)", modelcode, name, field, typ, required, sorting).Error
}

func UpdateExtField(id int, modelcode, name, field, typ string, required, sorting int) error {
	return db.DB.Exec("UPDATE ay_extfield SET modelcode=?, name=?, field=?, type=?, required=?, sorting=? WHERE id=?", modelcode, name, field, typ, required, sorting, id).Error
}

func UpdateExtFieldSingleField(id int, field, value string) error {
	return db.DB.Exec("UPDATE ay_extfield SET "+field+" = ? WHERE id=?", value, id).Error
}

func DeleteExtField(id int) error {
	// ⚠️ 熔断：严禁执行 ALTER TABLE ay_content_ext DROP COLUMN
	// 仅删除 ay_extfield 表中的字段定义记录，物理表结构不做任何修改
	return db.DB.Exec("DELETE FROM ay_extfield WHERE id=?", id).Error
}
