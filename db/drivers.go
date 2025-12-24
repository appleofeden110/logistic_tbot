package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"logistictbot/parser"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofrs/uuid"
	"github.com/tiendc/go-deepcopy"
)

type DriverConversationState string

const (
	StateWaitingLoc        DriverConversationState = "waiting_loc"
	StateCompletingForm    DriverConversationState = "waiting_form_end"
	StatePause             DriverConversationState = "on_rest"
	StateWorking           DriverConversationState = "working"
	StateLoad              DriverConversationState = "performing_load"
	StateUnload            DriverConversationState = "performing_unload"
	StateCollect           DriverConversationState = "performing_collect"
	StateDropoff           DriverConversationState = "performing_dropoff"
	StateCleaning          DriverConversationState = "performing_cleaning"
	StateOnTheRoad         DriverConversationState = "on_the_road"
	StateWaitingPhoto      DriverConversationState = "waiting_photo"
	StateWaitingAttachment DriverConversationState = "waiting_attach"
	StateEndingDay         DriverConversationState = "waiting_km"
	StateTracking          DriverConversationState = "tracking"
)

var (
	ErrNoDriverSession = errors.New("driver's session is absent, impossible to complete the function")
)

type DriverSession struct {
	ID                     int           `json:"id"`
	DriverID               string        `json:"driver_id"`
	Date                   time.Time     `json:"date"`
	Started                time.Time     `json:"started"`
	Paused                 sql.NullTime  `json:"paused,omitempty"`
	Worktime               Duration      `json:"worktime"`
	Drivetime              Duration      `json:"drivetime"`
	Pausetime              Duration      `json:"pausetime"`
	KilometrageAccumulated int           `json:"kilometrage_accumulated"`
	StartingKilometrage    sql.NullInt64 `json:"starting_kilometrage,omitempty"`
	EndKilometrage         sql.NullInt64 `json:"end_kilometrage,omitempty"`
}

type Driver struct {
	Id              uuid.UUID               `db:"id"`
	UserId          uuid.UUID               `db:"user_id"`
	ChatId          int64                   `db:"chat_id"`
	CarId           string                  `db:"car_id"`
	CreatedAt       time.Time               `db:"created_at"`
	UpdatedAt       time.Time               `db:"updated_at"`
	State           DriverConversationState `db:"state"`
	PerformedTaskId int                     `db:"performing_task_id"`

	User    *User
	Tasks   *parser.Shipment // should not be empty if performing a shipment
	Session *DriverSession
}

func (d *Driver) StoreDriver(db DBExecutor, bot *tgbotapi.BotAPI) error {
	id, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("err creating a new uuid for a driver: %w", err)
	}
	d.Id = id

	stmt, err := db.Prepare(`
		INSERT INTO drivers (id, user_id, chat_id) 
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("err preparing statement for insert driver: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(id.String(), d.UserId.String(), d.User.ChatId)
	if err != nil {
		return fmt.Errorf("err executing prep insert driver stmt: %w", err)
	}

	updateStmt, err := db.Prepare(`
		UPDATE users 
		SET driver_id = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?
	`)
	if err != nil {
		return fmt.Errorf("err preparing statement for update user driver_id: %w", err)
	}
	defer updateStmt.Close()

	_, err = updateStmt.Exec(id.String(), d.UserId.String())
	if err != nil {
		return fmt.Errorf("err executing update user driver_id stmt: %w", err)
	}

	d.Id = id
	d.User.DriverId = id

	err = d.User.SendRequestToSuperAdmins(db, bot)
	if err != nil {
		return fmt.Errorf("Err sending request to accept user to superadmins: %v\n", err)
	}

	return nil
}

func (d *Driver) SetPerformingTask(globalStorage *sql.DB) error {
	query := `
		UPDATE drivers
		SET performing_task_id = ?
		WHERE id = ?
	`

	if d.Id == uuid.Nil {
		return fmt.Errorf("Err getting id: %v\n", d.Id)
	}

	_, err := globalStorage.Exec(query, d.PerformedTaskId, d.Id)
	if err != nil {
		return fmt.Errorf("err setting drivers task: %v\n", err)
	}

	return nil

}

func (d *Driver) DeletePerformingTask(globalStorage *sql.DB) error {
	query := `
		UPDATE drivers 
		SET performing_task_id = NULL 
		WHERE id = ?
	`

	if d.Id == uuid.Nil {
		return fmt.Errorf("Err getting id: %v\n", d.Id)
	}

	_, err := globalStorage.Exec(query, d.Id)
	if err != nil {
		return fmt.Errorf("err deleting drivers task: %v\n", err)
	}

	return nil

}

func (d *Driver) ChangeDriverStatus(globalStorage *sql.DB) error {
	query := `
		UPDATE drivers
		SET state = ?
		WHERE id = ?
	`
	if d.Id == uuid.Nil {
		return fmt.Errorf("Err getting id: %v\n", d.Id)
	}

	_, err := globalStorage.Exec(query, d.State, d.Id)
	if err != nil {
		return fmt.Errorf("err changing drivers status: %v\n", err)
	}

	return nil
}

func GetDriverById(db DBExecutor, driverId uuid.UUID) (*Driver, error) {
	query := `
		SELECT 
			d.id, d.user_id, d.car_id, d.created_at, d.updated_at, d.chat_id, d.state,
			u.id, u.chat_id, u.name, u.driver_id, u.manager_id, u.created_at, u.updated_at, u.is_super_admin, u.tg_tag
		FROM drivers d
		JOIN users u ON d.user_id = u.id
		WHERE d.id = ?
	`
	var driver Driver
	var driverIdStr, userIdStr string
	var carIdStr sql.NullString
	var userDriverIdStr, userManagerIdStr sql.NullString

	err := db.QueryRow(query, driverId.String()).Scan(
		&driverIdStr, &userIdStr, &carIdStr, &driver.CreatedAt, &driver.UpdatedAt, &driver.ChatId, &driver.State,
		&driver.User.Id, &driver.User.ChatId, &driver.User.Name,
		&userDriverIdStr, &userManagerIdStr,
		&driver.User.CreatedAt, &driver.User.UpdatedAt, &driver.User.IsSuperAdmin, &driver.User.TgTag,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no driver found for id %s", driverId.String())
		}
		return nil, fmt.Errorf("error querying driver by id: %w", err)
	}

	driver.Id, err = uuid.FromString(driverIdStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing driver id: %w", err)
	}
	driver.UserId, err = uuid.FromString(userIdStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing user id: %w", err)
	}

	if carIdStr.Valid {
		driver.CarId = carIdStr.String
	}

	if userDriverIdStr.Valid {
		driverId, err := uuid.FromString(userDriverIdStr.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing user driver_id: %w", err)
		}
		driver.User.DriverId = driverId
	}
	if userManagerIdStr.Valid {
		managerId, err := uuid.FromString(userManagerIdStr.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing user manager_id: %w", err)
		}
		driver.User.ManagerId = managerId
	}
	return &driver, nil
}

func GetDriverByChatId(db DBExecutor, chatId int64) (*Driver, error) {
	query := `
		SELECT 
			d.id, d.user_id, d.car_id, d.created_at, d.updated_at, d.chat_id, d.state, d.performing_task_id,
			u.id, u.chat_id, u.name, u.driver_id, u.manager_id, u.created_at, u.updated_at, u.is_super_admin, u.tg_tag
		FROM drivers d
		JOIN users u ON d.chat_id = u.chat_id
		WHERE d.chat_id = ?
	`
	driver := new(Driver)
	driver.User = new(User)
	var driverIdStr, userIdStr string
	var carIdStr sql.NullString
	var userDriverIdStr, userManagerIdStr sql.NullString
	var performedTaskId sql.Null[int]

	err := db.QueryRow(query, chatId).Scan(
		&driverIdStr, &userIdStr, &carIdStr, &driver.CreatedAt, &driver.UpdatedAt, &driver.ChatId, &driver.State, &performedTaskId,
		&driver.User.Id, &driver.User.ChatId, &driver.User.Name,
		&userDriverIdStr, &userManagerIdStr,
		&driver.User.CreatedAt, &driver.User.UpdatedAt, &driver.User.IsSuperAdmin, &driver.User.TgTag,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no driver found for chat_id %d", chatId)
		}
		return nil, fmt.Errorf("error querying driver by chat_id: %w", err)
	}

	driver.Id, err = uuid.FromString(driverIdStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing driver id: %w", err)
	}
	driver.UserId, err = uuid.FromString(userIdStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing user id: %w", err)
	}

	if performedTaskId.Valid {
		driver.PerformedTaskId = performedTaskId.V
	}

	if carIdStr.Valid {
		driver.CarId = carIdStr.String
	}

	if userDriverIdStr.Valid {
		driverId, err := uuid.FromString(userDriverIdStr.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing user driver_id: %w", err)
		}
		driver.User.DriverId = driverId
	}
	if userManagerIdStr.Valid {
		managerId, err := uuid.FromString(userManagerIdStr.String)
		if err != nil {
			return nil, fmt.Errorf("error parsing user manager_id: %w", err)
		}
		driver.User.ManagerId = managerId
	}
	return driver, nil
}

func GetAllDrivers(db DBExecutor) ([]*Driver, error) {
	query := `
		SELECT 
			d.id, d.user_id, d.car_id, d.created_at, d.updated_at, d.chat_id, d.state, d.performing_task_id,
			u.id, u.chat_id, u.name, u.driver_id, u.manager_id, u.created_at, u.updated_at
		FROM drivers d
		JOIN users u ON d.user_id = u.id
		ORDER BY u.name
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying all drivers: %v", err)
	}
	defer rows.Close()

	var drivers []*Driver
	for rows.Next() {
		driver := new(Driver)
		driver.User = new(User)
		var driverIdStr, userIdStr string
		var carIdStr sql.NullString
		var userDriverIdStr, userManagerIdStr sql.NullString

		var performedTaskId sql.Null[int]

		err := rows.Scan(
			&driverIdStr, &userIdStr, &carIdStr, &driver.CreatedAt, &driver.UpdatedAt, &driver.ChatId, &driver.State, &performedTaskId,
			&driver.User.Id, &driver.User.ChatId, &driver.User.Name,
			&userDriverIdStr, &userManagerIdStr,
			&driver.User.CreatedAt, &driver.User.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning driver row: %v", err)
		}

		driver.Id, err = uuid.FromString(driverIdStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing driver id: %v", err)
		}
		driver.UserId, err = uuid.FromString(userIdStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing user id: %v", err)
		}

		if performedTaskId.Valid {
			driver.PerformedTaskId = performedTaskId.V
		}

		if carIdStr.Valid {
			driver.CarId = carIdStr.String
		}

		if userDriverIdStr.Valid {
			driverId, err := uuid.FromString(userDriverIdStr.String)
			if err != nil {
				return nil, fmt.Errorf("error parsing user driver_id: %v", err)
			}
			driver.User.DriverId = driverId
		}
		if userManagerIdStr.Valid {
			managerId, err := uuid.FromString(userManagerIdStr.String)
			if err != nil {
				return nil, fmt.Errorf("error parsing user manager_id: %v", err)
			}
			driver.User.ManagerId = managerId
		}

		drivers = append(drivers, driver)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating driver rows: %v", err)
	}
	return drivers, nil
}

func (d *Driver) UpdateCarId(db DBExecutor, newCarId string) error {
	tx, ok := db.(*sql.Tx)
	var txErr error

	stmtDriver, err := db.Prepare(`
		UPDATE drivers 
		SET car_id = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?
	`)
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("err preparing statement for update driver car_id: %v (txErr: %v)\n", err, txErr)
	}
	defer stmtDriver.Close()

	result, err := stmtDriver.Exec(newCarId, d.Id.String())
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("err executing update driver car_id stmt: %v (txErr: %v)\n", err, txErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("err getting rows affected: %v (txErr: %v)\n", err, txErr)
	}

	if rowsAffected == 0 {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("no driver found with id %s (txErr: %v)\n", d.Id, txErr)
	}

	d.CarId = newCarId

	stmtCar, err := db.Prepare(`
		UPDATE cars
		SET current_driver = ?
		WHERE id = ?
	`)
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("err preparing statement for update driver for a car in cars table: %v (txErr: %v)\n", err, txErr)
	}
	defer stmtCar.Close()

	result, err = stmtCar.Exec(d.Id.String(), d.CarId)
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("err executing update driver car_id stmt: %v (txErr: %v)\n", err, txErr)
	}

	if tx, ok := db.(*sql.Tx); ok {
		return tx.Commit()
	}
	return nil
}

func (d *Driver) UnpauseSession(db DBExecutor) (*DriverSession, error) {
	if d.CarId == "" {
		if d.ChatId == 0 {
			return nil, ErrNoDriverSession
		}
		driver, err := GetDriverByChatId(db, d.ChatId)
		if err != nil {
			return nil, fmt.Errorf("err getting driver by chat id: %v", err)
		}
		err = deepcopy.Copy(d, driver)
		if err != nil {
			return nil, fmt.Errorf("err making a deepcopy: %v", err)
		}
	}

	query := `
		INSERT INTO drivers_sessions
		(driver_id, starting_kilometrage)  
		VALUES (?, ?)
	`

	car, err := GetCarById(db, d.CarId)
	if err != nil {
		return nil, fmt.Errorf("err getting a car by id: %v", err)
	}

	result, err := db.Exec(query, d.Id, car.Kilometrage)
	if err != nil {
		return nil, fmt.Errorf("err inserting new session: %v", err)
	}

	sessionID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("err getting last insert id: %v", err)
	}

	session, err := GetSessionById(db, int(sessionID))
	if err != nil {
		return nil, fmt.Errorf("err getting created session: %v", err)
	}

	return session, nil
}

func GetSessionById(db DBExecutor, id int) (*DriverSession, error) {
	session := &DriverSession{}
	query := `
		SELECT id, driver_id, date, started, paused, worktime, drivetime, pausetime,
		       kilometrage_accumulated, starting_kilometrage, end_kilometrage
		FROM drivers_sessions
		WHERE id = ?
	`

	err := db.QueryRow(query, id).Scan(
		&session.ID,
		&session.DriverID,
		&session.Date,
		&session.Started,
		&session.Paused,
		&session.Worktime,
		&session.Drivetime,
		&session.Pausetime,
		&session.KilometrageAccumulated,
		&session.StartingKilometrage,
		&session.EndKilometrage,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("session %d not found", id)
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	return session, nil
}

func (d *Driver) GetLastActiveSession(db DBExecutor) (*DriverSession, error) {
	session := new(DriverSession)
	query := `
		SELECT id, driver_id, date, started, paused, worktime, drivetime, pausetime,
		       kilometrage_accumulated, starting_kilometrage, end_kilometrage
		FROM drivers_sessions
		WHERE id = (
			SELECT MAX(id) 
			FROM drivers_sessions 
			WHERE driver_id = ? AND paused IS NULL
		)
	`

	err := db.QueryRow(query, d.Id).Scan(
		&session.ID,
		&session.DriverID,
		&session.Date,
		&session.Started,
		&session.Paused,
		&session.Worktime,
		&session.Drivetime,
		&session.Pausetime,
		&session.KilometrageAccumulated,
		&session.StartingKilometrage,
		&session.EndKilometrage,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return new(DriverSession), nil
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	d.Session = session

	return session, nil
}

func UpdatePausedTime(db DBExecutor) error {
	var (
		pausedStr string
		id        int
	)

	err := db.QueryRow("SELECT id, paused FROM drivers_sessions ORDER BY id DESC LIMIT 1 OFFSET 1").Scan(&id, &pausedStr)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("err getting last paused time from the db: %v\n", err)
		}
		log.Println("sessions table is empty, or has only one entry, making checking for previous pause redundant")
		return nil
	}

	paused, err := time.Parse("2006-01-02 15:04:05", pausedStr)
	if err != nil {
		return fmt.Errorf("failed to parse datetime '%s': %v\n", pausedStr, err)
	}

	minutes := time.Since(paused)
	pausedTime := minutes.Round(time.Minute).String()

	updateQuery := `
		UPDATE drivers_sessions
		SET pausetime = ?
		WHERE id = ?
	`

	_, err = db.Exec(updateQuery, pausedTime, id)
	return err
}

func (d *Driver) PauseSession(db DBExecutor) (*DriverSession, error) {
	var sessionId sql.NullInt64
	session := d.Session

	if d.CarId == "" {
		if d.ChatId == 0 {
			return nil, ErrNoDriverSession
		}
		driver, err := GetDriverByChatId(db, d.ChatId)
		if err != nil {
			return nil, fmt.Errorf("err getting driver by chat id: %v", err)
		}
		err = deepcopy.Copy(d, driver)
		if err != nil {
			return nil, fmt.Errorf("err making a deepcopy: %v", err)
		}
	}

	sessionToPauseQuery := `
		SELECT MAX(id) 
		FROM drivers_sessions 
		WHERE driver_id = ? AND paused IS NULL
	`

	err := db.QueryRow(sessionToPauseQuery, d.Id).Scan(&sessionId)
	if err != nil {
		return nil, fmt.Errorf("err querying session id: %v", err)
	}

	if !sessionId.Valid {
		return nil, ErrNoDriverSession
	}

	if session == nil {
		log.Println("WARN: session on driver's struct is empty, there most likely would be no worktime, pausetime of drivertime")
		session, err = GetSessionById(db, int(sessionId.Int64))
		if err != nil {
			return nil, fmt.Errorf("err getting a session by id to stop it. Does it exist in the db?: %v\n", err)
		}
	}

	query := `
		UPDATE drivers_sessions
		SET paused = CURRENT_TIMESTAMP, 
			worktime = ?, 
			drivetime = ?, 
			pausetime = ?,
			kilometrage_accumulated = ?,
			end_kilometrage = ?
		WHERE id = ? 
	`

	// before that, km should be updated on the cars table
	car, err := GetCarById(db, d.CarId)
	if err != nil {
		return nil, fmt.Errorf("err getting a car by id: %v", err)
	}

	res, err := db.Exec(query,
		session.Worktime.String(),
		session.Drivetime.String(),
		session.Pausetime.String(),
		session.KilometrageAccumulated,
		car.Kilometrage,
		sessionId,
	)
	if err != nil {
		return nil, fmt.Errorf("err inserting new session: %v", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("pause session: no rows updated (session id=%d)", sessionId)
	}

	return session, nil
}

func ParseDuration(time string) Duration {
	hours, minutes, f := strings.Cut(time, ":")
	if !f {
		hours, minutes, f = strings.Cut(time, ".")
		if !f {
			return Duration{}
		}
	}

	return NewDurationFromString(hours, minutes)
}
