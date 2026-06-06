package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"student_portal/backend/config"
	"student_portal/backend/internals/tokens"
	"student_portal/backend/internals/utils"
)

type MailService struct {
	mailer       *Mailer
	cfg          *config.AppConfig
	magic_link   *template.Template
	was_this_you *template.Template
}

type MagicLinkData struct {
	MagicLinkURL string
	ExpiresAt    string
}

type WasThisYouData struct {
	ConfirmURL    string
	InvalidateURL string
	LoginTime     string
	Browser       string
	Platform      string
	Email         string
	ExpiresAt     string
}

func NewMailService(mailer *Mailer, cfg *config.AppConfig) (*MailService, error) {
	magicTmpl, err := template.ParseFiles("internals/mail/templates/magic_link.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse magic_link template: %w", err)
	}

	wasThisYouTmpl, err := template.ParseFiles("internals/mail/templates/was_this_you.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse was_this_you template: %w", err)
	}

	return &MailService{mailer, cfg, magicTmpl, wasThisYouTmpl}, nil
}

func (s *MailService) SendMagicLink(to string, rawToken string, expiresAt time.Time) error {
	magicURL := tokens.BuildMagicLinkURL(s.cfg.Server.APP_URL, rawToken)
	// expiresStr := expiresAt.In(istLocation).Format("03:04 PM MST")
	// Always show in IST — your users are all on campus in India
	var istLocation, _ = time.LoadLocation("Asia/Kolkata")
	expiresStr := expiresAt.In(istLocation).Format("03:04 PM MST")
	var htmlBuf bytes.Buffer
	err := s.magic_link.Execute(&htmlBuf, MagicLinkData{MagicLinkURL: magicURL, ExpiresAt: expiresStr})
	if err != nil {
		return fmt.Errorf("failed to render magic_link template: %w", err)
	}
	textBody := fmt.Sprintf(
		"Campus Council Portal — IIT Kanpur\n\n"+
			"Your login link:\n%s\n\n"+
			"This link expires at %s and can only be used once.\n\n"+
			"If you did not request this, ignore this email.",
		magicURL, expiresStr,
	)
	return s.mailer.Send(&Message{
		to:       to,
		subject:  "⚠ New login detected — Campus Council Portal",
		htmlBody: htmlBuf.String(),
		textBody: textBody,
	})

}

func (s *MailService) SendWasThisYou(to string, rawInvalidationToken string, expiresAt time.Time, r *http.Request) error {
	confirmURL := tokens.BuildConfirmationURL(s.cfg.Server.APP_URL, rawInvalidationToken)
	invalidateURL := tokens.BuildInvalidationURL(s.cfg.Server.APP_URL, rawInvalidationToken)
	fp := utils.ExtractFingerprints(r)
	browser := parseBrowserFromUA(fp.UserAgent) // internal helper
	platform := fp.Platform
	if platform == "" {
		platform = "Unknown device"
	}
	istLocation, _ := time.LoadLocation("Asia/Kolkata")
	loginTimeStr := time.Now().In(istLocation).Format("Monday, 02 Jan 2006, 03:04 PM MST")
	maskedEmail := maskEmail(to)
	// student@iitk.ac.in → stu***@iitk.ac.in
	expiresStr := expiresAt.In(istLocation).Format("03:04 PM MST")
	var htmlBuf bytes.Buffer
	err := s.was_this_you.Execute(&htmlBuf, WasThisYouData{
		ConfirmURL:    confirmURL,
		InvalidateURL: invalidateURL,
		LoginTime:     loginTimeStr,
		Browser:       browser,
		Platform:      platform,
		Email:         maskedEmail,
		ExpiresAt:     expiresStr,
	})
	if err != nil {
		return fmt.Errorf("failed to render was_this_you template: %w", err)
	}
	textBody := fmt.Sprintf(
		"Campus Council Portal — IIT Kanpur\n\n"+
			"SECURITY ALERT: New login detected on your account.\n\n"+
			"Time:     %s\n"+
			"Browser:  %s\n"+
			"Platform: %s\n"+
			"Account:  %s\n\n"+
			"Was this you?\n\n"+
			"YES — this was me:\n%s\n\n"+
			"NO — kill this session:\n%s\n\n"+
			"This alert expires at %s.\n"+
			"If you take no action, the session continues.\n\n"+
			"Never share this email with anyone.",
		loginTimeStr, browser, platform, maskedEmail,
		confirmURL, invalidateURL, expiresStr,
	)
	return s.mailer.Send(&Message{
		to:       to,
		subject:  "⚠ New login detected — Campus Council Portal",
		htmlBody: htmlBuf.String(),
		textBody: textBody,
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
