package member

import (
	"pbootcms-go/core/db"
)

func AutoMigrate() {
	db.DB.AutoMigrate(
		&Member{},
		&Comment{},
		&MemberGroup{},
		&MemberField{},
	)
}
