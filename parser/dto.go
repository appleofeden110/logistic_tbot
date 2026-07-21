package parser

import "time"

var validTaskTypes = map[string]bool{
	TaskLoad:     true,
	TaskUnload:   true,
	TaskCollect:  true,
	TaskDropoff:  true,
	TaskCleaning: true,
}

type UpdateShipmentInput struct {
	Started       *time.Time        `json:"Started"`
	Finished      *time.Time        `json:"Finished"`
	GeneralRemark *string           `json:"GeneralRemark"`
	Tasks         []UpdateTaskInput `json:"tasks"`
}

type UpdateTaskInput struct {
	Id                 int64      `json:"Id"` // 0 / omitted = new task
	Type               string     `json:"Type"`
	Address            string     `json:"Address"`
	DestinationAddress string     `json:"DestinationAddress"`
	Product            string     `json:"Product"`
	TankStatus         string     `json:"TankStatus"`
	Remark             string     `json:"Remark"`
	Start              *time.Time `json:"Start"`
	End                *time.Time `json:"End"`
	LoadReference      string     `json:"LoadReference"`
	LoadStartDate      *time.Time `json:"LoadStartDate"`
	LoadEndDate        *time.Time `json:"LoadEndDate"`
	UnloadReference    string     `json:"UnloadReference"`
	UnloadStartDate    *time.Time `json:"UnloadStartDate"`
	UnloadEndDate      *time.Time `json:"UnloadEndDate"`
	EditStatus         string     `json:"EditStatus"`
}
