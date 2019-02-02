/*
 * Charles-Antoine Mathieu <charles-antoine.mathieu@ovh.net>
 */

package common

import "sort"

type ServerStats struct {
	Users            int               `json:"users"`
	Uploads          int               `json:"uploads"`
	AnonymousUploads int               `json:"anonymousUploads"`
	Files            int               `json:"files"`
	TotalSize        int64             `json:"totalSize"`
	AnonymousSize    int64             `json:"anonymousTotalSize"`
	FileTypeByCount  []FileTypeByCount `json:"fileTypeByCount"`
	FileTypeBySize   []FileTypeBySize  `json:"fileTypeBySize"`
}

type FileTypeByCount struct {
	Type  string `json:"type" bson:"_id"`
	Total int    `json:"total" bson:"total"`
}

type FileTypeBySize struct {
	Type  string `json:"type" bson:"_id"`
	Total int64  `json:"total" bson:"total"`
}

// Helpers to build the Server Stats

// AddUpload add statistics of one upload to the ServerStats
func (stats *ServerStats) AddUpload(upload *Upload) {
	var uploadSize int64
	for _, file := range upload.Files {
		uploadSize += file.CurrentSize
	}

	stats.Uploads++
	stats.Files += len(upload.Files)
	stats.TotalSize += uploadSize

	if upload.User == "" {
		stats.AnonymousUploads++
		stats.AnonymousSize += uploadSize
	}
}

type byTypeValue struct {
	Count int
	Size  int64
}

type byTypeValuePair struct {
	key   string
	value *byTypeValue
}
type byTypeValuePairListByCount []byTypeValuePair

func (a byTypeValuePairListByCount) Len() int           { return len(a) }
func (a byTypeValuePairListByCount) Less(i, j int) bool { return a[i].value.Count > a[j].value.Count }
func (a byTypeValuePairListByCount) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type byTypeValuePairListBySize []byTypeValuePair

func (a byTypeValuePairListBySize) Len() int           { return len(a) }
func (a byTypeValuePairListBySize) Less(i, j int) bool { return a[i].value.Size > a[j].value.Size }
func (a byTypeValuePairListBySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// ByTypeAggregator helps to generate the FileTypeByCount and FileTypeBySize statistics
type ByTypeAggregator struct {
	values map[string]*byTypeValue
}

func NewByTypeAggregator() (aggr *ByTypeAggregator) {
	aggr = new(ByTypeAggregator)
	aggr.values = make(map[string]*byTypeValue)
	return aggr
}

// AddFile add a file statistics to the aggregator
func (aggr *ByTypeAggregator) AddFile(file *File) {
	if value, ok := aggr.values[file.Type]; ok {
		value.Count++
		value.Size += file.CurrentSize
	} else {
		aggr.values[file.Type] = &byTypeValue{1, file.CurrentSize}
	}
}

// GetFileTypeByCount get limit most FileTypeByCount
func (aggr *ByTypeAggregator) GetFileTypeByCount(limit int) []FileTypeByCount {
	array := make(byTypeValuePairListByCount, len(aggr.values))
	i := 0
	for k, v := range aggr.values {
		array[i] = byTypeValuePair{k, v}
		i++
	}
	sort.Sort(array)

	result := make([]FileTypeByCount, limit)

	for i, pair := range array {
		result[i] = FileTypeByCount{pair.key, pair.value.Count}

		i++
		if i >= limit {
			break
		}
	}

	return result
}

// GetFileTypeBySize get limit most FileTypeBySize
func (aggr *ByTypeAggregator) GetFileTypeBySize(limit int) []FileTypeBySize {
	array := make(byTypeValuePairListBySize, len(aggr.values))

	i := 0
	for k, v := range aggr.values {
		array[i] = byTypeValuePair{k, v}
		i++
	}
	sort.Sort(array)

	result := make([]FileTypeBySize, limit)

	for i, pair := range array {
		result[i] = FileTypeBySize{pair.key, pair.value.Size}

		i++
		if i >= limit {
			break
		}
	}

	return result
}
