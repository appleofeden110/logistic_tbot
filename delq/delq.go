package delq

import (
	"database/sql"
	"fmt"
	"log"
	"logistictbot/db"
	"logistictbot/parser"
	"strings"
	"time"

	tgbotapi "github.com/appleofeden110/telegram-bot-api/v5"
)

type DeleteQueueNode struct {
	ID        int64     `db:"id"`
	MessageID int       `db:"message_id"`
	ChatID    int64     `db:"chat_id"`
	Scheduled time.Time `db:"scheduled"`
	IsDeleted bool      `db:"is_deleted"`

	Requirements Requirements
}

type RequirementType string

type Requirements struct {
	ID                int             `db:"id"`
	Type              RequirementType `db:"requirement_type"`
	TrackedTaskId     int
	TrackedShipmentId int64
	TrackedRefuelId   int
	Surplus           time.Time
	areMet            bool
}

const (
	SCHEDULE_SURPLUS = 3 * time.Second
)

var (
	DeleteQueue = make(chan DeleteQueueNode, 1000)

	TaskFinished     RequirementType = "task_finished"     // messages that are left after finishing task, that do not provide any further information
	ShipmentFinished RequirementType = "shipment_finished" // mostly stuff that is left after finishing shipment
	Refueled         RequirementType = "refueled"          // basically all of the unneccesary inputs
	Timing           RequirementType = "timing"            // well, it is what it is
)

func (n *DeleteQueueNode) CheckRequirements(globalStorage *sql.DB) {
	r := n.Requirements

	if r.areMet {
		log.Println("Requirements are already met")
		return
	}
	switch r.Type {
	case TaskFinished:
		task, err := parser.GetTaskById(globalStorage, r.TrackedTaskId)
		if err != nil {
			log.Printf("ERR: getting task by id (%d): %v\n", r.TrackedTaskId, err)
			return
		}
		if task.IsFinished() {
			r.areMet = true
			n.Scheduled = time.Now().Add(SCHEDULE_SURPLUS)
		}

	case ShipmentFinished:
		shipment, err := parser.GetShipment(globalStorage, r.TrackedShipmentId)
		if err != nil {
			log.Printf("ERR: getting shipment by id (%d): %v\n", r.TrackedShipmentId, err)
			return
		}

		if shipment.IsFinished() {
			r.areMet = true
			n.Scheduled = time.Now().Add(SCHEDULE_SURPLUS)
		}

	case Refueled:
		tr, err := db.GetTankRefuelById(globalStorage, r.TrackedRefuelId)
		if err != nil {
			if !strings.Contains(err.Error(), "sql: no rows in result set") {
				log.Printf("ERR: getting tank refuel by id: %v\n", err)
			}
			return
		}

		if tr.Address != "" {
			r.areMet = true
			n.Scheduled = time.Now().Add(SCHEDULE_SURPLUS)
		}

	}

	n.Requirements = r
}

// EnqueueToDelete is used to enqueue for deletion any messages that are useless in a long run, do not hold information and are used to lead user to successful use of the bot.
// Requirements holding the second argument need to be filled with id that needs to be tracked and deleted, as well as the type
//
// Example usage:
//
//	delq.EnqueueToDelete(globalStorage, chatId, messageId, delq.Requirements{
//	    Type:          delq.TaskFinished,
//	    TrackedTaskId: task.Id,
//	})
func EnqueueToDelete(globalStorage *sql.DB, chatId int64, messageId int, requirements Requirements) {
	log.Println("ENQUEUED FOR DELETION WITH REQUIREMENTS: ", messageId, chatId, requirements.Type)
	dqNode := DeleteQueueNode{
		MessageID:    messageId,
		ChatID:       chatId,
		Requirements: requirements,
	}
	DeleteQueue <- dqNode
	if err := dqNode.StoreDeleteQueueNode(globalStorage); err != nil {
		log.Printf("ERR: storing dqNode in EnqueueToDelete: %v\n", err)
	}
}

func DeleteWorker(globalStorage *sql.DB, bot *tgbotapi.BotAPI) {
	var pending = make([]DeleteQueueNode, 1000)

	timer := time.NewTicker(5 * time.Second)
	defer timer.Stop()

	for {
		select {
		case dqNode := <-DeleteQueue:
			if !dqNode.Requirements.areMet {
				pending = append(pending, dqNode)
				continue
			}
			processNode(globalStorage, bot, &dqNode)
		case <-timer.C:
			var stillPending []DeleteQueueNode
			for _, node := range pending {
				if node.Requirements.areMet {
					processNode(globalStorage, bot, &node)
				} else {
					node.CheckRequirements(globalStorage)
					stillPending = append(stillPending, node)
				}
			}
			pending = stillPending
		}
	}
}

func processNode(globalStorage *sql.DB, bot *tgbotapi.BotAPI, dqNode *DeleteQueueNode) {
	if _, err := bot.Request(tgbotapi.NewDeleteMessage(dqNode.ChatID, dqNode.MessageID)); err != nil {
		if strings.Contains(err.Error(), "Bad Request: message to delete not found") {
			dqNode.IsDeleted = true
			dqNode.UpdateIsDeleted(globalStorage)
			log.Println("WARN: message was deleted from the local and global storages only after not being found: ", dqNode.ChatID, dqNode.MessageID)
		}

		log.Printf("ERR: deleting a message with bot.Request for msgID %d in chatId %d: %v\n", dqNode.MessageID, dqNode.ChatID, err)

		DeleteQueue <- *dqNode
		return
	}

	dqNode.IsDeleted = true
	dqNode.UpdateIsDeleted(globalStorage)
}

func ScheduleForDeletion(chatId int64, messageId int, globalStorage *sql.DB) {
	log.Println("SCHEDULED FOR DELETION: ", messageId, chatId)
	dqNode := DeleteQueueNode{MessageID: messageId, ChatID: chatId, Scheduled: time.Now().Add(SCHEDULE_SURPLUS)}
	DeleteQueue <- dqNode
	err := dqNode.StoreDeleteQueueNode(globalStorage)
	if err != nil {
		log.Printf("ERR: err storing dqNode: %v\n", err)
	}
}

func FillDeleteQueue(db *sql.DB) error {
	nodes, err := GetAllDeleteQueueNodes(db)
	if err != nil {
		return fmt.Errorf("ERR: filling delete queue: %v\n", err)
	}
	for _, node := range nodes {
		if node.IsDeleted {
			continue
		}
		DeleteQueue <- *node
	}
	return nil
}

func GetAllDeleteQueueNodes(db *sql.DB) ([]*DeleteQueueNode, error) {
	query := `
		SELECT
			dq.id,
			dq.message_id,
			dq.chat_id,
			dq.scheduled,
			dq.is_deleted,
			r.id,
			r.type,
			r.surplus,
			r.are_met,
			r.tracked_task_id,
			r.tracked_shipment_id
			r.tracked_refuel_id
		FROM delete_queue dq
		LEFT JOIN requirements r ON dq.requirements_id = r.id
		ORDER BY dq.scheduled ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ERR: querying all delete_queue nodes: %v\n", err)
	}
	defer rows.Close()

	var nodes []*DeleteQueueNode
	for rows.Next() {
		var node DeleteQueueNode
		var (
			reqId                sql.NullInt64
			reqType              sql.NullString
			reqSurplus           sql.NullTime
			reqAreMet            sql.NullBool
			reqTrackedTaskId     sql.NullInt64
			reqTrackedShipmentId sql.NullInt64
			reqTrackedRefuelId   sql.NullInt64
		)
		err := rows.Scan(
			&node.ID,
			&node.MessageID,
			&node.ChatID,
			&node.Scheduled,
			&node.IsDeleted,
			&reqId,
			&reqType,
			&reqSurplus,
			&reqAreMet,
			&reqTrackedTaskId,
			&reqTrackedShipmentId,
			&reqTrackedRefuelId,
		)
		if err != nil {
			return nil, fmt.Errorf("ERR: scanning delete_queue row: %v\n", err)
		}
		if reqId.Valid {
			node.Requirements = Requirements{
				ID:                int(reqId.Int64),
				Type:              RequirementType(reqType.String),
				Surplus:           reqSurplus.Time,
				areMet:            reqAreMet.Bool,
				TrackedTaskId:     int(reqTrackedTaskId.Int64),
				TrackedShipmentId: reqTrackedShipmentId.Int64,
				TrackedRefuelId:   int(reqTrackedRefuelId.Int64),
			}
		}
		nodes = append(nodes, &node)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ERR: iterating delete_queue rows: %v\n", err)
	}
	return nodes, nil
}

func (n *DeleteQueueNode) UpdateRequirements(db *sql.DB) error {
	_, err := db.Exec("UPDATE requirements SET are_met=1 WHERE delete_queue_id=?", n.ID)
	if err != nil {
		return fmt.Errorf("ERR: updating are_met for requirements with id %d: %v\n", n.ID, err)
	}
	return nil
}

func (n *DeleteQueueNode) UpdateIsDeleted(db *sql.DB) error {
	_, err := db.Exec("UPDATE delete_queue SET is_deleted=1 WHERE message_id=? AND chat_id=?", n.MessageID, n.ChatID)
	if err != nil {
		return fmt.Errorf("ERR: updating is delete for dqNode with id %d: %v\n", n.ID, err)
	}
	return nil
}

func (n *DeleteQueueNode) StoreDeleteQueueNode(db *sql.DB) error {
	stmt, err := db.Prepare(`
		INSERT INTO delete_queue (message_id, chat_id, scheduled)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("ERR: preparing statement for insert delete_queue node: %v\n", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(n.MessageID, n.ChatID, n.Scheduled)
	if err != nil {
		return fmt.Errorf("ERR: executing prep insert delete_queue node stmt: %v\n", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("ERR: getting last insert id for delete_queue node: %v\n", err)
	}
	n.ID = id

	if n.Requirements.Type != "" {
		n.Requirements.ID = int(n.ID)
		if err := n.Requirements.StoreRequirements(db); err != nil {
			return fmt.Errorf("ERR: storing requirements for dqNode: %v\n", err)
		}
		// update delete_queue row with requirements_id
		_, err = db.Exec(
			"UPDATE delete_queue SET requirements_id = ? WHERE id = ?",
			n.Requirements.ID, n.ID,
		)
		if err != nil {
			return fmt.Errorf("ERR: updating requirements_id on delete_queue node: %v\n", err)
		}
	}

	return nil
}
func (r *Requirements) StoreRequirements(db *sql.DB) error {
	stmt, err := db.Prepare(`
		INSERT INTO requirements (type, surplus, are_met, delete_queue_id, tracked_task_id, tracked_shipment_id, tracked_refuel_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("ERR: preparing statement for insert requirements: %v\n", err)
	}
	defer stmt.Close()

	var trackedTaskId, trackedShipmentId, trackedRefuelId sql.NullInt64
	if r.TrackedTaskId != 0 {
		trackedTaskId = sql.NullInt64{Int64: int64(r.TrackedTaskId), Valid: true}
	}
	if r.TrackedShipmentId != 0 {
		trackedShipmentId = sql.NullInt64{Int64: r.TrackedShipmentId, Valid: true}
	}
	if r.TrackedRefuelId != 0 {
		trackedRefuelId = sql.NullInt64{Int64: r.TrackedShipmentId, Valid: true}
	}

	res, err := stmt.Exec(
		string(r.Type),
		r.Surplus,
		r.areMet,
		r.ID, // delete_queue_id, filled after StoreDeleteQueueNode
		trackedTaskId,
		trackedShipmentId,
		trackedRefuelId,
	)
	if err != nil {
		return fmt.Errorf("ERR: executing prep insert requirements stmt: %v\n", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("ERR: getting last insert id for requirements: %v\n", err)
	}
	r.ID = int(id)
	return nil
}
