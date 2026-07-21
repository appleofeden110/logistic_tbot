package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxAge time.Duration = 24 * time.Hour

var ErrInvalidInitData = errors.New("invalid initData")
var ErrExpiredInitData = errors.New("expired initData")

func VerifyInitData(initData, botToken string) (chatId int64, err error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return 0, ErrInvalidInitData
	}

	receivedHash := values.Get("hash")
	if receivedHash == "" {
		return 0, ErrInvalidInitData
	}
	values.Del("hash")

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = k + "=" + values.Get(k)
	}
	dataCheckString := strings.Join(pairs, "\n")

	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	mac := hmac.New(sha256.New, secretKey.Sum(nil))
	mac.Write([]byte(dataCheckString))
	computedHash := hex.EncodeToString(mac.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(computedHash), []byte(receivedHash)) != 1 {
		return 0, ErrInvalidInitData
	}

	authDateUnix, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return 0, ErrInvalidInitData
	}
	if time.Since(time.Unix(authDateUnix, 0)) > maxAge {
		return 0, ErrExpiredInitData
	}

	var raw struct {
		Id int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(values.Get("user")), &raw); err != nil {
		return 0, ErrInvalidInitData
	}

	return raw.Id, nil
}
