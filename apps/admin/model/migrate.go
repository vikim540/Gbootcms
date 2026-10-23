package model

import (
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/member"
	"gbootcms/apps/admin/model/system"
	"gbootcms/core/db"
)

// AutoMigrate 統一自動遷移所有資料表（收斂原 content/member/system 分散定義）
func AutoMigrate() error {
	// 確保 ay_content 表 rel_id 欄位存在（相容舊庫，忽略重複欄位錯誤）
	db.DB.Exec("ALTER TABLE ay_content ADD COLUMN rel_id INTEGER DEFAULT 0")

	// 確保多語言精準跨語言關聯 rel_id 欄位（冪等更新）
	db.DB.Exec("UPDATE ay_content SET rel_id = 13 WHERE id IN (13, 44, 65)")
	db.DB.Exec("UPDATE ay_content SET rel_id = 9 WHERE id IN (9, 40, 61)")
	db.DB.Exec("UPDATE ay_content SET rel_id = 1 WHERE id IN (1, 33, 56)")

	return db.DB.AutoMigrate(
		// 1. system
		&system.AdminUser{},
		&system.Menu{},
		&system.MenuAction{},
		&system.Role{},
		&system.RoleLevel{},
		&system.Syslog{},
		&system.Area{},
		&system.Config{},
		&system.Database{},
		&system.Session{},

		// 2. content
		&content.Company{},
		&content.Content{},
		&content.ContentSort{},
		&content.ExtField{},
		&content.Form{},
		&content.FormField{},
		&content.Link{},
		&content.Message{},
		&content.Model{},
		&content.Site{},
		&content.Slide{},
		&content.SlideGroup{},
		&content.Tags{},
		&content.MediaMark{},

		// 3. member
		&member.Member{},
		&member.MemberComment{},
		&member.MemberGroup{},
		&member.MemberField{},
	)
}
