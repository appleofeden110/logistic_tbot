package db

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofrs/uuid"
)

type Car struct {
	Id              string    `db:"id" form:"Введіть повний номер машини (Приклад: AB123CD 456789)"`
	CurrentDriverId uuid.UUID `db:"current_driver"`
	Kilometrage     int64     `db:"current_kilometrage" form:"Введіть поточний кілометраж однією цифрою"`
}

func GetAllCars(exec DBExecutor) ([]*Car, error) {
	cars := make([]*Car, 0)
	rows, err := exec.Query("SELECT * FROM cars")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("No cars in the database")
		}
		return nil, err

	}
	defer rows.Close()

	for rows.Next() {
		car := new(Car)
		var currentDriverId sql.NullString

		err = rows.Scan(&car.Id, &currentDriverId, &car.Kilometrage)
		if err != nil {
			return nil, fmt.Errorf("fetching cars: %v\n", err)
		}

		if currentDriverId.Valid {
			car.CurrentDriverId, err = uuid.FromString(currentDriverId.String)
			if err != nil {
				return nil, fmt.Errorf("uuid from string, fetching cars: %v\n", err)
			}

		} else {
			car.CurrentDriverId = uuid.Nil
		}

		cars = append(cars, car)
	}

	return cars, nil
}

func GetCarById(exec DBExecutor, carId string) (*Car, error) {
	car := new(Car)
	var currentDriverId sql.NullString

	row := exec.QueryRow("SELECT id, current_driver, current_kilometrage FROM cars WHERE id = ?", carId)

	err := row.Scan(&car.Id, &currentDriverId, &car.Kilometrage)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("car with id %s not found", carId)
		}
		return nil, fmt.Errorf("ERR: scanning car: %w", err)
	}

	if currentDriverId.Valid {
		car.CurrentDriverId, err = uuid.FromString(currentDriverId.String)
		if err != nil {
			return nil, fmt.Errorf("ERR: parsing current_driver_id: %w", err)
		}
	} else {
		car.CurrentDriverId = uuid.Nil
	}

	return car, nil
}

func (c *Car) UpdateCarKilometrage(exec DBExecutor) error {
	_, err := exec.Exec(`
	UPDATE cars
	SET current_kilometrage = ?
	WHERE id = ?`,
		c.Kilometrage, c.Id,
	)

	if err != nil {
		return fmt.Errorf("ERR: giving updating kilometrage for %s: %v\n", c.Id, err)
	}

	return nil
}

/*
ONLY FOR PEOPLE WHO ARE SUPER_ADMINS

AddCarToDB checks based on your chat_id if you are superadmin or not and it is used in pair with bot to send you messages
*/
func (c *Car) AddCarToDB(chatId int64, bot *tgbotapi.BotAPI, executor DBExecutor) error {
	u := &User{ChatId: chatId}
	err := u.FindSuperAdmin(executor)

	if err != nil {
		if errors.Is(err, ErrNotSuperAdmin) {
			bot.Send(tgbotapi.NewMessage(chatId, "Ви не є адміністратором для виконання цієї дії"))
			return ErrNotSuperAdmin
		}
		return fmt.Errorf("ERR: checking if you are SA or not. Probably not actually...: %v\n", err)
	}

	stmt, err := executor.Prepare("INSERT INTO cars (id, current_kilometrage) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("ERR: prepping stmt for adding a car to the db: %v\n", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.Id, c.Kilometrage)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			bot.Send(tgbotapi.NewMessage(chatId, "Такий автомобіль вже є в базі даних"))
		}
		return fmt.Errorf("ERR: executing stmt to add car to the db: %v\n", err)
	}

	_, err = bot.Send(tgbotapi.NewMessage(chatId, fmt.Sprintf("Успішно добавили машину %s з кілометражом %d до бази даних", c.Id, c.Kilometrage)))
	return err
}

// same as with AddCarToDB but made to be adding cars in bulk through csv files. NOT TESTED
func (c *Car) AddCarsFromTelegramCSV(chatId int64, bot *tgbotapi.BotAPI, globalStorage *sql.DB, fileID string) error {
	u := &User{ChatId: chatId}
	err := u.FindSuperAdmin(globalStorage)
	if err != nil {
		if errors.Is(err, ErrNotSuperAdmin) {
			bot.Send(tgbotapi.NewMessage(chatId, "Ви не є адміністратором для виконання цієї дії"))
			return ErrNotSuperAdmin
		}
		return fmt.Errorf("ERR: checking if you are SA: %v", err)
	}

	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return fmt.Errorf("ERR: getting file from Telegram: %v", err)
	}

	fileURL := file.Link(bot.Token)
	resp, err := http.Get(fileURL)
	if err != nil {
		return fmt.Errorf("ERR: downloading file: %v", err)
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("ERR: reading CSV: %v", err)
	}

	tx, err := globalStorage.Begin()
	if err != nil {
		return fmt.Errorf("ERR: starting transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO cars (id, current_kilometrage) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("ERR: preparing statement: %v", err)
	}
	defer stmt.Close()

	successCount := 0
	var errSlice []string

	for i, record := range records[1:] {
		if len(record) < 2 {
			errSlice = append(errSlice, fmt.Sprintf("Row %d: insufficient columns", i+2))
			continue
		}

		carId := record[0]
		kilometrage, err := strconv.ParseInt(record[1], 10, 64)
		if err != nil {
			errSlice = append(errSlice, fmt.Sprintf("Row %d: invalid kilometrage '%s'", i+2, record[1]))
			continue
		}

		_, err = stmt.Exec(carId, kilometrage)
		if err != nil {
			errSlice = append(errSlice, fmt.Sprintf("Row %d (%s): %v", i+2, carId, err))
			continue
		}
		successCount++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ERR: committing transaction: %v", err)
	}

	message := fmt.Sprintf("Успішно додано %d машин(и) до бази даних", successCount)
	if len(errSlice) > 0 {
		message += fmt.Sprintf("\n\nПомилки (%d):\n%s", len(errSlice), strings.Join(errSlice, "\n"))
	}

	bot.Send(tgbotapi.NewMessage(chatId, message))
	return nil
}

func FormatKilometrage(km int) string {
	kmString := strconv.Itoa(km)

	switch len(kmString) {
	case 4:
		kmString = kmString[0:1] + "," + kmString[1:]
	case 5:
		kmString = kmString[0:2] + "," + kmString[2:]
	case 6:
		kmString = kmString[0:3] + "," + kmString[3:]
	default:
		break
	}

	return kmString + " km"
}

func ParseKilometrage(s string) (int64, error) {
	s = strings.TrimSpace(s)

	s = strings.TrimSuffix(s, " km")
	s = strings.TrimSuffix(s, " км")
	s = strings.TrimSuffix(s, "km")
	s = strings.TrimSuffix(s, "км")
	s = strings.TrimSuffix(s, " KM")
	s = strings.TrimSuffix(s, " КМ")
	s = strings.TrimSuffix(s, "KM")
	s = strings.TrimSuffix(s, "КМ")

	s = strings.TrimSpace(s)

	var digits strings.Builder
	for _, ch := range s {
		if unicode.IsDigit(ch) {
			digits.WriteRune(ch)
		}
	}

	result := digits.String()
	if result == "" {
		return 0, fmt.Errorf("no digits found in input")
	}

	km, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse kilometrage: %v", err)
	}

	return km, nil
}

func ParseTemperature(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, " °C")
	s = strings.TrimSuffix(s, " °c")
	s = strings.TrimSuffix(s, " C")
	s = strings.TrimSuffix(s, " c")
	s = strings.TrimSuffix(s, "°C")
	s = strings.TrimSuffix(s, "°c")
	s = strings.TrimSuffix(s, "C")
	s = strings.TrimSuffix(s, "c")
	s = strings.TrimSuffix(s, " degrees")
	s = strings.TrimSuffix(s, " градусів")
	s = strings.TrimSuffix(s, " градусов")
	s = strings.TrimSpace(s)

	var digits strings.Builder
	hasDecimal := false
	hasNegative := false

	for i, ch := range s {
		if unicode.IsDigit(ch) {
			digits.WriteRune(ch)
		} else if ch == '.' || ch == ',' {
			if !hasDecimal {
				digits.WriteRune('.')
				hasDecimal = true
			}
		} else if (ch == '-' || ch == '−') && i == 0 {
			if !hasNegative {
				digits.WriteRune('-')
				hasNegative = true
			}
		}
	}

	result := digits.String()
	if result == "" || result == "-" {
		return 0, fmt.Errorf("no digits found in input")
	}

	temp, err := strconv.ParseFloat(result, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse temperature: %v", err)
	}

	return temp, nil
}

func ParseWeight(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, " kg")
	s = strings.TrimSuffix(s, " кг")
	s = strings.TrimSuffix(s, " KG")
	s = strings.TrimSuffix(s, " КГ")
	s = strings.TrimSuffix(s, "kg")
	s = strings.TrimSuffix(s, "кг")
	s = strings.TrimSuffix(s, "KG")
	s = strings.TrimSuffix(s, "КГ")
	s = strings.TrimSuffix(s, " kilograms")
	s = strings.TrimSuffix(s, " кілограм")
	s = strings.TrimSuffix(s, " килограммов")
	s = strings.TrimSpace(s)

	var digits strings.Builder
	hasDecimal := false

	for _, ch := range s {
		if unicode.IsDigit(ch) {
			digits.WriteRune(ch)
		} else if ch == '.' || ch == ',' {
			if !hasDecimal {
				digits.WriteRune('.')
				hasDecimal = true
			}
		}
	}

	result := digits.String()
	if result == "" {
		return 0, fmt.Errorf("no digits found in input")
	}

	weight, err := strconv.Atoi(result)
	if err != nil {
		return 0, fmt.Errorf("failed to parse weight: %v", err)
	}

	return weight, nil
}
