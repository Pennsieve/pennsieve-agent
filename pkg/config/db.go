// Package config contains method implementations related to the local database that can be shared between CMDs
// in the CLI and which are not implemented in the gRPC server.

package config

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pennsieve/pennsieve-agent/migrations"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
)

// InitializeDB initialized local SQL DB and creates userinfo for current user.
// This method returns a sql.config instance and:
// * Ensures that this config has the correct tables
func InitializeDB() (*sql.DB, error) {
	// Initialize connection to the database
	dbPath := viper.GetString("agent.db_path")
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&mode=rwc&_journal_mode=WAL")
	if err != nil {
		log.Error("Unable to open database")
	}

	userSettingsStore := store.NewUserSettingsStore(db)

	// Get current user-settings. This is either 0, or 1 entry.
	_, err = userSettingsStore.Get()
	if err != nil {

		// We expect that there is an error if we are running agent with ENV variables.
		useConfig := viper.GetBool("agent.useConfigFile")
		target := &store.NoClientSessionError{}

		if err == sql.ErrNoRows || strings.ContainsAny(err.Error(), "no such table") {
			// The database does not exist or no userSettings are defined in the table.

			// If the table does not exist, run migrations.
			if strings.ContainsAny(err.Error(), "no such table") {
				log.Info("Setting up the local database and running migrations.")
				migrations.Run(db)
			}

		} else if errors.As(err, &target) && !useConfig {
			log.Info("No user record in db, but using environment variables.")

		} else {
			log.Fatalln(err)
		}
	}

	return db, nil
}
