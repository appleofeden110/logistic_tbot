package data_analysis

import (
	"database/sql"
	"fmt"
	"logistictbot/config"
	"logistictbot/db"
	"time"

	"github.com/gofrs/uuid"
	ex "github.com/xuri/excelize/v2"
)

type RefuelStatement struct {
	Id                 int     `excel:"ID"`
	ShipmentId         string  `excel:"Nr zlecenia"`
	FuelCard           string  `excel:"Паливна картка"`
	Car                string  `excel:"Машина"`
	CurrentKilometrage int64   `excel:"Кілометраж"`
	Address            string  `excel:"Адреса"`
	Diesel             float64 `excel:"Дизель (л)"`
	AdBlu              float64 `excel:"AdBlue (л)"`
	CreatedAt          string  `excel:"Дата"`
}

func convertRefuelToStatement(r db.TankRefuel) RefuelStatement {
	shipmentId := ""
	if r.ShipmentId != nil {
		shipmentId = fmt.Sprintf("%d", *r.ShipmentId)
	}

	car := ""
	if r.Driver != nil {
		car = r.Driver.CarId
	}

	return RefuelStatement{
		Id:                 r.Id,
		ShipmentId:         shipmentId,
		FuelCard:           r.FuelCard,
		Car:                car,
		CurrentKilometrage: r.CurrentKilometrage,
		Address:            r.Address,
		Diesel:             r.Diesel,
		AdBlu:              r.AdBlu,
		CreatedAt:          r.CreatedAt.Format("02-01-2006 15:04"),
	}
}

func writeRefuelRow(f *ex.File, sheet string, row int, data RefuelStatement) error {
	values := []interface{}{
		data.Id,
		data.ShipmentId,
		data.FuelCard,
		data.Car,
		data.CurrentKilometrage,
		data.Address,
		data.Diesel,
		data.AdBlu,
		data.CreatedAt,
	}

	for i, value := range values {
		cell, _ := ex.CoordinatesToCellName(i+1, row)
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			return fmt.Errorf("ERR: setting cell value at col %d row %d: %w", i+1, row, err)
		}
	}
	return nil
}

func CreateRefuelsStatement(from, to time.Time, storage *sql.DB) (string, error) {
	refuels, err := db.GetAllTankRefuels(storage)
	if err != nil {
		return "", fmt.Errorf("ERR: getting tank refuels: %w", err)
	}

	if !from.IsZero() || !to.IsZero() {
		filtered := refuels[:0]
		for _, r := range refuels {
			if (!from.IsZero() && r.CreatedAt.Before(from)) ||
				(!to.IsZero() && r.CreatedAt.After(to)) {
				continue
			}
			filtered = append(filtered, r)
		}
		refuels = filtered
	}

	f := ex.NewFile()
	defer f.Close()

	sheet := "Заправки"
	index, err := f.NewSheet(sheet)
	if err != nil {
		return "", fmt.Errorf("ERR: creating sheet: %w", err)
	}
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	headers := GetHeaders(RefuelStatement{})
	if err := WriteHeaders(f, sheet, headers); err != nil {
		return "", fmt.Errorf("ERR: writing headers: %w", err)
	}

	currentRow := 2
	for _, refuel := range refuels {
		statement := convertRefuelToStatement(refuel)
		if err := writeRefuelRow(f, sheet, currentRow, statement); err != nil {
			return "", fmt.Errorf("ERR: writing row %d: %w", currentRow, err)
		}
		currentRow++
	}

	for i := 0; i < len(headers); i++ {
		col, _ := ex.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 20)
	}

	var filename string
	if from.IsZero() && to.IsZero() {
		filename = fmt.Sprintf(config.GetOutDocsPath() + "refuels_all.xlsx")
	} else {
		filename = fmt.Sprintf(
			config.GetOutDocsPath()+"refuels_%s_%s.xlsx",
			from.Format("02-01-2006"),
			to.Format("02-01-2006"),
		)
	}

	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("ERR: saving refuels xlsx: %w", err)
	}

	return filename, nil
}
func CreateRefuelsStatementByDriver(driverId uuid.UUID, storage *sql.DB) (string, error) {
	refuels, err := db.GetTankRefuelsByDriver(storage, driverId)
	if err != nil {
		return "", fmt.Errorf("ERR: getting tank refuels for driver %s: %w", driverId, err)
	}

	f := ex.NewFile()
	defer f.Close()

	sheet := "Заправки"
	index, err := f.NewSheet(sheet)
	if err != nil {
		return "", fmt.Errorf("ERR: creating sheet: %w", err)
	}
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	headers := GetHeaders(RefuelStatement{})
	if err := WriteHeaders(f, sheet, headers); err != nil {
		return "", fmt.Errorf("ERR: writing headers: %w", err)
	}

	currentRow := 2
	for _, refuel := range refuels {
		statement := convertRefuelToStatement(refuel)
		if err := writeRefuelRow(f, sheet, currentRow, statement); err != nil {
			return "", fmt.Errorf("ERR: writing row %d: %w", currentRow, err)
		}
		currentRow++
	}

	for i := 0; i < len(headers); i++ {
		col, _ := ex.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 20)
	}

	carId := driverId.String()[:8]
	if len(refuels) > 0 && refuels[0].Driver != nil && refuels[0].Driver.CarId != "" {
		carId = refuels[0].Driver.CarId
	}

	filename := fmt.Sprintf(config.GetOutDocsPath()+"refuels_%s.xlsx", carId)
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("ERR: saving refuels xlsx: %w", err)
	}

	return filename, nil
}
