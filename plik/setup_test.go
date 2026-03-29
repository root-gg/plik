package plik

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	"github.com/root-gg/plik/server/data/file"
	"github.com/root-gg/plik/server/data/s3"
	"github.com/root-gg/plik/server/data/swift"
	data_test "github.com/root-gg/plik/server/data/testing"
	"github.com/root-gg/plik/server/metadata"
	"github.com/root-gg/plik/server/server"
)

// /!\ Backends ARE NOT automatically cleared between tests /!\
var dataBackend data.Backend

// Default metadata backend config
var metadataBackendConfig = &metadata.Config{Driver: "sqlite3", ConnectionString: "/tmp/plik.test.db", EraseFirst: true}

// testConfig holds the full configuration loaded from PLIKD_CONFIG (if set)
var testConfig *common.Configuration

func TestMain(m *testing.M) {
	var err error

	// Setup cleaning
	code := 0
	cleanMetadata := func() {}
	cleanData := func() {}
	defer func() {
		cleanMetadata()
		cleanData()
		os.Exit(code)
	}()

	testConfigPath := os.Getenv("PLIKD_CONFIG")
	if testConfigPath != "" {
		fmt.Println("loading test config : " + testConfigPath)
		testConfig, err = common.LoadConfiguration(testConfigPath)
		if err != nil {
			fmt.Printf("Unable to load test configuration : %s\n", err)
			os.Exit(1)
		}

		// Override default metadata backend config
		metadataBackendConfig = metadata.NewConfig(testConfig.MetadataBackendConfig)
		metadataBackendConfig.EraseFirst = true
	} else {
		testConfig = common.NewConfiguration()
		testConfig.DataBackend = "testing"
		if os.Getenv("data_backend") != "" {
			testConfig.DataBackend = os.Getenv("data_backend")
			if os.Getenv("data_backend_config") != "" {
				var dataBackendConfig = make(map[string]any)
				err = json.Unmarshal([]byte(os.Getenv("data_backend_config")), &dataBackendConfig)
				if err != nil {
					fmt.Printf("Unable to deserialize data_backend_config : %s\n", err)
					os.Exit(1)
				}
				testConfig.DataBackendConfig = dataBackendConfig
			}
		}
	}

	// Setup data backend
	switch testConfig.DataBackend {
	case "file":
		dir, err := os.MkdirTemp("", "pliktest_file_")
		if err != nil {
			fmt.Printf("Unable to setup file data backend : %s\n", err)
			os.Exit(1)
		}

		cleanData = func() {
			err := os.RemoveAll(dir)
			if err != nil {
				fmt.Println(err)
			}
		}

		dataBackend = file.NewBackend(&file.Config{Directory: dir})
		fmt.Println("running tests with file data backend")
	case "swift":
		swiftConfig := swift.NewConfig(testConfig.DataBackendConfig)
		dataBackend = swift.NewBackend(swiftConfig)
		fmt.Println("running tests with swift data backend")
	case "testing":
		dataBackend = data_test.NewBackend()
	case "s3":
		s3Config := s3.NewConfig(testConfig.DataBackendConfig)
		dataBackend, err = s3.NewBackend(s3Config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Invalid data backend : %s\n", testConfig.DataBackend)
		os.Exit(1)
	}

	// Run tests
	code = m.Run()
}

// /!\ Backends ARE NOT automatically cleared between tests /!\
func newPlikServerAndClient() (ps *server.PlikServer, pc *Client) {
	config := common.NewConfiguration()
	config.ListenAddress = "127.0.0.1"
	config.ListenPort = 0 // Use ephemeral port to avoid port conflicts
	config.AutoClean(false)
	//config.Debug = true

	// Copy settings from test config (PLIKD_CONFIG) if available
	if testConfig != nil {
		// Copy OIDC settings; FeatureAuthentication and FeatureLocalLogin
		// are intentionally left for each test to set as needed.
		config.OIDCClientID = testConfig.OIDCClientID
		config.OIDCClientSecret = testConfig.OIDCClientSecret
		config.OIDCProviderURL = testConfig.OIDCProviderURL
		config.OIDCProviderName = testConfig.OIDCProviderName

		// Use the configured listen port when running with an external
		// config (e.g., test-backends with Keycloak) so the port matches
		// Keycloak's allowed redirect URIs.
		if testConfig.ListenPort != 0 {
			config.ListenPort = testConfig.ListenPort
		}
	}

	_ = config.Initialize(nil)
	ps = server.NewPlikServer(config)

	metadataBackend, err := metadata.NewBackend(metadataBackendConfig, config.NewLogger())
	if err != nil {
		panic(err)
	}
	ps.WithMetadataBackend(metadataBackend)

	ps.WithDataBackend(dataBackend)
	// Client URL will be set after Start() once the actual port is known
	pc = NewClient(fmt.Sprintf("http://127.0.0.1:%d", common.APIMockServerDefaultPort))
	return ps, pc
}

// /!\ Backends ARE NOT automatically cleared between tests /!\
func start(ps *server.PlikServer) (err error) {
	err = ps.Start()
	if err != nil {
		return err
	}

	err = common.CheckHTTPServer(ps.GetConfig().ListenPort)
	if err != nil {
		return err
	}

	return nil
}

// startWithClient starts the server and updates the client URL with the actual port
func startWithClient(ps *server.PlikServer, pc *Client) (err error) {
	err = start(ps)
	if err != nil {
		return err
	}
	pc.URL = ps.GetConfig().GetServerURL().String()
	return nil
}

// /!\ Backends ARE NOT automatically cleared between tests /!\
func shutdown(ps *server.PlikServer) {
	err := ps.ShutdownNow()
	if err != nil {
		panic("unable to shutdown server " + err.Error())
	}
}

type LockedReader struct {
	lock   chan struct{}
	reader io.Reader
}

func NewLockedReader(reader io.Reader) (lr *LockedReader) {
	lr = new(LockedReader)
	lr.lock = make(chan struct{})
	lr.reader = reader
	return lr
}

func (lr *LockedReader) Read(p []byte) (n int, err error) {
	<-lr.lock
	return lr.reader.Read(p)
}

func (lr *LockedReader) Unleash() {
	close(lr.lock)
}

func NewSlowReaderRandom(reader io.Reader) (lr *LockedReader) {
	timeout := time.Duration(rand.Intn(1000)) * time.Millisecond
	return NewSlowReader(reader, timeout)
}

func NewSlowReader(reader io.Reader, timeout time.Duration) (lr *LockedReader) {
	lr = NewLockedReader(reader)
	go func() {
		<-time.After(timeout)
		lr.Unleash()
	}()
	return lr
}
