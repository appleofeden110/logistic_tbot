package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"logistictbot/config"
	"logistictbot/db"
	"strconv"
	"strings"

	tgbotapi "github.com/appleofeden110/telegram-bot-api/v5"
)

func HandleGroupCommands(chatId int64, command string, messageId int, fromId int64, globalStorage *sql.DB, topicId int) error {
	cmd, f := strings.CutPrefix(command, "g:")
	if !f {
		log.Println("ERR: There should a prefix for group command, but there is none")
		return nil
	}

	if groupChatIdToCar, f := strings.CutPrefix(cmd, "car:"); f {
		temp := strings.Split(groupChatIdToCar, ":")
		if temp[0] == groupChatIdToCar {
			log.Println("ERR: g:car:gchatid:car is incorrect, cannot find : between gchatid and car")
			return nil
		}
		groupChatId, err := strconv.ParseInt(strings.TrimSpace(temp[0]), 10, 64)
		if err != nil {
			return fmt.Errorf("ERR: there should be a number here!")
		}
		carId := temp[1]

		car, err := db.GetCarById(globalStorage, carId)
		if err != nil {
			return err
		}

		g := db.DriverGroup{CurrentCar: car, GroupChatId: groupChatId}
		err = g.CreateDriverGroup(globalStorage)
		if err != nil {
			return fmt.Errorf("ERR: Creating driver group: %v\n", err)
		}
		driver, err := db.GetDriverById(globalStorage, car.CurrentDriverId)
		if err != nil {
			return fmt.Errorf("ERR: getting driver by his id from the car object: %v\n", err)
		}

		_, err = Bot.Send(
			tgbotapi.NewMessage(
				groupChatId,
				config.Translate(config.GetLang(driver.ChatId), "group_inited", car.Id),
			),
		)
	}

	return nil
}
