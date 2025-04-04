// Package config contains method implementations related to the local database that can be shared between CMDs
// in the CLI and which are not implemented in the gRPC server.

package config

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
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
	fmt.Println("MigrationPath Print:", migrationPath)

	home, err := os.UserHomeDir()
	p, err := filepath.Abs(home)
	p = filepath.ToSlash(p)
	p = path.Join(p, ".pennsieve", "migrations")

	testPath := fmt.Sprintf("file://%s", p)

	fmt.Println("testing output", testPath)

	// m, err := migrate.NewWithDatabaseInstance(
	// 	fmt.Sprintf("file://%s", p),
	// 	"postgres", driver)

	filePathForURL := filepath.ToSlash(migrationPath)
	fmt.Println("filePathForURL Print:", filePathForURL)

	fileURL := &url.URL{
		Scheme: "file",
		Path:   "/" + filePathForURL,
	}
	fmt.Println("fileURL Print:", fileURL.String())

	log.Println("BEFORE SQL OPEN")
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&mode=rwc&_journal_mode=WAL")
	if err != nil {
		log.Error("Unable to open database")
	}

	log.Println("BEFORE DRIVER")
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Println("BEFORE MIGRATION NEW DATABASE INIT")
	m, err := migrate.NewWithDatabaseInstance(
		testPath,
		"sqlite3", driver)

	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Println("BEFORE M UP")

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("No change in database schema: ", err)
		} else {
			log.Error(err)
			return nil, err
		}
	}

	log.Println("AFTER MUP")

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
