package content

import (
	"errors"
	"fmt"
	"pbootcms-go/core/db"
	"strconv"
	"time"
)

type Model struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Mcode      string `gorm:"column:mcode" json:"mcode"`
	Name       string `gorm:"column:name" json:"name"`
	Type       int    `gorm:"column:type" json:"type"`
	URLName    string `gorm:"column:urlname" json:"urlname"`
	ListTpl    string `gorm:"column:listtpl;default:''" json:"listtpl"`
	ContentTpl string `gorm:"column:contenttpl;default:''" json:"contenttpl"`
	Status     int    `gorm:"column:status" json:"status"`
	Issystem   int    `gorm:"column:issystem" json:"issystem"`
	CreateUser string `gorm:"column:create_user" json:"create_user"`
	UpdateUser string `gorm:"column:update_user" json:"update_user"`
	CreateTime string `gorm:"column:create_time" json:"create_time"`
	UpdateTime string `gorm:"column:update_time" json:"update_time"`
}

func modelNow() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func GetAllModels() []Model {
	var list []Model
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(mcode,'') AS mcode, COALESCE(name,'') AS name, COALESCE(type,1) AS type, COALESCE(urlname,'') AS urlname, COALESCE(listtpl,'') AS listtpl, COALESCE(contenttpl,'') AS contenttpl, COALESCE(status,1) AS status, COALESCE(issystem,0) AS issystem, COALESCE(create_user,'') AS create_user, COALESCE(update_user,'') AS update_user, COALESCE(create_time,'') AS create_time, COALESCE(update_time,'') AS update_time FROM ay_model ORDER BY id ASC").Scan(&list)
	return list
}

func GetModelById(id int) Model {
	var m Model
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(mcode,'') AS mcode, COALESCE(name,'') AS name, COALESCE(type,1) AS type, COALESCE(urlname,'') AS urlname, COALESCE(listtpl,'') AS listtpl, COALESCE(contenttpl,'') AS contenttpl, COALESCE(status,1) AS status, COALESCE(issystem,0) AS issystem, COALESCE(create_user,'') AS create_user, COALESCE(update_user,'') AS update_user, COALESCE(create_time,'') AS create_time, COALESCE(update_time,'') AS update_time FROM ay_model WHERE id = ?", id).Scan(&m)
	return m
}

func GetModelByMcode(mcode string) Model {
	var m Model
	db.DB.Raw("SELECT COALESCE(id,0) AS id, COALESCE(mcode,'') AS mcode, COALESCE(name,'') AS name, COALESCE(type,1) AS type, COALESCE(urlname,'') AS urlname, COALESCE(listtpl,'') AS listtpl, COALESCE(contenttpl,'') AS contenttpl, COALESCE(status,1) AS status, COALESCE(issystem,0) AS issystem, COALESCE(create_user,'') AS create_user, COALESCE(update_user,'') AS update_user, COALESCE(create_time,'') AS create_time, COALESCE(update_time,'') AS update_time FROM ay_model WHERE mcode = ?", mcode).Scan(&m)
	return m
}

func AddModel(mcode, name, urlname, listtpl, contenttpl, updateUser string, typ, status int) error {
	now := modelNow()
	return db.DB.Exec(
		"INSERT INTO ay_model (mcode, name, type, urlname, listtpl, contenttpl, status, issystem, create_user, update_user, create_time, update_time) VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)",
		mcode, name, typ, urlname, listtpl, contenttpl, status, updateUser, updateUser, now, now,
	).Error
}

func UpdateModel(id int, name, urlname, listtpl, contenttpl, updateUser string, typ, status int) error {
	return db.DB.Exec(
		"UPDATE ay_model SET name=?, type=?, urlname=?, listtpl=?, contenttpl=?, status=?, update_user=?, update_time=? WHERE id=?",
		name, typ, urlname, listtpl, contenttpl, status, updateUser, modelNow(), id,
	).Error
}

func UpdateModelSingleField(id int, field, value string, updateUser string) error {
	return db.DB.Exec("UPDATE ay_model SET "+field+" = ?, update_user=?, update_time=? WHERE id=?", value, updateUser, modelNow(), id).Error
}

// DeleteModel deletes a model after checking issystem and content_sort association.
func DeleteModel(id int) error {
	// 1. Check if system model
	var m Model
	db.DB.Raw("SELECT issystem FROM ay_model WHERE id = ?", id).Scan(&m)
	if m.Issystem == 1 {
		return errors.New("系統內置模型禁止刪除")
	}

	// 2. Check content_sort association
	var mcode string
	db.DB.Raw("SELECT mcode FROM ay_model WHERE id = ?", id).Scan(&mcode)
	if mcode != "" {
		var count int64
		db.DB.Raw("SELECT COUNT(*) FROM ay_content_sort WHERE mcode = ?", mcode).Scan(&count)
		if count > 0 {
			return fmt.Errorf("該模型已被 %d 個欄目關聯，無法刪除", count)
		}
	}

	return db.DB.Exec("DELETE FROM ay_model WHERE id=?", id).Error
}

// GetNextMcode returns the next available mcode by incrementing the last one.
// Convention: mcode format is "3D" + number (e.g. 3D1, 3D2, 3D3).
func GetNextMcode() string {
	var last struct{ Mcode string }
	db.DB.Raw("SELECT mcode FROM ay_model ORDER BY id DESC LIMIT 1").Scan(&last)
	if last.Mcode == "" {
		return "3D1"
	}
	// Try to extract numeric suffix
	num := 0
	for i, ch := range last.Mcode {
		if ch >= '0' && ch <= '9' {
			n, _ := strconv.Atoi(last.Mcode[i:])
			num = n + 1
			prefix := last.Mcode[:i]
			return prefix + strconv.Itoa(num)
		}
	}
	// Fallback: append "1"
	return last.Mcode + "1"
}

// CheckUrlnameConflict checks if a urlname conflicts with other models (excluding self)
// or with any content_sort filename.
func CheckUrlnameConflict(urlname string, excludeID int) string {
	if urlname == "" {
		return ""
	}
	// Check against other models
	var count int64
	db.DB.Raw("SELECT COUNT(*) FROM ay_model WHERE urlname = ? AND id != ?", urlname, excludeID).Scan(&count)
	if count > 0 {
		return "URL名稱已被其他模型使用"
	}
	// Check against content_sort filename
	db.DB.Raw("SELECT COUNT(*) FROM ay_content_sort WHERE filename = ? AND filename != ''", urlname).Scan(&count)
	if count > 0 {
		return "URL名稱與欄目文件名稱衝突"
	}
	return ""
}
