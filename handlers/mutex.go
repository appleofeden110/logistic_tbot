package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"logistictbot/config"
	"logistictbot/db"
	"logistictbot/delq"
	"logistictbot/parser"
	"logistictbot/tracking"
	"sync"

	tgbotapi "github.com/appleofeden110/telegram-bot-api/v5"
	"github.com/gofrs/uuid"
)

var (
	// takes in DRIVER's id
	taskSessions   = make(map[uuid.UUID]*parser.TaskSection)
	taskSessionsMu sync.Mutex

	// DO NOT USE
	pendingRefuelCard = make(map[int64]int) // I will delete that fucking form system one day, gimme another month of pondering please
	// - until then DO NOT TOUCH

	pendingRefuel = make(map[uuid.UUID]*db.TankRefuel) // driverId -> tankRefuel
	refuelMu      sync.Mutex

	replyingToMessage   = make(map[int64]int64) // chatId -> commsId
	replyingToMessageMu sync.Mutex

	nonRepliedMessages   = make(map[uuid.UUID]*CommunicationMsg) // userId -> comms
	nonRepliedMessagesMu sync.Mutex

	writingToChatMap   = make(map[int64]int64) // senderChatId -> receiverChatId
	writingToChatMapMu sync.RWMutex

	managerSessions   = make(map[int64]*db.Manager)
	managerSessionsMu sync.Mutex

	driverSessions   = make(map[int64]*db.Driver)
	driverSessionsMu sync.Mutex

	devSession   = make(map[int64]*db.DevSesh)
	devSessionMu sync.Mutex

	waitingForInput = make(map[int64]*db.FormState)
	inputMu         sync.Mutex

	trackingSessions      = make(map[int64]*tracking.TrackingSession)
	trackingSessionsMutex sync.Mutex

	toDelete   = make(map[int]tgbotapi.MessageID) // taskId -> message with performing a task
	toDeleteMu sync.Mutex
)

func FillSessions(globalStorage *sql.DB) error {
	drivers, err := db.GetAllDrivers(globalStorage)
	if err != nil {
		return fmt.Errorf("ERR: getting all drivers: %v\n", err)
	}

	driverSessionsMu.Lock()
	taskSessionsMu.Lock()
	for _, d := range drivers {
		d.Session, err = d.GetLastActiveSession(globalStorage)
		if err != nil {
			driverSessionsMu.Unlock()
			taskSessionsMu.Unlock()
			return fmt.Errorf("ERR: getting a session for driver %s: %v\n", d.User.Name, err)
		}
		driverSessions[d.ChatId] = d
		if d.PerformedTaskId != 0 {
			task, err := parser.GetTaskById(globalStorage, d.PerformedTaskId)
			if err != nil {
				driverSessionsMu.Unlock()
				taskSessionsMu.Unlock()
				return fmt.Errorf("ERR: getting task that is being performed: %v\n", err)
			}
			taskSessions[d.Id] = task
		}

		config.SetChatLang(d.ChatId, config.LangCode(d.User.Language))
	}
	taskSessionsMu.Unlock()
	driverSessionsMu.Unlock()

	dg, err := db.GetAllDriverGroups(globalStorage)
	if err != nil {
		return fmt.Errorf("ERR: getting all the driver groups: %v\n", err)
	}

	for _, g := range dg {
		config.SetChatLang(g.GroupChatId, g.GroupLang)
	}

	log.Printf("Driver sessions and their tasks are filled (d-len: %d; t-len: %d)\n", len(driverSessions), len(taskSessions))

	managers, err := db.GetAllManagers(globalStorage)
	if err != nil {
		return fmt.Errorf("ERR: getting all the managers: %v\n", err)
	}

	managerSessionsMu.Lock()
	for _, m := range managers {
		managerSessions[m.ChatId] = m
		config.SetChatLang(m.ChatId, config.LangCode(m.User.Language))
	}
	managerSessionsMu.Unlock()

	log.Printf("Manager sessions are filled (len: %d)\n", len(managerSessions))

	comms, err := GetAllNonRepliedMessages(globalStorage)
	if err != nil {
		return fmt.Errorf("ERR: getting all non replied messages: %v\n", err)
	}
	nonRepliedMessagesMu.Lock()
	for _, c := range comms {
		nonRepliedMessages[c.Receiver.Id] = c
	}
	nonRepliedMessagesMu.Unlock()

	log.Printf("Non-replied messages are filled (len: %d)\n", len(nonRepliedMessages))

	err = delq.FillDeleteQueue(globalStorage)
	if err != nil {
		return fmt.Errorf("ERR: getting all the dq nodes: %v\n", err)
	}

	log.Printf("Delete Queue is filled (len: %d)\n", len(delq.DeleteQueue))

	return nil
}
