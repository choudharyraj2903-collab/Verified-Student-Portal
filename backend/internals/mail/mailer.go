package mail


import (
	"crypto/tls"
	"fmt"

	"github.com/go-gomail/gomail"
	"student_portal/backend/config"
)


type Mailer struct {
	host       string
	port       int
	username   string
	password   string
	fromAddr   string
	fromName   string
	encryption string
	dialer     *gomail.Dialer
}

type Message struct {
	to       string
	subject  string
	htmlBody string
	textBody string
}

func NewMailer(cfg *config.MailConfig) (*Mailer, error){
	dailer := gomail.NewDialer(cfg.MAIL_HOST, cfg.MAIL_PORT, cfg.MAIL_USERNAME, cfg.MAIL_PASSWORD)
	switch cfg.MAIL_ENCRYPTION {
		case "tls":
			dailer.SSL = true          // port 465 — full TLS from the start
		case "starttls":
			dailer.SSL = false         // port 587 — starts plain, upgrades via STARTTLS
            dailer.TLSConfig = &tls.Config{
                ServerName: cfg.MAIL_HOST,
                MinVersion: tls.VersionTLS12,   // reject TLS 1.0 and 1.1
				}
        case "none":
			dailer.SSL = false         // port 25 — only for local dev, blocked in prod by config validation
			}
	
	
	return &Mailer{cfg.MAIL_HOST, cfg.MAIL_PORT, cfg.MAIL_USERNAME, cfg.MAIL_PASSWORD, cfg.MAIL_FROM_ADDRESS, cfg.MAIL_FROM_NAME, cfg.MAIL_ENCRYPTION, dailer}, nil
}

func (m *Mailer) Send(msg *Message) error {
	gm := gomail.NewMessage()
	gm.SetAddressHeader("From", m.fromAddr, m.fromName)
	gm.SetHeader("To", msg.to)
	gm.SetHeader("Subject", msg.subject)
	gm.SetHeader("X-Mailer", "CampusCouncilPortal/1.0")    // identifies sender in headers
	gm.SetHeader("X-Priority", "1")                         // for Was This You — marks as high priority

	gm.SetBody("text/plain", msg.textBody)
	gm.AddAlternative("text/html", msg.htmlBody)
	if err := m.dialer.DialAndSend(gm); err != nil {
		return fmt.Errorf("failed to send email to %s: %w", msg.to, err)
	}
	return nil
}

func (m *Mailer) HealthCheck() error {
	return nil
}