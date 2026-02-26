package log

import (
	"strings"

	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/notification"
)

// Ensure Log Provider implements notification.Provider interface
var _ notification.Provider = (*Provider)(nil)

// Provider logs notifications instead of sending them.
// Useful for development, testing, and debugging.
type Provider struct {
	Logger *logger.Logger
}

// NewProvider instantiates a new Log notification provider.
func NewProvider(log *logger.Logger) *Provider {
	return &Provider{Logger: log}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "log"
}

// Send logs the notification message.
func (p *Provider) Send(msg *notification.Message) error {
	p.Logger.Infof("Notification to [%s] subject=%q text=%q",
		strings.Join(msg.To, ", "), msg.Subject, msg.Text)
	return nil
}
