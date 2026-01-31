package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GetOutDocsPath() string {
	if path := os.Getenv("OUTDOCS_PATH"); path != "" {
		return path
	}

	return "./storage/"
}

func GetFullPath(filename string) string {
	return filepath.Join(GetOutDocsPath(), filename)
}

func DownloadFile(url, fileName string) (string, error) {
	var fullPath string

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	fullPath = GetFullPath(fileName)
	out, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write to file: %w", err)
	}

	return fullPath, nil
}

func VERY_BAD(chatId int64, bot *tgbotapi.BotAPI) {
	bot.Send(tgbotapi.NewMessage(chatId, "Щось пішло сильно не так, якщо подальші дії бот не сприйматиме - пишіть або звоніть розробнику: @pinkfloydfan або +447990932300"))
}
