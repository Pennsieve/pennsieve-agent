// Package db contains method implementations related to the local database that can be shared between CMDs
// in the CLI and which are not implemented in the gRPC server.

package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
)

var DB *sql.DB

func InitializeDB() (*sql.DB, error) {
	// Initialize connection to the database
	var err error
	home, err := os.UserHomeDir()
	dbPath := filepath.Join(home, ".pennsieve/pennsieve_agent.db")
	DB, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=on")

	return DB, err
}
