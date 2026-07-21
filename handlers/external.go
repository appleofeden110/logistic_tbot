package handlers

import (
	"database/sql"
	"logistictbot/config"
	"logistictbot/db"
	"net/http"
	"strings"
)

func WithAuth(globalStorage *sql.DB, botToken string, next func(w http.ResponseWriter, r *http.Request, u *db.User, globalStorage *sql.DB)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hdr := r.Header.Get("Authorization")
		if !strings.HasPrefix(hdr, "tma ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		initData := strings.TrimPrefix(hdr, "tma ")

		chatId, err := config.VerifyInitData(initData, botToken)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		u := new(db.User)
		u.ChatId = chatId

		err = u.GetUserByChatId(globalStorage)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		next(w, r, u, globalStorage)
	}
}
