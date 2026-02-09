package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"logistictbot/db"
	"logistictbot/parser"
	"logistictbot/tracking"
	"sync"

	"github.com/gofrs/uuid"
)

var (
	// takes in DRIVER's id
	taskSessions   = make(map[uuid.UUID]*parser.TaskSection)
	taskSessionsMu sync.Mutex

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
)

func FillSessions(globalStorage *sql.DB) error {
	drivers, err := db.GetAllDrivers(globalStorage)
	if err != nil {
		return fmt.Errorf("err getting all drivers: %v\n", err)
	}

	driverSessionsMu.Lock()
	for _, d := range drivers {
		d.Session, err = d.GetLastActiveSession(globalStorage)
		if err != nil {
			return fmt.Errorf("err getting a session for driver %s: %v\n", d.User.Name, err)
		}
		driverSessions[d.ChatId] = d
	}
	driverSessionsMu.Unlock()

	log.Printf("Driver sessions are filled (len: %d)\n", len(driverSessions))

	managers, err := db.GetAllManagers(globalStorage)
	if err != nil {
		return fmt.Errorf("err getting all the managers: %v\n", err)
	}

	managerSessionsMu.Lock()
	for _, m := range managers {
		managerSessions[m.ChatId] = m
	}
	managerSessionsMu.Unlock()

	log.Printf("Manager sessions are filled (len: %d)\n", len(managerSessions))

	return nil
}
