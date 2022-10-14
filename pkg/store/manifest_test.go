package store

import (
	"database/sql"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/migrations"
	"os"
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
	// pseudo-code, some implementation excluded:
	//
	// 1. create test.db if it does not exist
	// 2. run our DDL statements to create the required tables if they do not exist
	// 3. run our tests
	// 4. truncate the test db tables

	db, err := sql.Open("sqlite3", "file:../test.db?cache=shared")
	migrations.Run(db)

	db.Exec("")

	if err != nil {
		return -1, fmt.Errorf("could not connect to database: %w", err)
	}

	// truncates all test data after the tests are run
	defer func() {
		for _, t := range []string{"books", "authors"} {
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
