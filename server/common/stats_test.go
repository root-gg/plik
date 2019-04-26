/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015> Copyright holders list can be found in AUTHORS file
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerStatsAddUpload(t *testing.T) {
	stats := &ServerStats{}

	upload1 := NewUpload()
	file1 := upload1.NewFile()
	file1.CurrentSize = 1
	file2 := upload1.NewFile()
	file2.CurrentSize = 2

	upload2 := NewUpload()
	upload2.User = "user"
	file3 := upload2.NewFile()
	file3.CurrentSize = 1

	stats.AddUpload(upload1)
	stats.AddUpload(upload2)

	require.Equal(t, 2, stats.Uploads, "invalid upload count")
	require.Equal(t, 3, stats.Files, "invalid file count")
	require.Equal(t, int64(4), stats.TotalSize, "invalid file size")
	require.Equal(t, 1, stats.AnonymousUploads, "invalid anonymous file count")
	require.Equal(t, int64(3), stats.AnonymousSize, "invalid anonymous file size")
}

func TestByTypeAggregator(t *testing.T) {
	aggr := NewByTypeAggregator()

	type pair struct {
		typ   string
		size  int64
		count int
	}

	plan := []pair{
		{"type1", 1, 1},
		{"type2", 1000, 5},
		{"type3", 1000 * 1000, 10},
		{"type4", 1000 * 1000 * 1000, 15},
	}

	for _, item := range plan {
		for i := 0; i < item.count; i++ {
			file := NewFile()
			file.Type = item.typ
			file.CurrentSize = item.size
			aggr.AddFile(file)
		}
	}

	fileTypeByCounts := aggr.GetFileTypeByCount(1)
	require.Len(t, fileTypeByCounts, 1, "invalid length")
	require.Equal(t, "type4", fileTypeByCounts[0].Type, "invalid type")
	require.Equal(t, 15, fileTypeByCounts[0].Total, "invalid total")

	fileTypeByCounts = aggr.GetFileTypeByCount(2)
	require.Len(t, fileTypeByCounts, 2, "invalid length")
	require.Equal(t, "type3", fileTypeByCounts[1].Type, "invalid type")
	require.Equal(t, 10, fileTypeByCounts[1].Total, "invalid total")

	fileTypeBySize := aggr.GetFileTypeBySize(1)
	require.Len(t, fileTypeBySize, 1, "invalid length")
	require.Equal(t, "type4", fileTypeBySize[0].Type, "invalid type")
	require.Equal(t, int64(15*1000*1000*1000), fileTypeBySize[0].Total, "invalid total")

	fileTypeBySize = aggr.GetFileTypeBySize(2)
	require.Len(t, fileTypeBySize, 2, "invalid length")
	require.Equal(t, "type3", fileTypeBySize[1].Type, "invalid type")
	require.Equal(t, int64(10*1000*1000), fileTypeBySize[1].Total, "invalid total")
}
