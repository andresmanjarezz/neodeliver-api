package utils

import "regexp"

func ValidateEmail(email *string) (bool, error) {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	return regexp.MatchString(emailRegex, *email)
}

func ValidatePhone(phone *string) (bool, error) {
	phoneRegex := `^\+\d{1,3}-\d{1,3}-\d{1,3}-\d{1,10}$`
	return regexp.MatchString(phoneRegex, *phone)
}
