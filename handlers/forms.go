package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"logistictbot/config"
	"logistictbot/db"
	"logistictbot/parser"
	"logistictbot/utils"
	"reflect"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const SkipKeyword = "Пропустити"

type TankRefuelForm struct {
	ChosenFuelCardId   int
	CurrentKilometrage int64  `form:"Введіть поточний кілометраж" form_question:"Введіть поточний кілометраж"`
	Address            string `form:"Введіть адресу заправки"            form_question:"Введіть адресу заправки"`
	Diesel             string `form:"Введіть кількість Дізелю ви заправили (введіть тільки числом)"             form_question:"Введіть кількість дизелю (літри), або пропустіть"`
	AdBlu              string `form:"Введіть кількість AdBlu ви взяли (введіть тільки числом)"              form_question:"Введіть кількість AdBlue (літри), або пропустіть"`
}

func storeRefuel(f db.Form, storage *sql.DB, bot *tgbotapi.BotAPI, driverSesh *db.Driver, chosenFuelCardId int) error {
	data, ok := f.Data.(TankRefuelForm)
	if !ok {
		return fmt.Errorf("ERR: refuel form data is wrong type: %T", f.Data)
	}

	diesel, err := strconv.ParseFloat(data.Diesel, 64)
	if err != nil {
		diesel = 0
	}
	adBlu, err := strconv.ParseFloat(data.AdBlu, 64)
	if err != nil {
		adBlu = 0
	}

	if chosenFuelCardId == 0 {
		inputMu.Lock()
		cardId, exists := pendingRefuelCard[driverSesh.ChatId]
		if exists {
			chosenFuelCardId = cardId
		}
		inputMu.Unlock()
	}

	shipment, err := parser.GetLatestShipmentByDriverId(storage, driverSesh.Id)
	if err != nil {
		shipment = &parser.Shipment{ShipmentId: 0}
	}

	refuel := db.TankRefuel{
		FuelCardId:         chosenFuelCardId,
		ShipmentId:         &shipment.ShipmentId,
		CurrentKilometrage: data.CurrentKilometrage,
		Address:            data.Address,
		Diesel:             diesel,
		AdBlu:              adBlu,
		Driver:             driverSesh,
	}

	if err := refuel.StoreTankRefuel(storage); err != nil {
		return fmt.Errorf("ERR: storing refuel: %w", err)
	}

	_, err = bot.Send(tgbotapi.NewMessage(f.ChatID, config.Translate(config.GetLang(f.ChatID), "refuel_saved")))
	return err
}

func createForm[T any](chatId int64, entity T, markup tgbotapi.InlineKeyboardMarkup, text, logMsg string) error {
	questionAnswers := make(map[string]string)
	tags, err := utils.GetAllFormTags[T](entity)
	if err != nil {
		return fmt.Errorf("ERR: getting tags: %v", err)
	}

	for _, question := range tags {
		questionAnswers[question] = ""
	}

	_, err = RegisterFormMessage(chatId, questionAnswers, markup, text)
	if err != nil {
		return fmt.Errorf("ERR: could not create a form message: %v", err)
	}

	log.Println("New form:", logMsg)
	return nil
}

func HandleFormInput(chatId int64, text string, state *db.FormState, globalStorage *sql.DB, from *tgbotapi.User) error {
	if text != SkipKeyword {
		state.Answers[state.Index] = text
		state.CurrentField = text
	}

	state.Index++

	if state.Index >= len(state.Answers) {
		return finishForm(chatId, state, globalStorage, from)
	}

	return askNextQuestion(chatId, state)
}

func finishForm(chatId int64, state *db.FormState, globalStorage *sql.DB, from *tgbotapi.User) error {
	var err error

	log.Println("chatId: ", chatId)

	if !state.Finished {
		var chosenFormText string
		var chosenFormMarkup tgbotapi.InlineKeyboardMarkup
		questionAnswers := make(map[string]string)
		for i, question := range state.Questions {
			questionAnswers[question] = state.Answers[i]
		}

		switch state.Form.WhichTable {

		case db.RefuelsTable:
			chosenFormText = config.Translate(config.GetLang(chatId), "form:done")
			chosenFormMarkup = FormRefuelDone(config.GetLang(chatId))
		case db.DriversTable:
			chosenFormText = config.Translate(config.GetLang(chatId), "form:done")
			chosenFormMarkup = FormDriverDone(config.GetLang(chatId))
		case db.ManagersTable:
			chosenFormText = config.Translate(config.GetLang(chatId), "form:done")
			chosenFormMarkup = FormManagerDone(config.GetLang(chatId))
		case db.CarsTable:
			chosenFormText = config.Translate(config.GetLang(chatId), "form:done")
			chosenFormMarkup = FormAddCarDone(config.GetLang(chatId))
		default:
			return fmt.Errorf("ERR: unknown form type: %T", state.Form.Data)
		}

		if chosenFormMarkup.InlineKeyboard == nil {
			fmt.Println(chosenFormMarkup, chosenFormText)
			log.Println("markup not initialized for form type")
		}

		_, err := RegisterFormMessage(chatId, questionAnswers, chosenFormMarkup, chosenFormText)
		if err != nil {
			return err
		}
		return nil
	}
	inputMu.Lock()
	delete(waitingForInput, chatId)
	inputMu.Unlock()
	state, err = getData(chatId, from, state)
	if err != nil {
		return err
	}

	var answerString string
	for i := 0; i < len(state.Answers); i++ {
		answerString += fmt.Sprintf("%s: %s\n", state.Questions[i], state.Answers[i])
	}

	err = db.CheckFormTable(globalStorage)
	if err != nil {
		return fmt.Errorf("ERR: checking if the form states table exists: %v\n", err)
	}

	if state.Form.WhichTable == db.RefuelsTable {
		refuelData, ok := state.Form.Data.(TankRefuelForm)
		if !ok {
			return fmt.Errorf("ERR: refuel form data wrong type after finish")
		}
		driver, err := db.GetDriverByChatId(globalStorage, chatId)
		if err != nil {
			return fmt.Errorf("ERR: getting driver for refuel store: %w", err)
		}
		return storeRefuel(state.Form, globalStorage, Bot, driver, refuelData.ChosenFuelCardId)
	}
	return state.Form.StoreForm(globalStorage, Bot)
}

func askNextQuestion(chatId int64, state *db.FormState) error {
	var nextQuestion tgbotapi.MessageConfig

	currentIndex := state.Index

	if currentIndex < len(state.Answers) && state.Answers[currentIndex] != "" {
		questionText := config.Translate(config.GetLang(chatId),
			"prev_answer",
			state.Questions[currentIndex],
			state.Answers[currentIndex],
		)
		nextQuestion = tgbotapi.NewMessage(chatId, questionText)

		skipButton := tgbotapi.NewKeyboardButton(config.Translate(config.GetLang(chatId), "skip"))
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(skipButton),
		)
		keyboard.OneTimeKeyboard = true
		nextQuestion.ReplyMarkup = keyboard
	} else {
		nextQuestion = tgbotapi.NewMessage(chatId, state.Questions[currentIndex])
	}

	_, err := Bot.Send(nextQuestion)
	return err
}

func getData(chatId int64, from *tgbotapi.User, state *db.FormState) (*db.FormState, error) {
	switch state.Form.WhichTable {
	case db.RefuelsTable:
		refuelForm := TankRefuelForm{}
		if err := populateFields(&refuelForm, nil, state.FieldNames, state.Answers); err != nil {
			return nil, err
		}

		if existing, ok := state.Form.Data.(TankRefuelForm); ok {
			refuelForm.ChosenFuelCardId = existing.ChosenFuelCardId
		}
		state.Form.Data = refuelForm
	case db.DriversTable:
		log.Println(from)
		driver := db.Driver{User: &db.User{ChatId: chatId, TgTag: from.UserName}, ChatId: chatId}
		if err := populateFields(&driver, driver.User, state.FieldNames, state.Answers); err != nil {
			return nil, err
		}
		state.Form.Data = driver
	case db.ManagersTable:
		manager := db.Manager{User: &db.User{ChatId: chatId, TgTag: from.UserName}, ChatId: chatId}
		if err := populateFields(&manager, manager.User, state.FieldNames, state.Answers); err != nil {
			return nil, err
		}
		state.Form.Data = manager
	case db.CarsTable:
		car := db.Car{}
		if err := populateFields(&car, nil, state.FieldNames, state.Answers); err != nil {
			return nil, err
		}
		state.Form.Data = car
	}
	return state, nil
}

func populateFields(entity interface{}, user *db.User, fieldNames []string, answers []string) error {
	entityValue := reflect.ValueOf(entity).Elem()
	var userValue reflect.Value
	if user != nil {
		userValue = reflect.ValueOf(user).Elem()
	}

	for i, fieldName := range fieldNames {
		answer := answers[i]

		if strings.Contains(fieldName, ".") {
			parts := strings.Split(fieldName, ".")
			if len(parts) == 2 && parts[0] == "User" {
				userField := userValue.FieldByName(parts[1])
				if userField.IsValid() && userField.CanSet() {
					if err := setField(userField, answer, parts[1]); err != nil {
						return err
					}
				}
			}
		} else {
			entityField := entityValue.FieldByName(fieldName)
			if entityField.IsValid() && entityField.CanSet() {
				if err := setField(entityField, answer, fieldName); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func setField(field reflect.Value, answer string, fieldName string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(answer)
	case reflect.Int, reflect.Int64:
		if strings.Contains(strings.ToLower(fieldName), "kilometrage") ||
			strings.Contains(strings.ToLower(fieldName), "km") {
			val, err := db.ParseKilometrage(answer)
			if err != nil {
				return fmt.Errorf("ERR: invalid kilometrage for %s: %v", fieldName, err)
			}
			field.SetInt(val)
		} else {
			val, err := strconv.ParseInt(answer, 10, 64)
			if err != nil {
				return fmt.Errorf("ERR: invalid number for %s: %v", fieldName, err)
			}
			field.SetInt(val)
		}
	default:
		return fmt.Errorf("ERR: unsupported type for field %s: %v", fieldName, field.Kind())
	}
	return nil
}

func GatherInfo(f db.Form) error {
	switch f.WhichTable {
	case db.DriversTable:
		return gatherFormInfo[db.Driver](f, db.Driver{})
	case db.ManagersTable:
		return gatherFormInfo[db.Manager](f, db.Manager{})
	case db.CarsTable:
		return gatherFormInfo[db.Car](f, db.Car{})
	case db.RefuelsTable:
		return gatherFormInfo[TankRefuelForm](f, TankRefuelForm{})
	default:
		return fmt.Errorf("ERR: unsupported table type: %s", f.WhichTable)
	}
}

func gatherFormInfo[T any](f db.Form, entity T) error {
	tags, err := utils.GetAllFormTags[T](entity)
	if err != nil {
		return fmt.Errorf("ERR: getting form tags from the struct: %w", err)
	}

	fieldNames, questions := getAllFieldsAndQuestions(tags)

	inputMu.Lock()
	state, exists := waitingForInput[f.ChatID]
	if !exists {
		waitingForInput[f.ChatID] = &db.FormState{
			Form:       f,
			FieldNames: fieldNames,
			Answers:    make([]string, len(questions)),
			Questions:  questions,
			Index:      0,
			Finished:   false,
		}
		state = waitingForInput[f.ChatID]
	} else {
		state.Index = 0
	}

	inputMu.Unlock()

	var firstQuestion tgbotapi.MessageConfig

	if exists && len(state.Answers) > 0 && state.Answers[0] != "" {
		questionText := fmt.Sprintf(
			"%s (Попередня відповідь: %s)\n\nНатисніть пропустити якщо цього не потрібно міняти",
			questions[0],
			state.Answers[0],
		)
		firstQuestion = tgbotapi.NewMessage(f.ChatID, questionText)

		skipButton := tgbotapi.NewKeyboardButton(SkipKeyword)
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(skipButton),
		)
		keyboard.OneTimeKeyboard = true
		firstQuestion.ReplyMarkup = keyboard
	} else {
		firstQuestion = tgbotapi.NewMessage(f.ChatID, questions[0])
	}

	_, err = Bot.Send(firstQuestion)
	return err
}

func getAllFieldsAndQuestions(tags map[string]string) (fieldNames, questions []string) {
	for fieldName, question := range tags {
		questions = append(questions, question)
		fieldNames = append(fieldNames, fieldName)
	}
	return fieldNames, questions
}
