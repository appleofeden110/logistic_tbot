package handlers

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"logistictbot/config"
	"logistictbot/db"
	"logistictbot/docs"
	"logistictbot/duration"
	"logistictbot/parser"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofrs/uuid"
)

var photoGroups = struct {
	sync.Mutex
	m map[string][]*tgbotapi.Message
}{
	m: make(map[string][]*tgbotapi.Message),
}

var Bot *tgbotapi.BotAPI

func HandleShipmentDetails(chatId, shipmentId int64, globalStorage *sql.DB) error {
	shipment, err := parser.GetShipment(globalStorage, shipmentId)
	if err != nil {
		return fmt.Errorf("Error: getting shipment for the details (shipmentId: %d): %v\n", shipmentId, err)
	}

	f := docs.File{Id: shipment.ShipmentDocId}
	err = f.GetFile(globalStorage)
	if err != nil {
		return fmt.Errorf("Error: getting file from ShipmentDocId (%d): %v\n", shipment.ShipmentDocId, err)
	}

	driver, err := db.GetDriverById(globalStorage, shipment.DriverId)
	if err != nil {
		return fmt.Errorf("Error: getting driver by id: %v\n", err)
	}

	msg := tgbotapi.NewDocument(chatId, tgbotapi.FileID(f.TgFileId))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.Caption = fmt.Sprintf("<b><i>Shipment</i></b> №%d:\n<b>Водій</b>: %s (@%s) - %s\nЗавдання:\n\n", shipment.ShipmentId, driver.User.Name, driver.User.TgTag, driver.CarId)

	log.Println(shipment.Tasks)
	for i, task := range shipment.Tasks {
		msg.Caption += fmt.Sprintf("%d. <b><i>%s</i></b>\n<b>Адреса в документі</b>: %s\n\n", i+1, strings.ToUpper(string(task.Type[0]))+task.Type[1:], task.Address)
	}

	_, err = Bot.Send(msg)
	return err
}

func HandleCommand(chatId int64, command string, globalStorage *sql.DB) error {
	cmd, found := strings.CutPrefix(command, "/")
	if !found {
		return fmt.Errorf("it is not a command: %s\n", command)
	}

	cmd, found = strings.CutSuffix(cmd, "@logistictbot")
	if found {
		log.Printf("GROUP cmd: %s", cmd)
	}

	switch cmd {
	case "start":
		u := new(db.User)
		u.ChatId = chatId
		err := u.GetUserByChatId(globalStorage)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("Something wrong with the db: %v\n", err)
			}
			u = nil
		}
		err = HandleStart(chatId, globalStorage, u)
		if err != nil {
			panic(err)
		}
	case "mngrreset":
		err := db.SetAllManagersToDormant(globalStorage)
		if err != nil {
			Bot.Send(tgbotapi.NewMessage(chatId, fmt.Sprintf("Не вийшло резетнути всі статуси менеджерів, ось помилка: %v\n", err)))
			return err
		}
		_, err = Bot.Send(tgbotapi.NewMessage(chatId, "Всі менеджери тепер в дефолтному статусі"))
		return err
	case "test":

	case "add_car":
		c := db.Car{}
		err := createForm(chatId, c, formMarkupAddCar, formTextAddCar, "adding a car to the db (ONLY SA)")
		if err != nil {
			return err
		}
	case "createform:driver_registration":
		d := db.Driver{User: &db.User{ChatId: chatId}}
		err := createForm(chatId, d, formMarkupDriver, formTextDriver, "driver's registration")
		if err != nil {
			return err
		}
	case "createform:manager_registration":
		m := db.Manager{User: &db.User{ChatId: chatId}}
		err := createForm(chatId, m, formMarkupManager, formTextManager, "manager's registration")
		if err != nil {
			return err
		}
	case "send_task":
	case "menu":
		return HandleCommand(chatId, "/start", globalStorage)
	case "dev:init":
		devSesh, err := db.GetDev(globalStorage, chatId)
		if err != nil {
			log.Println(err)
			return nil
		}

		devSessionMu.Lock()
		devSession[devSesh.ChatId] = devSesh
		devSessionMu.Unlock()

		msg := tgbotapi.NewMessage(devSesh.ChatId, devInitMessage)
		msg.ReplyMarkup = devInit
		_, err = Bot.Send(msg)
		return err
	default:
		return fmt.Errorf("unrecognized command")
	}
	return nil
}
func HandleManagerCommands(chatId int64, command string, messageId int, globalStorage *sql.DB) error {
	cmd, f := strings.CutPrefix(command, "manager:")
	if !f {
		return fmt.Errorf("not the right format of a dev cmd, should be \"dev:<command>\", not %s\n", command)
	}

	managerSessionsMu.Lock()
	managerSesh, exists := managerSessions[chatId]
	managerSessionsMu.Unlock()

	if !exists {
		return fmt.Errorf("not a manager session, register")
	}

	switch cmd {
	case "create":
		managerSesh.State = db.StateWaitingDoc

		err := managerSesh.ChangeManagerStatus(globalStorage)
		if err != nil {
			return err
		}

		msg := tgbotapi.NewMessage(chatId, "📄 Надішліть документ, який хочете відправити водію.")
		_, err = Bot.Send(msg)
		return err
	case "viewdrivers":
		drivers, err := db.GetAllDrivers(globalStorage)
		if err != nil {
			return fmt.Errorf("Err getting all driver by the ask of the manager: %v\n", err)
		}

		msg := tgbotapi.NewMessage(chatId, "Список водіїв:\n\n")
		msg.ParseMode = tgbotapi.ModeHTML
		for _, d := range drivers {
			if d.CarId != "" {
				msg.Text += fmt.Sprintf("<b>%s</b> (@%s) - %s\n", d.User.Name, d.User.TgTag, d.CarId)
				continue
			}
			msg.Text += fmt.Sprintf("<b>%s</b> (@%s) - Машину водію не призначено\n", d.User.Name, d.User.TgTag)
		}

		Bot.Send(msg)

	case "viewall":
		shipments, err := parser.GetAllShipments(globalStorage)
		if err != nil {
			return fmt.Errorf("Err getting all shipments for all drivers: %v\n", err)
		}

		msg, err := CreateShipmentListMessage(shipments, 0, chatId, "page:viewall")
		if err != nil {
			return fmt.Errorf("Err creating shipment list message: %v\n", err)
		}

		_, err = Bot.Send(msg)
		if err != nil {
			return fmt.Errorf("Err sending message: %v\n", err)
		}

	case "viewactive":
		shipments, err := parser.GetAllActiveShipments(globalStorage)
		if err != nil {
			return fmt.Errorf("Err getting all shipments for all drivers: %v\n", err)
		}

		msg, err := CreateShipmentListMessage(shipments, 0, chatId, "page:viewactive")
		if err != nil {
			return fmt.Errorf("Err creating shipment list message: %v\n", err)
		}

		_, err = Bot.Send(msg)
		if err != nil {
			return fmt.Errorf("Err sending message: %v\n", err)
		}
	case "mstmt":

		availableMonths, err := parser.GetAvailableMonths(globalStorage)
		if err != nil {
			return fmt.Errorf("Err getting available months for the shipments: %v\n", err)
		}

		msg := tgbotapi.NewMessage(chatId, "За який місяць?")

		markup := make([][]tgbotapi.InlineKeyboardButton, 0)
		buttons := make([]tgbotapi.InlineKeyboardButton, 0)

		for i := 0; i < len(availableMonths); i++ {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s %d", duration.MonthToUkrainian(availableMonths[i].Month), availableMonths[i].Year),
				fmt.Sprintf("mstmt:%d.%d", availableMonths[i].Month, availableMonths[i].Year),
			))

			if (i+1)%3 == 0 {
				markup = append(markup, buttons)
				buttons = make([]tgbotapi.InlineKeyboardButton, 0)
			}
		}

		if len(buttons) > 0 {
			markup = append(markup, buttons)
		}

		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(markup...)
		Bot.Send(msg)

	}
	return nil
}

func HandleManagerInputState(session *db.Manager, msg *tgbotapi.Message, globalStorage *sql.DB) (updatedSession *db.Manager, err error) {
	fmt.Printf("Manager session input msg (%s - %d): %s\n", session.State, session.ChatId, msg.Text)
	switch session.State {
	case db.StateWaitingDoc:
		if msg.Document != nil {
			session.PendingMessage = &db.PendingMessage{
				FromChatId:      msg.Chat.ID,
				MessageType:     "document",
				FromUser:        msg.From,
				DocOriginalName: msg.Document.FileName,
				DocMimetype:     docs.Mimetype(msg.Document.MimeType),
				FileId:          msg.Document.FileID,
			}
			session.State = db.StateWaitingNotes

			err = session.ChangeManagerStatus(globalStorage)
			if err != nil {
				return session, err
			}

			id, err := session.PendingMessage.StoreDocForShipment(globalStorage, Bot)
			if err != nil {
				return session, err
			}

			readDocMsg := tgbotapi.NewMessage(msg.Chat.ID, "⬇️ Натисніть тут що б прочитати документ")
			readDocMsg.ParseMode = tgbotapi.ModeHTML
			readDocMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Прочитати документ", "readdoc:"+strconv.Itoa(id))))

			_, err = Bot.Send(readDocMsg)
			if err != nil {
				config.VERY_BAD(msg.Chat.ID, Bot)
			}

			msg := tgbotapi.NewMessage(msg.Chat.ID, "✏️ Введіть нотатки або опис до документа:")
			_, err = Bot.Send(msg)
			return session, err
		} else {
			msg := tgbotapi.NewMessage(msg.Chat.ID, "Будь ласка, надішліть саме документ.")
			_, err = Bot.Send(msg)
			return session, err
		}
	case db.StateWaitingNotes:
		if msg.Text != "" {
			if session.PendingMessage == nil {
				session.State = db.StateDormantManager
				return session, session.ChangeManagerStatus(globalStorage)
			}
			session.PendingMessage.Caption = msg.Text
			session.State = db.StateWaitingDriver

			err = session.ChangeManagerStatus(globalStorage)
			if err != nil {
				return session, err
			}

			return session, session.ShowDriverList(globalStorage, msg.Chat.ID, *Bot)
		}
	}
	return session, err
}

func HandleDriverCommands(chatId int64, command string, messageId int, globalStorage *sql.DB) error {
	var cmd string
	var f bool

	cmd, f = strings.CutPrefix(command, "driver:")
	if !f {
		return fmt.Errorf("command is incorrect: %v\n", command)
	}

	log.Println("fuownaf")
	// made to transfer ids of tasks and shipments, handled case by case
	cmdString, _idString, idFound := strings.Cut(cmd, ":")
	log.Println(cmd)
	if idFound {
		cmd = cmdString
	}

	log.Println(cmd)
	driverSessionsMu.Lock()
	driverSesh, exists := driverSessions[chatId]
	driverSessionsMu.Unlock()

	if !exists {
		return fmt.Errorf("not a driver session, register")
	}

	switch cmd {
	case "begintask":
		taskId, err := strconv.Atoi(_idString)
		if err != nil {
			return err
		}

		task, err := parser.GetTaskById(globalStorage, taskId)
		if err != nil {
			return fmt.Errorf("err getting task by id (%d): %v\n", taskId, err)
		}

		switch task.Type {
		case parser.TaskLoad:
			driverSesh.State = db.StateLoad
		case parser.TaskUnload:
			driverSesh.State = db.StateUnload
		case parser.TaskCollect:
			driverSesh.State = db.StateCollect
		case parser.TaskDropoff:
			driverSesh.State = db.StateDropoff
		case parser.TaskCleaning:
			driverSesh.State = db.StateCleaning
		default:
			return fmt.Errorf("err wrong type of task: %s\n", task.Type)
		}

		driverSesh.PerformedTaskId = taskId

		err = driverSesh.SetPerformingTask(globalStorage)
		if err != nil {
			return fmt.Errorf("err changing driver's performing task: %v\n", err)
		}

		err = driverSesh.ChangeDriverStatus(globalStorage)
		if err != nil {
			return fmt.Errorf("err changing driver's status while performing a task: %v\n", err)
		}

		car, err := db.GetCarById(globalStorage, driverSesh.CarId)
		if err != nil {
			return fmt.Errorf("err getting a car for ending a task: %v\n", err)
		}

		msg := tgbotapi.NewMessage(chatId, fmt.Sprintf("Введіть поточний кілометраж автомобіля. \n<b><i>(попередньо: %d km)</i></b>\n\n(Доступні формати: 12345; 12,345; 12,345 км; 12,345 km)", car.Kilometrage))
		msg.ParseMode = tgbotapi.ModeHTML
		_, err = Bot.Send(msg)
		return err

	case "endtask":
		msg := tgbotapi.NewMessage(chatId, "Введіть вагу продукту\n(Доступні формати: 1234.5; 1,234.5; 1234.5 kg; 1,234.5 кг; 1234 kg)")
		msg.ParseMode = tgbotapi.ModeHTML
		_, err := Bot.Send(msg)
		return err
	case "add_doctotask":
		driverSesh.State = db.StateWaitingAttachment
		err := driverSesh.ChangeDriverStatus(globalStorage)
		if err != nil {
			return err
		}

		msg := tgbotapi.NewMessage(chatId, "Відправте документ який ви хочете прикріпити\n\nПри закінчені воно відправиться менеджеру разом з завданням")
		msg.ParseMode = tgbotapi.ModeHTML
		_, err = Bot.Send(msg)
		return err
	case "add_picstotask":
		driverSesh.State = db.StateWaitingPhoto
		err := driverSesh.ChangeDriverStatus(globalStorage)
		if err != nil {
			return err
		}

		msg := tgbotapi.NewMessage(chatId, "📸 Відправте фотографії які ви маєте прикріпити, та натисніть <b>\"Відправити Фотографії\"</b> знизу")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Відправити фотографії", "driver:send_pics")))
		msg.ParseMode = tgbotapi.ModeHTML
		_, err = Bot.Send(msg)
		return err
	case "send_pics":
		task, err := parser.GetTaskById(globalStorage, driverSesh.PerformedTaskId)
		if err != nil {
			return fmt.Errorf("err getting task by id (%d): %v\n", driverSesh.PerformedTaskId, err)
		}

		files, err := docs.GetFilesAttachedToTask(globalStorage, task.Id)
		if err != nil {
			return fmt.Errorf("err getting attached files: %v\n", err)
		}

		photos, _ := splitFiles(files)

		if len(photos) == 0 {
			msg := tgbotapi.NewMessage(chatId, "Немає фотографій для відправки")
			_, err = Bot.Send(msg)
			return err
		}

		// Confirm to driver
		confirmMsg := tgbotapi.NewMessage(chatId, fmt.Sprintf("Відправлено %d фотографій менеджеру ✅", len(photos)))
		_, err = Bot.Send(confirmMsg)
		if err != nil {
			return err
		}

		managerText := fmt.Sprintf("📸 Фотографії від водія %s (%s)\nМашина: %s\nЗавдання: %s (Shipment №%d)",
			driverSesh.User.Name,
			driverSesh.User.TgTag,
			driverSesh.CarId,
			task.Type,
			task.Id,
		)

		err = sendPhotosToManager(TestManagerChatId, photos, managerText)
		if err != nil {
			return fmt.Errorf("err sending photos to manager: %v\n", err)
		}

		// Reset driver state back to task state
		switch task.Type {
		case parser.TaskLoad:
			driverSesh.State = db.StateLoad
		case parser.TaskUnload:
			driverSesh.State = db.StateUnload
		case parser.TaskCollect:
			driverSesh.State = db.StateCollect
		case parser.TaskDropoff:
			driverSesh.State = db.StateDropoff
		case parser.TaskCleaning:
			driverSesh.State = db.StateCleaning
		default:
			return fmt.Errorf("err wrong type of task: %s\n", task.Type)
		}

		err = driverSesh.ChangeDriverStatus(globalStorage)
		return err

	case "beginday":
		car, err := db.GetCarById(globalStorage, driverSesh.CarId)
		if err != nil {
			return fmt.Errorf("err getting a car for the day beggining: %v\n", err)
		}
		additionalInfo := fmt.Sprintf("%s\nПочатковий кілометраж: %s\n\n", time.Now().Format(time.DateTime), db.FormatKilometrage(int(car.Kilometrage)))

		driverSesh.State = db.StateWorking
		err = driverSesh.ChangeDriverStatus(globalStorage)
		if err != nil {
			return fmt.Errorf("err changing driver's status: %v\n", err)
		}

		_, err = driverSesh.UnpauseSession(globalStorage)
		if err != nil {
			return fmt.Errorf("err starting a day: %v\n", err)
		}

		msg := tgbotapi.NewMessage(chatId, fmt.Sprintf("%sЛаскаво просимо, водію %s\nЩо ви хочете зробити?", additionalInfo, driverSesh.User.Name))
		msg.ReplyMarkup = driverStartMarkupWorking

		_, err = Bot.Send(msg)
		if err != nil {
			return err
		}

		driverSesh.State = db.StateWorking
		return driverSesh.ChangeDriverStatus(globalStorage)
	case "endDay":
		driverSesh.State = db.StateEndingDay
		err := driverSesh.ChangeDriverStatus(globalStorage)
		if err != nil {
			return nil
		}

		car, err := db.GetCarById(globalStorage, driverSesh.CarId)
		if err != nil {
			return fmt.Errorf("Error: getting car for the end of the day: %v\n", err)
		}

		msg := tgbotapi.NewMessage(chatId, fmt.Sprintf("Введіть поточний кілометраж автомобіля. \n<b><i>(попередньо: %d km)</i></b>\n\n(Доступні формати: 12345; 12,345; 12,345 км; 12,345 km)", car.Kilometrage))
		msg.ParseMode = tgbotapi.ModeHTML
		_, err = Bot.Send(msg)
		return err
	case "viewactive":
		shipments, err := parser.GetAllActiveShipmentsByCarId(driverSesh.CarId, globalStorage)
		if err != nil {
			return fmt.Errorf("Err getting all shipments for all drivers: %v\n", err)
		}

		msg, err := CreateShipmentListMessage(shipments, 0, chatId, "page:viewactivebycar:"+driverSesh.CarId)
		if err != nil {
			return fmt.Errorf("Err creating shipment list message: %v\n", err)
		}

		_, err = Bot.Send(msg)
		if err != nil {
			return fmt.Errorf("Err sending message: %v\n", err)
		}

	case "viewall":
		shipments, err := parser.GetAllShipmentsByCarId(driverSesh.CarId, globalStorage)
		if err != nil {
			return fmt.Errorf("Err getting all shipments for all drivers: %v\n", err)
		}

		msg, err := CreateShipmentListMessage(shipments, 0, chatId, "page:viewallbycar:"+driverSesh.CarId)
		if err != nil {
			return fmt.Errorf("Err creating shipment list message: %v\n", err)
		}

		_, err = Bot.Send(msg)
		if err != nil {
			return fmt.Errorf("Err sending message: %v\n", err)
		}
	}
	return nil
}

func HandleDriverInputState(driver *db.Driver, msg *tgbotapi.Message, globalStorage *sql.DB) (*db.Driver, error) {
	var err error
	log.Printf("Driver driver input msg (%s - %s): %s\n", driver.State, driver.CarId, msg.Text)

	switch driver.State {
	case db.StateWaitingAttachment:
		task, err := parser.GetTaskById(globalStorage, driver.PerformedTaskId)
		if err != nil {
			return driver, fmt.Errorf("err getting task by id (%d): %v\n", driver.PerformedTaskId, err)
		}

		if msg.Document != nil {
			file, err := Bot.GetFile(tgbotapi.FileConfig{FileID: msg.Document.FileID})
			if err != nil {
				return driver, fmt.Errorf("error getting file info: %v", err)
			}

			fileURL := file.Link(Bot.Token)
			log.Printf("File download URL: %s", fileURL)

			fullPath := "./handlers/outdocs/" + strings.Split(fileURL, "/")[6]

			sentDoc := docs.File{
				TgFileId:     msg.Document.FileID,
				From:         msg.Chat.ID,
				Name:         strings.Split(fileURL, "/")[6],
				OriginalName: msg.Document.FileName,
				Path:         fullPath,
				Mimetype:     docs.Mimetype(msg.Document.MimeType),
				Filetype:     docs.Document,
			}
			err = sentDoc.StoreFile(globalStorage)
			if err != nil {
				return driver, fmt.Errorf("err storing file: %v\n", err)
			}

			err = sentDoc.AttachFileToTask(globalStorage, task.Id)
			if err != nil {
				return driver, fmt.Errorf("err attaching file to task %d: %v\n", task.Id, err)
			}

			_, err = Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Документ додано до завдання!"))
			return driver, err
		}

		switch task.Type {
		case parser.TaskLoad:
			driver.State = db.StateLoad
		case parser.TaskUnload:
			driver.State = db.StateUnload
		case parser.TaskCollect:
			driver.State = db.StateCollect
		case parser.TaskDropoff:
			driver.State = db.StateDropoff
		case parser.TaskCleaning:
			driver.State = db.StateCleaning
		default:
			return driver, fmt.Errorf("err wrong type of task: %s\n", task.Type)
		}

		err = driver.ChangeDriverStatus(globalStorage)
		return driver, err
	case db.StateWaitingPhoto:
		task, err := parser.GetTaskById(globalStorage, driver.PerformedTaskId)
		if err != nil {
			return driver, fmt.Errorf("err getting task by id (%d): %v\n", driver.PerformedTaskId, err)
		}

		if len(msg.Photo) > 0 {
			if msg.MediaGroupID != "" {
				// Collect photos from media group
				photoGroups.Lock()
				photoGroups.m[msg.MediaGroupID] = append(photoGroups.m[msg.MediaGroupID], msg)
				photoGroups.Unlock()

				go func(groupID string, taskId int) {
					time.Sleep(1000 * time.Millisecond)

					photoGroups.Lock()
					msgs := photoGroups.m[groupID]
					delete(photoGroups.m, groupID)
					photoGroups.Unlock()

					for _, m := range msgs {
						if err := savePhotoToTask(m, taskId, globalStorage); err != nil {
							log.Printf("err saving album photo: %v", err)
						}
					}

					if len(msgs) > 0 {
						Bot.Send(tgbotapi.NewMessage(
							msg.Chat.ID,
							fmt.Sprintf("Додано %d фотографій 📸", len(msgs)),
						))
					}
				}(msg.MediaGroupID, task.Id)

				return driver, nil
			} else {
				// Single photo
				if err := savePhotoToTask(msg, task.Id, globalStorage); err != nil {
					return driver, fmt.Errorf("err saving single photo: %v", err)
				}

				Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Додано 1 фотографію 📸"))
				return driver, nil
			}
		}
	case db.StateLoad, db.StateUnload, db.StateCollect, db.StateDropoff, db.StateCleaning:
		task := new(parser.TaskSection)

		task, err := parser.GetTaskById(globalStorage, driver.PerformedTaskId)
		if err != nil {
			return driver, fmt.Errorf("err getting task by id (%d): %v\n", driver.PerformedTaskId, err)
		}

		country, _ := parser.ExtractCountry(task.Address)

		shipment, err := parser.GetShipment(globalStorage, task.ShipmentId)
		if err != nil {
			return driver, fmt.Errorf("err getting shipment from a task: %v\n", err)
		}

		log.Println(task.CurrentKilometrage, task.CurrentTemperature, task.CurrentWeight)
		log.Println(task.Start.IsZero(), task.End.IsZero())

		if task.CurrentKilometrage == 0 && task.Start.IsZero() {
			km, err := db.ParseKilometrage(msg.Text)
			if err != nil {
				_, err = Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Неправильний формат кілометражу, спробуйте ще раз"))
				return driver, err
			}
			task.CurrentKilometrage = km

			err = task.UpdateCurrentKmById(globalStorage)
			if err != nil {
				return driver, fmt.Errorf("err updating kilometrage by task id: %v\n", err)
			}

			err = task.StartTaskById(globalStorage)
			if err != nil {
				return driver, fmt.Errorf("err starting a task: %v\n", err)
			}

			startTaskMsg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf(TaskSubmissionFormatText,
				task.ShipmentId,
				strings.ToUpper(task.Type),
				shipment.Chassis,
				shipment.Container,
				time.Now().Format("02.01.2006"),
				task.Start.Format("15:04"),
				"",
				db.FormatKilometrage(int(task.CurrentKilometrage)),
				task.Address,
				country.Name,
				country.Emoji,
				0,
				0.00),
			)
			startTaskMsg.ParseMode = tgbotapi.ModeHTML

			startTaskMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Закінчити виконання", fmt.Sprintf("driver:endtask:%d", task.Id)),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Додати документ", fmt.Sprintf("driver:add_doctotask:%d", task.Id)),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Додати фотографії", fmt.Sprintf("driver:add_picstotask:%d", task.Id)),
				),
			)
			var pinMsg tgbotapi.Message
			pinMsg, err = Bot.Send(startTaskMsg)
			if err != nil {
				return driver, err
			}

			pin := tgbotapi.PinChatMessageConfig{
				ChatID:              pinMsg.Chat.ID,
				MessageID:           pinMsg.MessageID,
				DisableNotification: true,
			}

			Bot.Send(pin)

			driverInfo := fmt.Sprintf("Водій %s (%s) почав завдання %s для маршруту %d\nМашина: %s\n\n", driver.User.Name, driver.User.TgTag, task.Type, shipment.ShipmentId, driver.CarId)
			startTaskMsg.ReplyMarkup = nil
			startTaskMsg.Text = driverInfo + startTaskMsg.Text
			startTaskMsg.ChatID = TestManagerChatId
			_, err = Bot.Send(startTaskMsg)
			return driver, err
		}

		if task.CurrentWeight == 0 && task.End.IsZero() {
			kg, err := db.ParseWeight(msg.Text)
			if err != nil {
				_, err = Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Неправильний формат ваги, спробуйте ще раз"))
				return driver, err
			}
			task.CurrentWeight = kg

			err = task.UpdateCurrentWeightById(globalStorage)
			if err != nil {
				return driver, fmt.Errorf("err updating weight by task id: %v\n", err)
			}

			tempMsg := tgbotapi.NewMessage(msg.Chat.ID, "Введіть температуру.\n(Доступні формати: -18.5; -18,5; -18.5°C; -18,5 °C; -18.5 C)")
			tempMsg.ParseMode = tgbotapi.ModeHTML
			_, err = Bot.Send(tempMsg)
			return driver, err
		}

		if task.CurrentTemperature == 0 && task.End.IsZero() {
			celcius, err := db.ParseTemperature(msg.Text)
			if err != nil {
				_, err = Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Неправильний формат температури, спробуйте ще раз"))
				return driver, err
			}
			task.CurrentTemperature = celcius
			err = task.UpdateCurrentWeightById(globalStorage)
			if err != nil {
				return driver, fmt.Errorf("err updating weight by task id: %v\n", err)
			}

		}

		err = task.FinishTaskById(globalStorage)
		if err != nil {
			return driver, err
		}

		err = driver.DeletePerformingTask(globalStorage)
		if err != nil {
			return driver, err
		}

		driver.State = db.StateWorking
		err = driver.ChangeDriverStatus(globalStorage)
		if err != nil {
			return driver, err
		}

		driverInfo := fmt.Sprintf("Від водія %s (%s)\nМашина: %s\n", driver.User.Name, driver.User.TgTag, driver.CarId)
		files := make([]*docs.File, 0)

		endMsg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Завдання успішно виконано!\n"+TaskSubmissionFormatText,
			task.ShipmentId,
			strings.ToUpper(task.Type),
			shipment.Chassis,
			shipment.Container,
			time.Now().Format("02.01.2006"),
			task.Start.Format("15:04"),
			task.End.Format("15:04"),
			db.FormatKilometrage(int(task.CurrentKilometrage)),
			task.Address,
			country.Name,
			country.Emoji,
			task.CurrentWeight,
			task.CurrentTemperature),
		)
		endMsg.ParseMode = tgbotapi.ModeHTML

		_, err = Bot.Send(endMsg)
		if err != nil {
			return driver, err
		}

		files, err = docs.GetFilesAttachedToTask(globalStorage, task.Id)
		if err != nil {
			return driver, fmt.Errorf("err getting attached to task files: %v\n", err)
		}

		photos, docsFiles := splitFiles(files)
		managerText := driverInfo + endMsg.Text

		if len(photos) > 0 {
			err = sendPhotosToManager(TestManagerChatId, photos, managerText)
			if err != nil {
				return driver, err
			}
		} else {
			msg := tgbotapi.NewMessage(TestManagerChatId, managerText)
			msg.ParseMode = tgbotapi.ModeHTML
			if _, err = Bot.Send(msg); err != nil {
				return driver, err
			}
		}

		if len(docsFiles) > 0 {
			if err = sendDocumentsToManager(TestManagerChatId, docsFiles); err != nil {
				return driver, err
			}
		}
		//_, err = Bot.Send(endMsg)
		/*managerSessionsMu.Lock()
		defer managerSessionsMu.Unlock()
		for mChatId := range managerSessions {
			endMsg.ChatID = mChatId
			_, err = Bot.Send(endMsg)
			if err != nil {
				return driver, fmt.Errorf("%d could not receive message: %v\n", mChatId, err)
			}
		}*/

		return driver, err

	case db.StateEndingDay:

		if driver.Session == nil {
			driver.Session, err = driver.GetLastActiveSession(globalStorage)
			if err != nil {
				return driver, fmt.Errorf("err getting last active driver: %v\n", err)
			}
		}

		session := driver.Session

		if session.EndKilometrage.Int64 == 0 {
			var oldKm int64
			km, err := db.ParseKilometrage(msg.Text)
			if err != nil {
				return driver, fmt.Errorf("err parsing kmtrage: %v\n", err)
			}

			car, err := db.GetCarById(globalStorage, driver.CarId)
			if err != nil {
				return driver, fmt.Errorf("err getting car by id from the driver's sesh: %v\n", err)
			}

			if car == nil {
				return driver, fmt.Errorf("car does not exist: %v\n", car)
			}

			oldKm = car.Kilometrage
			car.Kilometrage = km
			kmAccum := int(km - oldKm)
			if kmAccum < 0 {
				message, err := Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Новий кілометраж менше за ваш старий, спробуйте ще раз"))
				log.Printf("Trying to end the day again\n\tendDayerr: %v\n\tbotSendErr: %v\n\n",
					HandleDriverCommands(msg.Chat.ID, "driver:endDay", message.MessageID, globalStorage),
					err,
				)
				return driver, nil
			}
			session.EndKilometrage = sql.NullInt64{Valid: km > 0, Int64: km}
			session.KilometrageAccumulated = kmAccum

			err = car.UpdateCarKilometrage(globalStorage)
			if err != nil {
				return driver, err
			}

			_, err = Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Кілометраж для %s було оновлено з %d км, до %d км", car.Id, oldKm, car.Kilometrage)))

			if err != nil {
				return driver, err
			}
			_, err = Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Тепер введіть будь ласка тривалість праці (Work time), формат: 15:25 або 15.25")))
			return driver, err
		}

		if session.Worktime.Nanoseconds() == 0 {
			var workTime duration.Duration

			workTime = db.ParseDuration(msg.Text)
			if workTime.Nanoseconds() == 0 {
				_, err = Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Введіть будь ласка тривалість праці (Work time) ще раз, формат: 15:25 або 15.25")))
			}

			session.Worktime = workTime

			_, err = Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Тепер введіть, будь ласка, тривалість водіння (Drive time), формат: 15:25 або 15.25")))
			return driver, nil
		}
		if session.Drivetime.Nanoseconds() == 0 {
			var driveTime duration.Duration

			driveTime = db.ParseDuration(msg.Text)
			if driveTime.Nanoseconds() == 0 {
				_, err = Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Введіть будь ласка тривалість паузи (Pause time) ще раз, прийнятний формат - 15:25 або 15.25")))
			}

			session.Drivetime = driveTime
			_, err = Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Тепер введіть, будь ласка, тривалість паузи (Pause time), формат: 15:25 або 15.25")))
			return driver, nil
		}
		if session.Pausetime.Nanoseconds() == 0 {
			var pausedTime duration.Duration

			pausedTime = db.ParseDuration(msg.Text)
			if pausedTime.Nanoseconds() == 0 {
				_, err = Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Введіть будь ласка тривалість праці (Drive time) ще раз, прийнятний формат - 15:25 або 15.25")))
			}

			session.Paused = sql.NullTime{Valid: true, Time: time.Now()}

			session.Pausetime = pausedTime
			session, err = driver.PauseSession(globalStorage)
			if err != nil {
				return driver, fmt.Errorf("err pausing day's session: %v\n", err)
			}
			finishMsg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("%s\nІнформація по дню:\n\nПочаток зміни: %s\nКінець зміни: %s\nПочатковий кілометраж: %s\nКінцевий кілометраж: %s\nЗагальна дистанція: %s\n\nТривалість:\nПраці (Work) - %s годин\nВодіння (Drive) - %s годин\nПаузи (Pause) - %s годин\n\nДякуємо за вашу працю, гарного дня!",
				time.Now().Format("02/01/2006"),
				session.Started.Format("15:04"),
				session.Paused.Time.Format("15:04"),
				db.FormatKilometrage(int(session.StartingKilometrage.Int64)),
				db.FormatKilometrage(int(session.EndKilometrage.Int64)),
				db.FormatKilometrage(session.KilometrageAccumulated),
				session.Worktime.Format(duration.ForPresentation),
				session.Drivetime.Format(duration.ForPresentation),
				session.Pausetime.Format(duration.ForPresentation),
			),
			)
			finishMsg.ParseMode = tgbotapi.ModeHTML

			driver.State = db.StatePause
			err = driver.ChangeDriverStatus(globalStorage)
			if err != nil {
				return driver, err
			}

			_, err = Bot.Send(finishMsg)
			if err != nil {
				return driver, err
			}

			return driver, err
		}

	}

	return driver, err
}

func savePhotoToTask(
	msg *tgbotapi.Message,
	taskId int,
	globalStorage *sql.DB,
) error {

	// highest resolution photo
	photo := msg.Photo[len(msg.Photo)-1]

	file, err := Bot.GetFile(tgbotapi.FileConfig{FileID: photo.FileID})
	if err != nil {
		return fmt.Errorf("error getting photo file info: %v", err)
	}

	fileURL := file.Link(Bot.Token)
	log.Printf("Photo download URL: %s", fileURL)

	filename := strings.Split(fileURL, "/")[6]

	fullPath := "/Users/appleofeden110/dev/logistictbot/handlers/outpics/" + filename

	sentPic := docs.File{
		TgFileId:     photo.FileID,
		From:         msg.Chat.ID,
		Name:         filename,
		OriginalName: filename,
		Path:         fullPath,
		Mimetype:     docs.Mimetype("image/jpeg"),
		Filetype:     docs.Image,
	}

	err = sentPic.StoreFile(globalStorage)
	if err != nil {
		return fmt.Errorf("err storing photo: %v", err)
	}

	return sentPic.AttachFileToTask(globalStorage, taskId)
}

func HandleSACommands(chatId int64, command string, messageId int, globalStorage *sql.DB) error {
	a, f := strings.CutPrefix(command, "sa:")
	if !f {
		return fmt.Errorf("command is incorrect: %v\n", command)
	}

	managerSessionsMu.Lock()
	_, exists := managerSessions[chatId]
	managerSessionsMu.Unlock()

	if !exists {
		return fmt.Errorf("not a manager session, register")
	}

	cmd, _idString, _ := strings.Cut(a, ":")
	switch cmd {
	case "approve":
		approvedChatId, err := strconv.ParseInt(_idString, 10, 64)
		if err != nil {
			return fmt.Errorf("Err parsing chatid from string: %s, err:%v\n", _idString, err)
		}

		u := &db.User{ChatId: approvedChatId}

		err = u.GetUserByChatId(globalStorage)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("Error getting this user from the db: %v\n", err)
			}
			return fmt.Errorf("Could not find this user, maybe already declined?: %v\n", err)
		}

		if u.DriverId != uuid.Nil {
			carQuestion := tgbotapi.NewMessage(chatId, "Яку машину ви призначаєте цьому водію?")
			cars, err := db.GetAllCars(globalStorage)
			if err != nil {
				return fmt.Errorf("Fetching cars for question: %v\n", err)
			}

			rows := make([][]tgbotapi.InlineKeyboardButton, 0)
			for _, v := range cars {
				car := fmt.Sprintf("%s (%d km)", v.Id, v.Kilometrage)
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(car, fmt.Sprintf("sa:carfor:%d:%s", approvedChatId, v.Id)),
				))
			}

			carQuestion.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
			carQuestion.ParseMode = tgbotapi.ModeHTML

			_, err = Bot.Send(carQuestion)
			return err

		} else if u.ManagerId != uuid.Nil {
			Bot.Send(tgbotapi.NewMessage(chatId, "Користувача "+u.Name+" було підтверджено на роль менеджера!"))
			_, err = Bot.Send(tgbotapi.NewMessage(approvedChatId, "Вашу реєстрацію було прийнято!"))
			if err != nil {
				log.Println(err)
			}
			return HandleMenu(approvedChatId, globalStorage, u)
		}
	case "decline":
		declinedChatId, err := strconv.ParseInt(_idString, 10, 64)
		if err != nil {
			return fmt.Errorf("Err parsing chatid from string: %s, err:%v\n", _idString, err)
		}

		u := &db.User{ChatId: declinedChatId}

		err = u.GetUserByChatId(globalStorage)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("Error getting this user from the db: %v\n", err)
			}
			return fmt.Errorf("Could not find this user, maybe already declined?: %v\n", err)
		}

		tx, err := globalStorage.Begin()
		if err != nil {
			return fmt.Errorf("err starting transaction: %v", err)
		}
		defer tx.Rollback()

		if u.DriverId != uuid.Nil {
			_, err = tx.Exec("DELETE from drivers where id = ?", u.DriverId.String())
			if err != nil {
				return fmt.Errorf("err deleting declined driver: %v\n", err)
			}

			driverSessionsMu.Lock()
			delete(driverSessions, declinedChatId)
			driverSessionsMu.Unlock()
		} else if u.ManagerId != uuid.Nil {
			_, err = tx.Exec("DELETE from managers where id = ?", u.ManagerId.String())
			if err != nil {
				return fmt.Errorf("err deleting declined manager: %v\n", err)
			}

			managerSessionsMu.Lock()
			delete(managerSessions, declinedChatId)
			managerSessionsMu.Unlock()
		}

		_, err = tx.Exec("DELETE from users where id = ?", u.Id.String())
		if err != nil {
			return fmt.Errorf("err deleting declined user: %v\n", err)
		}

		Bot.Send(tgbotapi.NewMessage(chatId, "Користувачу "+u.Name+" не було підтвердженно реєстрацію."))

		Bot.Send(tgbotapi.NewMessage(declinedChatId, "Вашу реєстрацію було не прийнято. Звʼяжіться з одним із менеджерів на пряму, або спробуйте ще раз."))

		return tx.Commit()
	case "carfor":
		driverId, carId, f := strings.Cut(_idString, ":")
		if !f {
			return fmt.Errorf("Err getting any info from sa:carfor command: %v\n", command)
		}

		id, err := strconv.ParseInt(driverId, 10, 64)
		if err != nil {
			return fmt.Errorf("Err parsing int chat id of a driver: %v\n", err)
		}

		driver, err := db.GetDriverByChatId(globalStorage, id)
		if err != nil {
			return fmt.Errorf("Error getting driver by id: %v\n", err)
		}

		err = driver.UpdateCarId(globalStorage, carId)
		if err != nil {
			return fmt.Errorf("Err updating car id: %v\n", err)
		}

		car, err := db.GetCarById(globalStorage, carId)
		if err != nil {
			return fmt.Errorf("Err getting car by id: %v\n", err)
		}

		Bot.Send(tgbotapi.NewMessage(chatId, "Користувача "+driver.User.Name+" було підтверджено на роль водія!"))

		_, err = Bot.Send(tgbotapi.NewMessage(driver.User.ChatId, fmt.Sprintf("Вашу реєстрацію було прийнято! Вам було призначено автомобіль %s з кілометражом %d км.", car.Id, car.Kilometrage)))
		if err != nil {
			return fmt.Errorf("Err sending driver a msg: %v\n", err)
		}
		return HandleMenu(id, globalStorage, driver.User)
	}
	return nil
}

func HandleDevCommands(chatId int64, command string, messageId int, globalStorage *sql.DB) error {
	cmd, f := strings.CutPrefix(command, "dev:")
	if !f {
		return fmt.Errorf("not the right format of a dev cmd, should be \"dev:<command>\", not %s\n", command)
	}

	devSessionMu.Lock()
	devSesh, exists := devSession[chatId]
	devSessionMu.Unlock()

	if !exists {
		return fmt.Errorf("not a dev session, use /dev:init first")
	}

	switch cmd {
	case "updatecleaningstations":
		devSesh.State = db.StateWaitingForCleaningCSV

		msg := tgbotapi.NewMessage(devSesh.ChatId, "Send the csv with updated list of cleaning stations")
		_, err := Bot.Send(msg)
		return err
	case "finish":
		devSessionMu.Lock()
		delete(devSession, devSesh.ChatId)
		devSessionMu.Unlock()

		msg := tgbotapi.NewMessage(chatId, "dev sesh ended")
		_, err := Bot.Send(msg)
		return err
	}
	return nil
}

func HandleCleaningDevCSV(chatId int64, doc *tgbotapi.Document, globalStorage *sql.DB) error {
	fileURL, err := Bot.GetFileDirectURL(doc.FileID)
	if err != nil {
		return fmt.Errorf("error getting file URL: %w", err)
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		return fmt.Errorf("error downloading file: %w", err)
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading CSV: %w", err)
	}

	tx, err := globalStorage.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO cleaning_stations (id, name, address, country, lat, lon, opening_hours)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			address = excluded.address,
			country = excluded.country,
			lat = excluded.lat,
			lon = excluded.lon,
			opening_hours = excluded.opening_hours
	`)
	if err != nil {
		return fmt.Errorf("error preparing statement: %w", err)
	}
	defer stmt.Close()

	for i, record := range records {
		if i == 0 {
			if _, err := strconv.Atoi(record[0]); err != nil {
				continue
			}
		}

		if len(record) != 7 {
			return fmt.Errorf("invalid CSV format at row %d: expected 7 columns, got %d", i+1, len(record))
		}

		id, err := strconv.Atoi(record[0])
		if err != nil {
			return fmt.Errorf("invalid ID at row %d: %w", i+1, err)
		}

		lat, err := strconv.ParseFloat(record[4], 64)
		if err != nil {
			return fmt.Errorf("invalid latitude at row %d: %w", i+1, err)
		}

		lon, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			return fmt.Errorf("invalid longitude at row %d: %w", i+1, err)
		}

		_, err = stmt.Exec(id, record[1], record[2], record[3], lat, lon, record[6])
		if err != nil {
			return fmt.Errorf("error inserting row %d: %w", i+1, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	_, err = Bot.Send(tgbotapi.NewMessage(chatId, "The list has been updated"))
	return err
}

func HandleStart(chatId int64, globalStorage *sql.DB, user *db.User) error {
	msg := tgbotapi.NewMessage(chatId, "Ласкаво просимо до допоміжного бота V&R Spedition.")

	if user == nil {
		msg.Text += "\n\nЗареєструйтесь що б відкрити основні функції бота, як Водій або Менеджер."
		msg.ReplyMarkup = welcomeMenuMarkup
	} else {
		return HandleMenu(chatId, globalStorage, user)
	}

	msg.ParseMode = tgbotapi.ModeHTML
	_, err := Bot.Send(msg)
	return err
}

func HandleMenu(chatId int64, globalStorage *sql.DB, u *db.User) error {

	var err error
	var role db.Role
	msg := tgbotapi.NewMessage(chatId, "Ласкаво просимо до допоміжного бота V&R Spedition.")
	if u == nil {
		u = new(db.User)
		u.ChatId = chatId

		err = u.GetUserByChatId(globalStorage)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("Error getting the user: %v\n", err)
			}
			role = db.NoRole
		}
	}

	if u.ManagerId != uuid.Nil {
		role = db.RoleManager
	}

	if u.DriverId != uuid.Nil {
		role = db.RoleDriver
	}

	switch role {
	case db.NoRole:
		return HandleStart(chatId, globalStorage, nil)
	case db.RoleDriver:
		driverSessionsMu.Lock()
		driver, exists := driverSessions[chatId]
		if !exists {
			driver, err = db.GetDriverByChatId(globalStorage, chatId)
			if err != nil {
				driverSessionsMu.Unlock()
				return fmt.Errorf("Err loading driver: %v\n", err)
			}
			driver.User = u
		}

		driverSessionsMu.Unlock()

		if driver.State != db.StatePause {
			msg.ReplyMarkup = driverStartMarkupWorking
		} else {
			msg.ReplyMarkup = driverStartMarkupPause
		}
		msg.Text = fmt.Sprintf("Ласкаво просимо, водію %s\nЩо ви хочете зробити?", u.Name)
	case db.RoleManager:
		managerSessionsMu.Lock()
		if manager, exists := managerSessions[chatId]; exists {
			manager.State = db.StateDormantManager
		} else {
			manager, err = db.GetManagerByChatId(globalStorage, chatId)
			if err != nil {
				managerSessionsMu.Unlock()
				return fmt.Errorf("Err loading manager: %v\n", err)
			}
			manager.User = u
		}
		managerSessionsMu.Unlock()

		msg.Text = fmt.Sprintf("Ласкаво просимо, менеджере %s\nЩо ви хочете зробити?", u.Name)
		msg.ReplyMarkup = managerStartMarkup
	}

	msg.ParseMode = tgbotapi.ModeHTML
	_, err = Bot.Send(msg)
	return err
}
