package parser

import (
	"encoding/json"
	"fmt"
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
	if s == "" || s == "0001-01-01T00:00:00Z" {
		ft.Valid = false
		return nil
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}

	var lastErr error
	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			ft.Time = t
			ft.Valid = true
			return nil
		}
		lastErr = err
	}
	return fmt.Errorf("unable to parse time %q: %v", s, lastErr)
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
