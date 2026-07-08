package plik

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func getJSONE2E(t *testing.T, client *http.Client, url string, out any) {
	t.Helper()
	resp, err := client.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected response: %s", string(body))
	err = json.Unmarshal(body, out)
	require.NoError(t, err)
}

func rangeDownloadE2E(t *testing.T, file *File, uploadToken string, rangeHeader string) {
	t.Helper()
	fileURL, err := file.GetURL()
	require.NoError(t, err)
	req, err := http.NewRequest("GET", fileURL.String(), nil)
	require.NoError(t, err)
	req.Header.Set("X-UploadToken", uploadToken)
	req.Header.Set("Range", rangeHeader)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusPartialContent, resp.StatusCode)
}

func findTrendingItemE2E(items []*common.TrendingItem, id string) *common.TrendingItem {
	for _, item := range items {
		if item.ID == id {
			return item
		}
	}
	return nil
}

func TestStatsE2E(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced
	_ = ps.GetConfig().Initialize()

	suffix := time.Now().UnixNano()
	login := fmt.Sprintf("stats-%d", suffix)
	password := "stats-password"

	user := common.NewUser(common.ProviderLocal, login)
	user.Login = login
	user.IsAdmin = true
	hash, err := common.HashPassword(password)
	require.NoError(t, err)
	user.Password = hash
	token := user.NewToken()

	err = startWithClient(ps, pc)
	require.NoError(t, err, "unable to start Plik server")

	err = ps.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create user")

	baseURL := ps.GetConfig().GetServerURL().String()
	browserClient, _ := loginAndGetXSRF(t, baseURL, login, password)

	pc.Token = token.Token
	data := "stats data for range and archive"
	upload, file, err := pc.UploadReader(fmt.Sprintf("stats-%d.txt", suffix), bytes.NewBufferString(data))
	require.NoError(t, err, "unable to upload file")

	var stats common.UserStats
	getJSONE2E(t, browserClient, baseURL+"/me/stats", &stats)
	require.Equal(t, 1, stats.Uploads)
	require.Equal(t, 1, stats.Files)
	require.Equal(t, int64(len(data)), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(len(data)), stats.Usage.Lifetime.TotalSize)
	require.NotNil(t, stats.Usage.StartedAt)
	require.False(t, stats.Usage.StartedAt.IsZero())

	reader, err := file.Download()
	require.NoError(t, err)
	_, err = io.Copy(io.Discard, reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())

	rangeDownloadE2E(t, file, upload.Metadata().UploadToken, "bytes=0-")
	rangeDownloadE2E(t, file, upload.Metadata().UploadToken, "bytes=1-")

	archiveReader, err := upload.DownloadZipArchive()
	require.NoError(t, err)
	_, err = io.Copy(io.Discard, archiveReader)
	require.NoError(t, err)
	require.NoError(t, archiveReader.Close())

	var trendingFiles []*common.TrendingItem
	getJSONE2E(t, browserClient, baseURL+"/stats/trending/files?window=1d&limit=100", &trendingFiles)
	trendingFile := findTrendingItemE2E(trendingFiles, file.Metadata().ID)
	require.NotNil(t, trendingFile, "missing file from trending")
	require.Equal(t, int64(3), trendingFile.DownloadCount)

	var trendingUploads []*common.TrendingItem
	getJSONE2E(t, browserClient, baseURL+"/stats/trending/uploads?window=all&limit=100", &trendingUploads)
	trendingUpload := findTrendingItemE2E(trendingUploads, upload.Metadata().ID)
	require.NotNil(t, trendingUpload, "missing upload from trending")
	require.Equal(t, int64(3), trendingUpload.DownloadCount)

	err = upload.Delete()
	require.NoError(t, err)

	getJSONE2E(t, browserClient, baseURL+"/me/stats", &stats)
	require.Equal(t, 0, stats.Uploads)
	require.Equal(t, 0, stats.Files)
	require.Equal(t, int64(0), stats.TotalSize)
	require.Equal(t, 1, stats.Usage.Lifetime.Uploads)
	require.Equal(t, 1, stats.Usage.Lifetime.Files)
	require.Equal(t, int64(len(data)), stats.Usage.Lifetime.TotalSize)
}
