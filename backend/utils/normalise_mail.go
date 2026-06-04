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
	if string == ""{
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
	for _, d := range allowedDomain {
		if d == domain {
			return true
		}
	}
	return false
}