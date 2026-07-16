// Package model re-exports the shared DB instance from core/db
// and provides type aliases for all model sub-package types.
// This allows controllers to use model.DB and model.XXX without
// importing sub-packages directly.
package model

import (
	"context"
	"gbootcms/config"
	"gbootcms/core/acodeplugin"
	"gbootcms/core/db"
	"sync"

	// Import sub-packages so their AutoMigrate / helpers are accessible.
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/member"
	"gbootcms/apps/admin/model/system"
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
type RoleArea = system.RoleArea
type Syslog = system.Syslog
type Area = system.Area
type Config = system.Config
type Database = system.Database
type Session = system.Session

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
type Redirect = content.Redirect

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
// Config cache — 避免每個請求 15+ 次 SQL 查 ay_config
// 後台修改配置後由 GORM 回調自動清除
// ──────────────────────────────────────────────

var (
	configCache   map[string]string
	configCacheMu sync.RWMutex
	configCacheOK bool
)

// preloadConfigCache 一次性載入所有配置到記憶體
func preloadConfigCache() {
	var configs []system.Config
	db.DB.Find(&configs)
	m := make(map[string]string, len(configs))
	for _, c := range configs {
		if c.Value != "" {
			m[c.Name] = c.Value
		}
	}
	configCacheMu.Lock()
	configCache = m
	configCacheOK = true
	configCacheMu.Unlock()
}

// ClearConfigCache 清除配置快取（由 GORM 回調觸發）
func ClearConfigCache() {
	configCacheMu.Lock()
	configCacheOK = false
	configCache = nil
	configCacheMu.Unlock()
}

// GetConfigValue reads a config value by name, returning defaultVal if not found or empty.
// 使用記憶體快取，避免每次調用都查 SQL
func GetConfigValue(name, defaultVal string) string {
	configCacheMu.RLock()
	if !configCacheOK {
		configCacheMu.RUnlock()
		preloadConfigCache()
		configCacheMu.RLock()
	}
	if v, ok := configCache[name]; ok {
		configCacheMu.RUnlock()
		return v
	}
	configCacheMu.RUnlock()
	return defaultVal
}

// ──────────────────────────────────────────────
// Areas cache — 區域列表極少變化，6+ 處查詢共用一份記憶體快取
// ──────────────────────────────────────────────

var (
	areasCache     []Area
	areasCacheMu   sync.RWMutex
	areasCacheReady bool
)

// GetCachedAreas 返回快取的區域列表（未命中則查 DB 並快取）
func GetCachedAreas() []Area {
	areasCacheMu.RLock()
	if areasCacheReady {
		defer areasCacheMu.RUnlock()
		return areasCache
	}
	areasCacheMu.RUnlock()

	var areas []Area
	db.DB.WithContext(acodeplugin.SkipAcode(context.Background())).Order("pcode, acode").Find(&areas)
	areasCacheMu.Lock()
	areasCache = areas
	areasCacheReady = true
	areasCacheMu.Unlock()
	return areas
}

// ClearAreasCache 清除區域快取（由 GORM 回調觸發）
func ClearAreasCache() {
	areasCacheMu.Lock()
	areasCacheReady = false
	areasCache = nil
	areasCacheMu.Unlock()
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
