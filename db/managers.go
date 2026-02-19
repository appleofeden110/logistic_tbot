package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"logistictbot/config"
	"logistictbot/docs"
	"logistictbot/parser"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofrs/uuid"
)

var (
	formTextAcceptTask = "Чи готові ви прийняти завдання?"
)

type ManagerConversationState string

const (
	StateDormantManager ManagerConversationState = "dormant_mng"

	StateWritingToDriver ManagerConversationState = "sending_driver_message"
	StateReplyingDriver  ManagerConversationState = "replying_driver"

	StateWaitingDoc    ManagerConversationState = "waiting_doc"
	StateWaitingNotes  ManagerConversationState = "waiting_notes"
	StateWaitingDriver ManagerConversationState = "waiting_driver"
)

type PendingMessage struct {
	FromChatId      int64
	ToChatId        int64
	MessageType     string // "document", "text", etc.
	FromUser        *tgbotapi.User
	DocOriginalName string // if msgType is "document"
	DocMimetype     docs.Mimetype
	Caption         string
	FileId          string
}

type Manager struct {
	Id             uuid.UUID                `db:"id"`
	UserId         uuid.UUID                `db:"user_id"`
	ChatId         int64                    `db:"chat_id"`
	State          ManagerConversationState `db:"state"`
	PendingMessage *PendingMessage
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`

	//Accepted bool

	User *User
}

func SetAllManagersToDormant(db DBExecutor) error {
	query := `
		UPDATE managers 
		SET state = ?, updated_at = CURRENT_TIMESTAMP
		WHERE state != ?
	`

	result, err := db.Exec(query, StateDormantManager, StateDormantManager)
	if err != nil {
		return fmt.Errorf("ERR: setting all managers to dormant state: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ERR: getting rows affected: %v", err)
	}

	log.Printf("Set %d manager(s) to dormant state", rowsAffected)
	return nil
}

func (m *Manager) ChangeManagerStatus(globalStorage *sql.DB) error {
	query := `
		UPDATE managers 
		SET state = ?
		WHERE id = ?
	`
	if m.Id == uuid.Nil {
		return fmt.Errorf("ERR: getting id for changing manager's status: %v\n", m.Id)
	}

	_, err := globalStorage.Exec(query, m.State, m.Id)
	if err != nil {
		return fmt.Errorf("ERR: changing manager's status: %v\n", err)
	}

	return nil
}

func GetManagerById(db DBExecutor, id uuid.UUID) (*Manager, error) {
	query := `
		SELECT 
			m.id, m.user_id, m.created_at, m.updated_at, m.chat_id, m.state,
			u.id, u.chat_id, u.name, u.driver_id, u.manager_id, u.created_at, u.updated_at
		FROM users u
		JOIN managers m ON u.manager_id = m.id
		WHERE m.id = $1
	`
	manager := new(Manager)
	manager.User = new(User)
	var managerIdStr, userIdStr string
	var userDriverIdStr, userManagerIdStr sql.NullString
	err := db.QueryRow(query, id.String()).Scan(
		&managerIdStr, &userIdStr, &manager.CreatedAt, &manager.UpdatedAt, &manager.ChatId, &manager.State,
		&manager.User.Id, &manager.User.ChatId, &manager.User.Name,
		&userDriverIdStr, &userManagerIdStr,
		&manager.User.CreatedAt, &manager.User.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no manager found for id %s", id.String())
		}
		return nil, fmt.Errorf("ERR: querying manager by id: %v", err)
	}
	manager.Id, err = uuid.FromString(managerIdStr)
	if err != nil {
		return nil, fmt.Errorf("ERR: parsing manager id: %v", err)
	}
	manager.UserId, err = uuid.FromString(userIdStr)
	if err != nil {
		return nil, fmt.Errorf("ERR: parsing user id: %v", err)
	}
	if userDriverIdStr.Valid {
		driverId, err := uuid.FromString(userDriverIdStr.String)
		if err != nil {
			return nil, fmt.Errorf("ERR: parsing user driver_id: %v", err)
		}
		manager.User.DriverId = driverId
	}
	if userManagerIdStr.Valid {
		managerId, err := uuid.FromString(userManagerIdStr.String)
		if err != nil {
			return nil, fmt.Errorf("ERR: parsing user manager_id: %v", err)
		}
		manager.User.ManagerId = managerId
	}
	return manager, nil
}

func GetManagerByChatId(db DBExecutor, chatId int64) (*Manager, error) {
	query := `
		SELECT 
			m.id, m.user_id, m.created_at, m.updated_at, m.chat_id, m.state,
			u.id, u.chat_id, u.name, u.driver_id, u.manager_id, u.created_at, u.updated_at
		FROM users u
		JOIN managers m ON u.manager_id = m.id
		WHERE u.chat_id = $1
	`

	manager := new(Manager)
	manager.User = new(User)
	var managerIdStr, userIdStr string
	var userDriverIdStr, userManagerIdStr sql.NullString

	err := db.QueryRow(query, chatId).Scan(
		&managerIdStr, &userIdStr, &manager.CreatedAt, &manager.UpdatedAt, &manager.ChatId, &manager.State,

		&manager.User.Id, &manager.User.ChatId, &manager.User.Name,
		&userDriverIdStr, &userManagerIdStr,
		&manager.User.CreatedAt, &manager.User.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no manager found for chat_id %d", chatId)
		}
		return nil, fmt.Errorf("ERR: querying manager by chat_id: %v", err)
	}

	manager.Id, err = uuid.FromString(managerIdStr)
	if err != nil {
		return nil, fmt.Errorf("ERR: parsing manager id: %v", err)
	}

	manager.UserId, err = uuid.FromString(userIdStr)
	if err != nil {
		return nil, fmt.Errorf("ERR: parsing user id: %v", err)
	}

	if userDriverIdStr.Valid {
		driverId, err := uuid.FromString(userDriverIdStr.String)
		if err != nil {
			return nil, fmt.Errorf("ERR: parsing user driver_id: %v", err)
		}
		manager.User.DriverId = driverId
	}

	if userManagerIdStr.Valid {
		managerId, err := uuid.FromString(userManagerIdStr.String)
		if err != nil {
			return nil, fmt.Errorf("ERR: parsing user manager_id: %v", err)
		}
		manager.User.ManagerId = managerId
	}

	return manager, nil
}

func (m *Manager) StoreManager(db DBExecutor, bot *tgbotapi.BotAPI) error {
	tx, ok := db.(*sql.Tx)
	var txErr error

	id, err := uuid.NewV4()
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("ERR: creating a new uuid for a manager: %v (txErr: %v)\n", err, txErr)
	}

	stmt, err := db.Prepare(`
		INSERT INTO managers (id, user_id, chat_id) 
		VALUES (?, ?, ?)
	`)
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("ERR: preparing statement for insert manager: %v (txErr: %v)\n", err, txErr)
	}
	defer stmt.Close()

	_, err = stmt.Exec(id.String(), m.UserId.String(), m.ChatId)
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("ERR: executing prep insert manager stmt: %v (txErr: %v)\n", err, txErr)
	}

	m.Id = id
	m.User.ManagerId = id

	updateStmt, err := db.Prepare(`
		UPDATE users 
		SET manager_id = $1, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $2
	`)
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("ERR: preparing statement for update user manager_id: %v (txErr: %v)\n", err, txErr)
	}
	defer updateStmt.Close()

	_, err = updateStmt.Exec(id.String(), m.UserId.String())
	if err != nil {
		if ok {
			txErr = tx.Rollback()
		}
		return fmt.Errorf("ERR: executing update user manager_id stmt: %v (txErr: %v)\n", err, txErr)
	}

	fmt.Println(1)
	err = m.User.SendRequestToSuperAdmins(db, bot)
	if err != nil {
		return fmt.Errorf("ERR: sending request to accept user to superadmins: %v\n", err)
	}

	return nil
}

func (u *User) IsManager(exec DBExecutor) (bool, error) {
	stmt, err := exec.Prepare("SELECT id, manager_id FROM users where chat_id=?")
	if err != nil {
		return false, fmt.Errorf("ERR: preparing statement to get id and manager_id: %v\n", err)
	}
	defer stmt.Close()
	row := stmt.QueryRow(u.ChatId)

	var managerIdNull sql.NullString
	if err := row.Scan(&u.Id, &managerIdNull); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("cannot even find a user somehow: %v\n", err)
		}
		return false, fmt.Errorf("ERR: scanning rows: %v\n", err)
	}

	if managerIdNull.Valid {
		managerId, err := uuid.FromString(managerIdNull.String)
		if err != nil {
			return false, fmt.Errorf("ERR: parsing manager_id: %w", err)
		}
		u.ManagerId = managerId
		return true, nil
	}

	u.ManagerId = uuid.Nil
	return false, nil
}

func (m *Manager) ShowDriverList(exec DBExecutor, callback string, caption string, chatId int64, bot *tgbotapi.BotAPI) error {
	drivers, err := GetAllDrivers(exec)
	if err != nil {
		return fmt.Errorf("ERR: getting all drivers: %v", err)
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, d := range drivers {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s, Номера авто: %s", d.User.Name, d.CarId),
				fmt.Sprintf("%s:%d", callback, d.User.ChatId),
			),
		))
	}

	msg := tgbotapi.NewMessage(chatId, "👤 "+caption)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	_, err = bot.Send(msg)
	return err
}

func (pm *PendingMessage) StoreDocForShipment(exec *sql.DB, bot *tgbotapi.BotAPI) (int, error) {
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: pm.FileId})
	if err != nil {
		return 0, fmt.Errorf("ERR: getting file info: %v", err)
	}

	fileURL := file.Link(bot.Token)
	log.Printf("File download URL: %s", fileURL)

	fullPath, err := config.DownloadFile(fileURL, strings.Split(fileURL, "/")[6])
	log.Println("ERR: Downloading: ", err)

	downloadedDoc := docs.File{
		TgFileId:     pm.FileId,
		From:         pm.FromChatId,
		Name:         strings.Split(fileURL, "/")[6],
		OriginalName: pm.DocOriginalName,
		Path:         fullPath,
		Mimetype:     pm.DocMimetype,
		Filetype:     docs.Document,
	}

	err = downloadedDoc.StoreFile(exec)
	return downloadedDoc.Id, err
}

func (pm *PendingMessage) SendDocToDriver(exec *sql.DB, bot *tgbotapi.BotAPI) error {
	if manager, err := GetManagerByChatId(exec, pm.FromChatId); err != nil && manager == nil {
		bot.Send(tgbotapi.NewMessage(pm.FromChatId, "Користувач повинен бути менеджером для використання цієї команди"))
	} else {
		fmt.Println(manager)
	}

	f := docs.File{TgFileId: pm.FileId}
	err := f.GetFile(exec)
	if err != nil {
		return fmt.Errorf("ERR: getting file from file id of tg: %v\n", err)
	}

	driver, err := GetDriverByChatId(exec, pm.ToChatId)
	if err != nil {
		return fmt.Errorf("ERR: getting driver from chat id: %v\n", err)
	}

	shipment, err := parser.GetSequenceOfTasks(f.Path)
	if err != nil {
		return fmt.Errorf("ERR: reading the shipment doc: %v\n", err)
	}

	shipment.CarId = driver.CarId
	shipment.DriverId = driver.Id
	shipment.ShipmentDocId = f.Id
	fmt.Println(f.Id, f.Path)

	err = shipment.StoreShipment(exec)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			_, err = bot.Send(tgbotapi.NewMessage(pm.FromChatId, "Такий маршрут та документ вже існують, спробуйте ще раз"))
			return err
		}
		return fmt.Errorf("store shipment: %w", err)
	}

	docMsg := tgbotapi.NewDocument(pm.ToChatId, tgbotapi.FileID(pm.FileId))
	docMsg.Caption = formTextAcceptTask + "\n\nНотатки від логіста:\n\n" + pm.Caption
	docMsg.ParseMode = tgbotapi.ModeHTML
	docMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Прочитати документ", fmt.Sprintf("readdoc:%d", f.Id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Почати маршрут", fmt.Sprintf("shipment:accept:%d", shipment.ShipmentId)),
		),
	)

	_, err = bot.Send(docMsg)
	return err
}

func GetAllManagers(db DBExecutor) ([]*Manager, error) {
	query := `
		SELECT 
			m.id, m.user_id, m.created_at, m.updated_at, m.chat_id, m.state,
			u.id, u.chat_id, u.name, u.driver_id, u.manager_id, u.created_at, u.updated_at
		FROM managers m
		JOIN users u ON m.user_id = u.id
		ORDER BY u.name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ERR: querying all managers: %v", err)
	}
	defer rows.Close()

	var managers []*Manager
	for rows.Next() {
		manager := new(Manager)
		manager.User = new(User)
		var managerIdStr, userIdStr string
		var userDriverIdStr, userManagerIdStr sql.NullString

		err := rows.Scan(
			&managerIdStr, &userIdStr, &manager.CreatedAt, &manager.UpdatedAt, &manager.ChatId, &manager.State,

			&manager.User.Id, &manager.User.ChatId, &manager.User.Name,
			&userDriverIdStr, &userManagerIdStr,
			&manager.User.CreatedAt, &manager.User.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("ERR: scanning manager row: %v", err)
		}

		manager.Id, err = uuid.FromString(managerIdStr)
		if err != nil {
			return nil, fmt.Errorf("ERR: parsing manager id: %v", err)
		}

		manager.UserId, err = uuid.FromString(userIdStr)
		if err != nil {
			return nil, fmt.Errorf("ERR: parsing user id: %v", err)
		}

		if userDriverIdStr.Valid {
			driverId, err := uuid.FromString(userDriverIdStr.String)
			if err != nil {
				return nil, fmt.Errorf("ERR: parsing user driver_id: %v", err)
			}
			manager.User.DriverId = driverId
		}

		if userManagerIdStr.Valid {
			managerId, err := uuid.FromString(userManagerIdStr.String)
			if err != nil {
				return nil, fmt.Errorf("ERR: parsing user manager_id: %v", err)
			}
			manager.User.ManagerId = managerId
		}

		managers = append(managers, manager)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ERR: iterating manager rows: %v", err)
	}

	return managers, nil
}
