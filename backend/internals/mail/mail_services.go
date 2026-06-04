package mail


import (
    "bytes"
    "fmt"
    "html/template"
    "net/http"
    "strings"
    "time"

    "yourmodule/config"
    "yourmodule/tokens"
    "yourmodule/utils"
)


type MailServices struct {
	mailer *mailer
	cfg *config.AppConfig
	magic_link *template.Template
	was_this_you *template.Template
}

type MagicLinkData struct {
	MagicLinkURL string
	ExpiresAt string
}

type WasThisYouData struct {
	ConfirmURL string
	InvalidateURL string
	LoginTime string
	Browser string
	Platform string
	Email string
	ExpiresAt string
}

func NewMailService(mailer *Mailer, cfg *config.AppConfig) (*MailService, error) {
	magicTmpl, err := template.ParseFiles("mail/templates/magic_link.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse magic_link template: %w", err)
	}

    wasThisYouTmpl, err := template.ParseFiles("mail/templates/was_this_you.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse was_this_you template: %w", err)
	}

	return &MailService{mailer, cfg, magicTmpl, wasThisYouTmpl}, nil
}

func (s *MailService) SendMagicLink(to string, rawToken string, expiresAt time.Time) error {
	magicURL := tokens.BuildMagicLinkURL(s.cfg.Server.AppURL, rawToken)
	expiresStr := expiresAt.In(istLocation).Format("03:04 PM MST")
    // Always show in IST — your users are all on campus in India
	var istLocation,_ := time.LoadLocation("Asia/Kolkata")
	var htmlBuf bytes.Buffer
	err := s.magicTmpl.Execute(&htmlBuf, MagicLinkData{MagicLinkURL: magicURL,ExpiresAt:    expiresStr,})
	if err != nil {
		return fmt.Errorf("failed to render magic_link template: %w", err)
	}
	textBody := fmt.Sprintf(
    "Campus Council Portal — IIT Kanpur\n\n" +
    "Your login link:\n%s\n\n" +
    "This link expires at %s and can only be used once.\n\n" +
    "If you did not request this, ignore this email.",
    magicURL, expiresStr,
	)
	return s.mailer.Send(&Message{
    To:       to,
    Subject:  "Your login link — Campus Council Portal",
    HTMLBody: htmlBuf.String(),
    TextBody: textBody,
})
}

func (s *MailService) SendWasThisYou(to string, rawInvalidationToken string, expiresAt time.Time, r *http.Request) error {
	confirmURL    := tokens.BuildConfirmationURL(s.cfg.Server.AppURL, rawInvalidationToken)
	invalidateURL := tokens.BuildInvalidationURL(s.cfg.Server.AppURL, rawInvalidationToken)
	fp := utils.ExtractFingerprint(r)
	browser  := parseBrowserFromUA(fp.UserAgent)   // internal helper
	platform := fp.Platform
	if platform == "" {
		platform = "Unknown device"
	}
	loginTimeStr := time.Now().In(istLocation).Format("Monday, 02 Jan 2006, 03:04 PM MST")
	maskedEmail := maskEmail(to)
    // student@iitk.ac.in → stu***@iitk.ac.in
	expiresStr := expiresAt.In(istLocation).Format("03:04 PM MST")
	var htmlBuf bytes.Buffer
	err := s.wasThisYouTmpl.Execute(&htmlBuf, WasThisYouData{
		ConfirmURL:    confirmURL,
		InvalidateURL: invalidateURL,
		LoginTime:     loginTimeStr,
		Browser:       browser,
		Platform:      platform,
		Email:         maskedEmail,
		ExpiresAt:     expiresStr,
	})
	textBody := fmt.Sprintf(
    "Campus Council Portal — IIT Kanpur\n\n" +
    "SECURITY ALERT: New login detected on your account.\n\n" +
    "Time:     %s\n" +
    "Browser:  %s\n" +
    "Platform: %s\n" +
    "Account:  %s\n\n" +
    "Was this you?\n\n" +
    "YES — this was me:\n%s\n\n" +
    "NO — kill this session:\n%s\n\n" +
    "This alert expires at %s.\n" +
    "If you take no action, the session continues.\n\n" +
    "Never share this email with anyone.",
    loginTimeStr, browser, platform, maskedEmail,
    confirmURL, invalidateURL, expiresStr,
)
    return s.mailer.Send(&Message{
    To:       to,
    Subject:  "⚠ New login detected — Campus Council Portal",
    HTMLBody: htmlBuf.String(),
    TextBody: textBody,
})
}

func parseBrowserFromUA(ua string) string {
	switch {
case strings.Contains(ua, "Edg/"):
    return "Microsoft Edge"
case strings.Contains(ua, "OPR/") || strings.Contains(ua, "Opera"):
    return "Opera"
case strings.Contains(ua, "Chrome/"):
    return "Chrome"
case strings.Contains(ua, "Firefox/"):
    return "Firefox"
case strings.Contains(ua, "Safari/") && !strings.Contains(ua, "Chrome"):
    return "Safari"
default:
    return "Unknown browser"
}
}

func maskEmail(email string) string {
	emailParts := strings.Split(email, "@")
	return emailParts[0] + "*****@" + emailParts[1]
}


