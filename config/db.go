package config

import (
	"database/sql"
	"os"
	"path/filepath"
)

var DB *sql.DB

func InitializeDB() (*sql.DB, error) {
	// Initialize connection to the database
	var err error
	home, err := os.UserHomeDir()
	dbPath := filepath.Join(home, ".pennsieve/pennsieve_agent.db")
	DB, err = sql.Open("sqlite3", dbPath)
	return DB, err
}
