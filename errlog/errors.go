package errlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const API_URL = "https://api.telegram.org/bot"

type LogWriter struct {
	endpoint string
	Client   *http.Client
}

type LogMessage struct {
	Text   string `json:"text"`
	ChatId int64  `json:"chat_id"`
}

type ErrMonitor struct {
	*log.Logger
}

var (
	ErrNoConnection = errors.New("You don't have internet connection to launch this bot, Telegram API has timed out")

	TOKEN string

	WARN = new(ErrMonitor)
	ERR  = new(ErrMonitor)
	INFO = new(ErrMonitor)
)

func NewLogWriter(endpoint string) *LogWriter {
	return &LogWriter{endpoint: endpoint, Client: &http.Client{}}
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	chat_id, err := strconv.Atoi(os.Getenv("LOG_BOT_CHAT_ID"))
	if err != nil {
		log.Println("CHAT_ID could not be read from the .env")
		return 0, err
	}

	if p[len(p)-1] == '\n' {
		p = p[:len(p)-1]
	}

	msg := LogMessage{
		ChatId: int64(chat_id),
		Text:   string(p),
	}

	data, err := json.Marshal(&msg)
	if err != nil {
		log.Println("LOG WRITE FUNC ERR: ", err)
		return 0, err
	}

	resp, err := w.Client.Post(w.endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		if strings.Contains(err.Error(), "i/o timeout") {
			return 0, ErrNoConnection
		}
		return 0, err
	}

	if resp.StatusCode == 400 {
		return 0, fmt.Errorf("incorrectly sent message, most likely wrong struct name or json tag")
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, err
	}

	return len(p), nil
}
