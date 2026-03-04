package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

type DriverGroup struct {
	Id              int
	GroupChatId     int64
	TankTopicId     int
	LoadingTopicId  int
	DocumentTopicId int
	PhotoTopicId    int
	CurrentCar      *Car
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

var (
	ErrGroupExists = errors.New("this driver's group already exists")
	ErrIdFilled    = errors.New("this topic already set in this group. gotta do it manually man")
)

func (g *DriverGroup) GetDriverGroupByCar(globalStorage *sql.DB) error {
	row := globalStorage.QueryRow(`
		SELECT 
			dg.id, dg.group_chat_id, dg.tank_topic_id, dg.loading_topic_id, dg.document_topic_id, dg.photo_topic_id, dg.created_at, dg.updated_at,
			c.id, c.current_driver, c.current_kilometrage
		FROM driver_groups dg
		LEFT JOIN cars c ON c.id = dg.current_car_id
		WHERE c.id = ?`, g.CurrentCar.Id)

	var (
		tankTopicId     sql.NullInt64
		loadingTopicId  sql.NullInt64
		documentTopicId sql.NullInt64
		photoTopicId    sql.NullInt64

		carId          sql.NullString
		carDriverId    sql.NullString
		carKilometrage sql.NullInt64
	)

	err := row.Scan(
		&g.Id,
		&g.GroupChatId,
		&tankTopicId,
		&loadingTopicId,
		&documentTopicId,
		&photoTopicId,
		&g.CreatedAt,
		&g.UpdatedAt,
		&carId,
		&carDriverId,
		&carKilometrage,
	)
	if err != nil {
		return fmt.Errorf("GetDriverGroupByCar: %w", err)
	}

	if tankTopicId.Valid {
		g.TankTopicId = int(tankTopicId.Int64)
	}
	if loadingTopicId.Valid {
		g.LoadingTopicId = int(loadingTopicId.Int64)
	}
	if documentTopicId.Valid {
		g.DocumentTopicId = int(documentTopicId.Int64)
	}
	if photoTopicId.Valid {
		g.PhotoTopicId = int(photoTopicId.Int64)
	}

	if carId.Valid {
		if g.CurrentCar == nil {
			g.CurrentCar = &Car{}
		}
		g.CurrentCar.Id = carId.String
		if carDriverId.Valid {
			g.CurrentCar.CurrentDriverId, err = uuid.FromString(carDriverId.String)
			if err != nil {
				return fmt.Errorf("GetDriverGroupByCar parsing car driver id: %w", err)
			}
		}
		if carKilometrage.Valid {
			g.CurrentCar.Kilometrage = carKilometrage.Int64
		}
	}

	return nil
}

func (g *DriverGroup) GetDriverGroup(globalStorage *sql.DB) error {
	row := globalStorage.QueryRow(`
		SELECT 
			dg.id, dg.group_chat_id, dg.tank_topic_id, dg.loading_topic_id, dg.document_topic_id, dg.photo_topic_id, dg.created_at, dg.updated_at,
			c.id, c.current_driver, c.current_kilometrage
		FROM driver_groups dg
		LEFT JOIN cars c ON c.id = dg.current_car_id
		WHERE dg.group_chat_id = ?`, g.GroupChatId)

	var (
		tankTopicId     sql.NullInt64
		loadingTopicId  sql.NullInt64
		documentTopicId sql.NullInt64
		photoTopicId    sql.NullInt64
		carId           sql.NullString
		carDriverId     sql.NullString
		carKilometrage  sql.NullInt64
	)

	err := row.Scan(
		&g.Id,
		&g.GroupChatId,
		&tankTopicId,
		&loadingTopicId,
		&documentTopicId,
		&photoTopicId,
		&g.CreatedAt,
		&g.UpdatedAt,
		&carId,
		&carDriverId,
		&carKilometrage,
	)
	if err != nil {
		return fmt.Errorf("GetDriverGroup: %w", err)
	}

	if tankTopicId.Valid {
		g.TankTopicId = int(tankTopicId.Int64)
	}
	if loadingTopicId.Valid {
		g.LoadingTopicId = int(loadingTopicId.Int64)
	}
	if documentTopicId.Valid {
		g.DocumentTopicId = int(documentTopicId.Int64)
	}
	if photoTopicId.Valid {
		g.PhotoTopicId = int(photoTopicId.Int64)
	}

	if carId.Valid {
		car := &Car{}
		car.Id = carId.String
		if carDriverId.Valid {
			car.CurrentDriverId, err = uuid.FromString(carDriverId.String)
			if err != nil {
				return fmt.Errorf("GetDriverGroup parsing car driver id: %w", err)
			}
		}
		if carKilometrage.Valid {
			car.Kilometrage = carKilometrage.Int64
		}
		g.CurrentCar = car
	}

	return nil
}

func (g *DriverGroup) CreateDriverGroup(globalStorage *sql.DB) error {
	//u := &User{Id: superAdminId}
	/*err := u.FindSuperAdmin(globalStorage)
	if err != nil {
		if errors.Is(err, ErrNotSuperAdmin) {
			return ErrNotSuperAdmin
		}
		return fmt.Errorf("ERR: checking if you are SA or not. Probably not actually...: %v\n", err)
	}*/

	stmt, err := globalStorage.Prepare("INSERT INTO driver_groups(group_chat_id, current_car_id) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("ERR: prepping stmt for adding a group to the db: %v\n", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(g.GroupChatId, g.CurrentCar.Id)
	if err != nil {
		// might not be possible, check later
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrGroupExists
		}
		return fmt.Errorf("ERR: executing stmt to add group to the db: %v\n", err)
	}

	return err
}

func (g *DriverGroup) fillTopicId(globalStorage *sql.DB, column string, topicId int, field *int) {
	row := globalStorage.QueryRow(fmt.Sprintf("SELECT %s FROM driver_groups WHERE group_chat_id = ?", column), g.GroupChatId)

	var existing sql.NullInt64
	if err := row.Scan(&existing); err != nil {
		return
	}

	if existing.Valid {
		*field = int(existing.Int64)
		return
	}

	_, err := globalStorage.Exec(fmt.Sprintf("UPDATE driver_groups SET %s = ? WHERE group_chat_id = ?", column), topicId, g.GroupChatId)
	if err != nil {
		return
	}
	*field = topicId
}
func (g *DriverGroup) FillTankTopicId(globalStorage *sql.DB, topicId int) {
	g.fillTopicId(globalStorage, "tank_topic_id", topicId, &g.TankTopicId)
}

func (g *DriverGroup) FillLoadingTopicId(globalStorage *sql.DB, topicId int) {
	g.fillTopicId(globalStorage, "loading_topic_id", topicId, &g.LoadingTopicId)
}

func (g *DriverGroup) FillDocumentTopicId(globalStorage *sql.DB, topicId int) {
	g.fillTopicId(globalStorage, "document_topic_id", topicId, &g.DocumentTopicId)
}

func (g *DriverGroup) FillPhotoTopicId(globalStorage *sql.DB, topicId int) {
	g.fillTopicId(globalStorage, "photo_topic_id", topicId, &g.PhotoTopicId)
}
