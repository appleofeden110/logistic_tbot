package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"logistictbot/config"
	"logistictbot/db"
	"strconv"
	"strings"

	tgbotapi "github.com/appleofeden110/telegram-bot-api/v5"
	"github.com/gofrs/uuid"
)

type ChatMemberResult struct {
	User        tgbotapi.User
	Status      string `json:"status"` // can be administrator, member, kicked, left, creator, restricted
	IsAnonymous bool   `json:"is_anonymous"`
}

func FindTankTopic(chatId int64, globalStorage *sql.DB) int {
	topicId := 0
	if strings.Contains(strconv.Itoa(int(chatId)), "-100") {
		g := db.DriverGroup{GroupChatId: chatId}
		err := g.GetDriverGroup(globalStorage)
		if err != nil {
			log.Printf("ERR: getting loading topic: %v\n", err)
			return 0
		}
		topicId = g.TankTopicId
	}

	return topicId
}

func FindLoadingTopic(chatId int64, globalStorage *sql.DB) int {
	topicId := 0
	if strings.Contains(strconv.Itoa(int(chatId)), "-100") {
		g := db.DriverGroup{GroupChatId: chatId}
		err := g.GetDriverGroup(globalStorage)
		if err != nil {
			log.Printf("ERR: getting loading topic: %v\n", err)
			return 0
		}
		topicId = g.LoadingTopicId
	}

	return topicId
}

func HandleGroupCommands(chatId int64, command string, messageId int, user *tgbotapi.User, globalStorage *sql.DB, topicId int) error {
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
				topicId,
			),
		)
	}

	if cmd == "register" {
		u := db.User{ChatId: user.ID, Name: fmt.Sprintf("%s %s", user.FirstName, user.LastName), TgTag: user.UserName}
		err := u.StoreUser(globalStorage)
		if err != nil {
			return fmt.Errorf("ERR: storing user for a group: %v\n", err)
		}

		users, err := db.GetAllUsers(globalStorage)
		if err != nil {
			return err
		}
		superAdmins, err := GetAllSuperAdminsOfGroup(chatId, users)
		if err != nil {
			return fmt.Errorf("ERR: getting all super admins of a group: %v\n", err)
		}

		var superAdminList string
		for _, sa := range superAdmins {
			superAdminList += fmt.Sprintf("@%s ", sa.TgTag)
		}

		roleMsg := tgbotapi.NewMessage(chatId, config.Translate(config.GetLang(chatId), "g:sa:assign_role", superAdminList, u.Name), topicId)
		roleMsg.ParseMode = tgbotapi.ModeHTML
		roleMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.GetLang(chatId), "g:sa:assign_role:driver"), fmt.Sprintf("g:sa:assign_role:d:%s", u.Id)),
				tgbotapi.NewInlineKeyboardButtonData(config.Translate(config.GetLang(chatId), "g:sa:assign_role:manager"), fmt.Sprintf("g:sa:assign_role:m:%s", u.Id)),
			),
		)

		Bot.Send(roleMsg)
	}

	if command, f := strings.CutPrefix(cmd, "sa:"); f {
		u := &db.User{ChatId: user.ID}
		if err := u.FindSuperAdmin(globalStorage); err != nil {
			if errors.Is(err, db.ErrNotSuperAdmin) {
				Bot.Send(tgbotapi.NewMessage(chatId, config.Translate(config.GetLang(chatId), "g:not_sa"), topicId))
			}
			return fmt.Errorf("ERR: err finding a sa for a group for a command: %v\n", err)
		}

		if userIdStr, f := strings.CutPrefix(command, "assign_role:d:"); f {
			driverUser := &db.User{Id: uuid.FromStringOrNil(userIdStr)}
			err := driverUser.GetUserById(globalStorage)
			if err != nil {
				return fmt.Errorf("ERR: getting user by id: %v\n", err)
			}
			d := &db.Driver{UserId: driverUser.Id, ChatId: driverUser.ChatId}
			err = d.StoreDriver(globalStorage, Bot)
			if err != nil {
				return fmt.Errorf("ERR: storing driver: %v\n", err)
			}

			carQuestion := tgbotapi.NewMessage(chatId, config.Translate(config.GetLang(chatId), "car_question"), topicId)
			cars, err := db.GetAllCars(globalStorage)
			if err != nil {
				return fmt.Errorf("Fetching cars for question: %v\n", err)
			}
			rows := make([][]tgbotapi.InlineKeyboardButton, 0)
			for _, v := range cars {
				car := fmt.Sprintf("%s (%d km)", v.Id, v.Kilometrage)
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(car, fmt.Sprintf("g:sa:carfor:%s:%s", userIdStr, v.Id)),
				))
			}
			carQuestion.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
			carQuestion.ParseMode = tgbotapi.ModeHTML
			_, err = Bot.Send(carQuestion)
			return err
		}

		if userIdStr, f := strings.CutPrefix(command, "assign_role:m:"); f {
			driverUser := &db.User{Id: uuid.FromStringOrNil(userIdStr)}
			err := driverUser.GetUserById(globalStorage)
			if err != nil {
				return fmt.Errorf("ERR: getting user by id: %v\n", err)
			}
			m := &db.Manager{UserId: driverUser.Id, ChatId: driverUser.ChatId}
			err = m.StoreManager(globalStorage, Bot)
			if err != nil {
				return fmt.Errorf("ERR: storing manager: %v\n", err)
			}
			_, err = Bot.Send(tgbotapi.NewMessage(chatId, config.Translate(config.GetLang(chatId), "registration_accepted:managertoSA", driverUser.Name), topicId))
			return err
		}

		if _idString, f := strings.CutPrefix(command, "carfor:"); f {
			driverUserId, carId, f := strings.Cut(_idString, ":")
			if !f {
				return fmt.Errorf("ERR: getting any info from sa:carfor command: %v\n", command)
			}
			u := &db.User{Id: uuid.FromStringOrNil(driverUserId)}

			err := u.GetUserById(globalStorage)
			if err != nil {
				return fmt.Errorf("ERR: getting driver by id: %v\n", err)
			}
			driver, err := db.GetDriverById(globalStorage, u.DriverId)
			if err != nil {
				return fmt.Errorf("ERR: getting driver by id: %v\n", err)
			}
			err = driver.UpdateCarId(globalStorage, carId)
			if err != nil {
				return fmt.Errorf("ERR: updating car id: %v\n", err)
			}
			car, err := db.GetCarById(globalStorage, carId)
			if err != nil {
				return fmt.Errorf("ERR: getting car by id: %v\n", err)
			}
			Bot.Send(tgbotapi.NewMessage(chatId, config.Translate(config.GetLang(chatId),
				"registration_accepted:drivertoSA",
				driver.User.Name,
			), topicId))
			driverSessionsMu.Lock()
			driverSessions[driver.ChatId] = driver
			driverSessionsMu.Unlock()
			_, err = Bot.Send(tgbotapi.NewMessage(driver.User.ChatId,
				config.Translate(config.GetLang(driver.User.ChatId),
					"registration_accepted:driver",
					car.Id, int(car.Kilometrage), topicId)),
			)
			if err != nil {
				return fmt.Errorf("ERR: sending driver a msg: %v\n", err)
			}
			return HandleMenu(chatId, globalStorage, driver.User)
		}
	}

	return nil
}

func GetAllSuperAdminsOfGroup(groupChatId int64, users []*db.User) ([]*db.User, error) {
	superAdmins := make([]*db.User, 0)
	for _, us := range users {

		resp, err := Bot.Request(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: groupChatId, UserID: us.ChatId}})
		if err != nil {
			return nil, fmt.Errorf("ERR: gettin")
		}

		var member ChatMemberResult
		if err := json.Unmarshal(resp.Result, &member); err != nil {
			return nil, fmt.Errorf("ERR: getting chat member: %v\n", err)
		}

		if member.Status == "left" || member.Status == "kicked" {
			continue
		}

		if !us.IsSuperAdmin {
			continue
		}

		superAdmins = append(superAdmins, us)

	}
	return superAdmins, nil
}

func GetAllManagersOfGroup(groupChatId int64, users []*db.User) ([]*db.User, error) {
	managers := make([]*db.User, 0)
	for _, us := range users {

		resp, err := Bot.Request(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: groupChatId, UserID: us.ChatId}})
		if err != nil {
			return nil, fmt.Errorf("ERR: gettin")
		}

		var member ChatMemberResult
		if err := json.Unmarshal(resp.Result, &member); err != nil {
			return nil, fmt.Errorf("ERR: getting chat member: %v\n", err)
		}

		if member.Status == "left" || member.Status == "kicked" {
			continue
		}

		if us.ManagerId.IsNil() {
			continue
		}

		managers = append(managers, us)

	}
	return managers, nil
}
