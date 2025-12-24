package data_analysis

import (
	"database/sql"
	"fmt"
	"log"

	ex "github.com/xuri/excelize/v2"
)

type CleaningStation struct {
	Id           int     `db:"id"`
	Name         string  `db:"name"`
	Address      string  `db:"address"`
	Country      string  `db:"country"`
	Lat          float64 `db:"lat"`
	Lon          float64 `db:"lon"`
	OpeningHours string  `db:"opening_hours"`
}

func CreateExcelSheet(globalStorage *sql.DB, tableName string) error {

	rows, err := globalStorage.Query(fmt.Sprintf("SELECT * FROM cleaning_stations"))
	if err != nil {
		return fmt.Errorf("Err querying the table %s, err: %v\n", tableName, err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("Err getting all the columns: %v\n", err)
	}

	f := ex.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalf("err closing file?: %v\n", err)
		}
	}()

	index, err := f.NewSheet(tableName)
	if err != nil {
		return fmt.Errorf("Error creating a sheet: %v\n", err)
	}

	for i, col := range columns {
		cell, _ := ex.CoordinatesToCellName(i+1, 1)
		err = f.SetCellValue(tableName, cell, col)
		if err != nil {
			return err
		}
	}

	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	rowNum := 2
	for rows.Next() {
		err = rows.Scan(valuePtrs...)
		if err != nil {
			return err
		}

		for i, val := range values {
			cell, _ := ex.CoordinatesToCellName(i+1, rowNum)

			switch v := val.(type) {
			case nil:
				err = f.SetCellValue(tableName, cell, "")
			case []byte:
				err = f.SetCellValue(tableName, cell, string(v))
			default:
				err = f.SetCellValue(tableName, cell, v)
			}
			if err != nil {
				return fmt.Errorf("Err setting value in cell: %s\n", cell)
			}
		}
		rowNum++
	}

	f.SetActiveSheet(index)

	if err := f.SaveAs("Book1.xlsx"); err != nil {
		return fmt.Errorf("Error creating book1: %v\n", err)
	}
	log.Println("Done. Book1 created")
	return nil
}
