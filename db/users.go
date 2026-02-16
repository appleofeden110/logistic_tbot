package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

type Role string

const (
	RoleSuperAdmin Role = "СуперАдмін"
	RoleManager    Role = "Менеджер"
	RoleDriver     Role = "Водій"
	NoRole         Role = "no_role"
)

type User struct {
	Id           uuid.UUID `db:"id"`
	ChatId       int64     `db:"chat_id"`
	Name         string    `db:"name" form:"Введіть ваше імʼя"`
	TgTag        string    `db:"tg_tag"`
	DriverId     uuid.UUID `db:"driver_id"`
	ManagerId    uuid.UUID `db:"manager_id"`
	IsSuperAdmin bool      `db:"is_super_admin"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func GetAllUsers(globalStorage *sql.DB) ([]*User, error) {
	query := `
		SELECT
			id,
			chat_id,
			name,
			driver_id,
			manager_id,
			created_at,
			updated_at,
			is_super_admin,
			tg_tag
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := globalStorage.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ERR: querying all users: %v\n", err)
	}
	defer rows.Close()

	var users []*User

	for rows.Next() {
		var user User
		var driverIdNull, managerIdNull, tgTagNull sql.NullString

		err := rows.Scan(
			&user.Id,
			&user.ChatId,
			&user.Name,
			&driverIdNull,
			&managerIdNull,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.IsSuperAdmin,
			&tgTagNull,
		)
		if err != nil {
			return nil, fmt.Errorf("ERR: scanning user row: %v\n", err)
		}

		// Handle driver_id
		if driverIdNull.Valid {
			driverId, err := uuid.FromString(driverIdNull.String)
			if err != nil {
				return nil, fmt.Errorf("ERR: parsing driver_id: %v\n", err)
			}
			user.DriverId = driverId
		} else {
			user.DriverId = uuid.Nil
		}

		// Handle manager_id
		if managerIdNull.Valid {
			managerId, err := uuid.FromString(managerIdNull.String)
			if err != nil {
				return nil, fmt.Errorf("ERR: parsing manager_id: %v\n", err)
			}
			user.ManagerId = managerId
		} else {
			user.ManagerId = uuid.Nil
		}

		// Handle tg_tag
		if tgTagNull.Valid {
			user.TgTag = tgTagNull.String
		} else {
			user.TgTag = ""
		}

		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ERR: iterating user rows: %v\n", err)
	}

	return users, nil
}
func (u *User) GetUserById(globalStorage *sql.DB) error {
	row := globalStorage.QueryRow("SELECT id, chat_id, name, driver_id, manager_id, created_at, updated_at, is_super_admin, tg_tag FROM users WHERE id = ?", u.Id)
	var driverIdNull, managerIdNull, tgTagNull sql.NullString
	err := row.Scan(
		&u.Id,
		&u.ChatId,
		&u.Name,
		&driverIdNull,
		&managerIdNull,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.IsSuperAdmin,
		&tgTagNull,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return fmt.Errorf("error scanning user: %w", err)
	}
	if driverIdNull.Valid {
		driverId, err := uuid.FromString(driverIdNull.String)
		if err != nil {
			return fmt.Errorf("error parsing driver_id: %w", err)
		}
		u.DriverId = driverId
	} else {
		u.DriverId = uuid.Nil
	}
	if managerIdNull.Valid {
		managerId, err := uuid.FromString(managerIdNull.String)
		if err != nil {
			return fmt.Errorf("error parsing manager_id: %w", err)
		}
		u.ManagerId = managerId
	} else {
		u.ManagerId = uuid.Nil
	}
	if tgTagNull.Valid {
		u.TgTag = tgTagNull.String
	} else {
		u.TgTag = ""
	}
	return nil
}

func (u *User) GetUserByChatId(globalStorage *sql.DB) error {
	row := globalStorage.QueryRow("SELECT id, chat_id, name, driver_id, manager_id, created_at, updated_at, is_super_admin, tg_tag FROM users WHERE chat_id = ?", u.ChatId)

	var driverIdNull, managerIdNull, tgTagNull sql.NullString

	err := row.Scan(
		&u.Id,
		&u.ChatId,
		&u.Name,
		&driverIdNull,
		&managerIdNull,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.IsSuperAdmin,
		&tgTagNull,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return fmt.Errorf("error scanning user: %w", err)
	}

	if driverIdNull.Valid {
		driverId, err := uuid.FromString(driverIdNull.String)
		if err != nil {
			return fmt.Errorf("error parsing driver_id: %w", err)
		}
		u.DriverId = driverId
	} else {
		u.DriverId = uuid.Nil
	}

	if managerIdNull.Valid {
		managerId, err := uuid.FromString(managerIdNull.String)
		if err != nil {
			return fmt.Errorf("error parsing manager_id: %w", err)
		}
		u.ManagerId = managerId
	} else {
		u.ManagerId = uuid.Nil
	}

	if tgTagNull.Valid {
		u.TgTag = tgTagNull.String
	} else {
		u.TgTag = ""
	}

	return nil
}

// User storage
func (u *User) StoreUser(db DBExecutor) error {
	tx, ok := db.(*sql.Tx)
	var txErr error

	id, err := uuid.NewV4()
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("err creating a new uuid for a user: %v (txErr: %v)\n", err, txErr)
	}
	fmt.Println(u.Name)
	fmt.Println(u.ChatId)

	stmt, err := db.Prepare(`
		INSERT INTO users (id, chat_id, name, tg_tag) 
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("err preparing statement for insert user: %v (txErr: %v)\n", err, txErr)
	}
	defer stmt.Close()

	_, err = stmt.Exec(id.String(), u.ChatId, u.Name, u.TgTag)
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("err executing prep insert user stmt: %v (txErr: %v)\n", err, txErr)
	}

	u.Id = id
	return nil
}
