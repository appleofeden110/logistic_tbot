package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"logistictbot/config"
	data_analysis "logistictbot/data-analysis"
	"logistictbot/db"
	"logistictbot/docs"
	"logistictbot/parser"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var TestManagerChatId int64 = 2042374598

func HandleCallbackQuery(cbq *tgbotapi.CallbackQuery, globalStorage *sql.DB) error {

	var err error
	id := cbq.Message.MessageID

	user := &db.User{ChatId: cbq.Message.From.ID}
	err = user.GetUserByChatId(globalStorage)
	if err != nil {
		user.Name = "NEU"
		user.TgTag = "@nil"
	}

	log.Printf("(%d - %s - %s) pressed a button %s. msg id: %d", cbq.Message.From.ID, user.Name, user.TgTag, cbq.Data, id)

	switch {
	case strings.HasPrefix(cbq.Data, "mstmt:"):
		after, _ := strings.CutPrefix(cbq.Data, "mstmt:")
		m, y, found := strings.Cut(after, ".")
		if !found {
			log.Printf("Місяця немає тут: %s\n", cbq.Data)
			return fmt.Errorf("invalid month format")
		}
		month, _ := strconv.Atoi(m)
		year, _ := strconv.Atoi(y)

		filename, err := data_analysis.CreateMonthlyStatement(time.Month(month), year, globalStorage)
		if err != nil {
			return fmt.Errorf("error creating statement: %v", err)
		}

		Bot.Send(tgbotapi.NewDocument(cbq.Message.Chat.ID, tgbotapi.FilePath(filename)))

	case strings.HasPrefix(cbq.Data, "driver:"):
		return HandleDriverCommands(cbq.Message.Chat.ID, cbq.Data, cbq.Message.MessageID, globalStorage)
	case strings.HasPrefix(cbq.Data, "manager:"):
		return HandleManagerCommands(cbq.Message.Chat.ID, cbq.Data, cbq.Message.MessageID, globalStorage)
	case strings.HasPrefix(cbq.Data, "sa:"):
		return HandleSACommands(cbq.Message.Chat.ID, cbq.Data, cbq.Message.MessageID, globalStorage)
	case strings.HasPrefix(cbq.Data, "dev:"):
		return HandleDevCommands(cbq.Message.Chat.ID, cbq.Data, cbq.Message.MessageID, globalStorage)
	case strings.HasPrefix(cbq.Data, "page:"):
		return HandlePaginationCommands(cbq.Message.Chat.ID, cbq.Data, cbq.Message.MessageID, globalStorage)
	case strings.HasPrefix(cbq.Data, "shipment:details:"):
		shipmentIdString, f := strings.CutPrefix(cbq.Data, "shipment:details:")
		if !f {
			fmt.Printf("Error: there is no shipment id when it should be in here: %s (chatId: %d)\n", cbq.Data, cbq.Message.Chat.ID)
			config.VERY_BAD(cbq.Message.Chat.ID, Bot)
		}

		shipmentId, err := strconv.ParseInt(shipmentIdString, 10, 64)
		if err != nil {
			return fmt.Errorf("Error: parsing shipment id (og str: %s) was not successful: %v\n", shipmentIdString, err)
		}
		return HandleShipmentDetails(cbq.Message.Chat.ID, shipmentId, globalStorage)
	case strings.HasPrefix(cbq.Data, "startform:"):
		after, found := strings.CutPrefix(cbq.Data, "startform:")
		var whichTable db.TableType
		if found {
			whichTable = db.TableType(after)
		}

		f := db.Form{ChatID: cbq.Message.Chat.ID, FormMsgId: cbq.Message.MessageID, FormMsgText: cbq.Message.Text, WhichTable: whichTable}
		return GatherInfo(f)
	case strings.HasPrefix(cbq.Data, "editform:"):
		after, found := strings.CutPrefix(cbq.Data, "editform:")
		var whichTable db.TableType
		if found {
			whichTable = db.TableType(after)
		}

		f := db.Form{ChatID: cbq.Message.Chat.ID, FormMsgId: cbq.Message.MessageID, FormMsgText: cbq.Message.Text, WhichTable: whichTable}
		return GatherInfo(f)
	case strings.HasPrefix(cbq.Data, "acceptform:"):
		inputMu.Lock()
		inputSesh, exists := waitingForInput[cbq.Message.Chat.ID]
		inputMu.Unlock()
		if !exists {
			panic("the one that should, does not exist")
		}

		inputSesh.Finished = true
		_, err = Bot.Send(tgbotapi.NewMessage(cbq.Message.Chat.ID, "Опрацювання..."))
		if err != nil {
			return fmt.Errorf("Err sending acceptform message to a user: %v\n", err)
		}
		return finishForm(cbq.Message.Chat.ID, inputSesh, globalStorage, cbq.Message.From)

	case strings.HasPrefix(cbq.Data, "createform:"):
		err = HandleCommand(cbq.Message.Chat.ID, fmt.Sprintf("/%s", cbq.Data), globalStorage)
	case strings.HasPrefix(cbq.Data, "shipment:accept:"):

		driverSessionsMu.Lock()
		driver, exists := driverSessions[cbq.Message.Chat.ID]
		driverSessionsMu.Unlock()

		if !exists {
			msg := tgbotapi.NewMessage(cbq.Message.Chat.ID, "Ви не водій в системі. Cкоріше всього щось не так з ботом. Якщо нічого не працюватиме - напишіть менеджеру або розробнику (@NazKan_Uk | @pinkfloydfan). \n\nЯкщо ви не зареєстровані зробіть це через команду /start, це скоріш всього вирішить проблему")
			msg.ParseMode = tgbotapi.ModeHTML
			Bot.Send(msg)
			panic("Something went very wrong, driver action triggered with button, but no driver present for this chat_id: " + strconv.Itoa(int(cbq.Message.Chat.ID)))
		}

		if driver.State == db.StatePause {
			_, err = Bot.Send(tgbotapi.NewMessage(cbq.Message.Chat.ID, "Ви повинні почати зміну перед будь якими діями. Можете зробити через меню (команди /start або /menu)"))
			return err
		}

		shipmentIdString, _ := strings.CutPrefix(cbq.Data, "shipment:accept:")
		var shipmentId int64

		if shipmentId, err = strconv.ParseInt(shipmentIdString, 10, 64); err != nil {
			return fmt.Errorf("err parsing shipment id: %v\n", err)
		}

		shipment, err := parser.GetShipment(globalStorage, shipmentId)
		if err != nil {
			return fmt.Errorf("err getting shipment by id: %v\n", err)
		}

		if !shipment.Started.IsZero() {
			_, err = Bot.Send(tgbotapi.NewMessage(cbq.Message.Chat.ID, "Цей маршрут вже почався"))
			return err
		}

		err = shipment.StartShipment(globalStorage)
		if err != nil {
			return fmt.Errorf("err starting shipment: %v\n", err)
		}

		shipment.Tasks, err = parser.GetAllTasksByShipmentId(globalStorage, shipmentId)
		if err != nil {
			return fmt.Errorf("err getting all the tasks for the shipment: %v\n", err)
		}

		msg := tgbotapi.NewMessage(cbq.Message.Chat.ID, "Початок маршруту: "+shipment.Started.Format("02/01/2006 15:04"))
		markup := make([][]tgbotapi.InlineKeyboardButton, 0)
		markup = append(markup, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Закінчити маршрут", "shipment:end:"+shipmentIdString)))

		for _, task := range shipment.Tasks {
			msg.Text += fmt.Sprintf("\n\n<i><b>- Завдання: %s</b></i>\n", task.Type)
			msg.Text += parser.ReadTaskShort(task)
			markup = append(markup, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Почати "+task.Type, "driver:begintask:"+strconv.Itoa(task.Id))))
		}
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(markup...)

		var pinMsg tgbotapi.Message
		pinMsg, err = Bot.Send(msg)
		if err != nil {
			return err
		}

		_, err = Bot.Send(tgbotapi.PinChatMessageConfig{
			ChatID:              pinMsg.Chat.ID,
			MessageID:           pinMsg.MessageID,
			DisableNotification: true,
		})
		return err
	case strings.HasPrefix(cbq.Data, "shipment:end:"):
		shipmentIdString, _ := strings.CutPrefix(cbq.Data, "shipment:end:")
		var shipmentId int64

		if shipmentId, err = strconv.ParseInt(shipmentIdString, 10, 64); err != nil {
			return fmt.Errorf("err parsing shipment id: %v\n", err)
		}

		shipment, err := parser.GetShipment(globalStorage, shipmentId)
		if err != nil {
			return fmt.Errorf("err getting shipment by id: %v\n", err)
		}
		log.Println(shipment.Started)

		if shipment.Started.IsZero() {
			_, err = Bot.Send(tgbotapi.NewMessage(cbq.Message.Chat.ID, "Цей маршрут ще геть не почався, неможливо закінчити"))
			return err
		}

		if !shipment.Finished.IsZero() {
			_, err = Bot.Send(tgbotapi.NewMessage(cbq.Message.Chat.ID, "Цей маршрут вже був закінчений"))
			return err
		}

		err = shipment.FinishShipment(globalStorage)
		if err != nil {
			return fmt.Errorf("err starting shipment: %v\n", err)
		}

		_, err = Bot.Send(tgbotapi.NewMessage(cbq.Message.Chat.ID, fmt.Sprintf("Маршрут %d було закінчено!", shipmentId)))
		return err
	case strings.HasPrefix(cbq.Data, "task:begin:"):

	case strings.HasPrefix(cbq.Data, "readdoc:"):
		docIdString, f := strings.CutPrefix(cbq.Data, "readdoc:")
		if f {
			docId, err := strconv.Atoi(docIdString)
			if err != nil {
				return fmt.Errorf("err getting id from docIdstring: %v\n", err)
			}

			f := docs.File{Id: docId}
			err = f.GetFile(globalStorage)
			if err != nil {
				return fmt.Errorf("err getting a file to read: %v\n", err)
			}

			return parser.ReadDocAndSend(f.Path, cbq.Message.Chat.ID, Bot)
		}
	case strings.HasPrefix(cbq.Data, "task_data:"):

		suffix, found := strings.CutPrefix(cbq.Data, "task_data:")
		if !found {
			return fmt.Errorf("err founding task data: %s\n", suffix)
		}
		task, _, found := strings.Cut(suffix, ":")
		fmt.Println("task found? ", found)

		sections, err := parser.GetSequenceOfTasks("")
		if err != nil {
			return fmt.Errorf("Error sections: %v\n", err)
		}
		_, secRes := parser.ReadDoc(sections)

		msg := tgbotapi.NewMessage(cbq.Message.Chat.ID, "Чи правильні дані введяні для "+task+"?\n\n")
		var temp string
		for k, v := range secRes {
			if k == task {
				for _, line := range strings.Split(v, "\n") {
					temp += fmt.Sprintf("%s\n", line)
				}
			}
		}
		msg.ParseMode = tgbotapi.ModeHTML

		msg.Text += temp

		/*	trackingSessionsMutex.Lock()
				sesh, wasTracking := trackingSessions[cbq.Message.Chat.ID]
				trackingSessionsMutex.Unlock()

			var totalDistance string
			if wasTracking {
				totalDistance = fmt.Sprintf("\n\nПоточний кілометраж по маршруту: %.2f км", sesh.TotalDistance)
			}
			msg.Text += totalDistance
		*/
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Так", "yesend:"+task),
				tgbotapi.NewInlineKeyboardButtonData("Ні, змінити дані", "edit:"+task),
			),
		)
		Bot.Send(msg)

	case strings.HasPrefix(cbq.Data, "yesend:"):

		suffix, found := strings.CutPrefix(cbq.Data, "yesend:")
		if !found {
			return fmt.Errorf("err founding task data: %s\n", suffix)
		}
		task, _, found := strings.Cut(suffix, ":")
		fmt.Println("task found? ", found)

		sections, err := parser.GetSequenceOfTasks("")
		if err != nil {
			return fmt.Errorf("Error sections: %v\n", err)
		}
		_, secRes := parser.ReadDoc(sections)

		var temp string
		for k, v := range secRes {
			if k == task {
				for _, line := range strings.Split(v, "\n") {
					temp += fmt.Sprintf("%s\n", line)
				}
			}
		}
		//
		//trackingSessionsMutex.Lock()
		//sesh, wasTracking := trackingSessions[cbq.Message.Chat.ID]
		//trackingSessionsMutex.Unlock()
		//
		//var totalDistance string
		//if wasTracking {
		//	totalDistance = fmt.Sprintf("Поточний кілометраж по маршруту: %.2f км\n", sesh.TotalDistance)

		//}

		// to driver
		Bot.Send(tgbotapi.NewMessage(cbq.Message.Chat.ID, "Інфо відправлено Логісту Nazar Kaniuka"))
		// to manager
		Bot.Send(tgbotapi.NewMessage(tasks[cbq.Message.Chat.ID], fmt.Sprintf("Дані від Назар Канюка (790133 LU454TW) для задачі %s на завданні %d:\n\n%s\n\nЧас закінчення: %s;\nВсього тривалість: %s\n\n", task, sections.ShipmentId, temp, time.Now().Format("2006-01-02 15:04:05"), time.Since(now).String() /*, totalDistance*/)))
	case strings.HasPrefix(cbq.Data, "reply:"):
		commsIdStr, _ := strings.CutPrefix(cbq.Data, "reply:")
		commsId, err := strconv.Atoi(commsIdStr)
		if err != nil {
			return fmt.Errorf("ERR: getting comms id: %v\n", err)
		}

		comms := &CommunicationMsg{Id: int64(commsId)}
		err = comms.GetCommsMessage(globalStorage)
		if err != nil {
			return fmt.Errorf("ERR: getting comms message for a reply")
		}

		replyingToMessageMu.Lock()
		replyingToMessage[comms.Receiver.ChatId] = comms.Id
		replyingToMessageMu.Unlock()

		isM, err := comms.Receiver.IsManager(globalStorage)
		if err != nil {
			return fmt.Errorf("ERR: couldn't find if the guy is the manager or the driver: %v\n", err)
		}

		if isM {
			managerSessionsMu.Lock()
			manager, f := managerSessions[comms.Receiver.ChatId]
			managerSessionsMu.Unlock()

			if f {
				manager.State = db.StateReplyingDriver
				err = manager.ChangeManagerStatus(globalStorage)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("ERR: couldn't find the manager with this chatid: %v\n", comms.Receiver.ChatId)
			}
		} else {
			driverSessionsMu.Lock()
			driver, f := driverSessions[comms.Receiver.ChatId]
			driverSessionsMu.Unlock()

			if f {
				driver.State = db.StateReplyingManager
				err = driver.ChangeDriverStatus(globalStorage)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("Couldn't find the driver with this id: %v\n", comms.Receiver.ChatId)
			}
		}

		msg := tgbotapi.NewMessage(comms.Receiver.ChatId, "✏️ Напишіть <b>одним повідомленням</b> що ви хочете відправити")
		msg.ParseMode = tgbotapi.ModeHTML
		_, err = Bot.Send(msg)
		return err
	case strings.HasPrefix(cbq.Data, "writeback:"):
		chatIdStr, _ := strings.CutPrefix(cbq.Data, "writeback:")
		receiverChatId, err := strconv.ParseInt(chatIdStr, 10, 64)
		if err != nil {
			return fmt.Errorf("ERR: parsing int64 for chatid (%s): %v\n", chatIdStr, err)
		}

		log.Println("R: " + strconv.Itoa(int(receiverChatId)))

		writingToChatMapMu.Lock()
		writingToChatMap[cbq.Message.Chat.ID] = receiverChatId
		writingToChatMapMu.Unlock()

		err = getSessionAndSetWritingState(cbq.Message.Chat.ID, 0, globalStorage)
		if err != nil {
			return err
		}

		msg := tgbotapi.NewMessage(cbq.Message.Chat.ID, "✏️ Напишіть <b>одним повідомленням</b> що ви хочете відправити")
		msg.ParseMode = tgbotapi.ModeHTML
		Bot.Send(msg)

	case strings.HasPrefix(cbq.Data, "senddrivermsg:"):
		// after trimming - will contain both comms_msg_id and driver's chat_id
		msgIdChatId := strings.TrimPrefix(cbq.Data, "senddrivermsg:")

		msgIdStr, chatIdStr, _ := strings.Cut(msgIdChatId, ":")
		var msgId, chatId int64

		msgId, err = strconv.ParseInt(msgIdStr, 10, 64)
		chatId, err = strconv.ParseInt(chatIdStr, 10, 64)
		if err != nil {
			return fmt.Errorf("Error parsing communication msg id (%d) and chat id (%d) of the receiver: %v\n", msgId, chatId, err)
		}

		return SendWithCommsAndChat(globalStorage, msgId, chatId)
	case strings.HasPrefix(cbq.Data, "sendmanagermsg:"):
		// after trimming - will contain both comms_msg_id and manager's chat_id
		msgIdChatId := strings.TrimPrefix(cbq.Data, "sendmanagermsg:")

		msgIdStr, chatIdStr, _ := strings.Cut(msgIdChatId, ":")
		var msgId, chatId int64

		msgId, err = strconv.ParseInt(msgIdStr, 10, 64)
		chatId, err = strconv.ParseInt(chatIdStr, 10, 64)
		if err != nil {
			return fmt.Errorf("Error parsing communication msg id (%d) and chat id (%d) of the receiver: %v\n", msgId, chatId, err)
		}

		return SendWithCommsAndChat(globalStorage, msgId, chatId)
	case strings.HasPrefix(cbq.Data, "selectdriverfortask:"):
		driverChatIdStr := strings.TrimPrefix(cbq.Data, "selectdriverfortask:")
		driverChatId, _ := strconv.ParseInt(driverChatIdStr, 10, 64)

		managerSessionsMu.Lock()
		session, exists := managerSessions[cbq.Message.Chat.ID]
		managerSessionsMu.Unlock()

		if exists && session.State == db.StateWaitingDriver {
			pm := session.PendingMessage
			pm.ToChatId = driverChatId

			if err := pm.SendDocToDriver(globalStorage, Bot); err != nil {
				if strings.Contains(err.Error(), "UNIQUE constraint failed") {
					log.Printf("From %d to %d doc unique error: %v\n", pm.FromChatId, pm.ToChatId, err)
					session.State = db.StateDormantManager

					err = session.ChangeManagerStatus(globalStorage)
					if err != nil {
						return fmt.Errorf("err changing status from waiting driver to dormant_mng: %v\n", err)
					}
					return nil
				}
				return fmt.Errorf("Error sending document: %v\n", err)
			}

			session.State = db.StateDormantManager

			err = session.ChangeManagerStatus(globalStorage)
			if err != nil {
				return fmt.Errorf("err changing status from waiting driver to dormant_mng: %v\n", err)
			}
			msg := tgbotapi.NewMessage(cbq.Message.Chat.ID, "✅ Завдання відправлено водію!")
			_, err = Bot.Send(msg)
			return err
		}
	case strings.HasPrefix(cbq.Data, "video:"):
		videoData, f := strings.CutPrefix(cbq.Data, "video:")
		if f {
			video := CreateVideoToSend(cbq.Message.Chat.ID, videoData)
			video.Caption = "Туторіал відео по тому як відправляти Живу Локацію"

			// Send video first WITHOUT the button
			sentMsg, err := Bot.Send(video)
			if err != nil {
				return err
			}

			editMarkup := tgbotapi.NewEditMessageReplyMarkup(
				sentMsg.Chat.ID,
				sentMsg.MessageID,
				tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(
							"Видалити відео",
							fmt.Sprintf("deletevid:%d", sentMsg.MessageID),
						),
					),
				),
			)
			_, err = Bot.Send(editMarkup)

			return err
		}
		log.Println("Cannot find video name: ", cbq.Data)
		return nil
	case strings.HasPrefix(cbq.Data, "deletevid:"):
		messageToDel, f := strings.CutPrefix(cbq.Data, "deletevid:")
		if f {
			messageId, err := strconv.Atoi(messageToDel)
			if err != nil {
				log.Println("could not find the video to delete: ", cbq.Data)
				return nil
			}
			_, err = Bot.Send(tgbotapi.NewDeleteMessage(cbq.Message.Chat.ID, messageId))
			return err
		}
		log.Println("could not find the video to delete: ", cbq.Data)
		return nil
	case strings.HasPrefix(cbq.Data, "end_route:"):
		var shipment int64
		shipmentId, f := strings.CutPrefix(cbq.Data, "end_route:")
		if f {
			shipment, err = strconv.ParseInt(shipmentId, 0, 64)
			if err != nil {
				log.Println(err)
			}
		}

		trackingSessionsMutex.Lock()
		sesh, isTracking := trackingSessions[cbq.Message.Chat.ID]
		trackingSessionsMutex.Unlock()

		if isTracking {
			//sesh.Stop()
			log.Printf("Chat id: %d\n", cbq.Message.Chat.ID)

			endMsg := tgbotapi.NewMessage(cbq.Message.Chat.ID, "Маршрут закінченно, будь ласка виключте активний маячок")
			fmt.Println(cbq.Message.Chat.ID, sesh.LiveLocationMsgID)
			endMsg.ReplyToMessageID = sesh.LiveLocationMsgID
			//endMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("Маячок?", CreatePrivateMessageLink(cbq.Message.Chat.ID, sesh.LiveLocationMsgID))))
			// to driver
			Bot.Send(endMsg)
			// to manager
			Bot.Send(tgbotapi.NewMessage(tasks[cbq.Message.Chat.ID], fmt.Sprintf("Закінчувальний кілометраж для водія Назар Канюка по маршруту %d: %.2f км", shipment, sesh.TotalDistance)))

		} else {
			log.Println("ЩО ЗАКІНЧУВАТИ ТО?")
		}

	default:
		break
	}
	return err
}
