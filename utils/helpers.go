package utils

import (
	"regexp"

	isolang "github.com/emvi/iso-639-1"
)

func ValidateEmail(email *string) bool {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(emailRegex, *email)
	return match
}

func ValidatePhone(phone *string) bool {
	phoneRegex := `^\+\d{1,3}-\d{1,3}-\d{1,3}-\d{1,10}$`
	match, _ := regexp.MatchString(phoneRegex, *phone)
	return match
}

func ValidateLanguageCode(language *string) bool {
	return isolang.ValidCode(*language)
}

func ValidateNotificationToken(token *string) bool {
	tokenRegex := `^ExponentPushToken\[[A-Za-z0-9]+\]$`
	match, _ := regexp.MatchString(tokenRegex, *token)
	return match
}

func ValidateMongoDBQuery(query *string) bool {
	queryRegex := `\{\s*\$[a-zA-Z]+\s*:\s*\[.*\]\s*\}`
	match, _ := regexp.MatchString(queryRegex, *query)
	return match
}
