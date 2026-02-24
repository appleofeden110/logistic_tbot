package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Lang map[string]string
type LangCode string // can be en, pl or ua. If you wanna add languages -> use https://en.wikipedia.org/wiki/IETF_language_tag

const (
	English   LangCode = "en"
	Polish    LangCode = "pl"
	Ukrainian LangCode = "uk"
)

var (
	locales          = map[LangCode]Lang{}
	UsersLanguages   = make(map[int64]LangCode) // chat_id -> LangCode
	UsersLanguagesMu sync.RWMutex
)

func GetLang(chatId int64) LangCode {
	UsersLanguagesMu.RLock()
	defer UsersLanguagesMu.RUnlock()
	if lang, ok := UsersLanguages[chatId]; ok {
		return lang
	}
	return Ukrainian
}

func SetUserLang(chatId int64, lang LangCode) {
	UsersLanguagesMu.Lock()
	defer UsersLanguagesMu.Unlock()
	UsersLanguages[chatId] = lang
}

func LoadLocales() error {
	for _, lang := range []LangCode{Ukrainian, Polish, English} {
		data, err := os.ReadFile("locales/" + string(lang) + ".json")
		if err != nil {
			return err
		}
		var l Lang
		if err := json.Unmarshal(data, &l); err != nil {
			return err
		}
		locales[lang] = l
	}
	return nil
}

// Translate literally uses Sprintf for its strings. In locale files just use regular Go formatting, same as with fmt.Printf or fmt.Sprintf
func Translate(userLang LangCode, key string, args ...any) string {
	l, ok := locales[userLang]

	if !ok {
		l = locales["en"]
	}

	str, ok := l[key]
	if !ok {
		return key
	}

	return fmt.Sprintf(str, args...)
}
