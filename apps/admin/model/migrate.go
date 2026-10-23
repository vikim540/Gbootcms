package model

import (
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/member"
	"gbootcms/apps/admin/model/system"
	"gbootcms/core/db"
)

// AutoMigrate 統一自動遷移所有資料表（收斂原 content/member/system 分散定義）
func AutoMigrate() error {
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
		&content.Label{},
		&content.Link{},
		&content.Message{},
		&content.Model{},
		&content.Redirect{},
		&content.Single{},
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
