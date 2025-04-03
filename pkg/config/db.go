// Package config contains method implementations related to the local database that can be shared between CMDs
// in the CLI and which are not implemented in the gRPC server.

package config

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// InitializeDB initialized local SQL DB and creates userinfo for current user.
// This method returns a sql.config instance and:
// * Ensures that this config has the correct tables
func InitializeDB() (*sql.DB, error) {
	// Initialize connection to the database
	fmt.Println("Initializing DB...")
	dbPath := viper.GetString("agent.db_path")
	migrationPath := viper.GetString("migration.path")
	log.Println(migrationPath)
	log.Println(viper.GetString("migration.local"))

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&mode=rwc&_journal_mode=WAL")
	if err != nil {
		log.Error("Unable to open database")
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	m, err := migrate.NewWithDatabaseInstance(
		migrationPath,
		"sqlite3", driver)

	if err != nil {
		log.Error(err)
		return nil, err
	}
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("No change in database schema: ", err)
		} else {
			log.Error(err)
			return nil, err
		}
	}

	err = db.Ping()
	if err != nil {
		log.Errorf("unable to connect to database at %s: %s", dbPath, err)
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
			log.Info(err)

		} else if errors.As(err, &target) && !useConfig {
			log.Info("No user record in db, but using environment variables.")

		} else {
			log.Fatalln(err)
		}
	}

	return db, nil
}
