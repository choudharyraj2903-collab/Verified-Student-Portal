package utils

import (
	"net/http"
	"strings"
)

type FinngerPrintData struct {
	UserAgent      string
	AcceptLanguage string
	AcceptEncoding string
	Platform       string
	UAMobile       string
	UABranded      string
}

func normalise(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func ExtractFingerprints(r *http.Request) FinngerPrintData {
	return FinngerPrintData{
		UserAgent:      normalise(r.Header.Get("User-Agent")),
		AcceptLanguage: normalise(r.Header.Get("Accept-Language")),
		AcceptEncoding: normalise(r.Header.Get("Accept-Encoding")),
		Platform:       normalise(r.Header.Get("Platform")),
		UAMobile:       normalise(r.Header.Get("UAMobile")),
		UABranded:      normalise(r.Header.Get("UABranded")),
	}
}

func BuildFingerprintString(fp FinngerPrintData) string {
	return fp.UserAgent + "|" + fp.AcceptLanguage + "|" + fp.AcceptEncoding + "|" + fp.Platform + "|" + fp.UAMobile + "|" + fp.UABranded
}

func HashFingerprint(fp string) string {
	return Hash256(fp)
}

func FingerprintMatch(r *http.Request, fp string) bool {
	return HashFingerprint(BuildFingerprintString(ExtractFingerprints(r))) == fp
}

func FingerprintEntropy(fp FinngerPrintData) int {
	total := 0
	for _, v := range []string{fp.UserAgent, fp.AcceptLanguage, fp.AcceptEncoding, fp.Platform, fp.UAMobile, fp.UABranded} {
		if v != "" {
			total++
		}
	}
	return total
}
