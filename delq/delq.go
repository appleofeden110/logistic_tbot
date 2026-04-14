package delq

import (
	"database/sql"
	"fmt"
	"log"
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
	Surplus           time.Time
	areMet            bool
}

const (
	SCHEDULE_SURPLUS = 30 * time.Minute
)

var (
	DeleteQueue = make(chan DeleteQueueNode, 1000)

	TaskFinished     RequirementType = "task_finished"     // messages that are left after finishing task, that do not provide any further information
	ShipmentFinished RequirementType = "shipment_finished" // mostly stuff that is left after finishing shipment
	Timing           RequirementType = "timing"            // well, it is what it is
)

func (r *Requirements) Check(globalStorage *sql.DB) {
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
		r.areMet = task.IsFinished()

	case ShipmentFinished:
		shipment, err := parser.GetShipment(globalStorage, r.TrackedShipmentId)
		if err != nil {
			log.Printf("ERR: getting shipment by id (%d): %v\n", r.TrackedShipmentId, err)
			return
		}
		r.areMet = shipment.IsFinished()
	}
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
		Scheduled:    time.Now().Add(SCHEDULE_SURPLUS),
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
					node.Requirements.Check(globalStorage)
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
			id,
			message_id,
			chat_id,
			scheduled,
			is_deleted
		FROM delete_queue
		ORDER BY scheduled ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ERR: querying all delete_queue nodes: %v\n", err)
	}
	defer rows.Close()

	var nodes []*DeleteQueueNode
	for rows.Next() {
		var node DeleteQueueNode
		err := rows.Scan(
			&node.ID,
			&node.MessageID,
			&node.ChatID,
			&node.Scheduled,
			&node.IsDeleted,
		)
		if err != nil {
			return nil, fmt.Errorf("ERR: scanning delete_queue row: %v\n", err)
		}
		nodes = append(nodes, &node)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ERR: iterating delete_queue rows: %v\n", err)
	}
	return nodes, nil
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
	return nil
}
