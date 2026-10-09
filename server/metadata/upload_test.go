package metadata

import (
	"fmt"
	"testing"
	"time"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"gorm.io/gorm"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func createUpload(t *testing.T, b *Backend, upload *common.Upload) {
	upload.InitializeForTests()
	err := b.CreateUpload(upload)
	require.NoError(t, err, "create upload error : %s", err)
}

func TestBackend_CreateUpload(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	file := upload.NewFile()

	createUpload(t, b, upload)

	require.NotZero(t, upload.ID, "missing upload id")
	require.NotZero(t, upload.CreatedAt, "missing creation date")
	require.NotZero(t, file.ID, "missing file id")
	require.Equal(t, upload.ID, file.UploadID, "missing file id")
	require.NotZero(t, file.CreatedAt, "missing creation date")
}

func TestBackend_GetUpload(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	_ = upload.NewFile()

	createUpload(t, b, upload)

	result, err := b.GetUpload(upload.ID)
	require.NoError(t, err, "get upload error")

	require.Equal(t, upload.ID, result.ID, "invalid upload id")
	require.Zero(t, result.Files, "invalid upload files")
	require.Equal(t, upload.UploadToken, result.UploadToken, "invalid upload token")
}

func TestBackend_GetUpload_NotFound(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload, err := b.GetUpload("not found")
	require.NoError(t, err, "get upload error")
	require.Nil(t, upload, "upload not nil")
}

func TestBackend_GetUploads_MissingPagingQuery(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	_, _, err := b.GetUploads(UploadFilters{}, false, nil)
	require.Error(t, err, "get upload error expected")
}

func TestBackend_DeleteUpload(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	_ = upload.NewFile()

	createUpload(t, b, upload)

	err := b.RemoveUpload(upload.ID)
	require.NoError(t, err, "get upload error")

	upload, err = b.GetUpload(upload.ID)
	require.NoError(t, err, "get upload error")
	require.Nil(t, upload, "upload not nil")
}

func TestBackend_GetUploads(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		upload.NewFile()
		createUpload(t, b, upload)
	}

	limit := 10
	uploads, cursor, err := b.GetUploads(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", 100-i), uploads[i].Comments, "invalid upload sequence")
	}

	//  Test forward cursor
	uploads, cursor, err = b.GetUploads(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithAfterCursor(*cursor.After))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.NotNil(t, cursor.Before, "invalid nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", 100-limit-i), uploads[i].Comments, "invalid upload sequence")
	}

	//  Test backward cursor
	uploads, cursor, err = b.GetUploads(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithBeforeCursor(*cursor.Before))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", 100-i), uploads[i].Comments, "invalid upload sequence")
	}
}

func TestBackend_GetUploadsWithFiles(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	upload.NewFile()
	createUpload(t, b, upload)

	uploads, cursor, err := b.GetUploads(UploadFilters{}, false, common.NewPagingQuery())
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, 1, "invalid upload count")
	require.Len(t, uploads[0].Files, 0, "invalid file count")
	require.Nil(t, cursor.After, "invalid non nil after cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")

	uploads, _, err = b.GetUploads(UploadFilters{}, true, common.NewPagingQuery())
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, 1, "invalid upload count")
	require.Len(t, uploads[0].Files, 1, "invalid file count")
}

func TestBackend_GetUploads_User(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := &common.User{ID: "user"}

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		if i%10 == 0 {
			upload.User = user.ID
		}
		createUpload(t, b, upload)
	}

	limit := 10
	uploads, cursor, err := b.GetUploads(UploadFilters{User: user.ID}, false, common.NewPagingQuery().WithLimit(limit))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")

	for i := range limit {
		expected := 100 - i*10
		require.Equal(t, fmt.Sprintf("%d", expected), uploads[i].Comments, "invalid upload sequence")
	}
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.Nil(t, cursor.After, "invalid non nil after cursor")
}

func TestBackend_GetUploads_Token(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	token := &common.Token{Token: "token"}

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		if i%10 == 0 {
			upload.Token = token.Token
		}
		createUpload(t, b, upload)
	}

	limit := 10
	uploads, cursor, err := b.GetUploads(UploadFilters{Token: token.Token}, false, &common.PagingQuery{Limit: &limit})
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")

	for i := 0; i < limit; i++ {
		expected := 100 - i*10
		require.Equal(t, fmt.Sprintf("%d", expected), uploads[i].Comments, "invalid upload sequence")
	}
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.Nil(t, cursor.After, "invalid non nil after cursor")
}

func TestBackend_GetUploadsAsc(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		f := upload.NewFile()
		f.Size = int64(i)
		f.Status = common.FileUploaded
		createUpload(t, b, upload)
	}

	limit := 10
	uploads, cursor, err := b.GetUploads(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithOrder("asc"))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", i+1), uploads[i].Comments, "invalid upload sequence")
	}

	//  Test forward cursor
	uploads, cursor, err = b.GetUploads(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithOrder("asc").WithAfterCursor(*cursor.After))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.NotNil(t, cursor.Before, "invalid nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", i+limit+1), uploads[i].Comments, "invalid upload sequence")
	}

	//  Test backward cursor
	uploads, cursor, err = b.GetUploads(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithOrder("asc").WithBeforeCursor(*cursor.Before))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", i+1), uploads[i].Comments, "invalid upload sequence")
	}
}

func TestBackend_GetUploadsSortedBySize(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		f := upload.NewFile()
		f.Size = int64(i)
		f.Status = common.FileUploaded
		createUpload(t, b, upload)
	}

	limit := 10
	uploads, cursor, err := b.GetUploadsSortedBySize(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", 100-i), uploads[i].Comments, "invalid upload sequence")
	}

	//  Test forward cursor
	uploads, cursor, err = b.GetUploadsSortedBySize(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithAfterCursor(*cursor.After))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.NotNil(t, cursor.Before, "invalid nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", 100-limit-i), uploads[i].Comments, "invalid upload sequence")
	}

	//  Test backward cursor
	uploads, cursor, err = b.GetUploadsSortedBySize(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithBeforeCursor(*cursor.Before))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", 100-i), uploads[i].Comments, "invalid upload sequence")
	}
}

func TestBackend_GetUploadsSortedBySizeWithFiles(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	f := upload.NewFile()
	f.Status = common.FileUploaded
	createUpload(t, b, upload)

	uploads, cursor, err := b.GetUploadsSortedBySize(UploadFilters{}, false, common.NewPagingQuery())
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, 1, "invalid upload count")
	require.Len(t, uploads[0].Files, 0, "invalid file count")
	require.Nil(t, cursor.After, "invalid non nil after cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")

	uploads, _, err = b.GetUploads(UploadFilters{}, true, common.NewPagingQuery())
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, 1, "invalid upload count")
	require.Len(t, uploads[0].Files, 1, "invalid file count")
}

func TestBackend_GetUploadsSortedBySize_EmptyUploads(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	// Create 3 uploads with files of varying sizes
	for i := 1; i <= 3; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("with-files-%d", i)}
		f := upload.NewFile()
		f.Size = int64(i * 100)
		f.Status = common.FileUploaded
		createUpload(t, b, upload)
	}

	// Create 2 uploads WITHOUT files
	for i := 1; i <= 2; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("empty-%d", i)}
		createUpload(t, b, upload)
	}

	// All 5 uploads should appear when sorting by size (desc)
	uploads, _, err := b.GetUploadsSortedBySize(UploadFilters{}, false, common.NewPagingQuery().WithLimit(10))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, 5, "uploads without files should still appear")

	// Desc order: largest first, empty uploads (size 0) at the end
	require.Equal(t, "with-files-3", uploads[0].Comments)
	require.Equal(t, "with-files-2", uploads[1].Comments)
	require.Equal(t, "with-files-1", uploads[2].Comments)

	// All 5 should also appear in asc order
	uploads, _, err = b.GetUploadsSortedBySize(UploadFilters{}, false, common.NewPagingQuery().WithLimit(10).WithOrder("asc"))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, 5, "uploads without files should still appear in asc order")

	// Asc order: empty uploads (size 0) first, then smallest to largest
	require.Equal(t, "with-files-3", uploads[len(uploads)-1].Comments)
}

func TestBackend_GetUploadsSortedBySizeAsc(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		f := upload.NewFile()
		f.Size = int64(i)
		f.Status = common.FileUploaded
		createUpload(t, b, upload)
	}

	limit := 10
	uploads, cursor, err := b.GetUploadsSortedBySize(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithOrder("asc"))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", i+1), uploads[i].Comments, "invalid upload sequence")
	}

	//  Test forward cursor
	uploads, cursor, err = b.GetUploadsSortedBySize(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithOrder("asc").WithAfterCursor(*cursor.After))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.NotNil(t, cursor.Before, "invalid nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", i+limit+1), uploads[i].Comments, "invalid upload sequence")
	}

	//  Test backward cursor
	uploads, cursor, err = b.GetUploadsSortedBySize(UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithOrder("asc").WithBeforeCursor(*cursor.Before))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.NotNil(t, cursor.After, "invalid nil after cursor")

	for i := range limit {
		require.Equal(t, fmt.Sprintf("%d", i+1), uploads[i].Comments, "invalid upload sequence")
	}
}

func TestBackend_GetUploadsSortedBySize_User(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	user := &common.User{ID: "user"}

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		f := upload.NewFile()
		f.Size = int64(i)
		f.Status = common.FileUploaded
		if i%10 == 0 {
			upload.User = user.ID
		}
		createUpload(t, b, upload)
	}

	limit := 10
	uploads, cursor, err := b.GetUploadsSortedBySize(UploadFilters{User: user.ID}, false, common.NewPagingQuery().WithLimit(limit))
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")

	for i := range limit {
		expected := 100 - i*10
		require.Equal(t, fmt.Sprintf("%d", expected), uploads[i].Comments, "invalid upload sequence")
	}
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.Nil(t, cursor.After, "invalid non nil after cursor")
}

func TestBackend_GetUploadsSortedBySize_Token(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	token := &common.Token{Token: "token"}

	for i := 1; i <= 100; i++ {
		upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
		f := upload.NewFile()
		f.Size = int64(i)
		f.Status = common.FileUploaded
		if i%10 == 0 {
			upload.Token = token.Token
		}
		createUpload(t, b, upload)
	}

	limit := 10
	uploads, cursor, err := b.GetUploadsSortedBySize(UploadFilters{Token: token.Token}, false, &common.PagingQuery{Limit: &limit})
	require.NoError(t, err, "get upload error")
	require.Len(t, uploads, limit, "invalid upload count")
	require.NotNil(t, cursor, "invalid nil cursor")

	for i := 0; i < limit; i++ {
		expected := 100 - i*10
		require.Equal(t, fmt.Sprintf("%d", expected), uploads[i].Comments, "invalid upload sequence")
	}
	require.Nil(t, cursor.Before, "invalid non nil before cursor")
	require.Nil(t, cursor.After, "invalid non nil after cursor")
}

// uploadSortFamily parameterizes the two upload sort methods that differ by
// exactly one seeded column (GetUploadsSortedByDownloads /
// GetUploadsSortedByDownloadedBytes), so their identical test shapes —
// pagination, zero-value ascending ordering, file preloading, user/token
// filtering, and the desc tie-breaker rule — run once per family via
// TestBackend_GetUploadsSortedByColumn instead of being hand-duplicated.
type uploadSortFamily struct {
	name   string
	sortBy func(b *Backend, filters UploadFilters, withFiles bool, pagingQuery *common.PagingQuery) ([]*common.Upload, *paginator.Cursor, error)
	seed   func(upload *common.Upload, value int64)
	value  func(upload *common.Upload) int64
}

var uploadSortFamilies = []uploadSortFamily{
	{
		name: "Downloads",
		sortBy: func(b *Backend, filters UploadFilters, withFiles bool, pagingQuery *common.PagingQuery) ([]*common.Upload, *paginator.Cursor, error) {
			return b.GetUploadsSortedByDownloads(filters, withFiles, pagingQuery)
		},
		seed:  func(upload *common.Upload, value int64) { upload.DownloadCount = value },
		value: func(upload *common.Upload) int64 { return upload.DownloadCount },
	},
	{
		name: "DownloadedBytes",
		sortBy: func(b *Backend, filters UploadFilters, withFiles bool, pagingQuery *common.PagingQuery) ([]*common.Upload, *paginator.Cursor, error) {
			return b.GetUploadsSortedByDownloadedBytes(filters, withFiles, pagingQuery)
		},
		seed:  func(upload *common.Upload, value int64) { upload.DownloadedBytes = value },
		value: func(upload *common.Upload) int64 { return upload.DownloadedBytes },
	},
}

// TestBackend_GetUploadsSortedByColumn table-drives the two upload sort
// families, which differ by exactly one paginator Rule. Both families
// exercise the same five behaviors: basic + cursor pagination, zero-value
// ascending ordering, file preloading, user/token filtering, and the desc
// tie-breaker rule (CreatedAt, then ID). The tie-breaker case previously
// existed only for the DownloadedBytes copy; both families now cover it.
func TestBackend_GetUploadsSortedByColumn(t *testing.T) {
	for _, family := range uploadSortFamilies {
		t.Run(family.name, func(t *testing.T) {
			t.Run("Paginated", func(t *testing.T) {
				b := newTestMetadataBackend()
				defer shutdownTestMetadataBackend(b)

				for i := 1; i <= 100; i++ {
					upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
					family.seed(upload, int64(i))
					createUpload(t, b, upload)
				}

				limit := 10
				uploads, cursor, err := family.sortBy(b, UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit))
				require.NoError(t, err, "get upload error")
				require.Len(t, uploads, limit, "invalid upload count")
				require.NotNil(t, cursor, "invalid nil cursor")
				require.Nil(t, cursor.Before, "invalid non nil before cursor")
				require.NotNil(t, cursor.After, "invalid nil after cursor")

				for i := range limit {
					require.Equal(t, fmt.Sprintf("%d", 100-i), uploads[i].Comments, "invalid upload sequence")
				}

				//  Test forward cursor
				uploads, cursor, err = family.sortBy(b, UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithAfterCursor(*cursor.After))
				require.NoError(t, err, "get upload error")
				require.Len(t, uploads, limit, "invalid upload count")
				require.NotNil(t, cursor, "invalid nil cursor")
				require.NotNil(t, cursor.Before, "invalid nil before cursor")
				require.NotNil(t, cursor.After, "invalid nil after cursor")

				for i := range limit {
					require.Equal(t, fmt.Sprintf("%d", 100-limit-i), uploads[i].Comments, "invalid upload sequence")
				}

				//  Test backward cursor
				uploads, cursor, err = family.sortBy(b, UploadFilters{}, false, common.NewPagingQuery().WithLimit(limit).WithBeforeCursor(*cursor.Before))
				require.NoError(t, err, "get upload error")
				require.Len(t, uploads, limit, "invalid upload count")
				require.NotNil(t, cursor, "invalid nil cursor")
				require.Nil(t, cursor.Before, "invalid non nil before cursor")
				require.NotNil(t, cursor.After, "invalid nil after cursor")

				for i := range limit {
					require.Equal(t, fmt.Sprintf("%d", 100-i), uploads[i].Comments, "invalid upload sequence")
				}
			})

			t.Run("AscAndZero", func(t *testing.T) {
				b := newTestMetadataBackend()
				defer shutdownTestMetadataBackend(b)

				values := []int64{3, 0, 2, 0, 1}
				for i, value := range values {
					upload := &common.Upload{Comments: fmt.Sprintf("%d", i+1)}
					family.seed(upload, value)
					createUpload(t, b, upload)
				}

				uploads, _, err := family.sortBy(b, UploadFilters{}, false, common.NewPagingQuery().WithLimit(10).WithOrder("asc"))
				require.NoError(t, err, "get upload error")
				require.Len(t, uploads, len(values), "uploads without a seeded value should still appear")
				require.Equal(t, int64(0), family.value(uploads[0]))
				require.Equal(t, int64(0), family.value(uploads[1]))
				require.Equal(t, int64(3), family.value(uploads[len(uploads)-1]))
			})

			t.Run("WithFiles", func(t *testing.T) {
				b := newTestMetadataBackend()
				defer shutdownTestMetadataBackend(b)

				upload := &common.Upload{}
				family.seed(upload, 1)
				f := upload.NewFile()
				f.Status = common.FileUploaded
				createUpload(t, b, upload)

				uploads, cursor, err := family.sortBy(b, UploadFilters{}, false, common.NewPagingQuery())
				require.NoError(t, err, "get upload error")
				require.Len(t, uploads, 1, "invalid upload count")
				require.Len(t, uploads[0].Files, 0, "invalid file count")
				require.Nil(t, cursor.After, "invalid non nil after cursor")
				require.Nil(t, cursor.Before, "invalid non nil before cursor")

				uploads, _, err = family.sortBy(b, UploadFilters{}, true, common.NewPagingQuery())
				require.NoError(t, err, "get upload error")
				require.Len(t, uploads, 1, "invalid upload count")
				require.Len(t, uploads[0].Files, 1, "invalid file count")
			})

			t.Run("UserAndToken", func(t *testing.T) {
				b := newTestMetadataBackend()
				defer shutdownTestMetadataBackend(b)

				user := &common.User{ID: "user"}
				token := &common.Token{Token: "token"}

				for i := 1; i <= 100; i++ {
					upload := &common.Upload{Comments: fmt.Sprintf("%d", i)}
					family.seed(upload, int64(i))
					if i%10 == 0 {
						upload.User = user.ID
					}
					if i%20 == 0 {
						upload.Token = token.Token
					}
					createUpload(t, b, upload)
				}

				limit := 10
				uploads, cursor, err := family.sortBy(b, UploadFilters{User: user.ID}, false, common.NewPagingQuery().WithLimit(limit))
				require.NoError(t, err, "get upload error")
				require.Len(t, uploads, limit, "invalid upload count")
				require.NotNil(t, cursor, "invalid nil cursor")

				for i := range limit {
					expected := 100 - i*10
					require.Equal(t, fmt.Sprintf("%d", expected), uploads[i].Comments, "invalid upload sequence")
				}
				require.Nil(t, cursor.Before, "invalid non nil before cursor")
				require.Nil(t, cursor.After, "invalid non nil after cursor")

				uploads, cursor, err = family.sortBy(b, UploadFilters{Token: token.Token}, false, &common.PagingQuery{Limit: &limit})
				require.NoError(t, err, "get upload error")
				require.Len(t, uploads, 5, "invalid upload count")
				require.NotNil(t, cursor, "invalid nil cursor")
				require.Equal(t, "100", uploads[0].Comments, "invalid upload sequence")
				require.Equal(t, "20", uploads[len(uploads)-1].Comments, "invalid upload sequence")
				require.Nil(t, cursor.Before, "invalid non nil before cursor")
				require.Nil(t, cursor.After, "invalid non nil after cursor")
			})

			// TieBreaker pins the deterministic tie-break rule for equal sort
			// values: CreatedAt (newest first for desc), then ID. Previously
			// only exercised for DownloadedBytes; both families now cover it.
			t.Run("TieBreaker", func(t *testing.T) {
				b := newTestMetadataBackend()
				defer shutdownTestMetadataBackend(b)

				base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				older := &common.Upload{Comments: "older"}
				family.seed(older, 500)
				older.CreatedAt = base
				createUpload(t, b, older)

				newer := &common.Upload{Comments: "newer"}
				family.seed(newer, 500)
				newer.CreatedAt = base.Add(time.Hour)
				createUpload(t, b, newer)

				uploads, _, err := family.sortBy(b, UploadFilters{}, false, common.NewPagingQuery().WithLimit(10))
				require.NoError(t, err, "get upload error")
				require.Len(t, uploads, 2)
				require.Equal(t, "newer", uploads[0].Comments, "equal values must break ties by newest CreatedAt first (desc)")
				require.Equal(t, "older", uploads[1].Comments)
			})
		})
	}
}

func TestBackend_DeleteExpiredUploads(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	// Each upload carries one uploaded file so cleanup must adjust current
	// counters, not just row counts.
	newUploadWithFile := func(size int64) *common.Upload {
		u := &common.Upload{}
		f := u.NewFile()
		f.Size = size
		f.Status = common.FileUploaded
		createUpload(t, b, u)
		return u
	}

	newUploadWithFile(1000) // never referenced again: only its presence among the retained uploads matters

	upload2 := newUploadWithFile(2000)
	deadline2 := time.Now().Add(time.Hour)
	upload2.ExpireAt = &deadline2
	err := b.UpdateUploadExpirationDate(upload2)
	require.NoError(t, err, "update upload error")

	upload3 := newUploadWithFile(4000)
	deadline3 := time.Now().Add(-time.Hour)
	upload3.ExpireAt = &deadline3
	err = b.UpdateUploadExpirationDate(upload3)
	require.NoError(t, err, "update upload error")

	before, err := b.GetServerStatistics()
	require.NoError(t, err, "get server stats error")
	require.Equal(t, 3, before.Uploads, "current uploads before cleanup")
	require.Equal(t, 3, before.Files, "current files before cleanup")
	require.Equal(t, int64(7000), before.TotalSize, "current size before cleanup")

	removed, err := b.RemoveExpiredUploads()
	require.Nil(t, err, "delete expired upload error")
	require.Equal(t, 1, removed, "removed expired upload count mismatch")

	// Only the expired upload/file leaves current usage; the two retained uploads
	// are untouched and lifetime counters never move.
	after, err := b.GetServerStatistics()
	require.NoError(t, err, "get server stats error")
	require.Equal(t, 2, after.Uploads, "current uploads after cleanup")
	require.Equal(t, 2, after.Files, "current files after cleanup")
	require.Equal(t, int64(3000), after.TotalSize, "current size after cleanup")
	require.Equal(t, 3, after.Usage.Lifetime.Uploads, "lifetime uploads unchanged")
	require.Equal(t, 3, after.Usage.Lifetime.Files, "lifetime files unchanged")
	require.Equal(t, int64(7000), after.Usage.Lifetime.TotalSize, "lifetime size unchanged")
}

func TestBackend_PurgeDeletedUploads(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}

	// All upload files need to be deleted from the data backend
	// So we can purge the upload and file metadata from the metadata backend
	file := upload.NewFile()
	file.Status = common.FileDeleted
	createUpload(t, b, upload)

	// Noop
	purged, err := b.DeleteRemovedUploads()
	require.NoError(t, err, "purge deleted upload error")
	require.Equal(t, 0, purged, "invalid purged count")

	u, err := b.GetUpload(upload.ID)
	require.NoError(t, err, "unable to get upload")
	require.Equal(t, upload.ID, u.ID, "unable to get upload")

	f, err := b.GetFile(file.ID)
	require.NoError(t, err, "unable to get file")
	require.Equal(t, file.ID, f.ID, "unable to get file")

	err = b.RemoveUpload(upload.ID)
	require.NoError(t, err, "delete upload error")

	purged, err = b.DeleteRemovedUploads()
	require.NoError(t, err, "purge deleted upload error")
	require.Equal(t, 1, purged, "invalid purged count")

	u, err = b.GetUpload(upload.ID)
	require.NoError(t, err, "unable to get upload")
	require.Nil(t, u, "upload is not nil")

	f, err = b.GetFile(file.ID)
	require.NoError(t, err, "unable to get file")
	require.Nil(t, f, "file is not nil")
}

// Same as below but with uploaded or uploading file status
func TestBackend_PurgeDeletedUploads_FixFileStatus(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}

	// All upload files need to be deleted from the data backend
	// So we can purge the upload and file metadata from the metadata backend
	file := upload.NewFile()
	file.Status = common.FileDeleted
	createUpload(t, b, upload)

	err := b.RemoveUpload(upload.ID)
	require.NoError(t, err, "delete upload error")

	// But sometimes shit happen and all the files are not properly deleted :'(
	err = b.UpdateFileStatus(file, common.FileDeleted, common.FileUploaded)
	require.Nil(t, err, "unable to update file status")

	purged, err := b.DeleteRemovedUploads()
	require.NoError(t, err, "unexpected purge deleted upload error")
	require.Equal(t, 0, purged, "invalid purged upload count")

	// Upload has been soft deleted by RemoveUpload
	u := &common.Upload{}
	err = b.db.Unscoped().Take(u, &common.Upload{ID: upload.ID}).Error
	require.NoError(t, err, "unable to get upload")
	require.Equal(t, upload.ID, u.ID, "unable to get upload")

	// File status has been updated to removed
	f, err := b.GetFile(file.ID)
	require.NoError(t, err, "unable to get file")
	require.Equal(t, file.ID, f.ID, "unable to get file")
	require.Equal(t, common.FileRemoved, f.Status, "invalid file status")

	// Let's simulate the removal of the file
	err = b.UpdateFileStatus(f, common.FileRemoved, common.FileDeleted)
	require.NoError(t, err, "unable to update file status")

	purged, err = b.DeleteRemovedUploads()
	require.NoError(t, err, "purge deleted upload error")
	require.Equal(t, 1, purged, "invalid purged count")

	err = b.db.Take(u, &common.Upload{ID: upload.ID}).Error
	require.Equal(t, gorm.ErrRecordNotFound, err, "unable to get upload")

	f, err = b.GetFile(f.ID)
	require.NoError(t, err, "unable to get file")
	require.Nil(t, f, "file is not nil")
}

// Same as above but with missing or empty file status
func TestBackend_PurgeDeletedUploads_FixFileStatusMissing(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}

	// All upload files need to be deleted from the data backend
	// So we can purge the upload and file metadata from the metadata backend
	file := upload.NewFile()
	file.Status = common.FileDeleted
	createUpload(t, b, upload)

	err := b.RemoveUpload(upload.ID)
	require.NoError(t, err, "delete upload error")

	// But sometimes shit happen and all the files are not properly deleted :'(
	err = b.UpdateFileStatus(file, common.FileDeleted, common.FileMissing)
	require.Nil(t, err, "unable to update file status")

	purged, err := b.DeleteRemovedUploads()
	require.NoError(t, err, "unexpected purge deleted upload error")
	require.Equal(t, 0, purged, "invalid purged upload count")

	// Upload has been soft deleted by RemoveUpload
	u := &common.Upload{}
	err = b.db.Unscoped().Take(u, &common.Upload{ID: upload.ID}).Error
	require.NoError(t, err, "unable to get upload")
	require.Equal(t, upload.ID, u.ID, "unable to get upload")

	// File status has been updated to deleted
	f, err := b.GetFile(file.ID)
	require.NoError(t, err, "unable to get file")
	require.Equal(t, file.ID, f.ID, "unable to get file")
	require.Equal(t, common.FileDeleted, f.Status, "invalid file status")

	purged, err = b.DeleteRemovedUploads()
	require.NoError(t, err, "purge deleted upload error")
	require.Equal(t, 1, purged, "invalid purged count")

	err = b.db.Take(u, &common.Upload{ID: upload.ID}).Error
	require.Equal(t, gorm.ErrRecordNotFound, err, "unable to get upload")

	f, err = b.GetFile(f.ID)
	require.NoError(t, err, "unable to get file")
	require.Nil(t, f, "file is not nil")
}

func TestBackend_ForEachUpload(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	upload.Comments = "foo bar"
	upload.NewFile()
	createUpload(t, b, upload)

	count := 0
	f := func(upload *common.Upload) error {
		count++
		require.Equal(t, "foo bar", upload.Comments, "invalid upload comments")
		return nil
	}
	err := b.ForEachUpload(f)
	require.NoError(t, err, "for each upload error : %s", err)
	require.Equal(t, 1, count, "invalid upload count")

	f = func(upload *common.Upload) error {
		return fmt.Errorf("expected")
	}
	err = b.ForEachUpload(f)
	require.Errorf(t, err, "expected")
}

func TestBackend_ForEachUploadUnscoped(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload1 := &common.Upload{}
	upload1.NewFile()
	createUpload(t, b, upload1)

	upload2 := &common.Upload{}
	upload2.NewFile()
	createUpload(t, b, upload2)

	count := 0
	f := func(upload *common.Upload) error {
		count++
		return nil
	}
	err := b.ForEachUpload(f)
	require.NoError(t, err, "for each upload error : %s", err)
	require.Equal(t, 2, count, "invalid upload count")

	count = 0
	err = b.ForEachUploadUnscoped(f)
	require.NoError(t, err, "for each upload error : %s", err)
	require.Equal(t, 2, count, "invalid upload count")

	err = b.RemoveUpload(upload1.ID)
	require.NoError(t, err, "unable to delete upload1")

	count = 0
	err = b.ForEachUpload(f)
	require.NoError(t, err, "for each upload error : %s", err)
	require.Equal(t, 1, count, "invalid upload count")

	count = 0
	err = b.ForEachUploadUnscoped(f)
	require.NoError(t, err, "for each upload error : %s", err)
	require.Equal(t, 2, count, "invalid upload count")
}

func TestBackend_UpdateUploadExpirationDate(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	upload.TTL = 1
	createUpload(t, b, upload)

	upload, err := b.GetUpload(upload.ID)
	require.NoError(t, err)
	require.NotNil(t, upload.ExpireAt)

	require.False(t, upload.IsExpired())
	time.Sleep(time.Second)
	require.True(t, upload.IsExpired())

	upload.ExtendExpirationDate()
	err = b.UpdateUploadExpirationDate(upload)
	require.NoError(t, err)

	upload, err = b.GetUpload(upload.ID)
	require.NoError(t, err)
	require.NotNil(t, upload.ExpireAt)

	require.False(t, upload.IsExpired())
	time.Sleep(time.Second)
	require.True(t, upload.IsExpired())
}
