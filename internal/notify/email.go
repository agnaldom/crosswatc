package notify

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Email sends plain-text mail through SMTP (STARTTLS when offered by the server).
type Email struct {
	Addr     string // host:port
	From     string
	Username string
	Password string
	send     func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

func (Email) Channel() string { return "email" }

func (e Email) Send(_ context.Context, to []string, m Message) error {
	var auth smtp.Auth
	if e.Username != "" {
		host, _, err := net.SplitHostPort(e.Addr)
		if err != nil {
			return err
		}
		auth = smtp.PlainAuth("", e.Username, e.Password, host)
	}
	send := e.send
	if send == nil {
		send = smtp.SendMail
	}
	if err := send(e.Addr, auth, e.From, to, render(e.From, to, m)); err != nil {
		return fmt.Errorf("smtp: %w", err)
	}
	return nil
}

func render(from string, to []string, m Message) []byte {
	var b strings.Builder
	header := func(k, v string) {
		fmt.Fprintf(&b, "%s: %s\r\n", k, strings.NewReplacer("\r", "", "\n", "").Replace(v))
	}
	header("From", from)
	header("To", strings.Join(to, ", "))
	header("Subject", "[Crosswatc] "+m.Title())
	header("Date", time.Now().UTC().Format(time.RFC1123Z))
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=UTF-8")
	b.WriteString("\r\n")
	fmt.Fprintf(&b, "%s\r\n\r\n%s\r\n\r\n", m.Title(), m.Text)
	fmt.Fprintf(&b, "Recurso: %s (%s)\r\nAmbiente: %s\r\nSquad: %s\r\nCorrelation ID: %s\r\n", m.ResourceName, m.ResourceKind, m.Environment, m.Owner, m.CorrelationID)
	if m.PRURL != "" {
		fmt.Fprintf(&b, "Pull Request: %s\r\n", m.PRURL)
	}
	if m.Link != "" {
		fmt.Fprintf(&b, "Crosswatc: %s\r\n", m.Link)
	}
	return []byte(b.String())
}
