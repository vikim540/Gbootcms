package content

import (
	"gbootcms/core/db"
)

func AutoMigrate() {
	db.DB.AutoMigrate(
		&Company{},
		&Content{},
		&ContentSort{},
		&ExtField{},
		&Form{},
		&FormField{},
		&Label{},
		&Link{},
		&Message{},
		&Model{},
		&Redirect{},
		&Single{},
		&Site{},
		&Slide{},
		&Tags{},
		&MediaMark{},
	)
}
