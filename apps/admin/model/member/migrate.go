package member

import (
	"gbootcms/core/db"
)

func AutoMigrate() {
	db.DB.AutoMigrate(
		&Member{},
		&MemberComment{},
		&MemberGroup{},
		&MemberField{},
	)
}
