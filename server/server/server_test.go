package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	data_test "github.com/root-gg/plik/server/data/testing"
	"github.com/root-gg/plik/server/metadata"
)

func newPlikServer() (ps *PlikServer) {
	ps = NewPlikServer(getTestConfig())
	ps.config.ListenAddress = "127.0.0.1"
	ps.config.ListenPort = common.APIMockServerDefaultPort
	ps.config.AutoClean(false)

	metadataBackendConfig := metadata.NewConfig(ps.config.MetadataBackendConfig)
	metadataBackendConfig.EraseFirst = true
	metadataBackend, err := metadata.NewBackend(metadataBackendConfig, ps.config.NewLogger())
	if err != nil {
		panic(err)
	}
	ps.WithMetadataBackend(metadataBackend)

	err = ps.initializeDataBackend()
	if err != nil {
		panic(err)
	}

	ps.WithStreamBackend(data_test.NewBackend())

	return ps
}

func getTestConfig() (testConfig *common.Configuration) {
	testConfigPath := os.Getenv("PLIKD_CONFIG")
	if testConfigPath == "" {
		testConfig = common.NewConfiguration()
		testConfig.DataBackend = "testing"
	}

	fmt.Println("loading test config : " + testConfigPath)
	testConfig, err := common.LoadConfiguration(testConfigPath)
	if err != nil {
		fmt.Printf("Unable to load test configuration : %s\n", err)
		os.Exit(1)
	}

	return testConfig
}

// Some data backend return an error on GetFile and some delay that to the first read on the reader
func getTestFile(t *testing.T, ps *PlikServer, file *common.File, content string) (err error) {
	reader, err := ps.dataBackend.GetFile(file)
	if err != nil {
		return err
	}
	require.NotNil(t, reader, "missing file reader")
	defer reader.Close()

	result, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	require.Equal(t, content, string(result), "invalid file content")
	return nil
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

func TestDataBackend(t *testing.T) {
	ps := newPlikServer()
	defer ps.ShutdownNow()

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	content := "data data data"
	err := ps.dataBackend.AddFile(file, bytes.NewBufferString(content))
	require.NoError(t, err, "unable to save file")

	err = getTestFile(t, ps, file, content)
	require.NoError(t, err, "unable to get file")

	err = getTestFile(t, ps, file, content)
	require.NoError(t, err, "unable to get file")

	err = ps.dataBackend.RemoveFile(file)
	require.NoError(t, err, "unable to remove file")

	err = getTestFile(t, ps, file, content)
	require.Error(t, err, "able to get removed file")

	// Test remove file twice
	err = ps.dataBackend.RemoveFile(file)
	require.NoError(t, err, "unable to remove removed file")

	err = getTestFile(t, ps, file, content)
	require.Error(t, err, "able to get removed file")
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
	upload.InitializeForTests()

	require.True(t, upload.IsExpired(), "upload should be expired")

	err := ps.metadataBackend.CreateUpload(upload)
	require.NoError(t, err, "unable to save upload")

	content := "data data data"
	err = ps.dataBackend.AddFile(file, bytes.NewBufferString(content))
	require.NoError(t, err, "unable to save file")

	ps.Clean()

	u, err := ps.metadataBackend.GetUpload(upload.ID)
	require.NoError(t, err, "unexpected unable to get upload")
	require.Nil(t, u, "should be unable to get expired upload after clean")

	err = getTestFile(t, ps, file, content)
	require.Error(t, err, "missing get file error")
}

func TestCleanUploadingFiles(t *testing.T) {
	ps := newPlikServer()
	defer ps.ShutdownNow()

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploading
	upload.TTL = 1
	deadline := time.Now().Add(-10 * time.Minute)
	upload.ExpireAt = &deadline
	upload.InitializeForTests()

	require.True(t, upload.IsExpired(), "upload should be expired")

	err := ps.metadataBackend.CreateUpload(upload)
	require.NoError(t, err, "unable to save upload")

	content := "data data data"
	err = ps.dataBackend.AddFile(file, bytes.NewBufferString(content))
	require.NoError(t, err, "unable to save file")

	ps.Clean()

	u, err := ps.metadataBackend.GetUpload(upload.ID)
	require.NoError(t, err, "unexpected unable to get upload")
	require.Nil(t, u, "should be unable to get expired upload after clean")

	err = getTestFile(t, ps, file, content)
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
	upload.InitializeForTests()

	require.True(t, upload.IsExpired(), "upload should be expired")

	err = ps.metadataBackend.CreateUpload(upload)
	require.NoError(t, err, "unable to save upload")

	time.Sleep(2 * time.Second)

	u, err := ps.metadataBackend.GetUpload(upload.ID)
	require.NoError(t, err, "unexpected unable to get upload")
	require.Nil(t, u, "should be unable to get expired upload after clean")
}
