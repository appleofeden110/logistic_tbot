package state_machines

import (
	"context"
	"fmt"

	"github.com/looplab/fsm"
)

type ShipmentT struct {
	ShipmentId int
	Task       []*TaskT
	FSM        *fsm.FSM
}

type TaskT struct {
	TaskId int
	FSM    *fsm.FSM
}

func NewShipmentT(id int) *ShipmentT {
	s := &ShipmentT{
		ShipmentId: id,
	}

	s.FSM = fsm.NewFSM(
		"not_started",
		fsm.Events{
			{Name: "begin", Src: []string{"not_started"}, Dst: "in_progress"},
			{Name: "end", Src: []string{"in_progress"}, Dst: "finished"},
		},
		fsm.Callbacks{
			"before_begin": func(_ context.Context, e *fsm.Event) {
				e.FSM.SetMetadata("msg", "Shipment has began")
				e.FSM.SetMetadata("shipment_id", s.ShipmentId)
				fmt.Println(e)
			},
			"after_begin": func(_ context.Context, e *fsm.Event) {
				message, ok := e.FSM.Metadata("msg")
				if ok {
					fmt.Println("message = " + message.(string))
				}
				shipment, ok := e.FSM.Metadata("shipment_id")
				if ok {
					fmt.Printf("shipment id: %d\n", shipment)
				}
			},
		},
	)

	return s
}

// func NewTaskT(id int) *TaskT {
// 	t := &TaskT{
// 		TaskId: id,
// 		FSM: fsm.NewFSM(
// 			"not started",
// 			fsm.Events{
// 				{Name: "begin", Src: []string{""}}
// 			}
// 		),
// 	}

// }

// func (s *ShipmentT) AddTasksToShipment(taskTypes []string) {
// 	for i, taskType := range taskTypes {
// 		NewTaskT(id)
// 	}

// }
