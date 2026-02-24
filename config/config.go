package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const maxLogSize = 20 * 1024 * 1024 // 20mb

func GetOutDocsPath() string {
	if path := os.Getenv("OUTDOCS_PATH"); path != "" {
		return path
	}

	return "./storage/"
}

func GetLogsPath() string {
	if path := os.Getenv("LOGS_PATH"); path != "" {
		return path
	}

	return "./logs/"
}

func GetFullPathOutDocs(filename string) string {
	return filepath.Join(GetOutDocsPath(), filename)
}

func WriteLogs(line string) {
	logsPath := GetLogsPath()

	if err := os.MkdirAll(logsPath, 0750); err != nil {
		fmt.Printf("ERR: failed to create logs directory: %v\n", err)
		return
	}

	allFiles, err := os.ReadDir(logsPath)
	if err != nil {
		fmt.Printf("ERR: failed to read logs directory: %v\n", err)
		return
	}

	var logFiles []os.DirEntry
	for _, file := range allFiles {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "chat") && strings.HasSuffix(file.Name(), ".log") {
			logFiles = append(logFiles, file)
		}
	}

	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].Name() < logFiles[j].Name()
	})

	fileNumber := 0

	if len(logFiles) > 0 {
		lastLog := logFiles[len(logFiles)-1]
		lastLogInfo, err := lastLog.Info()
		if err != nil {
			fmt.Printf("ERR: failed to stat log file: %v\n", err)
			return
		}

		name := lastLog.Name()
		if strings.Contains(name, "_") {
			parts := strings.Split(strings.TrimSuffix(name, ".log"), "_")
			if len(parts) == 2 {
				fileNumber, _ = strconv.Atoi(parts[1])
			}
		}

		if lastLogInfo.Size() >= maxLogSize {
			fileNumber++
		}
	}

	var filename string
	if fileNumber == 0 {
		filename = "chat.log"
	} else {
		filename = fmt.Sprintf("chat_%d.log", fileNumber)
	}

	logPath := filepath.Join(logsPath, filename)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("ERR: failed to open log file (%s): %v\n", filename, err)
		return
	}
	defer f.Close()

	logLine := fmt.Sprintf("[%s] %s\n", time.Now().Format("02/01/2006 15:04:05"), line)
	if _, err := f.WriteString(logLine); err != nil {
		fmt.Printf("ERR: failed to write to log file (%s): %v\n", filename, err)
		return
	}
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

	fullPath = GetFullPathOutDocs(fileName)
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
	bot.Send(tgbotapi.NewMessage(chatId, Translate(GetLang(chatId), "VERY_BAD")))
}
