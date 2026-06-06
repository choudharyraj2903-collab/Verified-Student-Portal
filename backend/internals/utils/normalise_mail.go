package utils 

import (
	"strings"
)

func NormaliseMail(email string) string {
	email = strings.ToLower(email)
	email = strings.TrimSpace(email)
	return email
}

func ValidateMailFormat(email string) bool {
	if email == "" {
		return false 
	}
	if !strings.Contains(email, "@") {
		return false
	}
	if !strings.Contains(email, ".") {
		return false
	}
	return true
}

func ExtractEmailDomain(email string) string {
	return strings.Split(email, "@")[1]
}

func ValidateMailDomain(email string, allowedDomain string) bool {
	domain := ExtractEmailDomain(email)
	return allowedDomain == domain
}