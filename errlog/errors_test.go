package errlog

import (
	"errors"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestErrMonitor(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "https://api.telegram.org/bot"
	token := os.Getenv("LOG_BOT_API")
	logWriter := NewLogWriter(fmt.Sprintf("%s%s/sendMessage", endpoint, token))
	t.Log(logWriter.endpoint)

	ERR := &ErrMonitor{log.New(logWriter, "test: ", 0)}

	message := "ERR: you are screwed!"

	n, err := ERR.Writer().Write([]byte(message))
	if err != nil {
		if errors.Is(err, ErrNoConnection) {
			t.Fatal(err)
		}
		t.Error(err)
	}
	if n != len(message) {
		t.Errorf("message is too short, should be %d, and it is: %d", len(message), n)
	}
	err = ERR.Output(2, string(message))
	if err != nil {
		t.Fatal(err)
	}
	// ERR.Print(message)
}
