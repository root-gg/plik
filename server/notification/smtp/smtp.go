package smtp

import (
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strings"

	"github.com/root-gg/plik/server/notification"
	"github.com/root-gg/utils"
)

// Ensure SMTP Provider implements notification.Provider interface
var _ notification.Provider = (*Provider)(nil)

// Config describes configuration for the SMTP notification provider.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      bool
}

// NewConfig instantiates a new default configuration
// and overrides it with configuration passed as argument.
func NewConfig(params map[string]any) (config *Config) {
	config = new(Config)
	config.Port = 587
	config.TLS = true
	utils.Assign(config, params)
	return
}

// Provider sends notifications via SMTP.
type Provider struct {
	Config *Config
}

// NewProvider instantiates a new SMTP notification provider.
func NewProvider(config *Config) *Provider {
	return &Provider{Config: config}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "smtp"
}

// Send delivers a notification message via SMTP.
func (p *Provider) Send(msg *notification.Message) error {
	if len(msg.To) == 0 {
		return nil
	}

	addr := net.JoinHostPort(p.Config.Host, fmt.Sprintf("%d", p.Config.Port))

	// Build RFC 2045 multipart/alternative email
	boundary := "plik-boundary-" + fmt.Sprintf("%d", len(msg.HTML)+len(msg.Text))
	headers := make([]string, 0, 8)
	headers = append(headers, fmt.Sprintf("From: %s", p.Config.From))
	headers = append(headers, fmt.Sprintf("To: %s", strings.Join(msg.To, ", ")))
	headers = append(headers, fmt.Sprintf("Subject: %s", mime.QEncoding.Encode("utf-8", msg.Subject)))
	headers = append(headers, "MIME-Version: 1.0")
	headers = append(headers, fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q", boundary))
	headers = append(headers, "")

	var body strings.Builder
	body.WriteString(strings.Join(headers, "\r\n"))
	body.WriteString("\r\n")

	// Plain-text part (quoted-printable encoded)
	if msg.Text != "" {
		body.WriteString("--" + boundary + "\r\n")
		body.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		body.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		qpWriter := quotedprintable.NewWriter(&body)
		qpWriter.Write([]byte(msg.Text))
		qpWriter.Close()
		body.WriteString("\r\n")
	}

	// HTML part (quoted-printable encoded)
	if msg.HTML != "" {
		body.WriteString("--" + boundary + "\r\n")
		body.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		body.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		qpWriter := quotedprintable.NewWriter(&body)
		qpWriter.Write([]byte(msg.HTML))
		qpWriter.Close()
		body.WriteString("\r\n")
	}

	body.WriteString("--" + boundary + "--\r\n")

	var auth smtp.Auth
	if p.Config.Username != "" {
		auth = smtp.PlainAuth("", p.Config.Username, p.Config.Password, p.Config.Host)
	}

	if p.Config.TLS {
		return p.sendTLS(addr, auth, body.String(), msg.To)
	}

	return p.sendPlain(addr, auth, body.String(), msg.To)
}

// sendPlain sends email over a plain (non-TLS) connection.
// Unlike smtp.SendMail, this does NOT attempt STARTTLS, which avoids
// certificate validation errors with local SMTP servers.
func (p *Provider) sendPlain(addr string, auth smtp.Auth, body string, to []string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial to %s failed: %w", addr, err)
	}

	client, err := smtp.NewClient(conn, p.Config.Host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err = client.Mail(p.Config.From); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}

	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("SMTP RCPT TO <%s> failed: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	_, err = w.Write([]byte(body))
	if err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("SMTP close data failed: %w", err)
	}

	return client.Quit()
}

// sendTLS sends email over an explicit TLS connection (STARTTLS or direct TLS).
func (p *Provider) sendTLS(addr string, auth smtp.Auth, body string, to []string) error {
	tlsConfig := &tls.Config{ServerName: p.Config.Host}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial to %s failed: %w", addr, err)
	}

	client, err := smtp.NewClient(conn, p.Config.Host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err = client.Mail(p.Config.From); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}

	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("SMTP RCPT TO <%s> failed: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	_, err = w.Write([]byte(body))
	if err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("SMTP close data failed: %w", err)
	}

	return client.Quit()
}
