package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"
)

// Duration for 1h0m rough format
type Duration struct {
	time.Duration
}

type DurationFormat string

const (
	ForDB           DurationFormat = "%dh%dm"    //<hours>h<minutes>m
	ForPresentation DurationFormat = "%02d:%02d" // <hours>:<minutes> годин/hours/godzin
)

func NewDuration(hours, minutes int) Duration {
	return Duration{
		Duration: time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute,
	}
}

func NewDurationFromString(hours, minutes string) Duration {
	hourD, err := strconv.Atoi(hours)
	if err != nil {
		log.Printf("Invalid format: %v\n", hours)
		return Duration{}
	}
	minuteD, err := strconv.Atoi(minutes)
	if err != nil {
		log.Printf("Invalid format: %v\n", minutes)
		return Duration{}
	}

	return NewDuration(hourD, minuteD)
}

// uses ForDB format
func (d *Duration) String() string {
	if d.Duration == 0 {
		return "0h0m"
	}

	d.Duration = d.Duration.Round(time.Minute)

	hours := int(d.Duration.Hours())
	minutes := int(d.Duration.Minutes()) % 60

	return fmt.Sprintf("%dh%dm", hours, minutes)
}

// either ForDB or ForPresentation formats available
func (d *Duration) Format(layout DurationFormat) string {
	d.Duration = d.Duration.Round(time.Minute)

	return fmt.Sprintf(string(layout), int(d.Hours()), int(d.Minutes())%60)
}

// Scan implements sql.Scanner for reading from database
func (d *Duration) Scan(value interface{}) error {
	if value == nil {
		d.Duration = 0
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("Duration.Scan: expected string, got %T", value)
	}

	duration, err := time.ParseDuration(str)
	if err != nil {
		return fmt.Errorf("Duration.Scan: %w", err)
	}

	d.Duration = duration
	return nil
}

// Value implements driver.Valuer for writing to database
func (d *Duration) Value() (driver.Value, error) {
	return d.String(), nil
}

// MarshalJSON for JSON serialization
func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON for JSON deserialization
func (d *Duration) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	duration, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	d.Duration = duration
	return nil
}
