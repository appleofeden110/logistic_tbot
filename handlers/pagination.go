package handlers

import (
	"database/sql"
	"fmt"
	"logistictbot/config"
	"logistictbot/parser"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const ShipmentsPerPage = 5

func FormatShipmentForList(s *parser.Shipment, index int) string {
	status := "🔴 Не почато"
	if !s.Started.IsZero() && s.Finished.IsZero() {
		status = "🟡 В процесі"
	} else if !s.Finished.IsZero() {
		status = "🟢 Завершено"
	}

	return fmt.Sprintf(
		"%d. %s\n"+
			"   🚛 Авто: %s\n"+
			"   📦 Контейнер: %s\n"+
			"   📋 Завдань: %d\n",
		index+1,
		status,
		s.CarId,
		s.Container,
		len(s.Tasks),
	)
}

func CreateShipmentListMessage(shipments []*parser.Shipment, page int, chatId int64, callbackPrefix string) (tgbotapi.MessageConfig, error) {
	totalPages := (len(shipments) + ShipmentsPerPage - 1) / ShipmentsPerPage

	if page < 0 {
		page = 0
	}
	if page >= totalPages && totalPages > 0 {
		page = totalPages - 1
	}

	start := page * ShipmentsPerPage
	end := start + ShipmentsPerPage
	if end > len(shipments) {
		end = len(shipments)
	}

	var messageText strings.Builder
	messageText.WriteString(fmt.Sprintf("📋 Маршрути (сторінка %d/%d)\n", page+1, totalPages))
	messageText.WriteString(fmt.Sprintf("Всього маршрутів: %d\n\n", len(shipments)))

	if len(shipments) == 0 {
		messageText.WriteString("Немає маршрутів для відображення.")
	} else {
		for i := start; i < end; i++ {
			messageText.WriteString(FormatShipmentForList(shipments[i], i))
			messageText.WriteString("\n")
		}
	}

	msg := tgbotapi.NewMessage(chatId, messageText.String())

	if len(shipments) > 0 {
		var rows [][]tgbotapi.InlineKeyboardButton

		for i := start; i < end; i++ {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("Деталі маршруту #%d", i+1),
					fmt.Sprintf("shipment:details:%d", shipments[i].ShipmentId),
				),
			))
		}

		var navButtons []tgbotapi.InlineKeyboardButton

		if page > 0 {
			navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData(
				"⬅️ Назад",
				fmt.Sprintf("%s:%d", callbackPrefix, page-1),
			))
		}

		if page < totalPages-1 {
			navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData(
				"Вперед ➡️",
				fmt.Sprintf("%s:%d", callbackPrefix, page+1),
			))
		}

		if len(navButtons) > 0 {
			rows = append(rows, navButtons)
		}

		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	}

	return msg, nil
}

func HandlePaginationCommands(chatId int64, command string, msgId int, globalStorage *sql.DB) error {
	cmd, f := strings.CutPrefix(command, "page:")
	if !f {
		config.VERY_BAD(chatId, Bot)
	}
	switch {
	case strings.Contains(cmd, "viewall:"):
		parts := strings.Split(cmd, ":")
		if len(parts) < 2 {
			return fmt.Errorf("invalid pagination callback data")
		}

		page, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid page number: %v", err)
		}

		var shipments []*parser.Shipment
		var callbackPrefix string

		shipments, err = parser.GetAllShipments(globalStorage)
		callbackPrefix = "page:viewall"
		if err != nil {
			return fmt.Errorf("Err getting shipments: %v", err)
		}

		msg, err := CreateShipmentListMessage(shipments, page, chatId, callbackPrefix)
		if err != nil {
			return fmt.Errorf("Err creating message: %v", err)
		}

		edit := tgbotapi.NewEditMessageText(chatId, msgId, msg.Text)
		keyboard := msg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
		edit.ReplyMarkup = &keyboard

		_, err = Bot.Send(edit)
		if err != nil {
			return fmt.Errorf("Err editing message: %v", err)
		}
	case strings.Contains(cmd, "viewallbycar:"):

	}
	return nil
}
