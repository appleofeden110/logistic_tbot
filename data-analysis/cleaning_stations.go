package data_analysis

import (
	"database/sql"
	"fmt"
)

type CleaningStation struct {
	Id           int     `db:"id"`
	Name         string  `db:"name"`
	Address      string  `db:"address"`
	Country      string  `db:"country"`
	Lat          float64 `db:"lat"`
	Lon          float64 `db:"lon"`
	OpeningHours string  `db:"opening_hours"`
}

func (c *CleaningStation) GetById(globalStorage *sql.DB) error {
	row := globalStorage.QueryRow(`SELECT id, name, address, country, lat, lon, opening_hours FROM cleaning_stations WHERE id = ?`, c.Id)
	return row.Scan(&c.Id, &c.Name, &c.Address, &c.Country, &c.Lat, &c.Lon, &c.OpeningHours)
}

func (c *CleaningStation) GetByName(globalStorage *sql.DB) error {
	row := globalStorage.QueryRow(`SELECT id, name, address, country, lat, lon, opening_hours FROM cleaning_stations WHERE name = ?`, c.Name)
	return row.Scan(&c.Id, &c.Name, &c.Address, &c.Country, &c.Lat, &c.Lon, &c.OpeningHours)
}

func GetAllCleaningStations(globalStorage *sql.DB) ([]CleaningStation, error) {
	rows, err := globalStorage.Query(`SELECT id, name, address, country, lat, lon, opening_hours FROM cleaning_stations`)
	if err != nil {
		return nil, fmt.Errorf("GetAllCleaningStations: %w", err)
	}
	defer rows.Close()

	var stations []CleaningStation
	for rows.Next() {
		var c CleaningStation
		if err := rows.Scan(&c.Id, &c.Name, &c.Address, &c.Country, &c.Lat, &c.Lon, &c.OpeningHours); err != nil {
			return nil, fmt.Errorf("GetAllCleaningStations scan: %w", err)
		}
		stations = append(stations, c)
	}
	return stations, rows.Err()
}
