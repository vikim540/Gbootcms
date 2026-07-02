// Package model re-exports the shared DB instance from core/db
// and provides type aliases for all model sub-package types.
// This allows controllers to use model.DB and model.XXX without
// importing sub-packages directly.
package model

import (
	"pbootcms-go/config"
	"pbootcms-go/core/db"

	// Import sub-packages so their AutoMigrate / helpers are accessible.
	"pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/admin/model/member"
	"pbootcms-go/apps/admin/model/system"
)

// DB is the shared GORM database instance (re-exported from core/db).
var DB = db.DB

// InitDB initialises the database and syncs the DB pointer.
func InitDB(cfg *config.Config) error {
	if err := db.InitDB(cfg); err != nil {
		return err
	}
	DB = db.DB
	return nil
}

// CloseDB closes the database connection.
func CloseDB() {
	db.CloseDB()
}

// ──────────────────────────────────────────────
// Type aliases – system sub-package
// ──────────────────────────────────────────────
type AdminUser = system.AdminUser
type Menu = system.Menu
type MenuAction = system.MenuAction
type Role = system.Role
type RoleLevel = system.RoleLevel
type Syslog = system.Syslog
type Area = system.Area
type Config = system.Config
type Type = system.Type
type DictType = system.DictType
type Database = system.Database

// ──────────────────────────────────────────────
// Type aliases – content sub-package
// ──────────────────────────────────────────────
type Content = content.Content
type ContentSort = content.ContentSort
type Site = content.Site
type Company = content.Company
type Slide = content.Slide
type Link = content.Link
type Message = content.Message
type Tags = content.Tags
type Form = content.Form
type FormField = content.FormField
type Label = content.Label
type ContentModel = content.Model
type ExtField = content.ExtField
type MediaMark = content.MediaMark

// ──────────────────────────────────────────────
// Type aliases – member sub-package
// ──────────────────────────────────────────────
type Member = member.Member
type MemberGroup = member.MemberGroup
type MemberField = member.MemberField
type MemberComment = member.MemberComment
type Comment = member.MemberComment
type CommentView = member.CommentView

// ──────────────────────────────────────────────
// Helper functions (previously missing)
// ──────────────────────────────────────────────

// GetConfigValue reads a config value by name, returning defaultVal if not found or empty.
func GetConfigValue(name, defaultVal string) string {
	var cfg system.Config
	if db.DB.Where("name = ?", name).First(&cfg).Error == nil {
		if cfg.Value != "" {
			return cfg.Value
		}
	}
	return defaultVal
}

// GetDBName returns the current database file name from config.
func GetDBName() string {
	cfg := config.Get()
	return cfg.Database.DBName
}

// TableName adds the ay_ prefix to a table name.
func TableName(name string) string {
	return "ay_" + name
}
