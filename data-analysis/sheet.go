package data_analysis

import (
	"database/sql"
	"fmt"
	"logistictbot/config"
	"logistictbot/duration"
	"logistictbot/parser"
	"reflect"
	"slices"
	"strings"
	"time"

	ex "github.com/xuri/excelize/v2"
)

type ShipmentStatement struct {
	ShipmentId             int64             `excel:"Nr zlecenia"`
	LoadAddress            string            `excel:"Miejsce zaladunku"`
	LoadDatetime           string            `excel:"Data i godziny zal."`
	LoadDuration           duration.Duration `excel:"Czas zal."`
	UnloadAddress          string            `excel:"Miejsce rozladunku"`
	UnloadDatetime         string            `excel:"Data i godziny roz."`
	UnloadDuration         duration.Duration `excel:"Czas roz."`
	Weight                 string            `excel:"Waga"`
	DurationLoadAndUnload  duration.Duration `excel:"Czas zal+rozl"`
	CleaningStationAddress string            `excel:"Myjka"`
	KmVnR                  int               `excel:"KM V&R"`
	KmHoyer                int               `excel:"KM Hoyer"`
	Difference             int               `excel:"Różnica"`
	Fracht                 int64             `excel:"Fracht"`
}

// for now it won't be used
type CleaningStation struct {
	Id           int     `db:"id"`
	Name         string  `db:"name"`
	Address      string  `db:"address"`
	Country      string  `db:"country"`
	Lat          float64 `db:"lat"`
	Lon          float64 `db:"lon"`
	OpeningHours string  `db:"opening_hours"`
}

func GetHeaders(s interface{}) []string {
	t := reflect.TypeOf(s)
	headers := make([]string, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("excel")
		if tag != "" {
			headers = append(headers, tag)
		} else {
			headers = append(headers, field.Name)
		}
	}
	return headers
}

func WriteHeaders(f *ex.File, sheet string, headers []string) error {
	for i, header := range headers {
		cell, _ := ex.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return err
		}
	}
	return nil
}

func WriteRow(f *ex.File, sheet string, row int, data ShipmentStatement) error {
	v := reflect.ValueOf(data)
	t := reflect.TypeOf(data)

	var kmVnRCol, kmHoyerCol int
	for i := 0; i < t.NumField(); i++ {
		fieldName := t.Field(i).Name
		switch fieldName {
		case "KmVnR":
			kmVnRCol = i + 1
		case "KmHoyer":
			kmHoyerCol = i + 1
		}
	}

	for i := 0; i < v.NumField(); i++ {
		cell, _ := ex.CoordinatesToCellName(i+1, row)
		fieldName := t.Field(i).Name

		if fieldName == "Difference" {
			kmVnRCell, _ := ex.CoordinatesToCellName(kmVnRCol, row)
			kmHoyerCell, _ := ex.CoordinatesToCellName(kmHoyerCol, row)
			formula := fmt.Sprintf("%s-%s", kmHoyerCell, kmVnRCell)
			if err := f.SetCellFormula(sheet, cell, formula); err != nil {
				return err
			}
		} else {
			value := v.Field(i).Interface()

			if dur, ok := value.(duration.Duration); ok {
				formattedDuration := dur.Format(duration.ForDB)
				if err := f.SetCellValue(sheet, cell, formattedDuration); err != nil {
					return err
				}
			} else if station, ok := value.(CleaningStation); ok {
				if err := f.SetCellValue(sheet, cell, station.Name); err != nil {
					return err
				}
			} else {
				if err := f.SetCellValue(sheet, cell, value); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func CreateMonthlyStatement(month time.Month, year int, db *sql.DB) (string, error) {
	shipments, err := parser.GroupByMonth(month, year, db)
	if err != nil {
		return "", fmt.Errorf("error getting shipments: %w", err)
	}

	f := ex.NewFile()
	defer f.Close()

	sheet := "Statement"
	index, err := f.NewSheet(sheet)
	if err != nil {
		return "", fmt.Errorf("error creating sheet: %w", err)
	}
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	headers := GetHeaders(ShipmentStatement{})
	if err := WriteHeaders(f, sheet, headers); err != nil {
		return "", fmt.Errorf("error writing headers: %w", err)
	}

	currentRow := 2
	for _, shipment := range shipments {
		statements := ConvertShipmentToStatements(shipment)

		for _, statement := range statements {
			if err := WriteRow(f, sheet, currentRow, statement); err != nil {
				return "", fmt.Errorf("error writing row %d: %w", currentRow, err)
			}
			currentRow++
		}
	}

	for i := 0; i < len(headers); i++ {
		col, _ := ex.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 15)
	}

	filename := fmt.Sprintf(config.GetOutDocsPath()+"V R_statement_%s_%d.xlsx", month.String(), year)
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("error saving file: %w", err)
	}

	return filename, nil
}

func ConvertShipmentToStatements(shipment *parser.Shipment) []ShipmentStatement {
	statements := make([]ShipmentStatement, 0)

	fmt.Printf("Processing shipment %d with %d tasks\n", shipment.ShipmentId, len(shipment.Tasks))

	var loadTask *parser.TaskSection

	for _, task := range shipment.Tasks {
		taskType := strings.ToLower(strings.TrimSpace(task.Type))

		fmt.Printf("  Processing task type: '%s'\n", taskType)

		if taskType == "load" || taskType == "collect" {
			loadTask = task
			fmt.Printf("  Found load task at: %s\n", task.Address)
		} else if taskType == "unload" || taskType == "dropoff" {
			if loadTask != nil {
				fmt.Printf("  Found unload task, creating statement\n")

				statement := ShipmentStatement{
					ShipmentId:     shipment.ShipmentId,
					LoadAddress:    loadTask.Address,
					LoadDatetime:   formatDateTimeRange(loadTask.Start, loadTask.End),
					LoadDuration:   calculateDuration(loadTask.Start, loadTask.End),
					UnloadAddress:  task.Address,
					UnloadDatetime: formatDateTimeRange(task.Start, task.End),
					UnloadDuration: calculateDuration(task.Start, task.End),
					Weight:         task.Weight,
				}
				statement.DurationLoadAndUnload = duration.Duration{
					Duration: statement.LoadDuration.Duration + statement.UnloadDuration.Duration,
				}

				statements = append(statements, statement)
				fmt.Printf("  Statement added: Shipment %d, Load: %s, Unload: %s\n",
					statement.ShipmentId, statement.LoadAddress, statement.UnloadAddress)

				loadTask = nil
			} else {
				fmt.Printf("  WARNING: Found unload without preceding load task\n")
			}
		} else if taskType == "cleaning" {
			idStmt := slices.IndexFunc(statements, func(s ShipmentStatement) bool {
				return s.ShipmentId == task.ShipmentId
			})
			if idStmt == -1 {
				fmt.Printf("WARNING: found cleaning task, but could not find the shipment with the same shipment id: task - %d\n", task.ShipmentId)
				continue
			}
			statements[idStmt].CleaningStationAddress = task.Address
			fmt.Printf("  Found cleaning task at: %s\n", task.Address)
		}
	}

	fmt.Printf("Total statements created: %d\n", len(statements))
	return statements
}

func formatDateTime(start time.Time) string {
	if start.IsZero() {
		return ""
	}
	return start.Format("2006-01-02 15:04")
}

func formatDateTimeRange(start, end time.Time) string {
	if start.IsZero() {
		return ""
	}

	if end.IsZero() {
		return start.Format("2006-01-02 15:04")
	}

	if start.Format("2006-01-02") == end.Format("2006-01-02") {
		return fmt.Sprintf("%s %s - %s",
			start.Format("2006-01-02"),
			start.Format("15:04"),
			end.Format("15:04"))
	} else {
		return fmt.Sprintf("%s - %s",
			start.Format("2006-01-02 15:04"),
			end.Format("2006-01-02 15:04"))
	}
}
func calculateDuration(start, end time.Time) duration.Duration {
	if start.IsZero() || end.IsZero() {
		return duration.Duration{Duration: 0}
	}
	d := end.Sub(start)
	return duration.Duration{Duration: d}
}

/*func CreateExcelSheet(rows *sql.Rows, tableName string) error {
	columns, err := rows.Columns()
	if err != nil {
		log.Fatalf("Err getting all the columns: %v\n", err)
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


*/
