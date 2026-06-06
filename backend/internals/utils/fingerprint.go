package utils 

import (
	"strings"
	"net/http"
)

type FingerprintData struct {
	UserAgent string 
	AcceptLanguage string
	AcceptEncoding string
	Platform string
	UAMobile string 
	UABranded string
}

func normalise(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	return s
}


func ExtractFingerprints(r *http.Request) FingerprintData {
	return FingerprintData{
		UserAgent: normalise(r.Header.Get("User-Agent")),
		AcceptLanguage: normalise(r.Header.Get("Accept-Language")),
		AcceptEncoding: normalise(r.Header.Get("Accept-Encoding")),
		Platform: normalise(r.Header.Get("Platform")),
		UAMobile: normalise(r.Header.Get("UAMobile")),
		UABranded: normalise(r.Header.Get("UABranded")),
	}
}

func BuildFingerprintString(fp FingerprintData) string {
	return fp.UserAgent + "|" + fp.AcceptLanguage + "|" + fp.AcceptEncoding + "|" + fp.Platform + "|" + fp.UAMobile + "|" + fp.UABranded
}

func HashFingerprint(fp string) string {
	return Hash256String(fp)
}

func FingerprintMatch(r *http.Request, fp string) bool {
	return HashFingerprint(BuildFingerprintString(ExtractFingerprints(r))) == fp
}

func FingerprintEntropy(fp FingerprintData) int {
	total := 0
	for _, v := range []string{fp.UserAgent, fp.AcceptLanguage, fp.AcceptEncoding, fp.Platform, fp.UAMobile, fp.UABranded} {
		if v != "" {
			total += 1 
		}
	}
	return total
}



