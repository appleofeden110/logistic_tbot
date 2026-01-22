package main

import (
	"context"
	"database/sql"
	"log"
	"logistictbot/db"
	"logistictbot/handlers"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

// Excel tables - today
// users perfect - today
// shipments perfect - today
// in group? maybe later - later, maybe after stage 1
// direct comms between manager and driver probs - later, maybe tomorrow

func main() {
	var err error
	err = godotenv.Load()

	globalStorage, err := sql.Open("sqlite3", "./bot.db")
	if err != nil {
		log.Fatal(err)
	}
	defer globalStorage.Close()

	_, err = globalStorage.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		log.Fatal(err)
	}

	err = db.CheckAllTables(globalStorage)
	if err != nil {
		log.Fatalf("Err with the table: %v\n", err)
	}

	err = handlers.FillSessions(globalStorage)
	if err != nil {
		log.Fatalf("Err filling sessions: %v\n", err)
	}

	handlers.Bot, err = tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_API"))
	handlers.Check(err, true)

	handlers.Bot.Debug = false
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := handlers.Bot.GetUpdatesChan(u)

	_, err = tgbotapi.NewWebhook("")
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go handlers.ReceiveUpdates(ctx, updates, globalStorage)

	// Start the HTTP server and listen on the PORT specified by Heroku
	port := os.Getenv("PORT")
	if port == "" {
		port = "8443" // Use a default port if PORT is not set
	}
	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
