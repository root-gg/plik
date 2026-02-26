package testing

import (
	"sync"

	"github.com/root-gg/plik/server/notification"
)

// Ensure Testing Provider implements notification.Provider interface
var _ notification.Provider = (*Provider)(nil)

// Provider records notifications in memory for testing.
type Provider struct {
	Messages []*notification.Message
	err      error
	mu       sync.Mutex
}

// NewProvider instantiates a new Testing notification provider.
func NewProvider() *Provider {
	return &Provider{}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "testing"
}

// Send records the notification message.
func (p *Provider) Send(msg *notification.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.err != nil {
		return p.err
	}

	p.Messages = append(p.Messages, msg)
	return nil
}

// GetMessages returns all recorded messages.
func (p *Provider) GetMessages() []*notification.Message {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.Messages
}

// SetError sets the error that this provider will return on any subsequent Send call.
func (p *Provider) SetError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.err = err
}

// Reset clears all recorded messages and errors.
func (p *Provider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Messages = nil
	p.err = nil
}
