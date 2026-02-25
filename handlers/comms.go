package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"logistictbot/config"
	"logistictbot/db"
	"logistictbot/docs"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofrs/uuid"
)

type CommunicationMsg struct {
	Id             int64
	Receiver       *db.User
	Sender         *db.User
	MessageContent string
	ReplyContent   string
	CreatedAt      time.Time
	RepliedAt      time.Time
}

const tickRate = 2 * time.Minute

func PingNonReplies(globalStorage *sql.DB) {
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	for range ticker.C {
		messages, err := GetAllNonRepliedMessages(globalStorage)
		if err != nil {
			log.Printf("ERR: getting non-replied messages: %v\n", err)
			continue
		}

		for _, comms := range messages {
			if time.Since(comms.CreatedAt) > tickRate {
				msg := tgbotapi.NewMessage(
					comms.Receiver.ChatId,
					config.Translate(config.GetLang(comms.Receiver.ChatId), "comms:notification",
						comms.Sender.Name,
						comms.Sender.TgTag,
						comms.MessageContent,
					),
				)
				msg.ParseMode = tgbotapi.ModeHTML
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.GetLang(comms.Receiver.ChatId), "btn:reply"), "reply:"+strconv.Itoa(int(comms.Id))),
					),
				)

				_, err := Bot.Send(msg)
				if err != nil {
					log.Printf("ERR: sending reminder for message %d: %v\n", comms.Id, err)
				} else {
					log.Printf("Sent reminder for message %d to user %d\n", comms.Id, comms.Receiver.ChatId)
				}
			}
		}
	}
}

// GetCommsMessage just needs the id
func (comms *CommunicationMsg) GetCommsMessage(globalStorage *sql.DB) error {
	query := `
		SELECT
			cm.id,
			cm.reciever_id,
			cm.sender_id,
			cm.message_content,
			cm.reply_content,
			cm.created_at,
			cm.replied_at
		FROM communication_messages cm
		WHERE cm.id = ?
	`

	var receiverID, senderID sql.NullString
	var replyContent sql.NullString
	var repliedAt sql.NullTime

	err := globalStorage.QueryRow(query, comms.Id).Scan(
		&comms.Id,
		&receiverID,
		&senderID,
		&comms.MessageContent,
		&replyContent,
		&comms.CreatedAt,
		&repliedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("ERR: communication message with id %d not found", comms.Id)
		}
		return fmt.Errorf("ERR: fetching communication message: %w", err)
	}

	if replyContent.Valid {
		comms.ReplyContent = replyContent.String
	}

	if repliedAt.Valid {
		comms.RepliedAt = repliedAt.Time
	}

	if receiverID.Valid && receiverID.String != "" {
		receiverUUID, err := uuid.FromString(receiverID.String)
		if err != nil {
			return fmt.Errorf("ERR: parsing receiver_id: %w", err)
		}

		comms.Receiver = &db.User{Id: receiverUUID}
		if err := comms.Receiver.GetUserById(globalStorage); err != nil {
			return fmt.Errorf("ERR: fetching receiver user: %w", err)
		}
	}

	if senderID.Valid && senderID.String != "" {
		senderUUID, err := uuid.FromString(senderID.String)
		if err != nil {
			return fmt.Errorf("ERR: parsing sender_id: %w", err)
		}

		comms.Sender = &db.User{Id: senderUUID}
		if err := comms.Sender.GetUserById(globalStorage); err != nil {
			return fmt.Errorf("ERR: fetching sender user: %w", err)
		}
	}

	return nil
}

func SendWithCommsAndChat(globalStorage *sql.DB, msgId int64, chatId int64) error {

	receiver := &db.User{ChatId: chatId}
	err := receiver.GetUserByChatId(globalStorage)
	if err != nil {
		return fmt.Errorf("ERR: getting receiver by chat id: %v\n", err)
	}

	comms := &CommunicationMsg{Id: msgId, Receiver: receiver}
	err = comms.GetCommsMessage(globalStorage)
	if err != nil {
		return err
	}

	return comms.Send(globalStorage)
}

func GetAllNonRepliedMessages(globalStorage *sql.DB) ([]*CommunicationMsg, error) {
	query := `
		SELECT
			cm.id,
			cm.sender_id,
			cm.reciever_id,
			cm.message_content,
			cm.reply_content,
			cm.created_at,
			cm.replied_at
		FROM communication_messages cm
		WHERE cm.replied_at IS NULL
		AND cm.reciever_id IS NOT NULL
		ORDER BY cm.created_at DESC
	`
	rows, err := globalStorage.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ERR: querying non-replied messages: %v\n", err)
	}
	defer rows.Close()

	var messages []*CommunicationMsg
	for rows.Next() {
		var senderID, receiverID sql.NullString
		var replyContent sql.NullString
		var repliedAt sql.NullTime
		var msg CommunicationMsg

		err := rows.Scan(
			&msg.Id,
			&senderID,
			&receiverID,
			&msg.MessageContent,
			&replyContent,
			&msg.CreatedAt,
			&repliedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ERR: scanning message row: %v\n", err)
		}

		if senderID.Valid {
			senderUUID, err := uuid.FromString(senderID.String)
			if err != nil {
				return nil, fmt.Errorf("ERR: parsing sender_id: %v\n", err)
			}
			msg.Sender = &db.User{Id: senderUUID}
			err = msg.Sender.GetUserById(globalStorage)
			if err != nil {
				return nil, fmt.Errorf("ERR: getting sender user: %v\n", err)
			}
		}

		if receiverID.Valid {
			receiverUUID, err := uuid.FromString(receiverID.String)
			if err != nil {
				return nil, fmt.Errorf("ERR: parsing receiver_id: %v\n", err)
			}
			msg.Receiver = &db.User{Id: receiverUUID}
			err = msg.Receiver.GetUserById(globalStorage)
			if err != nil {
				return nil, fmt.Errorf("ERR: getting receiver user: %v\n", err)
			}
		}

		if replyContent.Valid {
			msg.ReplyContent = replyContent.String
		}
		if repliedAt.Valid {
			msg.RepliedAt = repliedAt.Time
		}

		messages = append(messages, &msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ERR: iterating message rows: %v\n", err)
	}

	return messages, nil
}

func GetNonRepliedMessagesByUserId(globalStorage *sql.DB, userId uuid.UUID) ([]*CommunicationMsg, error) {
	query := `
		SELECT
			cm.id,
			cm.sender_id,
			cm.reciever_id,
			cm.message_content,
			cm.reply_content,
			cm.created_at,
			cm.replied_at
		FROM communication_messages cm
		WHERE cm.reciever_id = ?
		AND cm.replied_at IS NULL
		ORDER BY cm.created_at DESC
	`

	rows, err := globalStorage.Query(query, userId.String())
	if err != nil {
		return nil, fmt.Errorf("ERR: querying non-replied messages for user: %v\n", err)
	}
	defer rows.Close()

	var messages []*CommunicationMsg
	for rows.Next() {
		var senderID, receiverID sql.NullString
		var replyContent sql.NullString
		var repliedAt sql.NullTime
		var msg CommunicationMsg

		err := rows.Scan(
			&msg.Id,
			&senderID,
			&receiverID,
			&msg.MessageContent,
			&replyContent,
			&msg.CreatedAt,
			&repliedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ERR: scanning message row: %v\n", err)
		}

		// Parse sender ID
		if senderID.Valid {
			senderUUID, err := uuid.FromString(senderID.String)
			if err != nil {
				return nil, fmt.Errorf("ERR: parsing sender_id: %v\n", err)
			}
			msg.Sender = &db.User{Id: senderUUID}
			err = msg.Sender.GetUserById(globalStorage)
			if err != nil {
				return nil, fmt.Errorf("ERR: getting sender user: %v\n", err)
			}
		}

		// Parse receiver ID
		if receiverID.Valid {
			receiverUUID, err := uuid.FromString(receiverID.String)
			if err != nil {
				return nil, fmt.Errorf("ERR: parsing receiver_id: %v\n", err)
			}
			msg.Receiver = &db.User{Id: receiverUUID}
			err = msg.Receiver.GetUserById(globalStorage)
			if err != nil {
				return nil, fmt.Errorf("ERR: getting receiver user: %v\n", err)
			}
		}

		// Handle nullable fields
		if replyContent.Valid {
			msg.ReplyContent = replyContent.String
		}
		if repliedAt.Valid {
			msg.RepliedAt = repliedAt.Time
		}

		messages = append(messages, &msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ERR: iterating message rows: %v\n", err)
	}

	return messages, nil
}

func (comms *CommunicationMsg) Send(globalStorage *sql.DB) error {

	msg := tgbotapi.NewMessage(comms.Receiver.ChatId, config.Translate(config.GetLang(comms.Receiver.ChatId), "comms:newmsg", comms.Sender.Name, comms.Sender.TgTag, comms.MessageContent))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.GetLang(comms.Receiver.ChatId), "btn:reply"), "reply:"+strconv.Itoa(int(comms.Id)))))

	_, err := globalStorage.Exec(
		`UPDATE communication_messages
		SET reciever_id = ? WHERE id = ?`,
		comms.Receiver.Id.String(),
		comms.Id,
	)
	if err != nil {
		return fmt.Errorf("ERR: inserting into communication messages: %v\n", err)
	}

	nonRepliedMessagesMu.Lock()
	nonRepliedMessages[comms.Receiver.Id] = comms
	nonRepliedMessagesMu.Unlock()

	_, err = Bot.Send(msg)
	if err != nil {
		return err
	}

	_, err = Bot.Send(tgbotapi.NewMessage(comms.Sender.ChatId, config.Translate(config.GetLang(comms.Sender.ChatId), "comms:success_msg")+comms.Receiver.Name))
	return err
}

func (comms *CommunicationMsg) Reply(globalStorage *sql.DB) error {
	msg := tgbotapi.NewMessage(comms.Sender.ChatId, config.Translate(config.GetLang(comms.Receiver.ChatId), "comms:reply", comms.Receiver.Name, comms.Receiver.TgTag, comms.ReplyContent))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.GetLang(comms.Receiver.ChatId), "btn:writeback"), "writeback:"+strconv.Itoa(int(comms.Receiver.ChatId)))))

	_, err := globalStorage.Exec(
		`UPDATE communication_messages
		SET replied_at = CURRENT_TIMESTAMP, reply_content = ? WHERE id = ?`,
		comms.ReplyContent,
		comms.Id,
	)
	if err != nil {
		return fmt.Errorf("ERR: inserting into communication messages: %v\n", err)
	}

	nonRepliedMessagesMu.Lock()
	delete(nonRepliedMessages, comms.Receiver.Id)
	nonRepliedMessagesMu.Unlock()

	_, err = Bot.Send(msg)
	if err != nil {
		return err
	}

	_, err = Bot.Send(tgbotapi.NewMessage(comms.Receiver.ChatId, config.Translate(config.GetLang(comms.Sender.ChatId), "comms:success_reply")+comms.Sender.Name))
	return err
}

func (comms *CommunicationMsg) CreateMessage(text string, globalStorage *sql.DB) error {
	if !comms.Sender.DriverId.IsNil() {
		return comms.createDriverMessage(text, globalStorage)
	} else if !comms.Sender.ManagerId.IsNil() {
		return comms.createManagerMessage(text, globalStorage)
	}

	return fmt.Errorf("ERR: User has both manager's and driver's ids nil (user id: %v)\n", comms.Sender.Id)
}

func (comms *CommunicationMsg) createManagerMessage(text string, globalStorage *sql.DB) error {
	m, err := db.GetManagerByChatId(globalStorage, comms.Sender.ChatId)
	if err != nil {
		return fmt.Errorf("ERR: getting manager by chat id: %v\n", err)
	}

	writingToChatMapMu.RLock()
	receiverChatId, hasReceiver := writingToChatMap[comms.Sender.ChatId]
	writingToChatMapMu.RUnlock()

	if hasReceiver {
		receiver := &db.User{ChatId: receiverChatId}
		err = receiver.GetUserByChatId(globalStorage)
		if err != nil {
			return fmt.Errorf("ERR: getting receiver by chat id: %v\n", err)
		}
		comms.Receiver = receiver

		res, err := globalStorage.Exec(
			`INSERT INTO communication_messages
			(sender_id, reciever_id, message_content) VALUES (?, ?, ?)`,
			comms.Sender.Id.String(),
			receiver.Id.String(),
			text,
		)
		if err != nil {
			return fmt.Errorf("ERR: inserting into communication messages: %v\n", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("ERR: getting last insert id: %v\n", err)
		}

		comms.Id = id
		comms.MessageContent = text

		writingToChatMapMu.Lock()
		delete(writingToChatMap, comms.Sender.ChatId)
		writingToChatMapMu.Unlock()

		m.State = db.StateDormantManager
		err = m.ChangeManagerStatus(globalStorage)
		if err != nil {
			return fmt.Errorf("ERR: resetting manager state: %v\n", err)
		}

		return comms.Send(globalStorage)
	}

	res, err := globalStorage.Exec(
		`INSERT INTO communication_messages
		(sender_id, message_content) VALUES(?, ?)`,
		comms.Sender.Id.String(),
		text,
	)
	if err != nil {
		return fmt.Errorf("ERR: inserting into communication messages: %v\n", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("ERR: getting last insert id: %v\n", err)
	}

	return m.ShowDriverList(globalStorage, "senddrivermsg:"+strconv.Itoa(int(id)), config.Translate(config.GetLang(m.ChatId), "select_receiver"), m.ChatId, Bot)
}

func (comms *CommunicationMsg) createDriverMessage(text string, globalStorage *sql.DB) error {
	d, err := db.GetDriverByChatId(globalStorage, comms.Sender.ChatId)
	if err != nil {
		return fmt.Errorf("ERR: getting driver by chat id: %v\n", err)
	}

	writingToChatMapMu.RLock()
	receiverChatId, hasReceiver := writingToChatMap[comms.Sender.ChatId]
	writingToChatMapMu.RUnlock()

	if hasReceiver {
		receiver := &db.User{ChatId: receiverChatId}
		err = receiver.GetUserByChatId(globalStorage)
		if err != nil {
			return fmt.Errorf("ERR: getting receiver by chat id: %v\n", err)
		}
		comms.Receiver = receiver

		log.Println(text, receiver)
		res, err := globalStorage.Exec(
			`INSERT INTO communication_messages
			(sender_id, reciever_id, message_content) VALUES (?, ?, ?)`,
			comms.Sender.Id.String(),
			receiver.Id.String(),
			text,
		)
		if err != nil {
			return fmt.Errorf("ERR: inserting into communication messages: %v\n", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("ERR: getting last insert id: %v\n", err)
		}

		comms.Id = id
		comms.MessageContent = text

		writingToChatMapMu.Lock()
		delete(writingToChatMap, comms.Sender.ChatId)
		writingToChatMapMu.Unlock()

		d.State = db.StateWorking
		err = d.ChangeDriverStatus(globalStorage)
		if err != nil {
			return fmt.Errorf("ERR: resetting driver state: %v\n", err)
		}

		return comms.Send(globalStorage)
	}

	res, err := globalStorage.Exec(
		`INSERT INTO communication_messages
		(sender_id, message_content) VALUES (?, ?)`,
		comms.Sender.Id.String(),
		text,
	)
	if err != nil {
		return fmt.Errorf("ERR: inserting into communication messages: %v\n", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("ERR: getting last insert id: %v\n", err)
	}

	return d.ShowManagerList(globalStorage, "sendmanagermsg:"+strconv.Itoa(int(id)), config.Translate(config.GetLang(d.ChatId), "select_receiver"), d.ChatId, Bot)
}

func getSessionAndSetWritingState(chatId int64, commsId int64, globalStorage *sql.DB) error {
	user := &db.User{ChatId: chatId}
	err := user.GetUserByChatId(globalStorage)
	if err != nil {
		return fmt.Errorf("ERR: getting user by chat id: %v\n", err)
	}

	isM, err := user.IsManager(globalStorage)
	if err != nil {
		return fmt.Errorf("ERR: couldn't find if the guy is the manager or the driver: %v\n", err)
	}

	if isM {
		managerSessionsMu.Lock()
		manager, f := managerSessions[chatId]
		managerSessionsMu.Unlock()

		if f {
			manager.State = db.StateWritingToDriver
			return manager.ChangeManagerStatus(globalStorage)
		}
		return fmt.Errorf("ERR: couldn't find the manager with this chatid: %v\n", chatId)

	}
	driverSessionsMu.Lock()
	driver, f := driverSessions[chatId]
	driverSessionsMu.Unlock()

	if f {
		driver.State = db.StateWritingToManager
		return driver.ChangeDriverStatus(globalStorage)
	}
	return fmt.Errorf("Couldn't find the driver with this id: %v\n", chatId)
}

func CreateVideoToSend(chatId int64, videoName string) *tgbotapi.VideoConfig {
	videoFP := tgbotapi.FilePath("./videos/" + videoName)

	video := tgbotapi.NewVideo(chatId, videoFP)
	return &video
}

func sendDocumentsToManager(
	chatID int64,
	docsFiles []*docs.File,
) error {

	for _, f := range docsFiles {
		doc := tgbotapi.NewDocument(chatID, tgbotapi.FileID(f.TgFileId))
		doc.Caption = f.OriginalName

		if _, err := Bot.Send(doc); err != nil {
			return fmt.Errorf("send document: %w", err)
		}
	}

	return nil
}

func sendPhotosToManager(
	chatID int64,
	photos []*docs.File,
	caption string,
) error {

	const maxGroupSize = 10

	for i := 0; i < len(photos); i += maxGroupSize {
		end := i + maxGroupSize
		if end > len(photos) {
			end = len(photos)
		}

		group := tgbotapi.MediaGroupConfig{
			ChatID: chatID,
		}

		for j, f := range photos[i:end] {
			photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FileID(f.TgFileId))
			// Caption only on first photo of first group
			if i == 0 && j == 0 {
				photo.Caption = caption
				photo.ParseMode = tgbotapi.ModeHTML
			}

			group.Media = append(group.Media, photo)
		}

		if _, err := Bot.SendMediaGroup(group); err != nil {
			return fmt.Errorf("send photo group: %w", err)
		}
	}

	return nil
}

func splitFiles(files []*docs.File) (photos, docsFiles []*docs.File) {
	for _, f := range files {
		switch f.Filetype {
		case docs.Image:
			photos = append(photos, f)
		case docs.Document:
			docsFiles = append(docsFiles, f)
		}
	}
	return
}
