package parser

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
)

type MonthYear struct {
	Month time.Month
	Year  int
}

func scanShipment(rows *sql.Rows) (*Shipment, error) {
	shipment := &Shipment{}
	var docLang, instrType string
	var createdAt, updatedAt, started, finished sql.NullString

	err := rows.Scan(
		&shipment.ShipmentId,
		&docLang,
		&instrType,
		&shipment.CarId,
		&shipment.DriverId,
		&shipment.Container,
		&shipment.Chassis,
		&shipment.Tankdetails,
		&shipment.GeneralRemark,
		&shipment.ShipmentDocId,
		&createdAt,
		&updatedAt,
		&started,
		&finished,
	)
	if err != nil {
		return nil, fmt.Errorf("ERR: scan shipment: %w", err)
	}

	shipment.DocLang = Language(docLang)
	shipment.InstructionType = InstructionType(instrType)

	if createdAt.Valid {
		shipment.CreatedAt, _ = parseTimeString(createdAt.String)
	}
	if updatedAt.Valid {
		shipment.UpdatedAt, _ = parseTimeString(updatedAt.String)
	}
	if started.Valid {
		shipment.Started, _ = parseTimeString(started.String)
	}
	if finished.Valid {
		shipment.Finished, _ = parseTimeString(finished.String)
	}

	return shipment, nil
}

func scanTask(taskRows *sql.Rows) (*TaskSection, error) {
	task := &TaskSection{}
	var taskType string
	var loadStartDate, loadEndDate, unloadStartDate, unloadEndDate sql.NullString
	var start, end sql.NullString
	var createdAt, updatedAt sql.NullString
	var compartment sql.NullInt64
	var currentKm, currentWeight sql.NullInt64
	var currentTemp sql.NullFloat64

	err := taskRows.Scan(
		&task.Id,
		&taskType,
		&task.ShipmentId,
		&task.Content,
		&task.CustomerReference,
		&task.LoadReference,
		&loadStartDate,
		&loadEndDate,
		&task.UnloadReference,
		&unloadStartDate,
		&unloadEndDate,
		&task.TankStatus,
		&task.Product,
		&task.Weight,
		&task.Volume,
		&task.Temperature,
		&compartment,
		&task.Remark,
		&task.Address,
		&task.DestinationAddress,
		&task.ShipmentDocId,
		&start,
		&end,
		&currentKm,
		&currentWeight,
		&currentTemp,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("ERR: scan task: %w", err)
	}

	task.Type = taskType

	if loadStartDate.Valid {
		task.LoadStartDate, _ = parseTimeString(loadStartDate.String)
	}
	if loadEndDate.Valid {
		task.LoadEndDate, _ = parseTimeString(loadEndDate.String)
	}
	if unloadStartDate.Valid {
		task.UnloadStartDate, _ = parseTimeString(unloadStartDate.String)
	}
	if unloadEndDate.Valid {
		task.UnloadEndDate, _ = parseTimeString(unloadEndDate.String)
	}
	if start.Valid {
		task.Start, _ = parseTimeString(start.String)
	}
	if end.Valid {
		task.End, _ = parseTimeString(end.String)
	}
	if createdAt.Valid {
		task.CreatedAt, _ = parseTimeString(createdAt.String)
	}
	if updatedAt.Valid {
		task.UpdatedAt, _ = parseTimeString(updatedAt.String)
	}

	if compartment.Valid {
		task.Compartment = int(compartment.Int64)
	}
	if currentKm.Valid {
		task.CurrentKilometrage = currentKm.Int64
	}
	if currentWeight.Valid {
		task.CurrentWeight = int(currentWeight.Int64)
	}
	if currentTemp.Valid {
		task.CurrentTemperature = currentTemp.Float64
	}

	return task, nil
}

func loadTasksForShipment(tx *sql.Tx, shipmentId int64) ([]*TaskSection, error) {
	taskQuery := `
		SELECT id, type, shipment_id, content, customer_ref, load_ref, 
		       load_start_date, load_end_date, unload_ref, unload_start_date, 
		       unload_end_date, tank_status, product, weight, volume, 
		       temperature, compartment, remark, address, destination_address, 
		       doc_id, start, end, current_kilometrage, current_weight, 
		       current_temperature, created_at, updated_at
		FROM tasks
		WHERE shipment_id = ?
		ORDER BY id
	`

	taskRows, err := tx.Query(taskQuery, shipmentId)
	if err != nil {
		return nil, fmt.Errorf("ERR: query tasks for shipment %d: %w", shipmentId, err)
	}
	defer taskRows.Close()

	tasks := make([]*TaskSection, 0)

	for taskRows.Next() {
		task, err := scanTask(taskRows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := taskRows.Err(); err != nil {
		return nil, fmt.Errorf("ERR: iterate tasks: %w", err)
	}

	return tasks, nil
}

func queryShipments(db *sql.DB, query string, args ...interface{}) ([]*Shipment, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("ERR: beginning transaction: %v", err)
	}
	defer tx.Rollback()

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("ERR: query shipments: %w", err)
	}
	defer rows.Close()

	shipments := make([]*Shipment, 0)

	for rows.Next() {
		shipment, err := scanShipment(rows)
		if err != nil {
			return nil, err
		}

		tasks, err := loadTasksForShipment(tx, shipment.ShipmentId)
		if err != nil {
			return nil, err
		}

		shipment.Tasks = tasks
		shipments = append(shipments, shipment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ERR: iterate shipments: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("ERR: commit transaction: %w", err)
	}

	return shipments, nil
}

func GetAllShipments(db *sql.DB) ([]*Shipment, error) {
	query := `
		SELECT id, document_language, instruction_type, car_id, driver_id, 
		       container, chassis, tankdetails, generalremark, doc_id, 
		       created_at, updated_at, started, finished
		FROM shipments
	`
	return queryShipments(db, query)
}

func GetAllShipmentsByCarId(carId string, db *sql.DB) ([]*Shipment, error) {
	query := `
		SELECT id, document_language, instruction_type, car_id, driver_id, 
		       container, chassis, tankdetails, generalremark, doc_id, 
		       created_at, updated_at, started, finished
		FROM shipments
		WHERE car_id = ?
	`
	return queryShipments(db, query, carId)
}

func GetAllActiveShipments(db *sql.DB) ([]*Shipment, error) {
	query := `
		SELECT id, document_language, instruction_type, car_id, driver_id, 
		       container, chassis, tankdetails, generalremark, doc_id, 
		       created_at, updated_at, started, finished
		FROM shipments
		WHERE finished IS NULL OR finished = ''
	`
	return queryShipments(db, query)
}

func GetAllActiveShipmentsByCarId(carId string, db *sql.DB) ([]*Shipment, error) {
	query := `
		SELECT id, document_language, instruction_type, car_id, driver_id, 
		       container, chassis, tankdetails, generalremark, doc_id, 
		       created_at, updated_at, started, finished
		FROM shipments
		WHERE car_id = ? AND (finished IS NULL OR finished = '')
	`
	return queryShipments(db, query, carId)
}

// GroupByMonth retrieves all shipments finished in a specific month and year
func GroupByMonth(month time.Month, year int, db *sql.DB) ([]*Shipment, error) {
	query := `
		SELECT id, document_language, instruction_type, car_id, driver_id, 
		       container, chassis, tankdetails, generalremark, doc_id, 
		       created_at, updated_at, started, finished
		FROM shipments
		WHERE strftime('%m', finished) = ? 
		  AND strftime('%Y', finished) = ?
		ORDER BY finished 
	`
	return queryShipments(db, query, fmt.Sprintf("%02d", month), fmt.Sprintf("%d", year))
}

func parseTimeString(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05.999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("ERR: unable to parse time: %s", s)
}

func GetAvailableMonths(db *sql.DB) ([]MonthYear, error) {
	query := `
		SELECT DISTINCT 
			strftime('%Y', finished) as year,
			strftime('%m', finished) as month
		FROM shipments
		WHERE started IS NOT NULL
		ORDER BY year DESC, month DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ERR: query available months: %w", err)
	}
	defer rows.Close()

	months := make([]MonthYear, 0)
	for rows.Next() {
		var yearStr, monthStr sql.NullString
		if err := rows.Scan(&yearStr, &monthStr); err != nil {
			return nil, fmt.Errorf("ERR: scan month: %w", err)
		}

		if !yearStr.Valid || !monthStr.Valid {
			continue
		}

		year, err := strconv.Atoi(yearStr.String)
		if err != nil {
			continue
		}

		month, err := strconv.Atoi(monthStr.String)
		if err != nil {
			continue
		}

		months = append(months, MonthYear{
			Year:  year,
			Month: time.Month(month),
		})
	}

	return months, rows.Err()
}

// StoreShipment stores a shipment and all its tasks in the database
// Returns the created shipment ID
func (s *Shipment) StoreShipment(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ERR: begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `INSERT INTO shipments 
		(id, document_language, instruction_type, car_id, driver_id, container, chassis, 
		tankdetails, generalremark, doc_id, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	log.Println(s)

	_, err = tx.Exec(
		query,
		s.ShipmentId,
		string(s.DocLang),
		string(s.InstructionType),
		s.CarId,
		s.DriverId.String(),
		s.Container,
		s.Chassis,
		s.Tankdetails,
		s.GeneralRemark,
		s.ShipmentDocId,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("ERR: insert shipment: %w", err)
	}

	taskQuery := `INSERT INTO tasks 
	(type, shipment_id, content, customer_ref, load_ref, load_start_date, load_end_date,
	unload_ref, unload_start_date, unload_end_date, tank_status, product, weight, volume,
	temperature, compartment, remark, address, destination_address, doc_id, current_kilometrage, current_weight, current_temperature, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	for _, task := range s.Tasks {
		if task.Type == "" {
			continue
		}
		task.ShipmentId = s.ShipmentId
		task.ShipmentDocId = s.ShipmentDocId
		fmt.Println(task)

		result, err := tx.Exec(
			taskQuery,
			task.Type,
			s.ShipmentId,
			task.Content,
			task.CustomerReference,
			task.LoadReference,
			formatTime(task.LoadStartDate),
			formatTime(task.LoadEndDate),
			task.UnloadReference,
			formatTime(task.UnloadStartDate),
			formatTime(task.UnloadEndDate),
			task.TankStatus,
			task.Product,
			task.Weight,
			task.Volume,
			task.Temperature,
			task.Compartment,
			task.Remark,
			task.Address,
			task.DestinationAddress,
			task.ShipmentDocId,
			task.CurrentKilometrage,
			task.CurrentTemperature,
			task.CurrentWeight,
			time.Now(),
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("ERR: insert task: %w", err)
		}

		taskId, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("ERR: get task id: %w", err)
		}
		task.Id = int(taskId)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ERR: commit transaction: %w", err)
	}

	return nil
}

// GetShipment retrieves a shipment by ID (without tasks)
func GetShipment(db *sql.DB, shipmentId int64) (*Shipment, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("ERR: beginning transaction for the shipment: %w\n", err)
	}
	defer tx.Rollback()

	query := `SELECT id, document_language, instruction_type, car_id, driver_id, 
		container, chassis, tankdetails, generalremark, doc_id, created_at, updated_at, started, finished 
		FROM shipments WHERE id = ?`
	row := tx.QueryRow(query, shipmentId)
	s := &Shipment{}
	var driverIdStr string
	var docLang, instrType string
	var createdAtStr, updatedAtStr sql.NullString

	var started sql.NullTime
	var finished sql.NullTime

	err = row.Scan(
		&s.ShipmentId,
		&docLang,
		&instrType,
		&s.CarId,
		&driverIdStr,
		&s.Container,
		&s.Chassis,
		&s.Tankdetails,
		&s.GeneralRemark,
		&s.ShipmentDocId,
		&createdAtStr,
		&updatedAtStr,
		&started,
		&finished,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("shipment not found")
		}
		return nil, fmt.Errorf("scan shipment: %w", err)
	}
	s.DocLang = Language(docLang)
	s.InstructionType = InstructionType(instrType)
	if driverIdStr != "" {
		driverId, err := uuid.FromString(driverIdStr)
		if err != nil {
			return nil, fmt.Errorf("parse driver id: %w", err)
		}
		s.DriverId = driverId
	}

	if createdAtStr.Valid && createdAtStr.String != "" {
		t, err := parseTime(createdAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		s.CreatedAt = t
	}

	if updatedAtStr.Valid && updatedAtStr.String != "" {
		t, err := parseTime(updatedAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse updated_at: %w", err)
		}
		s.UpdatedAt = t
	}

	if started.Valid {
		s.Started = started.Time
	}

	if finished.Valid {
		s.Finished = finished.Time
	}

	s.Tasks, err = loadTasksForShipment(tx, s.ShipmentId)
	if err != nil {
		return nil, fmt.Errorf("ERR: getting tasks for the shipment: %v\n", err)
	}

	return s, nil
}

func (s *Shipment) StartShipment(db *sql.DB) error {
	query := `
		UPDATE shipments
		SET started = ? 
		WHERE id = ?
	`

	s.Started = time.Now()

	_, err := db.Exec(query, s.Started.Format("2006-01-02 15:04:05.999999999-07:00"), s.ShipmentId)
	if err != nil {
		return fmt.Errorf("ERR: starting shipment: %v\n", err)
	}

	return nil
}

func (s *Shipment) FinishShipment(db *sql.DB) error {
	query := `
		UPDATE shipments
		SET finished = ? 
		WHERE id = ?
	`

	s.Finished = time.Now()

	_, err := db.Exec(query, s.Finished.Format("2006-01-02 15:04:05.999999999-07:00"), s.ShipmentId)
	if err != nil {
		return fmt.Errorf("ERR: starting shipment: %v\n", err)
	}

	return nil
}

// GetTaskById retrieves a single task by ID
func GetTaskById(db *sql.DB, taskId int) (*TaskSection, error) {
	query := `SELECT id, type, shipment_id, content, customer_ref, load_ref, 
	load_start_date, load_end_date, unload_ref, unload_start_date, unload_end_date,
	tank_status, product, weight, volume, temperature, compartment, remark, 
	address, destination_address, doc_id, start, end, current_kilometrage, current_temperature, current_weight, created_at, updated_at
	FROM tasks WHERE id = ?`

	row := db.QueryRow(query, taskId)

	task := &TaskSection{}

	var (
		loadStart, loadEnd, unloadStart, unloadEnd string
		startTime, endTime                         sql.NullString
	)

	err := row.Scan(
		&task.Id,
		&task.Type,
		&task.ShipmentId,
		&task.Content,
		&task.CustomerReference,
		&task.LoadReference,
		&loadStart,
		&loadEnd,
		&task.UnloadReference,
		&unloadStart,
		&unloadEnd,
		&task.TankStatus,
		&task.Product,
		&task.Weight,
		&task.Volume,
		&task.Temperature,
		&task.Compartment,
		&task.Remark,
		&task.Address,
		&task.DestinationAddress,
		&task.ShipmentDocId,
		&startTime,
		&endTime,
		&task.CurrentKilometrage,
		&task.CurrentTemperature,
		&task.CurrentWeight,
		&sql.NullString{}, // created_at
		&sql.NullString{}, // updated_at
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("scan task: %w", err)
	}

	task.LoadStartDate, _ = parseTime(loadStart)
	task.LoadEndDate, _ = parseTime(loadEnd)
	task.UnloadStartDate, _ = parseTime(unloadStart)
	task.UnloadEndDate, _ = parseTime(unloadEnd)

	if startTime.Valid {
		task.Start, _ = parseTime(startTime.String)
	}
	if endTime.Valid {
		task.End, _ = parseTime(endTime.String)
	}

	return task, nil
}

// GetAllTasksByShipmentId retrieves all tasks for a given shipment
func GetAllTasksByShipmentId(db *sql.DB, shipmentId int64) ([]*TaskSection, error) {

	query := `SELECT id, type, shipment_id, content, customer_ref, load_ref, 
	load_start_date, load_end_date, unload_ref, unload_start_date, unload_end_date,
	tank_status, product, weight, volume, temperature, compartment, remark, 
	address, destination_address, doc_id, start, end, current_kilometrage, current_weight, current_temperature	FROM tasks WHERE shipment_id = ? ORDER BY id ASC`

	rows, err := db.Query(query, shipmentId)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*TaskSection

	for rows.Next() {
		task := &TaskSection{}

		var (
			loadStart, loadEnd, unloadStart, unloadEnd string
			startTime, endTime                         sql.NullString
		)

		err := rows.Scan(
			&task.Id,
			&task.Type,
			&task.ShipmentId,
			&task.Content,
			&task.CustomerReference,
			&task.LoadReference,
			&loadStart,
			&loadEnd,
			&task.UnloadReference,
			&unloadStart,
			&unloadEnd,
			&task.TankStatus,
			&task.Product,
			&task.Weight,
			&task.Volume,
			&task.Temperature,
			&task.Compartment,
			&task.Remark,
			&task.Address,
			&task.DestinationAddress,
			&task.ShipmentDocId,
			&startTime,
			&endTime,
			&task.CurrentKilometrage,
			&task.CurrentTemperature,
			&task.CurrentWeight,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		task.LoadStartDate, _ = parseTime(loadStart)
		task.LoadEndDate, _ = parseTime(loadEnd)
		task.UnloadStartDate, _ = parseTime(unloadStart)
		task.UnloadEndDate, _ = parseTime(unloadEnd)

		if startTime.Valid {
			task.Start, _ = parseTime(startTime.String)
		}
		if endTime.Valid {
			task.End, _ = parseTime(endTime.String)
		}

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

// StartTaskById marks a task as started
func (t *TaskSection) StartTaskById(db *sql.DB) error {
	query := `UPDATE tasks SET start = ?, updated_at = ? WHERE id = ?`

	t.Start = time.Now()

	result, err := db.Exec(query, t.Start.Format(time.RFC3339), time.Now(), t.Id)
	if err != nil {
		return fmt.Errorf("update task start: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

func (t *TaskSection) UpdateCurrentKmById(db *sql.DB) error {
	query := `UPDATE tasks SET current_kilometrage= ? WHERE id = ?`

	result, err := db.Exec(query, t.CurrentKilometrage, t.Id)
	if err != nil {
		return fmt.Errorf("update km: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

func (t *TaskSection) UpdateCurrentWeightById(db *sql.DB) error {
	query := `UPDATE tasks SET current_weight= ? WHERE id = ?`

	result, err := db.Exec(query, t.CurrentWeight, t.Id)
	if err != nil {
		return fmt.Errorf("update kg: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

func (t *TaskSection) UpdateCurrentTempById(db *sql.DB) error {
	query := `UPDATE tasks SET current_temperature= ? WHERE id = ?`

	result, err := db.Exec(query, t.CurrentTemperature, t.Id)
	if err != nil {
		return fmt.Errorf("update c: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// FinishTaskById marks a task as finished
func (t *TaskSection) FinishTaskById(db *sql.DB) error {
	query := `UPDATE tasks SET end = ?, updated_at = ? WHERE id = ?`

	t.End = time.Now()

	result, err := db.Exec(query, t.End.Format(time.RFC3339), time.Now(), t.Id)
	if err != nil {
		return fmt.Errorf("update task end: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func parseTime(timeStr string) (time.Time, error) {
	if timeStr == "" || timeStr == "0001-01-01T00:00:00Z" {
		return time.Time{}, nil // Return zero time
	}

	formats := []string{
		"2006-01-02 15:04:05.999999-07:00",
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		t, err := time.Parse(format, timeStr)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}
