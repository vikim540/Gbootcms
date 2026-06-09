package content

import (
	"pbootcms-go/core/db"
	"regexp"
	"strings"
	"time"
)

type Label struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"column:name" json:"name"`
	Value       string `gorm:"column:value" json:"value"`
	Type        int    `gorm:"column:type" json:"type"`
	Description string `gorm:"column:description" json:"description"`
	CreateUser  string `gorm:"column:create_user" json:"create_user"`
	UpdateUser  string `gorm:"column:update_user" json:"update_user"`
	CreateTime  string `gorm:"column:create_time" json:"create_time"`
	UpdateTime  string `gorm:"column:update_time" json:"update_time"`
}

func nowStr() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func GetAllLabels() []Label {
	var list []Label
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(name,'') AS name, COALESCE(value,'') AS value, COALESCE(type,1) AS type, COALESCE(description,'') AS description, COALESCE(create_user,'') AS create_user, COALESCE(update_user,'') AS update_user, COALESCE(create_time,'') AS create_time, COALESCE(update_time,'') AS update_time FROM ay_label ORDER BY id ASC").Scan(&list)
	return list
}

func GetLabelById(id int) Label {
	var lb Label
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(name,'') AS name, COALESCE(value,'') AS value, COALESCE(type,1) AS type, COALESCE(description,'') AS description, COALESCE(create_user,'') AS create_user, COALESCE(update_user,'') AS update_user, COALESCE(create_time,'') AS create_time, COALESCE(update_time,'') AS update_time FROM ay_label WHERE id = ?", id).Scan(&lb)
	return lb
}

func AddLabel(name, value, updateUser string, typ int) error {
	now := nowStr()
	return db.DB.Exec("INSERT INTO ay_label (name, value, type, description, create_user, update_user, create_time, update_time) VALUES (?, ?, ?, '', ?, ?, ?, ?)", name, value, typ, updateUser, updateUser, now, now).Error
}

func UpdateLabel(id int, name, value, updateUser string) error {
	return db.DB.Exec("UPDATE ay_label SET name=?, value=?, update_user=?, update_time=? WHERE id=?", name, value, updateUser, nowStr(), id).Error
}

func DeleteLabel(id int) error {
	return db.DB.Exec("DELETE FROM ay_label WHERE id=?", id).Error
}

// BatchUpdateLabelValues 批量更新标签值（POST /admin/Label/index 的核心逻辑）
// 遍历所有 POST 键，过滤合法标签名，对 value 字段进行换行符替换后批量 UPDATE
func BatchUpdateLabelValues(postForm map[string]string, updateUser string) int {
	updated := 0
	validName := regexp.MustCompile(`^[\w\-]+$`)
	for name, value := range postForm {
		if !validName.MatchString(name) {
			continue
		}
		finalValue := strings.ReplaceAll(value, "\r\n", "<br>")
		finalValue = strings.ReplaceAll(finalValue, "\n", "<br>")
		result := db.DB.Exec("UPDATE ay_label SET value=?, update_user=?, update_time=? WHERE name=?", finalValue, updateUser, nowStr(), name)
		if result.RowsAffected > 0 {
			updated++
		}
	}
	return updated
}
