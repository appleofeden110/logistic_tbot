package tracking

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type State string

var (
	ErrNotLiveLocation = errors.New("локація не активна")
)

type TrackingSession struct {
	ChatID            int64
	Message           *tgbotapi.Message
	LiveLocationMsgID int
	TotalDistance     float64
	LastLat           float64
	LastLon           float64
	LastPeriod        int
	FirstLocation     bool
	IsAlive           bool
	StopChan          chan bool
	mu                sync.Mutex
}

func StartTracking(bot *tgbotapi.BotAPI, chatId int64, existingMsg *tgbotapi.Message) *TrackingSession {
	existsingMsgText := existingMsg.Text

	msg := tgbotapi.NewEditMessageText(chatId, existingMsg.MessageID, fmt.Sprintf("%s\n\nЗагальна дистанція (початкова): 0.00 км", existsingMsgText))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = existingMsg.ReplyMarkup
	sent, err := bot.Send(msg)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatId, "ERR: "+err.Error()))
	}

	sesh := &TrackingSession{
		ChatID:        chatId,
		Message:       &sent,
		TotalDistance: 0,
		FirstLocation: true,
		StopChan:      make(chan bool),
	}

	go sesh.UpdateLoop(bot, existsingMsgText)

	return sesh
}

func (t *TrackingSession) UpdateLoop(bot *tgbotapi.BotAPI, existingMsgText string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			text := fmt.Sprintf("%s\n\nЗагальна дистанція (обновлюється): %.2f км", existingMsgText, t.TotalDistance)
			t.mu.Unlock()

			edit := tgbotapi.NewEditMessageText(t.ChatID, t.Message.MessageID, text)
			edit.ParseMode = tgbotapi.ModeHTML
			edit.ReplyMarkup = t.Message.ReplyMarkup

			bot.Send(edit)
		case <-t.StopChan:
			t.mu.Lock()
			text := fmt.Sprintf("%s\n\nЗагальна дистанція (кінцева): %.2fkm", existingMsgText, t.TotalDistance)
			t.mu.Unlock()

			edit := tgbotapi.NewEditMessageText(t.ChatID, t.Message.MessageID, text)
			edit.ParseMode = tgbotapi.ModeHTML
			edit.ReplyMarkup = t.Message.ReplyMarkup

			bot.Send(edit)
			return
		}

	}
}

func (t *TrackingSession) UpdateLocation(lat, lon float64, livePeriod int, bot *tgbotapi.BotAPI) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	log.Println("Last period: ", t.LastPeriod, livePeriod)

	if !t.FirstLocation {
		distance := Haversine(t.LastLat, t.LastLon, lat, lon)
		t.TotalDistance += distance

		if livePeriod < t.LastPeriod {
			log.Println("live location has stopped: ", livePeriod, t.LastPeriod)
			t.IsAlive = false
		} else if livePeriod > t.LastPeriod {
			panic(fmt.Errorf("something went terribly wrong: %d, %d", livePeriod, t.LastPeriod))
		}

		t.LastPeriod = livePeriod
	} else {
		t.LastPeriod = livePeriod
		t.FirstLocation = false
	}

	t.LastLat = lat
	t.LastLon = lon
	return nil
}

func (t *TrackingSession) Stop() {
	close(t.StopChan)
}
