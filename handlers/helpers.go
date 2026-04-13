package handlers

import (
	"database/sql"
	"fmt"
	"logistictbot/config"
	"logistictbot/db"
	"logistictbot/parser"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/appleofeden110/telegram-bot-api/v5"
)

func GenStartTaskMsg(chatId int64, task *parser.TaskSection, globalStorage *sql.DB) (tgbotapi.MessageConfig, error) {
	var loadingTopicId int

	if strings.HasPrefix(strconv.Itoa(int(chatId)), "-100") {
		loadingTopicId = FindLoadingTopic(chatId, globalStorage)
	}

	shipment, err := parser.GetShipment(globalStorage, task.ShipmentId)
	if err != nil {
		return tgbotapi.MessageConfig{}, fmt.Errorf("ERR: getting shipment to display task: %v\n", err)
	}

	country, _ := parser.ExtractCountry(task.Address)

	startTaskMsg := tgbotapi.NewMessage(chatId,
		fmt.Sprintf(TaskSubmissionFormatText,
			task.ShipmentId,
			strings.ToUpper(task.Type),
			shipment.Chassis,
			shipment.Container,
			time.Now().In(config.WarsawLoc).Format("02.01.2006"),
			task.Start.In(config.WarsawLoc).Format("15:04"),
			"",
			db.FormatKilometrage(int(task.CurrentKilometrage)),
			task.Address,
			country.Name,
			country.Emoji,
			0,
			0.00,
		),
		loadingTopicId,
	)
	startTaskMsg.ParseMode = tgbotapi.ModeHTML

	startTaskMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.GetLang(chatId), "btn:driver:endtask"), fmt.Sprintf("driver:endtask:%d", task.Id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.GetLang(chatId), "btn:driver:add_docstotask"), fmt.Sprintf("driver:add_doctotask:%d", task.Id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.GetLang(chatId), "btn:driver:add_picstotask"), fmt.Sprintf("driver:add_picstotask:%d", task.Id)),
		),
	)

	return startTaskMsg, nil
}

func GenEndTaskMessage(chatId int64, task *parser.TaskSection, globalStorage *sql.DB) (tgbotapi.MessageConfig, error) {
	shipment, err := parser.GetShipment(globalStorage, task.ShipmentId)
	if err != nil {
		return tgbotapi.MessageConfig{}, fmt.Errorf("ERR: getting shipment from a task: %v\n", err)
	}

	country, _ := parser.ExtractCountry(task.Address)
	endMsg := tgbotapi.NewMessage(chatId, fmt.Sprintf(config.Translate(config.GetLang(chatId), "driver:task_done")+TaskSubmissionFormatText,
		task.ShipmentId,
		strings.ToUpper(task.Type),
		shipment.Chassis,
		shipment.Container,
		time.Now().In(config.WarsawLoc).Format("02.01.2006"),
		task.Start.In(config.WarsawLoc).Format("15:04"),
		task.End.In(config.WarsawLoc).Format("15:04"),
		db.FormatKilometrage(int(task.CurrentKilometrage)),
		task.Address,
		country.Name,
		country.Emoji,
		task.CurrentWeight,
		task.CurrentTemperature),
	)
	endMsg.ParseMode = tgbotapi.ModeHTML
	endMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.GetLang(chatId), "endmsg:edit"), "driver:task_edit:"+strconv.Itoa(task.Id)),
		),
	)

	return endMsg, nil
}
