package data_analysis

import (
	"database/sql"
	"log"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateExcelSheet(t *testing.T) {
	path, err := filepath.Abs("../bot.db")
	log.Println(path, err)

	globalStorage, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}
	defer globalStorage.Close()

	_, err = globalStorage.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		log.Fatal(err)
	}

	err = CreateExcelSheet(globalStorage, "cleaning_stations")
	if err != nil {
		t.Fatalf("Err creating an excel sheet: %v\n", err)
	}
}
