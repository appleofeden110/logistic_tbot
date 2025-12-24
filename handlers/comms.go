package handlers

import (
	"fmt"
	"logistictbot/docs"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func CreateVideoToSend(chatId int64, videoName string) *tgbotapi.VideoConfig {
	videoFP := tgbotapi.FilePath("./videos/" + videoName)

	video := tgbotapi.NewVideo(chatId, videoFP)
	return &video
}

func sendDocumentsToManager(
	chatID int64,
	docsFiles []*docs.File,
) error {

	for _, f := range docsFiles {
		doc := tgbotapi.NewDocument(chatID, tgbotapi.FileID(f.TgFileId))
		doc.Caption = f.OriginalName

		if _, err := Bot.Send(doc); err != nil {
			return fmt.Errorf("send document: %w", err)
		}
	}

	return nil
}

func sendPhotosToManager(
	chatID int64,
	photos []*docs.File,
	caption string,
) error {

	const maxGroupSize = 10

	for i := 0; i < len(photos); i += maxGroupSize {
		end := i + maxGroupSize
		if end > len(photos) {
			end = len(photos)
		}

		group := tgbotapi.MediaGroupConfig{
			ChatID: chatID,
		}

		for j, f := range photos[i:end] {
			photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FileID(f.TgFileId))
			// Caption only on first photo of first group
			if i == 0 && j == 0 {
				photo.Caption = caption
				photo.ParseMode = tgbotapi.ModeHTML
			}

			group.Media = append(group.Media, photo)
		}

		if _, err := Bot.SendMediaGroup(group); err != nil {
			return fmt.Errorf("send photo group: %w", err)
		}
	}

	return nil
}

func splitFiles(files []*docs.File) (photos, docsFiles []*docs.File) {
	for _, f := range files {
		switch f.Filetype {
		case docs.Image:
			photos = append(photos, f)
		case docs.Document:
			docsFiles = append(docsFiles, f)
		}
	}
	return
}
