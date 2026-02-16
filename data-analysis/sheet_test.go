package data_analysis

import (
	_ "github.com/mattn/go-sqlite3"
)

/*func TestCreateExcelSheet(t *testing.T) {
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

	rows, err := globalStorage.Query(fmt.Sprintf("SELECT * FROM cleaning_stations"))
	if err != nil {
		log.Fatalf("ERR: querying the table, err: %v\n", err)
	}
	defer rows.Close()

	// Get headers from struct
	headers := getHeaders(ShipmentStatement{})

	f := ex.NewFile()
	defer f.Close()

	sheet := "Sheet1"

	err = writeHeaders(f, sheet, headers)
	if err != nil {
		t.Fatalf("err writing headers from the struct: %v\n", err)
	}

	style, _ := f.NewStyle(&ex.Style{
		Font: &ex.Font{Bold: true},
	})
	err = f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", string(rune('A'+len(headers)-1))), style)
	if err != nil {
		t.Fatalf("Err setting style of the cells of the headers: %v\n", err)
	}

	if err := f.SaveAs("statement.xlsx"); err != nil {
		fmt.Println("ERR: ", err)
	}
}
*/
