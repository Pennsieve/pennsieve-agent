package migrations

import (
	"database/sql"
	"github.com/pennsieve/pennsieve-agent/config"
	"log"
)

func Run() {
	// Migrate notes
	migrate(config.DB, UserInfo)
	migrate(config.DB, UserSettings)
	migrate(config.DB, UploadRecords)
	// Other migrations can be added here.
}
func migrate(dbDriver *sql.DB, query string) {
	statement, err := dbDriver.Prepare(query)
	if err == nil {
		_, creationError := statement.Exec()
		if creationError == nil {
			log.Println("Table created successfully")
		} else {
			log.Println(creationError.Error())
		}
	} else {
		log.Println(err.Error())
	}
}
