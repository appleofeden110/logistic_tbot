package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofrs/uuid"
)

type TableType string
type Form struct {
	Id          uuid.UUID `db:"id"`
	ChatID      int64     `db:"chat_id"`
	FormMsgText string    `db:"message_text"`
	FormMsgId   int       `db:"message_id"`
	WhichTable  TableType `db:"which_table"`
	Data        any
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
type FormState struct {
	Form         Form
	FieldNames   []string
	CurrentField string
	Answers      []string
	Questions    []string
	Index        int
	Finished     bool
}

const (
	UsersTable    = TableType("users")
	DriversTable  = TableType("drivers")
	ManagersTable = TableType("managers")
	RefuelsTable  = TableType("tank_refuels")
	RoutesTable   = TableType("routes")
	CarsTable     = TableType("cars")
)

func (tt TableType) String() string {
	return string(tt)
}

func CheckFormTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS form_states (
		    id TEXT PRIMARY KEY,
			chat_id INTEGER,
			user_id TEXT,
			message_text TEXT,
			form_message_id INTEGER,
			which_table TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`)
	return err
}

func (f Form) StoreForm(db *sql.DB, bot *tgbotapi.BotAPI) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ERR: beginning transaction: %v\n", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO form_states (chat_id, message_text, form_message_id, which_table) VALUES  ($1, $2, $3, $4) 
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("ERR: prepping stmt for inserting into form states: %v \n", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(f.ChatID, f.FormMsgText, f.FormMsgId, f.WhichTable)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("ERR: executing insert stmt for a form: %v \n", err)
	}

	return insertIntoSpecTable(f.Data, tx, f.ChatID, bot)
}

// if err != nil in this function, transaction must be absolutely fucking rolled back
func insertIntoSpecTable(data any, tx *sql.Tx, chatId int64, bot *tgbotapi.BotAPI) error {
	var err error
	switch v := data.(type) {
	case Driver:
		v.User.ChatId = chatId
		err = v.User.StoreUser(tx)
		if err != nil {
			tx.Rollback()

			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				bot.Send(tgbotapi.NewMessage(chatId, "Такий користувача скоріш всього вже існує. Якщо це помилково - напишіть розробнику: @pinkfloydfan або @NazKan_Uk"))
			}
			return fmt.Errorf("ERR: storing user: %v\n", err)
		}

		v.UserId = v.User.Id
		err = v.StoreDriver(tx, bot)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("ERR: storing driver in the table: %v\n", err)
		}

		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("ERR: committing transaction: %v\n", err)
		}

		return nil
	case Manager:
		v.User.ChatId = chatId
		err = v.User.StoreUser(tx)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("ERR: storing user based on manager's form: %v\n", err)
		}
		v.UserId = v.User.Id

		err = v.StoreManager(tx, bot)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("ERR: storing manager in the table: %v\n", err)
		}

		err = tx.Commit()
		if err != nil {
			return err
		}

		return nil
	case Car:
		err = v.AddCarToDB(chatId, bot, tx)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("ERR: adding car to db: %v\n", err)
		}

		return tx.Commit()
	default:
		err = fmt.Errorf("Wrong type: %T. Possible tx error: %v\n", v, tx.Rollback())
	}
	return err
}

func CheckAllTables(db *sql.DB) (err error) {
	fmt.Println("checking every table if exists...")

	err = CheckTaskDocsTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table task_docs: %v\n", err)
	}
	log.Println("task_docs is ok.")

	err = CheckFormStatesTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table form_states: %v\n", err)
	}
	log.Println("form_states is ok.")

	err = CheckUsersTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table users: %v\n", err)
	}
	log.Println("users is ok.")

	err = CheckManagersTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table managers: %v\n", err)
	}
	log.Println("managers is ok.")

	err = CheckDriversTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table drivers: %v\n", err)
	}
	log.Println("drivers is ok.")

	err = CheckDriversSessionsTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table drivers_sessions: %v\n", err)
	}
	log.Println("drivers_sessions is ok.")

	err = CheckCarsTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table cars: %v\n", err)
	}
	log.Println("cars is ok.")

	err = CheckCleaningStationsTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table cleaning_stations: %v\n", err)
	}
	log.Println("cleaning_stations is ok.")

	err = CheckFilesTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table files: %v\n", err)
	}
	log.Println("files is ok.")

	err = CheckShipmentsTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table shipments: %v\n", err)
	}
	log.Println("shipments is ok.")

	err = CheckTasksTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table tasks: %v\n", err)
	}
	log.Println("tasks is ok.")

	err = CheckShipmentSessionsTable(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table shipment_sessions: %v\n", err)
	}
	log.Println("shipment_sessions is ok.")

	err = CheckCommunicationMessages(db)
	if err != nil {
		return fmt.Errorf("ERR: creating or checking the table shipment_sessions: %v\n", err)
	}
	log.Println("communication_messages is ok.")

	return nil
}
