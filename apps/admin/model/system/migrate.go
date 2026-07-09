package system

import (
	"gbootcms/core/db"
)

func AutoMigrate() {
	db.DB.AutoMigrate(
		&AdminUser{},
		&Menu{},
		&MenuAction{},
		&Role{},
		&RoleLevel{},
		&Syslog{},
		&Area{},
		&Config{},
		&Type{},
		&Database{},
		&DictType{},
	)
}
