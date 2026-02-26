package notification_test

import (
	"testing"
	"time"

	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/notification"
	notifTesting "github.com/root-gg/plik/server/notification/testing"
	"github.com/stretchr/testify/require"
)

func TestServiceNotify(t *testing.T) {
	config := &common.Configuration{}
	config.MaxUploadReceivers = 5
	log := logger.NewLogger()

	provider := notifTesting.NewProvider()
	svc, err := notification.NewService(provider, config, log)
	require.NoError(t, err)
	svc.Start()
	defer svc.Stop()

	upload := common.NewUpload()
	upload.Receivers = common.Receivers{"test@example.com"}

	file := upload.NewFile()
	file.Name = "test.txt"
	file.Size = 1024
	file.Status = common.FileUploaded

	svc.Notify(notification.Event{
		Type:   notification.EventUploadReady,
		Upload: upload,
	})

	// Wait for async processing
	time.Sleep(200 * time.Millisecond)

	msgs := provider.GetMessages()
	require.Len(t, msgs, 1)
	require.Contains(t, msgs[0].To, "test@example.com")
	require.Contains(t, msgs[0].Subject, "ready")
}

func TestServiceNotifyAllDownloaded(t *testing.T) {
	config := &common.Configuration{}
	log := logger.NewLogger()

	provider := notifTesting.NewProvider()
	svc, err := notification.NewService(provider, config, log)
	require.NoError(t, err)
	svc.Start()
	defer svc.Stop()

	upload := common.NewUpload()
	upload.NotifyCreator = true
	upload.Receivers = common.Receivers{"creator@example.com"}

	file := upload.NewFile()
	file.Name = "doc.pdf"
	file.Size = 2048
	file.Status = common.FileUploaded

	user := common.NewUser(common.ProviderLocal, "testuser")
	user.Email = "creator@example.com"

	svc.Notify(notification.Event{
		Type:   notification.EventAllDownloaded,
		Upload: upload,
		User:   user,
	})

	time.Sleep(200 * time.Millisecond)

	msgs := provider.GetMessages()
	require.Len(t, msgs, 1)
	require.Contains(t, msgs[0].Subject, "downloaded")
}

func TestServiceNotifyCreatorWithUser(t *testing.T) {
	config := &common.Configuration{}
	log := logger.NewLogger()

	provider := notifTesting.NewProvider()
	svc, err := notification.NewService(provider, config, log)
	require.NoError(t, err)
	svc.Start()
	defer svc.Stop()

	upload := common.NewUpload()
	upload.NotifyCreator = true
	// No receivers, just notify creator

	file := upload.NewFile()
	file.Name = "hello.txt"
	file.Size = 512
	file.Status = common.FileUploaded

	user := common.NewUser(common.ProviderLocal, "testuser")
	user.Email = "me@example.com"

	svc.Notify(notification.Event{
		Type:   notification.EventUploadReady,
		Upload: upload,
		User:   user,
	})

	time.Sleep(200 * time.Millisecond)

	msgs := provider.GetMessages()
	require.Len(t, msgs, 1)
	require.Contains(t, msgs[0].To, "me@example.com")
}

func TestServiceNoRecipientsSkips(t *testing.T) {
	config := &common.Configuration{}
	log := logger.NewLogger()

	provider := notifTesting.NewProvider()
	svc, err := notification.NewService(provider, config, log)
	require.NoError(t, err)
	svc.Start()
	defer svc.Stop()

	upload := common.NewUpload()
	// No receivers, no notifyCreator, no user

	svc.Notify(notification.Event{
		Type:   notification.EventUploadReady,
		Upload: upload,
	})

	time.Sleep(200 * time.Millisecond)

	msgs := provider.GetMessages()
	require.Len(t, msgs, 0, "should not send any notification when no recipients")
}
