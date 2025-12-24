package parser

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gofrs/uuid"
)

// StoreShipment stores a shipment and all its tasks in the database
// Returns the created shipment ID
func (s *Shipment) StoreShipment(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
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
		return fmt.Errorf("insert shipment: %w", err)
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
			return fmt.Errorf("insert task: %w", err)
		}

		taskId, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("get task id: %w", err)
		}
		task.Id = int(taskId)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetShipment retrieves a shipment by ID (without tasks)
func GetShipment(db *sql.DB, shipmentId int64) (*Shipment, error) {
	query := `SELECT id, document_language, instruction_type, car_id, driver_id, 
		container, chassis, tankdetails, generalremark, doc_id, created_at, updated_at, started, finished 
		FROM shipments WHERE id = ?`
	row := db.QueryRow(query, shipmentId)
	s := &Shipment{}
	var driverIdStr string
	var docLang, instrType string
	var createdAtStr, updatedAtStr sql.NullString

	var started sql.NullTime
	var finished sql.NullTime

	err := row.Scan(
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

	return s, nil
}

func (s *Shipment) StartShipment(db *sql.DB) error {
	query := `
		UPDATE shipments
		SET started = ? 
		WHERE id = ?
	`

	s.Started = time.Now()

	_, err := db.Exec(query, s.Started.String(), s.ShipmentId)
	if err != nil {
		return fmt.Errorf("err starting shipment: %v\n", err)
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

	_, err := db.Exec(query, s.Finished.String(), s.ShipmentId)
	if err != nil {
		return fmt.Errorf("err starting shipment: %v\n", err)
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
