package utils

import (
	"strings"
)

func NormaliseMail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
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
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func ValidateMailDomain(email string, allowedDomain string) bool {
	return ExtractEmailDomain(email) == allowedDomain
}
