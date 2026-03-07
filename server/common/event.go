package common

import (
	"fmt"
	"time"
)

// Event type constants
const (
	EventUploadCreated  = "upload_created"
	EventFileAdded      = "file_added"
	EventFileDownloaded = "file_downloaded"
)

// Event tracks upload lifecycle activity
type Event struct {
	ID        string    `json:"id" gorm:"primaryKey;size:16"`
	UploadID  string    `json:"uploadId" gorm:"size:256;index:idx_event_upload;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
	FileID    string    `json:"fileId,omitempty" gorm:"size:16"`
	FileName  string    `json:"fileName,omitempty" gorm:"size:1024"`
	FileSize  int64     `json:"fileSize,omitempty"`
	Type      string    `json:"type" gorm:"size:32;index:idx_event_type"`
	RemoteIP  string    `json:"remoteIp,omitempty"`
	User      string    `json:"user,omitempty"`
	Message   string    `json:"message" gorm:"-"`
	CreatedAt time.Time `json:"createdAt"`
}

// NewEvent creates a new event with a random ID
func NewEvent(uploadID string, eventType string) *Event {
	return &Event{
		ID:       GenerateRandomID(16),
		UploadID: uploadID,
		Type:     eventType,
	}
}

// Sanitize clears sensitive fields (RemoteIP, User) for non-admin callers
func (event *Event) Sanitize() {
	event.RemoteIP = ""
	event.User = ""
}

// ComputeMessage builds a human-readable detail string (complements the type badge)
func (event *Event) ComputeMessage() {
	switch event.Type {
	case EventUploadCreated:
		event.Message = ""
	case EventFileAdded, EventFileDownloaded:
		if event.FileName != "" {
			event.Message = fmt.Sprintf("%s (%s)", event.FileName, humanReadableSize(event.FileSize))
		}
	}
}

// humanReadableSize formats a byte count into a human-readable string
func humanReadableSize(bytes int64) string {
	if bytes < 0 {
		return "0 B"
	}
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "kMGTPE"[exp])
}
