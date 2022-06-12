package migrations

import (
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/pkg/db"
	"log"
)

func Run() {
	// Iterate over migration steps
	migrate(db.DB, UserInfo)
	migrate(db.DB, UserSettings)
	migrate(db.DB, Manifests)
	migrate(db.DB, ManifestFiles)
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
