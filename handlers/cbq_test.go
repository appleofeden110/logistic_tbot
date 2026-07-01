package handlers

import (
	"logistictbot/config"
	"logistictbot/db"
	"logistictbot/parser"
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	tgbotapi "github.com/appleofeden110/telegram-bot-api/v5"
	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
)

func jumbleUUID(s string) string {
	r := []rune(s)
	rand.Shuffle(len(r), func(i, j int) {
		if r[i] == '-' {
			r[i], r[j] = r[i], r[j]
		}
		r[i], r[j] = r[j], r[i]
	})
	return string(r)
}

func TestCanAcceptShipment(t *testing.T) {
	chat_id, _, globalStorage, err := config.LoadEverythingForTest()
	if err != nil {
		t.Fatal(err)
	}

	mock := &tgbotapi.BotAPI{}
	original := Bot
	Bot = mock
	defer func() { Bot = original }()

	// test id of the shipment
	shipment_id_str := "4344650"
	shipment_id, err := strconv.ParseInt(shipment_id_str, 10, 64)
	if err != nil {
		t.Fatal(err)
	}

	driver, err := db.GetDriverByChatId(globalStorage, chat_id)
	if err != nil {
		t.Fatal(err)
	}

	driverSessionsMu.Lock()
	driverSessions[chat_id] = driver
	driverSessionsMu.Unlock()

	shipment, err := parser.GetShipment(globalStorage, shipment_id)
	if err != nil {
		t.Error(err)
	}

	okShipment := shipment

	wrongDriver := shipment
	wrongDriver.Id = okShipment.Id + 1
	wrongDriver.DriverId = uuid.FromStringOrNil(jumbleUUID(wrongDriver.DriverId.String()))

	err = wrongDriver.StoreShipment(globalStorage)
	if err != nil {
		t.Fatal(err)
	}
	defer wrongDriver.DeleteShipment(globalStorage)

	alreadyStarted := shipment
	alreadyStarted.Id = okShipment.Id + 2
	alreadyStarted.Started = time.Now().Add(time.Minute)

	err = alreadyStarted.StoreShipment(globalStorage)
	if err != nil {
		t.Fatal(err)
	}
	defer alreadyStarted.DeleteShipment(globalStorage)

	tasksAreEmpty := shipment
	tasksAreEmpty.Id = okShipment.Id + 3
	tasksAreEmpty.Tasks = make([]*parser.TaskSection, 0)

	err = tasksAreEmpty.StoreShipment(globalStorage)
	if err != nil {
		t.Fatal(err)
	}
	defer tasksAreEmpty.DeleteShipment(globalStorage)

	tests := []struct {
		name     string
		shipment *parser.Shipment
		want     bool
	}{
		{"wrong driver", wrongDriver, false},
		{"already started", alreadyStarted, false},
		{"ok to accept", okShipment, true},
		{"tasks are empty", tasksAreEmpty, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cbq := &tgbotapi.CallbackQuery{
				ID:           "1",
				From:         &tgbotapi.User{ID: chat_id},
				Message:      &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chat_id}},
				Data:         "shipment:accept:" + strconv.FormatInt(tt.shipment.Id, 10),
				ChatInstance: "1",
			}
			if err := HandleCallbackQuery(cbq, globalStorage); err != nil {
				t.Error(err)
			}
			t.Log(err)
		})
	}
}
