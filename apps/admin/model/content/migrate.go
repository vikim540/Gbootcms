package content

import (
	"pbootcms-go/core/db"
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
		&Single{},
		&Site{},
		&Slide{},
		&Tags{},
	)
}
