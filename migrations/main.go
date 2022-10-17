package migrations

import (
	"database/sql"
	log "github.com/sirupsen/logrus"
)

func Run(db *sql.DB) {
	// Iterate over migration steps
	migrate(db, UserInfo)
	migrate(db, UserSettings)
	migrate(db, Manifests)
	migrate(db, ManifestFiles)
	// Other migrations can be added here.

	log.Info("Database initialized...")

}
func migrate(dbDriver *sql.DB, query string) {
	statement, err := dbDriver.Prepare(query)
	if err == nil {
		_, creationError := statement.Exec()
		if creationError != nil {
			log.Error(creationError.Error())
		}
	} else {
		log.Error(err.Error())
	}
}
