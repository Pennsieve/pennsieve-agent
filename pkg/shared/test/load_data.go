package test

import (
	"database/sql"
	"fmt"
	"os"
)

func LoadTestData(pgDB *sql.DB, path string) error {

	fmt.Println("Trying to load test data")
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("adsadsadas")
		return err
	}
	sqlStr := string(sqlBytes)
	_, err = pgDB.Exec(sqlStr)
	if err != nil {
		fmt.Println("ads222adsadas")

		fmt.Println(err)
		return err
	}

	return nil
}
