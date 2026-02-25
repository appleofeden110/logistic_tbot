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

const (
	ShipmentsPerPage = 5
	DriversPerPage   = 5
)

//func CreateDriversListMessage(drivers []*db.Driver, page int, chatId int64) (tgbotapi.MessageConfig, error) {
//	totalPages := (len(drivers) + DriversPerPage - 1) / DriversPerPage
//
//	if page < 0 {
//		page = 0
//	}
//	if page >= totalPages && totalPages > 0 {
//		page = totalPages - 1
//	}
//
//	start := page * DriversPerPage
//	end := start + DriversPerPage
//	if end > len(drivers) {
//		end = len(drivers)
//	}
//
//}

func FormatShipmentForList(s *parser.Shipment, index int, lang config.LangCode) string {
	status := config.Translate(lang, "shipment_notstarted")
	if !s.Started.IsZero() && s.Finished.IsZero() {
		status = config.Translate(lang, "shipment_active")
	} else if !s.Finished.IsZero() {
		status = config.Translate(lang, "shipment_done")
	}

	return config.Translate(lang,
		"shipment_format",
		index+1, s.ShipmentId,
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
	messageText.WriteString(config.Translate(config.GetLang(chatId), "shipment_view_header", page+1, totalPages))
	messageText.WriteString(config.Translate(config.GetLang(chatId), "shipment_view_total", len(shipments)))

	if len(shipments) == 0 {
		messageText.WriteString(config.Translate(config.GetLang(chatId), "shipment_view_noshipments"))
	} else {
		for i := start; i < end; i++ {
			messageText.WriteString(FormatShipmentForList(shipments[i], i, config.GetLang(chatId)))
			messageText.WriteString("\n")
		}
	}

	msg := tgbotapi.NewMessage(chatId, messageText.String())

	if len(shipments) > 0 {
		var rows [][]tgbotapi.InlineKeyboardButton

		for i := start; i < end; i++ {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					config.Translate(config.GetLang(chatId), "shipment_details", i+1),
					fmt.Sprintf("shipment:details:%d", shipments[i].ShipmentId),
				),
			))
		}

		var navButtons []tgbotapi.InlineKeyboardButton

		if page > 0 {
			navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData(
				config.Translate(config.GetLang(chatId), "page:prev"),
				fmt.Sprintf("%s:%d", callbackPrefix, page-1),
			))
		}

		if page < totalPages-1 {
			navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData(
				config.Translate(config.GetLang(chatId), "page:next"),
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
	case strings.Contains(cmd, "managers:"):
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
			return fmt.Errorf("ERR: getting shipments: %v", err)
		}

		msg, err := CreateShipmentListMessage(shipments, page, chatId, callbackPrefix)
		if err != nil {
			return fmt.Errorf("ERR: creating message: %v", err)
		}

		edit := tgbotapi.NewEditMessageText(chatId, msgId, msg.Text)
		keyboard := msg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
		edit.ReplyMarkup = &keyboard

		_, err = Bot.Send(edit)
		if err != nil {
			return fmt.Errorf("ERR: editing message: %v", err)
		}
	case strings.Contains(cmd, "viewallbycar:"):
		parts := strings.Split(cmd, ":")
		if len(parts) < 3 {
			return fmt.Errorf("invalid pagination callback data")
		}

		carId := parts[1]
		page, err := strconv.Atoi(parts[2])
		if err != nil {
			return fmt.Errorf("invalid page number: %v", err)
		}

		var shipments []*parser.Shipment
		var callbackPrefix string

		shipments, err = parser.GetAllShipmentsByCarId(carId, globalStorage)
		callbackPrefix = "page:viewallbycar:" + carId
		if err != nil {
			return fmt.Errorf("ERR: getting shipments: %v", err)
		}

		msg, err := CreateShipmentListMessage(shipments, page, chatId, callbackPrefix)
		if err != nil {
			return fmt.Errorf("ERR: creating message: %v", err)
		}

		edit := tgbotapi.NewEditMessageText(chatId, msgId, msg.Text)
		keyboard := msg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
		edit.ReplyMarkup = &keyboard

		_, err = Bot.Send(edit)
		if err != nil {
			return fmt.Errorf("ERR: editing message: %v", err)
		}

	case strings.Contains(cmd, "viewactivebycar:"):
		parts := strings.Split(cmd, ":")
		if len(parts) < 3 {
			return fmt.Errorf("invalid pagination callback data")
		}

		carId := parts[1]
		page, err := strconv.Atoi(parts[2])
		if err != nil {
			return fmt.Errorf("invalid page number: %v", err)
		}

		var shipments []*parser.Shipment
		var callbackPrefix string

		shipments, err = parser.GetAllActiveShipmentsByCarId(carId, globalStorage)
		callbackPrefix = "page:viewactivebycar:" + carId
		if err != nil {
			return fmt.Errorf("ERR: getting shipments: %v", err)
		}

		msg, err := CreateShipmentListMessage(shipments, page, chatId, callbackPrefix)
		if err != nil {
			return fmt.Errorf("ERR: creating message: %v", err)
		}

		edit := tgbotapi.NewEditMessageText(chatId, msgId, msg.Text)
		keyboard := msg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
		edit.ReplyMarkup = &keyboard

		_, err = Bot.Send(edit)
		if err != nil {
			return fmt.Errorf("ERR: editing message: %v", err)
		}
	case strings.Contains(cmd, "viewactive:"):
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

		shipments, err = parser.GetAllActiveShipments(globalStorage)
		callbackPrefix = "page:viewactive"
		if err != nil {
			return fmt.Errorf("ERR: getting shipments: %v", err)
		}

		msg, err := CreateShipmentListMessage(shipments, page, chatId, callbackPrefix)
		if err != nil {
			return fmt.Errorf("ERR: creating message: %v", err)
		}

		edit := tgbotapi.NewEditMessageText(chatId, msgId, msg.Text)
		keyboard := msg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
		edit.ReplyMarkup = &keyboard

		_, err = Bot.Send(edit)
		if err != nil {
			return fmt.Errorf("ERR: editing message: %v", err)
		}
	}
	return nil
}
