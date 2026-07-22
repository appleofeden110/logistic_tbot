package parser

import (
	"encoding/json"
	"time"
)

type FlexTime struct {
	time.Time
	Valid bool
}

var validTaskTypes = map[string]bool{
	TaskLoad:     true,
	TaskUnload:   true,
	TaskCollect:  true,
	TaskDropoff:  true,
	TaskCleaning: true,
}

type UpdateShipmentInput struct {
	Started       FlexTime          `json:"Started"`
	Finished      FlexTime          `json:"Finished"`
	GeneralRemark *string           `json:"GeneralRemark"`
	Tasks         []UpdateTaskInput `json:"tasks"`
}

type UpdateTaskInput struct {
	Id                 int64    `json:"Id"` // 0 / omitted = new task
	Type               string   `json:"Type"`
	Address            string   `json:"Address"`
	DestinationAddress string   `json:"DestinationAddress"`
	Product            string   `json:"Product"`
	TankStatus         string   `json:"TankStatus"`
	Remark             string   `json:"Remark"`
	Start              FlexTime `json:"Start"`
	End                FlexTime `json:"End"`
	LoadReference      string   `json:"LoadReference"`
	LoadStartDate      FlexTime `json:"LoadStartDate"`
	LoadEndDate        FlexTime `json:"LoadEndDate"`
	UnloadReference    string   `json:"UnloadReference"`
	UnloadStartDate    FlexTime `json:"UnloadStartDate"`
	UnloadEndDate      FlexTime `json:"UnloadEndDate"`
	EditStatus         string   `json:"EditStatus"`
}

func (ft *FlexTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		ft.Valid = false
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	ft.Time = t
	ft.Valid = true
	return nil
}

func (ft FlexTime) MarshalJSON() ([]byte, error) {
	if !ft.Valid {
		return []byte(`""`), nil
	}
	return json.Marshal(ft.Time.Format(time.RFC3339))
}

func (ft FlexTime) Ptr() *time.Time {
	if !ft.Valid {
		return nil
	}
	t := ft.Time
	return &t
}
