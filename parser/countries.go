package parser

import (
	"regexp"
	"strings"
)

const (
	CountryBE = "BE"
	CountryDE = "DE"
	CountryNL = "NL"
	CountryFR = "FR"
	CountryPL = "PL"
	CountryCZ = "CZ"
	CountryAT = "AT"
	CountrySK = "SK"
	CountryHU = "HU"
	CountrySI = "SI"
	CountryHR = "HR"
	CountryRO = "RO"
	CountryBG = "BG"
	CountryRS = "RS"
	CountryUA = "UA"
	CountryLT = "LT"
	CountryLV = "LV"
	CountryEE = "EE"
	CountryCH = "CH"
	CountryLU = "LU"
)

type Country struct {
	Code  string
	Name  string
	Emoji string
}

var Countries = map[string]Country{
	CountryBE: {Code: "BE", Name: "Belgium", Emoji: "🇧🇪"},
	CountryDE: {Code: "DE", Name: "Germany", Emoji: "🇩🇪"},
	CountryNL: {Code: "NL", Name: "Netherlands", Emoji: "🇳🇱"},
	CountryFR: {Code: "FR", Name: "France", Emoji: "🇫🇷"},
	CountryPL: {Code: "PL", Name: "Poland", Emoji: "🇵🇱"},
	CountryCZ: {Code: "CZ", Name: "Czechia", Emoji: "🇨🇿"},
	CountryAT: {Code: "AT", Name: "Austria", Emoji: "🇦🇹"},
	CountrySK: {Code: "SK", Name: "Slovakia", Emoji: "🇸🇰"},
	CountryHU: {Code: "HU", Name: "Hungary", Emoji: "🇭🇺"},
	CountrySI: {Code: "SI", Name: "Slovenia", Emoji: "🇸🇮"},
	CountryHR: {Code: "HR", Name: "Croatia", Emoji: "🇭🇷"},
	CountryRO: {Code: "RO", Name: "Romania", Emoji: "🇷🇴"},
	CountryBG: {Code: "BG", Name: "Bulgaria", Emoji: "🇧🇬"},
	CountryRS: {Code: "RS", Name: "Serbia", Emoji: "🇷🇸"},
	CountryUA: {Code: "UA", Name: "Ukraine", Emoji: "🇺🇦"},
	CountryLT: {Code: "LT", Name: "Lithuania", Emoji: "🇱🇹"},
	CountryLV: {Code: "LV", Name: "Latvia", Emoji: "🇱🇻"},
	CountryEE: {Code: "EE", Name: "Estonia", Emoji: "🇪🇪"},
	CountryCH: {Code: "CH", Name: "Switzerland", Emoji: "🇨🇭"},
	CountryLU: {Code: "LU", Name: "Luxembourg", Emoji: "🇱🇺"},
}

func GetCountryByCode(code string) (Country, bool) {
	country, exists := Countries[code]
	return country, exists
}

func GetCountryEmoji(code string) string {
	if country, exists := Countries[code]; exists {
		return country.Emoji
	}
	return ""
}

func GetCountryName(code string) string {
	if country, exists := Countries[code]; exists {
		return country.Name
	}
	return ""
}

func ExtractCountryCode(address string) string {
	// "DE 68219" or "DE-68219"
	pattern := regexp.MustCompile(`\b([A-Z]{2})[\s-]\d`)

	matches := pattern.FindStringSubmatch(address)
	if len(matches) > 1 {
		code := matches[1]
		if _, exists := Countries[code]; exists {
			return code
		}
	}

	words := strings.Fields(address)
	for _, word := range words {
		cleaned := strings.Trim(word, ",.")
		if len(cleaned) == 2 {
			upper := strings.ToUpper(cleaned)
			if _, exists := Countries[upper]; exists {
				return upper
			}
		}
	}

	return ""
}

func ExtractCountry(address string) (Country, bool) {
	code := ExtractCountryCode(address)
	if code == "" {
		return Country{}, false
	}
	country, exists := Countries[code]
	return country, exists
}
