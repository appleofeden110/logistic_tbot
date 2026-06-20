package db

import (
	"logistictbot/config"
	"testing"

	tgbotapi "github.com/appleofeden110/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
)

func TestTagPerson(t *testing.T) {
	chat_id, bot, globalStorage, err := config.LoadEverythingForTest()
	if err != nil {
		t.Fatal(err)
	}
	defer globalStorage.Close()

	u := User{ChatId: chat_id}
	if err := u.GetUserByChatId(globalStorage); err != nil {
		t.Fatal(err)
	}

	u.TgTag = ""

	msg := tgbotapi.NewMessage(chat_id, u.TagPerson()+", твій мозок здоровий, бро?")
	msg.ParseMode = tgbotapi.ModeHTML

	res, err := bot.Request(msg)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(res.Ok, res.ErrorCode, string(res.Result))
}
