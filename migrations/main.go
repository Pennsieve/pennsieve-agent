package migrations

import (
	"database/sql"
	"fmt"
	"log"
)

func Run(db *sql.DB) {
	// Iterate over migration steps
	migrate(db, UserInfo)
	migrate(db, UserSettings)
	migrate(db, Manifests)
	migrate(db, ManifestFiles)
	// Other migrations can be added here.

	fmt.Println("Database initialized...")

}
func migrate(dbDriver *sql.DB, query string) {
	statement, err := dbDriver.Prepare(query)
	if err == nil {
		_, creationError := statement.Exec()
		if creationError != nil {
			log.Println(creationError.Error())
		}
	} else {
		log.Println(err.Error())
	}
}
