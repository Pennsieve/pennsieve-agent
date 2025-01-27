package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pennsieve/pennsieve-agent/pkg/shared/test"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"testing"
)

var db *sql.DB

func TestMain(m *testing.M) {
	// os.Exit skips defer calls, so we need to call another function
	code, err := run(m)
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(code)
}

// TempFileName generates a temporary filename for use in testing or whatever
func TempFileName(prefix, suffix string) string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join("./", prefix+hex.EncodeToString(randBytes)+suffix)
}

func run(m *testing.M) (code int, err error) {

	// 1. create test.db if it does not exist
	// 2. run our DDL statements to create the required tables if they do not exist
	// 3. run our tests
	// 4. truncate the test db tables
	home, err := os.UserHomeDir()
	tempDbPath := filepath.Join(home, TempFileName("", ".db"))

	db, err = sql.Open("sqlite3", tempDbPath+"?_foreign_keys=on&mode=rwc&_journal_mode=WAL")
	if err != nil {
		fmt.Println("error opening db:", err)
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	mig, err := migrate.NewWithDatabaseInstance(
		"file://../../db/migrations",
		"sqlite3", driver)

	defer driver.Close()

	if err != nil {
		log.Fatal(err)
		return 1, err
	}
	if err := mig.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("No change in database schema: ", err)
		} else {
			log.Fatal(err)
		}
	}

	testDataPath := filepath.Join("..", "..", "test", "sql", "store-test-data.sql")
	err = test.LoadTestData(db, testDataPath)
	//if err != nil {
	//	Error is possible as other tests might have already loaded the testdata.
	//fmt.Println(err)
	//}

	if err != nil {
		return -1, fmt.Errorf("could not connect to database: %w", err)
	}

	// truncates all test data after the tests are run
	defer func() {
		err := db.Close()
		if err != nil {
			return
		}

		err = os.Remove(tempDbPath)
		if err != nil {
			return
		}

		//for _, t := range []string{"manifests", "manifest_files", "user_info", "user_settings", "ts_channel", "ts_range"} {
		//	_, _ = db.Exec(fmt.Sprintf("DELETE FROM %s", t))
		//}

	}()

	return m.Run(), nil
}
