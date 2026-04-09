package handlers

import (
	"fmt"
	"logistictbot/config"
	"logistictbot/db"
	"logistictbot/parser"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/appleofeden110/telegram-bot-api/v5"
)

var (
	devInitMessage = "What would you like to do?"
	devInit        = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("update cleaning stations", "dev:updatecleaningstations"),
		tgbotapi.NewInlineKeyboardButtonData("finish dev sesh", "dev:finish"),
	))

	// shipment id, task name, container, chassis, date formatted as "02.01.2006", times as 17:15, kilometrage, address, country with country emoji, weight and temperature (done from form, needed usually only for load and unload)
	TaskSubmissionFormatText = "Shipment %d\n\n%s\n\n%s %s\n\n%s\n%s - %s\n%s\n\n%s\n\n%s %s\n\n%d kg      %.2f ℃"
)

func CreateTaskMessage(chatId int64, task *parser.TaskSection, shipment *parser.Shipment) tgbotapi.MessageConfig {
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

	return endMsg
}

func DriverStartMarkupPause(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:active_routes"), "driver:viewactive"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:all_routes"), "driver:viewall"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:report_refuel"), "driver:refuel"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:write_manager"), "driver:sendmessage"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:request_cleaning"), "driver:washing"),
		),
	)
}

func DriverStartMarkupWorking(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:active_routes"), "driver:viewactive"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:all_routes"), "driver:viewall"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:report_refuel"), "driver:refuel"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:write_manager"), "driver:sendmessage"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:request_cleaning"), "driver:washing"),
		),
	)
}

func SuperAdminMarkupManager(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:active_routes"), "manager:viewactive"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:all_routes"), "manager:viewall"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:create_route"), "manager:create"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:driver_list"), "manager:viewdrivers"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:route_report"), "manager:mstmt"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:refuel_report"), "manager:mrefuel"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:write_driver"), "manager:sendmessage"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(lang, "btn:switch_to_driver"), "sa:switch_to_driver"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(lang, "btn:add_car"), "sa:add_car"),
		),
	)
}
func SuperAdminMarkupDriver(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:active_routes"), "driver:viewactive"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:all_routes"), "driver:viewall"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:report_refuel"), "driver:refuel"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:write_manager"), "driver:sendmessage"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:request_cleaning"), "driver:washing"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(lang, "btn:switch_to_manager"), "sa:switch_to_manager"),
		),
	)
}

func ManagerStartMarkup(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:active_routes"), "manager:viewactive"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:all_routes"), "manager:viewall"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:create_route"), "manager:create"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:driver_list"), "manager:viewdrivers"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:route_report"), "manager:mstmt"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:refuel_report"), "manager:mrefuel"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:write_driver"), "manager:sendmessage"),
		),
	)
}

func FormAddCar(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:fill_form"), "startform:cars"),
		),
	)
}

func FormAddCarDone(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:confirm_data"), "acceptform:cars"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:edit_data"), "editform:cars"),
		),
	)
}

func FormRefuel(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:fill_form"), "startform:tank_refuels"),
		),
	)
}

func FormRefuelDone(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:confirm_data"), "acceptform:tank_refuels"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:edit_data"), "editform:tank_refuels"),
		),
	)
}

func FormDriver(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:start_registration"), "startform:drivers"),
		),
	)
}

func FormDriverDone(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:confirm_data"), "acceptform:drivers"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:edit_data"), "editform:drivers"),
		),
	)
}

func FormManager(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:start_registration"), "startform:managers"),
		),
	)
}

func FormManagerDone(lang config.LangCode) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:confirm_data"), "acceptform:managers"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.LangCode(lang), "btn:edit_data"), "editform:managers"),
		),
	)
}
