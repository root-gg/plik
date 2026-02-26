package notification

import (
	"bytes"
	gocontext "context"
	"embed"
	"fmt"
	"html/template"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/nikoksr/notify"
	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
)

//go:embed templates/*.html
var templateFS embed.FS

// EventType defines the type of notification event.
type EventType int

const (
	// EventUploadReady is fired when all files in an upload have been uploaded.
	EventUploadReady EventType = iota
	// EventAllDownloaded is fired when all files in an upload have been downloaded at least once.
	EventAllDownloaded
)

// Event represents a notification event to be processed.
type Event struct {
	Type   EventType
	Upload *common.Upload
	User   *common.User // Creator, may be nil for anonymous uploads
}

// TemplateData is the data passed to email templates.
type TemplateData struct {
	Upload       *common.Upload
	DownloadURL  string
	ServerName   string
	ExpiresAt    string
	FileCount    int
	TotalSize    string
	Strings      map[string]string
}

// Service manages asynchronous notification dispatch.
type Service struct {
	provider Provider
	channels *notify.Notify // nikoksr/notify channels (optional, may be nil)
	logger   *logger.Logger
	config   *common.Configuration
	ch       chan Event
	done     chan struct{}

	uploadReadyTmpl    *template.Template
	allDownloadedTmpl  *template.Template
}

// NewService creates a new notification service.
func NewService(provider Provider, config *common.Configuration, log *logger.Logger) (*Service, error) {
	funcMap := template.FuncMap{
		"humanizeBytes": func(size int64) string {
			return humanize.Bytes(uint64(size))
		},
	}

	uploadReadyTmpl, err := template.New("upload_ready.html").Funcs(funcMap).ParseFS(templateFS, "templates/upload_ready.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse upload_ready template: %w", err)
	}

	allDownloadedTmpl, err := template.New("all_downloaded.html").Funcs(funcMap).ParseFS(templateFS, "templates/all_downloaded.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse all_downloaded template: %w", err)
	}

	return &Service{
		provider:          provider,
		logger:            log,
		config:            config,
		ch:                make(chan Event, 100),
		done:              make(chan struct{}),
		uploadReadyTmpl:   uploadReadyTmpl,
		allDownloadedTmpl: allDownloadedTmpl,
	}, nil
}

// SetChannels sets the nikoksr/notify channels for plain-text push notifications.
func (s *Service) SetChannels(channels *notify.Notify) {
	s.channels = channels
}

// Start begins the background notification worker.
func (s *Service) Start() {
	go s.worker()
}

// Stop gracefully shuts down the notification worker.
func (s *Service) Stop() {
	close(s.ch)
	<-s.done
}

// Notify enqueues a notification event for async processing.
// Non-blocking: if the channel is full, the event is dropped with a warning.
func (s *Service) Notify(event Event) {
	select {
	case s.ch <- event:
	default:
		s.logger.Warningf("notification channel full, dropping event type=%d upload=%s", event.Type, event.Upload.ID)
	}
}

func (s *Service) worker() {
	defer close(s.done)

	for event := range s.ch {
		s.processEvent(event)
	}
}

func (s *Service) processEvent(event Event) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Warningf("panic in notification worker: %v", r)
		}
	}()

	var err error
	switch event.Type {
	case EventUploadReady:
		err = s.sendUploadReady(event)
	case EventAllDownloaded:
		err = s.sendAllDownloaded(event)
	default:
		s.logger.Warningf("unknown notification event type: %d", event.Type)
		return
	}

	if err != nil {
		s.logger.Warningf("failed to send notification: %s", err)
	}

	// Dispatch to nikoksr/notify channels (plain text)
	if s.channels != nil {
		s.dispatchToChannels(event)
	}
}

func (s *Service) getDownloadURL(upload *common.Upload) string {
	var baseURL string
	if s.config.GetDownloadDomain() != nil {
		baseURL = s.config.GetDownloadDomain().String()
	} else {
		baseURL = s.config.GetServerURL().String()
	}
	return fmt.Sprintf("%s/#/?id=%s", baseURL, upload.ID)
}

func (s *Service) getServerName() string {
	if s.config.GetDownloadDomain() != nil {
		return s.config.GetDownloadDomain().Host
	}
	return "Plik"
}

func (s *Service) buildTemplateData(upload *common.Upload) TemplateData {
	var totalSize int64
	for _, f := range upload.Files {
		totalSize += f.Size
	}

	var expiresAt string
	if upload.ExpireAt != nil {
		expiresAt = upload.ExpireAt.Format(time.RFC1123)
	} else {
		expiresAt = "Never"
	}

	return TemplateData{
		Upload:      upload,
		DownloadURL: s.getDownloadURL(upload),
		ServerName:  s.getServerName(),
		ExpiresAt:   expiresAt,
		FileCount:   len(upload.Files),
		TotalSize:   humanize.Bytes(uint64(totalSize)),
		Strings:     defaultStrings(),
	}
}

func (s *Service) collectRecipients(event Event) []string {
	recipients := make([]string, 0)

	// Add creator email if NotifyCreator is set
	if event.Upload.NotifyCreator && event.User != nil && event.User.Email != "" {
		recipients = append(recipients, event.User.Email)
	}

	// Add receivers
	recipients = append(recipients, event.Upload.Receivers...)

	return recipients
}

func (s *Service) sendUploadReady(event Event) error {
	recipients := s.collectRecipients(event)
	if len(recipients) == 0 {
		return nil
	}

	data := s.buildTemplateData(event.Upload)

	var htmlBuf bytes.Buffer
	if err := s.uploadReadyTmpl.Execute(&htmlBuf, data); err != nil {
		return fmt.Errorf("failed to render upload_ready template: %w", err)
	}

	subject := fmt.Sprintf("[%s] Your upload is ready — %d file(s)", data.ServerName, data.FileCount)

	// Plain-text fallback
	text := fmt.Sprintf("Your upload with %d file(s) (%s) is ready for download.\n\nDownload: %s\nExpires: %s\n",
		data.FileCount, data.TotalSize, data.DownloadURL, data.ExpiresAt)

	msg := &Message{
		To:      recipients,
		Subject: subject,
		HTML:    htmlBuf.String(),
		Text:    text,
	}

	s.logger.Infof("sending upload_ready notification to %v for upload %s", recipients, event.Upload.ID)
	return s.provider.Send(msg)
}

func (s *Service) sendAllDownloaded(event Event) error {
	// Only notify the creator for "all downloaded" events
	if !event.Upload.NotifyCreator || event.User == nil || event.User.Email == "" {
		return nil
	}

	recipients := []string{event.User.Email}
	data := s.buildTemplateData(event.Upload)

	var htmlBuf bytes.Buffer
	if err := s.allDownloadedTmpl.Execute(&htmlBuf, data); err != nil {
		return fmt.Errorf("failed to render all_downloaded template: %w", err)
	}

	subject := fmt.Sprintf("[%s] All files have been downloaded", data.ServerName)

	text := fmt.Sprintf("All %d file(s) in your upload have been downloaded at least once.\n\nUpload: %s\n",
		data.FileCount, data.DownloadURL)

	msg := &Message{
		To:      recipients,
		Subject: subject,
		HTML:    htmlBuf.String(),
		Text:    text,
	}

	s.logger.Infof("sending all_downloaded notification to %v for upload %s", recipients, event.Upload.ID)
	return s.provider.Send(msg)
}

// defaultStrings returns the default (English) string map for templates.
// This is structured for future i18n support.
func defaultStrings() map[string]string {
	return map[string]string{
		"UploadReady":         "Your Upload is Ready",
		"UploadReadyDesc":     "Your files have been uploaded and are ready for download.",
		"AllDownloaded":       "All Files Downloaded",
		"AllDownloadedDesc":   "All files in your upload have been downloaded at least once.",
		"Files":               "Files",
		"FileName":            "Name",
		"FileSize":            "Size",
		"TotalSize":           "Total Size",
		"Expires":             "Expires",
		"Download":            "Download Files",
		"ViewUpload":          "View Upload",
		"PoweredBy":           "Powered by",
		"FileCount":           "File(s)",
	}
}

// dispatchToChannels sends a plain-text summary to all configured nikoksr/notify channels.
func (s *Service) dispatchToChannels(event Event) {
	data := s.buildTemplateData(event.Upload)

	var subject, body string
	switch event.Type {
	case EventUploadReady:
		subject = fmt.Sprintf("📦 Upload ready — %d file(s) (%s)", data.FileCount, data.TotalSize)
		body = fmt.Sprintf("Download: %s\nExpires: %s", data.DownloadURL, data.ExpiresAt)
		for _, f := range event.Upload.Files {
			body += fmt.Sprintf("\n  • %s (%s)", f.Name, humanize.Bytes(uint64(f.Size)))
		}
	case EventAllDownloaded:
		subject = fmt.Sprintf("✅ All %d file(s) downloaded", data.FileCount)
		body = fmt.Sprintf("Upload: %s", data.DownloadURL)
	default:
		return
	}

	if err := s.channels.Send(gocontext.Background(), subject, body); err != nil {
		s.logger.Warningf("failed to send channel notification: %s", err)
	}
}
