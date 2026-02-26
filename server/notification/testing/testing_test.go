package testing

import (
	"testing"

	"github.com/root-gg/plik/server/notification"
	"github.com/stretchr/testify/require"
)

func TestTestingProvider_Name(t *testing.T) {
	p := NewProvider()
	require.Equal(t, "testing", p.Name())
}

func TestTestingProvider_Send(t *testing.T) {
	p := NewProvider()

	msg := &notification.Message{
		To:      []string{"test@example.com"},
		Subject: "Test",
		HTML:    "<p>Hello</p>",
	}

	err := p.Send(msg)
	require.NoError(t, err)

	msgs := p.GetMessages()
	require.Len(t, msgs, 1)
	require.Equal(t, "test@example.com", msgs[0].To[0])
	require.Equal(t, "Test", msgs[0].Subject)
}

func TestTestingProvider_MultipleMessages(t *testing.T) {
	p := NewProvider()

	for i := 0; i < 3; i++ {
		err := p.Send(&notification.Message{
			To:      []string{"test@example.com"},
			Subject: "Test",
		})
		require.NoError(t, err)
	}

	msgs := p.GetMessages()
	require.Len(t, msgs, 3)
}

func TestTestingProvider_Reset(t *testing.T) {
	p := NewProvider()

	err := p.Send(&notification.Message{
		To:      []string{"test@example.com"},
		Subject: "Test",
	})
	require.NoError(t, err)

	require.Len(t, p.GetMessages(), 1)

	p.Reset()
	require.Len(t, p.GetMessages(), 0)
}
