package server

import (
	"bytes"
	"testing"
	"time"

	"github.com/root-gg/plik/server/metadata"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	data_test "github.com/root-gg/plik/server/data/testing"
)

func newPlikServer() (ps *PlikServer) {
	ps = NewPlikServer(common.NewConfiguration())
	ps.config.ListenAddress = "127.0.0.1"
	ps.config.ListenPort = common.APIMockServerDefaultPort
	ps.config.AutoClean(false)

	metadataBackendConfig := &metadata.Config{Driver: "sqlite3", ConnectionString: "/tmp/plik.test.db", EraseFirst: true}
	metadataBackend, err := metadata.NewBackend(metadataBackendConfig)
	if err != nil {
		panic(err)
	}
	ps.WithMetadataBackend(metadataBackend)

	ps.WithDataBackend(data_test.NewBackend())
	ps.WithStreamBackend(data_test.NewBackend())

	return ps
}

func TestNewPlikServer(t *testing.T) {
	config := common.NewConfiguration()
	ps := NewPlikServer(config)
	require.NotNil(t, ps, "invalid nil Plik server")
	require.NotNil(t, ps.GetConfig(), "invalid nil configuration")
}

func TestStartShutdownPlikServer(t *testing.T) {
	ps := newPlikServer()
	defer ps.ShutdownNow()

	err := ps.Start()
	require.NoError(t, err, "unable to start plik server")

	err = ps.Start()
	require.Error(t, err, "should not be able to start plik server twice")
	require.Equal(t, "can't start a Plik server twice", err.Error(), "invalid error")

	err = ps.ShutdownNow()
	require.NoError(t, err, "unable to shutdown plik server")

	err = ps.ShutdownNow()
	require.Error(t, err, "should not be able to shutdown plik server twice")
	require.Equal(t, "can't shutdown a Plik server twice", err.Error(), "invalid error")

	err = ps.Start()
	require.Error(t, err, "should not be able to start a shutdown plik server")
	require.Equal(t, "can't start a shutdown Plik server", err.Error(), "invalid error")
}

func TestNewPlikServerNoHTTPSCertificates(t *testing.T) {
	ps := NewPlikServer(common.NewConfiguration())
	ps.config.ListenAddress = "127.0.0.1"
	ps.config.ListenPort = 44142
	ps.config.AutoClean(false)
	ps.config.DataBackend = "testing"

	ps.config.SslEnabled = true

	err := ps.Start()
	require.Error(t, err, "unable to start plik server without ssl certificates")
}

func TestNewPlikServerWithCustomBackends(t *testing.T) {
	ps := newPlikServer()
	defer ps.ShutdownNow()

	ps.WithDataBackend(data_test.NewBackend())
	err := ps.initializeDataBackend()
	require.NoError(t, err, "invalid error")
	require.NotNil(t, ps.GetDataBackend(), "missing data backend")

	ps.WithStreamBackend(data_test.NewBackend())
	err = ps.initializeStreamBackend()
	require.NoError(t, err, "invalid error")
	require.NotNil(t, ps.GetStreamBackend(), "missing stream backend")

	err = ps.initializeMetadataBackend()
	require.NoError(t, err, "invalid error")
	require.NotNil(t, ps.GetMetadataBackend(), "missing metadata backend")

}

func TestClean(t *testing.T) {
	ps := newPlikServer()
	defer ps.ShutdownNow()

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.TTL = 1
	deadline := time.Now().Add(-10 * time.Minute)
	upload.ExpireAt = &deadline
	upload.PrepareInsertForTests()

	require.True(t, upload.IsExpired(), "upload should be expired")

	err := ps.metadataBackend.CreateUpload(upload)
	require.NoError(t, err, "unable to save upload")

	_, err = ps.dataBackend.AddFile(file, bytes.NewBufferString("data data data"))
	require.NoError(t, err, "unable to save file")

	ps.Clean()

	u, err := ps.metadataBackend.GetUpload(upload.ID)
	require.NoError(t, err, "unexpected unable to get upload")
	require.Nil(t, u, "should be unable to get expired upload after clean")

	_, err = ps.dataBackend.GetFile(file)
	require.Error(t, err, "missing get file error")
}

func TestAutoClean(t *testing.T) {
	ps := newPlikServer()
	defer ps.ShutdownNow()

	ps.cleaningRandomDelay = 1
	ps.cleaningMinOffset = 1
	ps.config.AutoClean(true)

	err := ps.Start()
	require.NoError(t, err, "unable to start plik server")

	upload := &common.Upload{}
	upload.TTL = 1
	deadline := time.Now().Add(-10 * time.Minute)
	upload.ExpireAt = &deadline
	upload.PrepareInsertForTests()

	require.True(t, upload.IsExpired(), "upload should be expired")

	err = ps.metadataBackend.CreateUpload(upload)
	require.NoError(t, err, "unable to save upload")

	time.Sleep(2 * time.Second)

	u, err := ps.metadataBackend.GetUpload(upload.ID)
	require.NoError(t, err, "unexpected unable to get upload")
	require.Nil(t, u, "should be unable to get expired upload after clean")
}
