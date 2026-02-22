package config

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

type Lang map[string]string
type LangCode string // can be en, pl or ua. If you wanna add languages -> use ISO 639 locale codes

const (
	English   LangCode = "en"
	Polish    LangCode = "pl"
	Ukrainian LangCode = "ua"
)

var (
	locales          = map[LangCode]Lang{}
	usersLanguages   = make(map[int64]LangCode) // chat_id -> LangCode
	usersLanguagesMu sync.RWMutex
)

func GetLang(chatId int64) LangCode {
	usersLanguagesMu.RLock()
	defer usersLanguagesMu.RUnlock()
	if lang, ok := usersLanguages[chatId]; ok {
		return lang
	}
	return "ua"
}

func SetUserLang(chatId int64, lang LangCode) {
	usersLanguagesMu.Lock()
	defer usersLanguagesMu.Unlock()
	usersLanguages[chatId] = lang
}

func LoadLocales() error {
	for _, lang := range []LangCode{"en", "ua", "pl"} {
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

func T(userLang LangCode, key string, args ...map[string]string) string {
	l, ok := locales[userLang]

	if !ok {
		l = locales["en"]
	}

	str, ok := l[key]
	if !ok {
		return key
	}

	if len(args) > 0 {
		for k, v := range args[0] {
			str = strings.ReplaceAll(str, "{{"+k+"}}", v)
		}
	}
	return str
}
