package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"logistictbot/docs"
	"logistictbot/tracking"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	tasks = make(map[int64]int64)

	now time.Time
)

func Check(err error, print bool, message ...string) bool {
	if err != nil {
		if print {
			log.Fatalf("ERR: %s: %v\n", strings.Join(message, "; "), err)
		}
		return true
	}
	return false
}

func ReceiveUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel, globalStorage *sql.DB) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			if err := HandleUpdate(update, globalStorage); err != nil {
				log.Println("ERR: ", err)
			}
		}
	}

}

func HandleUpdate(update tgbotapi.Update, globalStorage *sql.DB) error {
	var err error

	switch {
	case update.Message != nil:

		LogTelegramMessage(update.Message)
		return HandleMessage(update.Message, globalStorage)
	case update.EditedMessage != nil:
		if update.EditedMessage.Location != nil {
			loc := update.EditedMessage.Location

			log.Println("2:", loc.LivePeriod)

			trackingSessionsMutex.Lock()
			if session, exists := trackingSessions[update.EditedMessage.Chat.ID]; exists {
				session.LiveLocationMsgID = update.EditedMessage.MessageID
				err = session.UpdateLocation(loc.Latitude, loc.Longitude, loc.LivePeriod, Bot)
				if err != nil {
					if !errors.Is(err, tracking.ErrNotLiveLocation) {
						trackingSessionsMutex.Unlock()
						return fmt.Errorf("ERR: parsing location from an edited message: %v\n", err)
					}
					log.Println("ERR: ", tracking.ErrNotLiveLocation.Error())
				}
				trackingSessionsMutex.Unlock()
				return nil
			}
			trackingSessionsMutex.Unlock()
		}

	case update.CallbackQuery != nil:
		return HandleCallbackQuery(update.CallbackQuery, globalStorage)
	default:
		err = fmt.Errorf("wrong type of update")
	}

	return err

}

func HandleMessage(msg *tgbotapi.Message, globalStorage *sql.DB) (err error) {
	id := msg.MessageID
	user := msg.From
	text := msg.Text

	if user == nil {
		return fmt.Errorf("How is user not there?: %v\n", user)
	}

	log.Printf("%s(%d - %s - %d) wrote %s. msg id: %d", user.FirstName, user.ID, user.LanguageCode, msg.Chat.ID, text, id)

	inputMu.Lock()
	state, isWaitingForInput := waitingForInput[msg.From.ID]
	inputMu.Unlock()

	managerSessionsMu.Lock()
	managerSesh, isManagerSesh := managerSessions[msg.From.ID]
	managerSessionsMu.Unlock()

	driverSessionsMu.Lock()
	driverSesh, isDriverSesh := driverSessions[msg.From.ID]
	driverSessionsMu.Unlock()

	devSessionMu.Lock()
	devSesh, isDev := devSession[msg.From.ID]
	devSessionMu.Unlock()

	if isDriverSesh {
		driverSesh, err = HandleDriverInputState(driverSesh, msg, globalStorage)
	}

	if isManagerSesh {
		managerSesh, err = HandleManagerInputState(managerSesh, msg, globalStorage)
	}

	if isWaitingForInput {
		return HandleFormInput(msg.From.ID, msg.Text, state, globalStorage, user)
	}

	if isDev {
		switch {
		case msg.Document != nil && msg.Document.MimeType == string(docs.MimeTextCSV):
			err = HandleCleaningDevCSV(devSesh.ChatId, msg.Document, globalStorage)
			return err
		}
	}

	if strings.HasPrefix(msg.Text, "/") {
		err = HandleCommand(msg.Chat.ID, msg.Text, globalStorage)
	}

	return err
}

func RegisterFormMessage(chatId int64, fields map[string]string, markup tgbotapi.InlineKeyboardMarkup, messageText string) (formMsg tgbotapi.Message, err error) {
	var resultFields string
	for question, answer := range fields {
		resultFields += fmt.Sprintf("%s: \"%s\"\n", question, answer)
	}
	msg := tgbotapi.NewMessage(chatId, messageText+"\n\n"+resultFields)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = markup

	return Bot.Send(msg)
}
