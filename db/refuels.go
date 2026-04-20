package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

type FuelCard struct {
	Id   int    `db:"id"`
	Name string `db:"name"`
}

type TankRefuel struct {
	Id                 int
	ShipmentId         *int64
	FuelCardId         int
	FuelCard           string
	CurrentKilometrage int64
	Address            string
	Diesel             float64
	AdBlu              float64
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Driver             *Driver
	CarId              string
}

func (t *TankRefuel) StoreTankRefuel(db DBExecutor) error {
	stmt, err := db.Prepare(`
		INSERT INTO tank_refuels (shipment_id, fuel_card_id, driver_id, created_at, car_id)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?)
	`)
	if err != nil {
		return fmt.Errorf("ERR: preparing statement for insert tank_refuel: %w", err)
	}
	defer stmt.Close()

	var shipmentId sql.NullInt64
	if t.ShipmentId != nil {
		shipmentId = sql.NullInt64{Int64: int64(*t.ShipmentId), Valid: true}
	}

	result, err := stmt.Exec(
		shipmentId,
		t.FuelCardId,
		t.Driver.Id.String(),
		t.CarId,
	)
	if err != nil {
		return fmt.Errorf("ERR: executing insert tank_refuel stmt: %w", err)
	}

	newId, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("ERR: getting last insert id for tank_refuel: %w", err)
	}
	t.Id = int(newId)
	return nil
}

func (t *TankRefuel) UpdateAddress(db DBExecutor, address string) error {
	_, err := db.Exec(`
		UPDATE tank_refuels SET address = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, address, t.Id)
	if err != nil {
		return fmt.Errorf("ERR: updating address for tank_refuel %d: %w", t.Id, err)
	}
	t.Address = address
	return nil
}

func (t *TankRefuel) UpdateKilometrage(db DBExecutor, kilometrage int64) error {
	_, err := db.Exec(`
		UPDATE tank_refuels SET current_kilometrage = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, kilometrage, t.Id)
	if err != nil {
		return fmt.Errorf("ERR: updating kilometrage for tank_refuel %d: %w", t.Id, err)
	}
	t.CurrentKilometrage = kilometrage
	return nil
}

func (t *TankRefuel) UpdateDiesel(db DBExecutor, diesel float64) error {
	_, err := db.Exec(`
		UPDATE tank_refuels SET diesel = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, diesel, t.Id)
	if err != nil {
		return fmt.Errorf("ERR: updating diesel for tank_refuel %d: %w", t.Id, err)
	}
	t.Diesel = diesel
	return nil
}

func (t *TankRefuel) UpdateAdBlu(db DBExecutor, adBlu float64) error {
	_, err := db.Exec(`
		UPDATE tank_refuels SET adblu = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, adBlu, t.Id)
	if err != nil {
		return fmt.Errorf("ERR: updating adblu for tank_refuel %d: %w", t.Id, err)
	}
	t.AdBlu = adBlu
	return nil
}

func GetAllTankRefuels(db DBExecutor) ([]TankRefuel, error) {
	rows, err := db.Query(`
		SELECT
			tr.id,
			tr.shipment_id,
			tr.fuel_card_id,
			fc.name,
			tr.current_kilometrage,
			tr.address,
			tr.diesel,
			tr.adblu,
			tr.created_at,
			tr.car_id,
			COALESCE(tr.updated_at, tr.created_at),
			d.id, d.user_id,  d.created_at, d.updated_at, d.chat_id, d.state, d.performing_task_id
		FROM tank_refuels tr
		JOIN fuel_cards fc ON fc.id = tr.fuel_card_id
		JOIN drivers    d  ON d.id  = tr.driver_id
		ORDER BY tr.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("ERR: querying all tank_refuels: %w", err)
	}
	defer rows.Close()

	return scanRefuelRows(rows)
}

func GetTankRefuelById(db DBExecutor, id int) (*TankRefuel, error) {
	row := db.QueryRow(`
		SELECT
			tr.id,
			tr.shipment_id,
			tr.fuel_card_id,
			fc.name,
			tr.current_kilometrage,
			tr.address,
			tr.diesel,
			tr.adblu,
			tr.created_at,
			tr.car_id,
			COALESCE(tr.updated_at, tr.created_at),
			d.id, d.user_id, d.created_at, d.updated_at, d.chat_id, d.state, d.performing_task_id
		FROM tank_refuels tr
		JOIN fuel_cards fc ON fc.id = tr.fuel_card_id
		JOIN drivers    d  ON d.id  = tr.driver_id
		WHERE tr.id = ?
	`, id)

	var (
		r            TankRefuel
		d            Driver
		shipmentId   sql.NullInt64
		carIdStr     sql.NullString
		taskId       sql.NullInt64
		driverIdStr  string
		userIdStr    string
		updatedAtStr sql.NullString
		address      sql.NullString
	)

	err := row.Scan(
		&r.Id,
		&shipmentId,
		&r.FuelCardId,
		&r.FuelCard,
		&r.CurrentKilometrage,
		&address,
		&r.Diesel,
		&r.AdBlu,
		&r.CreatedAt,
		&carIdStr,
		&updatedAtStr,
		&driverIdStr,
		&userIdStr,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.ChatId,
		&d.State,
		&taskId,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("ERR: tank refuel with id %d not found: %w", id, sql.ErrNoRows)
		}
		return nil, fmt.Errorf("ERR: scanning tank_refuel row for id %d: %w", id, err)
	}

	if updatedAtStr.Valid {
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05.999999999-07:00",
		}
		for _, layout := range formats {
			if t, err := time.Parse(layout, updatedAtStr.String); err == nil {
				r.UpdatedAt = t
				break
			}
		}
	}

	if shipmentId.Valid {
		v := shipmentId.Int64
		r.ShipmentId = &v
	}
	if address.Valid {
		r.Address = address.String
	}

	d.Id, err = uuid.FromString(driverIdStr)
	if err != nil {
		return nil, fmt.Errorf("ERR: parsing driver id in tank_refuel scan: %w", err)
	}
	d.UserId, err = uuid.FromString(userIdStr)
	if err != nil {
		return nil, fmt.Errorf("ERR: parsing user id in tank_refuel scan: %w", err)
	}

	if carIdStr.Valid {
		r.CarId = carIdStr.String
	}
	if taskId.Valid {
		d.PerformedTaskId = int(taskId.Int64)
	}

	r.Driver = &d
	return &r, nil
}

func GetTankRefuelsByDriver(db DBExecutor, driverID uuid.UUID) ([]TankRefuel, error) {
	rows, err := db.Query(`
		SELECT
			tr.id,
			tr.shipment_id,
			tr.fuel_card_id,
			fc.name,
			tr.current_kilometrage,
			tr.address,
			tr.diesel,
			tr.adblu,
			tr.created_at,
			tr.car_id,
			COALESCE(tr.updated_at, tr.created_at),
			d.id, d.user_id, d.created_at, d.updated_at, d.chat_id, d.state, d.performing_task_id
		FROM tank_refuels tr
		JOIN fuel_cards fc ON fc.id = tr.fuel_card_id
		JOIN drivers    d  ON d.id  = tr.driver_id
		WHERE tr.driver_id = ?
		ORDER BY tr.created_at DESC
	`, driverID.String())
	if err != nil {
		return nil, fmt.Errorf("ERR: querying tank_refuels for driver %s: %w", driverID, err)
	}
	defer rows.Close()

	return scanRefuelRows(rows)
}

func GetAllFuelCards(db DBExecutor) ([]FuelCard, error) {
	rows, err := db.Query(`SELECT id, name FROM fuel_cards ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("ERR: querying all fuel_cards: %w", err)
	}
	defer rows.Close()

	var cards []FuelCard
	for rows.Next() {
		var fc FuelCard
		if err := rows.Scan(&fc.Id, &fc.Name); err != nil {
			return nil, fmt.Errorf("ERR: scanning fuel_card row: %w", err)
		}
		cards = append(cards, fc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ERR: iterating fuel_card rows: %w", err)
	}

	return cards, nil
}

func GetFuelCardById(db DBExecutor, id int) (*FuelCard, error) {
	fc := new(FuelCard)
	row := db.QueryRow(`SELECT id, name FROM fuel_cards WHERE id = ?`, id)
	if err := row.Scan(&fc.Id, &fc.Name); err != nil {
		return nil, fmt.Errorf("ERR: scanning fuel_card row: %w", err)
	}
	return fc, nil
}

func scanRefuelRows(rows *sql.Rows) ([]TankRefuel, error) {
	var refuels []TankRefuel

	for rows.Next() {
		var (
			r            TankRefuel
			d            Driver
			shipmentId   sql.NullInt64
			carIdStr     sql.NullString
			taskId       sql.NullInt64
			driverIdStr  string
			userIdStr    string
			updatedAtStr sql.NullString // fix: SQLite returns this as string
		)

		err := rows.Scan(
			&r.Id,
			&shipmentId,
			&r.FuelCardId,
			&r.FuelCard,
			&r.CurrentKilometrage,
			&r.Address,
			&r.Diesel,
			&r.AdBlu,
			&r.CreatedAt,
			&carIdStr,
			&updatedAtStr, // was &r.UpdatedAt
			&driverIdStr,
			&userIdStr,
			&d.CreatedAt,
			&d.UpdatedAt,
			&d.ChatId,
			&d.State,
			&taskId,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, fmt.Errorf("ERR: scanning tank_refuel row: %w", err)
		}

		// parse updated_at string from SQLite
		if updatedAtStr.Valid {
			// SQLite datetime formats
			formats := []string{
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05Z",
				"2006-01-02 15:04:05.999999999-07:00",
			}
			for _, layout := range formats {
				if t, err := time.Parse(layout, updatedAtStr.String); err == nil {
					r.UpdatedAt = t
					break
				}
			}
		}

		if shipmentId.Valid {
			v := shipmentId.Int64
			r.ShipmentId = &v
		}

		d.Id, err = uuid.FromString(driverIdStr)
		if err != nil {
			return nil, fmt.Errorf("ERR: parsing driver id in tank_refuel scan: %w", err)
		}
		d.UserId, err = uuid.FromString(userIdStr)
		if err != nil {
			return nil, fmt.Errorf("ERR: parsing user id in tank_refuel scan: %w", err)
		}
		if carIdStr.Valid {
			r.CarId = carIdStr.String
		}
		if taskId.Valid {
			d.PerformedTaskId = int(taskId.Int64)
		}

		r.Driver = &d
		refuels = append(refuels, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ERR: iterating tank_refuel rows: %w", err)
	}

	return refuels, nil
}
