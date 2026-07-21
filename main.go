package main

import (
	"context"
	"database/sql"
	"log"
	"logistictbot/config"
	"logistictbot/db"
	"logistictbot/delq"
	"logistictbot/errlog"

	"logistictbot/handlers"
	"net/http"
	"os"

	tgbotapi "github.com/appleofeden110/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var err error
	err = godotenv.Load()

	mux := http.NewServeMux()

	errlog.WARN.Logger = log.New(errlog.NewLogWriter(errlog.API_URL+os.Getenv("LOG_BOT_API")+"/sendMessage"), "WARN: ", 0)
	errlog.ERR.Logger = log.New(errlog.NewLogWriter(errlog.API_URL+os.Getenv("LOG_BOT_API")+"/sendMessage"), "ERR: ", 0)
	errlog.INFO.Logger = log.New(errlog.NewLogWriter(errlog.API_URL+os.Getenv("LOG_BOT_API")+"/sendMessage"), "INFO: ", 0)

	//handlers for the api

	mux.HandleFunc("GET /telegram/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Success")
		w.WriteHeader(200)
	})

	err = config.LoadLocales()
	if err != nil {
		errlog.ERR.Fatalf("loading the locales: %v\n", err)
		return
	}

	globalStorage, err := sql.Open("sqlite3", "./bot.db")
	if err != nil {
		log.Fatalln("ERR: ", err)
	}
	defer globalStorage.Close()

	_, err = globalStorage.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		log.Fatalln("ERR: ", err)
	}

	err = db.CheckAllTables(globalStorage)
	if err != nil {
		log.Fatalf("ERR: with the table: %v\n", err)
	}

	err = handlers.FillSessions(globalStorage)
	if err != nil {
		log.Fatalf("ERR: filling sessions: %v\n", err)
	}

	handlers.Bot, err = tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_API"))
	if err != nil {
		log.Fatalf("ERR: creating new bot api: %v\n", err)
	}

	handlers.Bot.Debug = false
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := handlers.Bot.GetUpdatesChan(u)

	if os.Getenv("ENV") == "dev" {
		_, err = handlers.Bot.Send(tgbotapi.DeleteWebhookConfig{})
		if err != nil {
			log.Println(err)
		}
	} else {
		_, err = tgbotapi.NewWebhook("")
		if err != nil {
			panic(err)
		}
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go handlers.PingNonReplies(globalStorage)
	go delq.DeleteWorker(globalStorage, handlers.Bot)
	go handlers.ReceiveUpdates(ctx, updates, globalStorage)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8443"
	}
	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("ERR:Failed to start server: %v", err)
	}
}
