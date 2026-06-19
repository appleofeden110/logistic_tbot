package delq

import (
	"database/sql"
	"os"
	"strconv"
	"testing"

	tgbotapi "github.com/appleofeden110/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func TestDeleteWorker(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		t.Fatal(err)
	}

	chat_id_str := os.Getenv("LOG_BOT_GROUP_CHAT_ID")
	chat_id, err := strconv.ParseInt(chat_id_str, 10, 64)
	if err != nil {
		t.Fatal(err)
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_API"))
	if err != nil {
		t.Fatalf("ERR: creating new bot api: %v\n", err)
	}

	globalStorage, err := sql.Open("sqlite3", "../bot.db")
	if err != nil {
		t.Error("ERR: ", err)
	}
	defer globalStorage.Close()

	go DeleteWorker(globalStorage, bot)

	msg := tgbotapi.NewMessage(chat_id, "TEST_EDIT")
	sent, err := bot.Send(msg)
	if err != nil {
		t.Error(err)
	}

	EnqueueToDelete(globalStorage, sent.Chat.ID, sent.MessageID, Requirements{
		Type:                 TaskEdited,
		TrackedEditMessageId: sent.MessageID,
	})

}
