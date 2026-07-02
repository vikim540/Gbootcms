package member

import (
	"pbootcms-go/core/db"
)

func AutoMigrate() {
	db.DB.AutoMigrate(
		&Member{},
		&MemberComment{},
		&MemberGroup{},
		&MemberField{},
	)
}
