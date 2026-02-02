package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	welcomeText       = `<b>Logistic Bot</b>`
	welcomeMenuMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Реєстрація Водія", "createform:driver_registration"),
			tgbotapi.NewInlineKeyboardButtonData("Реєстрація Менеджера", "createform:manager_registration"),
		),
	)

	driverStartMarkupPause = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Почати зміну", "driver:beginday"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Активні маршрути", "driver:viewactive"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Всі маршрути", "driver:viewall"),
		),
	)

	driverStartMarkupWorking = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Закінчити зміну", "driver:endDay"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Активні маршрути", "driver:viewactive"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Всі маршрути", "driver:viewall"),
		),
	)

	managerStartMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Активні маршрути", "manager:viewactive"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Всі маршрути", "manager:viewall"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Створити маршрут", "manager:create"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Список водіїв", "manager:viewdrivers"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Створити звіт за місяць", "manager:mstmt"),
		),
	)

	formTextAddCar   = "<b>Форма для додавання машини в Базу Даних.</b>"
	formMarkupAddCar = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Заповнити форму", "startform:cars"),
		),
	)
	formTextAddCarDone   = "<b>Чи все правильно в відповідях?</b>"
	formMarkupAddCarDone = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Так, підвердити дані", "acceptform:cars"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Ні, змінити дані", "editform:cars"),
		),
	)

	formTextDriver   = "<b>Форма для реєстрації водія.</b>"
	formMarkupDriver = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Почати реєстрацію", "startform:drivers"),
		),
	)
	formTextDriverDone   = "<b>Чи все правильно в відповідях?</b>"
	formMarkupDriverDone = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Так, підвердити дані", "acceptform:drivers"),
		),

		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Ні, змінити дані", "editform:drivers"),
		),
	)

	formTextManager   = "<b>Форма для реєстрації менеджера.</b>"
	formMarkupManager = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Почати реєстрацію", "startform:managers"),
		),
	)

	formTextManagerDone   = "<b>Чи все правильно в відповідях?</b>"
	formMarkupManagerDone = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Так, підвердити дані", "acceptform:managers"),
		),

		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Ні, змінити дані", "editform:managers"),
		),
	)

	formTextSendTaskToDriver   = "<b>Форма для відсилання завдання водію<b/>"
	formMarkupSendTaskToDriver = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Відправити документ", "startform:docs"),
		),
	)

	formTextRoute   = "<b>Завдання прийнято</b>"
	formMarkupRoute = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Почати завдання", "startform:routes")))

	devInitMessage = "What would you like to do?"
	devInit        = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("update cleaning stations", "dev:updatecleaningstations"),
		tgbotapi.NewInlineKeyboardButtonData("finish dev sesh", "dev:finish"),
	))

	// shipment id, task name, container, chassis, date formatted as "02.01.2006", times as 17:15, kilometrage, address, country with country emoji, weight and temperature (done from form, needed usually only for load and unload)
	TaskSubmissionFormatText = "Shipment %d\n\n%s\n\n%s %s\n\n%s\n%s - %s\n%s\n\n%s\n\n%s %s\n\n%d kg      %.2f ℃"
)
