package metadata

import (
	"fmt"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"

	"github.com/root-gg/plik/server/common"
)

// CreateEvent create a new event in DB
func (b *Backend) CreateEvent(event *common.Event) (err error) {
	return b.db.Create(event).Error
}

// GetUploadEvents return paginated events for a specific upload
func (b *Backend) GetUploadEvents(uploadID string, pagingQuery *common.PagingQuery) (events []*common.Event, cursor *paginator.Cursor, err error) {
	if pagingQuery == nil {
		return nil, nil, fmt.Errorf("missing paging query")
	}

	stmt := b.db.
		Model(&common.Event{}).
		Where(&common.Event{UploadID: uploadID})

	p := pagingQuery.Paginator()
	p.SetKeys("CreatedAt", "ID")

	result, c, err := p.Paginate(stmt, &events)
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}

	return events, &c, err
}

// CountUploadEvents return the total number of events for an upload
func (b *Backend) CountUploadEvents(uploadID string) (count int64, err error) {
	err = b.db.Model(&common.Event{}).
		Where(&common.Event{UploadID: uploadID}).
		Count(&count).Error
	return count, err
}

// GetEvents return paginated events across all uploads with optional filters
func (b *Backend) GetEvents(uploadID string, eventType string, pagingQuery *common.PagingQuery) (events []*common.Event, cursor *paginator.Cursor, err error) {
	if pagingQuery == nil {
		return nil, nil, fmt.Errorf("missing paging query")
	}

	stmt := b.db.Model(&common.Event{})

	if uploadID != "" {
		stmt = stmt.Where("upload_id = ?", uploadID)
	}
	if eventType != "" {
		stmt = stmt.Where("type = ?", eventType)
	}

	p := pagingQuery.Paginator()
	p.SetKeys("CreatedAt", "ID")

	result, c, err := p.Paginate(stmt, &events)
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}

	return events, &c, err
}

// CountEvents return the total number of events matching the optional filters
func (b *Backend) CountEvents(uploadID string, eventType string) (count int64, err error) {
	stmt := b.db.Model(&common.Event{})

	if uploadID != "" {
		stmt = stmt.Where("upload_id = ?", uploadID)
	}
	if eventType != "" {
		stmt = stmt.Where("type = ?", eventType)
	}

	err = stmt.Count(&count).Error
	return count, err
}

// ForEachEvent execute f for every event in the database
func (b *Backend) ForEachEvent(f func(event *common.Event) error) (err error) {
	rows, err := b.db.Model(&common.Event{}).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		event := &common.Event{}
		err = b.db.ScanRows(rows, event)
		if err != nil {
			return err
		}
		err = f(event)
		if err != nil {
			return err
		}
	}

	return nil
}
