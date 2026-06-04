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
	Dailer := gomail.NewDialer(cfg.MAIL_HOST, cfg.MAIL_PORT, cfg.MAIL_USERNAME, cfg.MAIL_PASSWORD)
	switch cfg.Encryption {
		case "tls":
			dialer.SSL = true          // port 465 — full TLS from the start
		case "starttls":
			dialer.SSL = false         // port 587 — starts plain, upgrades via STARTTLS
            dialer.TLSConfig = &tls.Config{
                ServerName: cfg.Host,
                MinVersion: tls.VersionTLS12,   // reject TLS 1.0 and 1.1
				}
        case "none":
			dialer.SSL = false         // port 25 — only for local dev, blocked in prod by config validation
			}
	
	
	return &Mailer{cfg.MAIL_HOST, cfg.MAIL_PORT, cfg.MAIL_USERNAME, cfg.MAIL_PASSWORD, cfg.MAIL_FROM_ADDRESS, cfg.MAIL_FROM_NAME, cfg.MAIL_ENCRYPTION, dialer}, nil
}

func (m *Mailer) Send(msg *Message) error {
	gm := gomail.NewMessage()
	gm.SetAddressHeader("From", m.fromAddr, m.fromName)
	gm.SetHeader("To", msg.To)
	gm.SetHeader("Subject", msg.Subject)
	gm.SetHeader("X-Mailer", "CampusCouncilPortal/1.0")    // identifies sender in headers
	gm.SetHeader("X-Priority", "1")                         // for Was This You — marks as high priority

	gm.SetBody("text/plain", msg.TextBody)
	gm.AddAlternative("text/html", msg.HTMLBody)
	if err := m.dialer.DialAndSend(gm); err != nil {
		return fmt.Errorf("failed to send email to %s: %w", msg.To, err)
	}
	return nil
}

func (m *Mailer) HealthCheck() error {
	return nil
}