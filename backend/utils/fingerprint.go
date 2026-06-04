package utils 

import (
	"strings"
	"net/http"
)

type FinngerPrintData struct {
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


func ExtractFingerprints(r *http.Request) FinngerPrintData {
	return FinngerPrintData{
		UserAgent: normalise(r.Header.Get("User-Agent")),
		AcceptLanguage: normalise(r.Header.Get("Accept-Language")),
		AcceptEncoding: normalise(r.Header.Get("Accept-Encoding")),
		Platform: normalise(r.Header.Get("Platform")),
		UAMobile: normalise(r.Header.Get("UAMobile")),
		UABranded: normalise(r.Header.Get("UABranded")),
	}
}

func BuildFingerprintString(fp FinngerPrintData) string {
	"Mozilla/5.0 (Windows NT 10.0)|en-US,en;q=0.9|gzip,deflate,br|Windows|?0|Chrome"
	return fp.UserAgent + "|" + fp.AcceptLanguage + "|" + fp.AcceptEncoding + "|" + fp.Platform + "|" + fp.UAMobile + "|" + fp.UABranded
}

func HashFingerprint(fp string) string {
	return Hash256(fp)
}

func FingerprintMatch(r *http.Request, fp string) bool {
	return HashFingerprint(BuildFingerprintString(ExtractFingerprints(r))) == fp
}

func FingerprintEntropy(fp FingerprintData) int {
	total := 0
	for _, v := range fp {
		if v != "" {
			total += 1 
		}
	}
	return total
}