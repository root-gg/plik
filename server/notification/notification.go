package notification

// Provider interface describes methods that notification backends
// must implement to be compatible with Plik.
type Provider interface {
	// Send delivers a notification message to the specified recipients.
	// Implementations must not modify the Message.
	Send(msg *Message) error

	// Name returns the provider name (e.g., "smtp", "log").
	Name() string
}

// Message represents an outgoing notification.
type Message struct {
	To      []string // Recipient email addresses
	Subject string   // Email subject line
	HTML    string   // HTML body
	Text    string   // Plain-text fallback body
}
