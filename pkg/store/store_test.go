package store

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pennsieve/pennsieve-agent/migrations"
	"os"
	"path/filepath"
	"testing"
)

var db *sql.DB

func TestMain(m *testing.M) {
	// os.Exit skips defer calls
	// so we need to call another function
	code, err := run(m)
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(code)
}

func run(m *testing.M) (code int, err error) {

	// 1. create test.db if it does not exist
	// 2. run our DDL statements to create the required tables if they do not exist
	// 3. run our tests
	// 4. truncate the test db tables
	home, err := os.UserHomeDir()
	dbPath := filepath.Join(home, ".pennsieve/pennsieve_test.db")
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&mode=rwc&_journal_mode=WAL")
	migrations.Run(db)

	db.Exec("")

	if err != nil {
		return -1, fmt.Errorf("could not connect to database: %w", err)
	}

	// truncates all test data after the tests are run
	defer func() {
		for _, t := range []string{"manifests", "manifest_files", "user_info", "user_settings"} {
			_, _ = db.Exec(fmt.Sprintf("DELETE FROM %s", t))
		}

		db.Close()
	}()

	return m.Run(), nil
}

//func TestRecordCreation(t *testing.T) {
//
//	store := NewManifestStore(db)
//
//}
