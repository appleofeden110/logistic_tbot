package handlers

import (
	"fmt"
	"log"
	"logistictbot/config"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func LogTelegramMessage(msg *tgbotapi.Message) {
	if msg == nil {
		log.Println("Received nil message")
		return
	}

	var logLines []string

	messageType := determineMessageType(msg)
	logLines = append(logLines, fmt.Sprintf("%s::MsgID:%d", messageType, msg.MessageID))
	logLines = append(logLines, fmt.Sprintf("ChatID:%d (%s)", msg.Chat.ID, msg.Chat.Type))

	if msg.From != nil {
		logLines = append(logLines, fmt.Sprintf("From:%s %s(@%s, ID: %d)",
			msg.From.FirstName, msg.From.LastName, msg.From.UserName, msg.From.ID))
	}

	logLines = append(logLines, fmt.Sprintf("Time:%s", msg.Time().Format("02/01/2006 15:04:05")))

	if msg.Text != "" {
		logLines = append(logLines, fmt.Sprintf("Text:%s", msg.Text))
	}
	if msg.Caption != "" {
		logLines = append(logLines, fmt.Sprintf("Caption:%s", msg.Caption))
	}

	if msg.Document != nil {
		logLines = append(logLines, fmt.Sprintf("File:%s (%s, %d bytes)",
			msg.Document.FileName, msg.Document.MimeType, msg.Document.FileSize))
	}

	if msg.Photo != nil && len(msg.Photo) > 0 {
		p := msg.Photo[len(msg.Photo)-1]
		logLines = append(logLines, fmt.Sprintf("Photo:%dx%d (%d bytes)", p.Width, p.Height, p.FileSize))
	}

	if msg.Video != nil {
		logLines = append(logLines, fmt.Sprintf("Video:%dx%d, %ds (%d bytes-%s)",
			msg.Video.Width, msg.Video.Height, msg.Video.Duration, msg.Video.FileSize, msg.Video.MimeType))
	}

	if msg.MediaGroupID != "" {
		logLines = append(logLines, fmt.Sprintf("MediaGroup:%s", msg.MediaGroupID))
	}

	if msg.ReplyToMessage != nil {
		logLines = append(logLines, fmt.Sprintf("ReplyToMsgID: %d", msg.ReplyToMessage.MessageID))
	}

	config.WriteLogs(strings.Join(logLines, " | "))
}

func determineMessageType(msg *tgbotapi.Message) string {
	if msg.Document != nil {
		return "DOC"
	}
	if msg.Photo != nil && len(msg.Photo) > 0 {
		if msg.MediaGroupID != "" {
			return "PHOTO(Group)"
		}
		return "PHOTO"
	}
	if msg.Video != nil {
		return "VIDEO"
	}
	if msg.Audio != nil {
		return "AUDIO"
	}
	if msg.Voice != nil {
		return "VOICE"
	}
	if msg.Sticker != nil {
		return "STICKER"
	}
	if msg.Text != "" {
		if len(msg.Entities) > 0 && msg.Entities[0].Type == "bot_command" {
			return "CMD"
		}
		return "TEXT"
	}
	return "OTHER"
}
