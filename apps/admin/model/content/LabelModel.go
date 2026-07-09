package content

import (
	"gbootcms/core/db"
	"log"
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

// AddLabelFull 新增標籤（含描述和類型）
func AddLabelFull(name, value, description, updateUser string, typ int) error {
	now := nowStr()
	return db.DB.Exec("INSERT INTO ay_label (name, value, type, description, create_user, update_user, create_time, update_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", name, value, typ, description, updateUser, updateUser, now, now).Error
}

func UpdateLabel(id int, name, value, updateUser string) error {
	return db.DB.Exec("UPDATE ay_label SET name=?, value=?, update_user=?, update_time=? WHERE id=?", name, value, updateUser, nowStr(), id).Error
}

func DeleteLabel(id int) error {
	return db.DB.Exec("DELETE FROM ay_label WHERE id=?", id).Error
}

// BatchUpdateLabelValues 批量更新標籤值（POST /admin/Label/index 的核心邏輯）
// 遍歷所有 POST 鍵，過濾合法標籤名，對 value 字段進行換行符替換後批量 UPDATE
func BatchUpdateLabelValues(postForm map[string]string, updateUser string) int {
	updated := 0
	validName := regexp.MustCompile(`^[\w\-]+$`)
	// formcheck 是 CSRF token，不是標籤名，跳過
	skipFields := map[string]bool{"formcheck": true}
	for name, value := range postForm {
		if skipFields[name] {
			continue
		}
		if !validName.MatchString(name) {
			continue
		}
		finalValue := strings.ReplaceAll(value, "\r\n", "<br>")
		finalValue = strings.ReplaceAll(finalValue, "\n", "<br>")
		result := db.DB.Exec("UPDATE ay_label SET value=?, update_user=?, update_time=? WHERE name=?", finalValue, updateUser, nowStr(), name)
		if result.Error != nil {
			log.Printf("[BatchUpdateLabelValues] UPDATE error for name=%s: %v", name, result.Error)
		}
		if result.RowsAffected > 0 {
			updated++
		}
	}
	return updated
}
