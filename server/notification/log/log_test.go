package log

import (
	"testing"

	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/notification"
	"github.com/stretchr/testify/require"
)

func TestLogProvider_Name(t *testing.T) {
	p := NewProvider(logger.NewLogger())
	require.Equal(t, "log", p.Name())
}

func TestLogProvider_Send(t *testing.T) {
	log := logger.NewLogger()
	p := NewProvider(log)

	msg := &notification.Message{
		To:      []string{"test@example.com"},
		Subject: "Test Notification",
		HTML:    "<p>Hello</p>",
		Text:    "Hello",
	}

	err := p.Send(msg)
	require.NoError(t, err)
}

func TestLogProvider_SendEmpty(t *testing.T) {
	log := logger.NewLogger()
	p := NewProvider(log)

	msg := &notification.Message{
		To:      []string{},
		Subject: "No recipients",
	}

	err := p.Send(msg)
	require.NoError(t, err)
}
