package plik

import (
	"encoding/json"
	"fmt"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	"github.com/root-gg/plik/server/data/file"
	"github.com/root-gg/plik/server/data/swift"
	data_test "github.com/root-gg/plik/server/data/testing"
	"github.com/root-gg/plik/server/data/weedfs"
	"github.com/root-gg/plik/server/metadata"
	"github.com/root-gg/plik/server/metadata/bolt"
	"github.com/root-gg/plik/server/metadata/mongo"
	metadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/root-gg/plik/server/server"
	"github.com/root-gg/utils"
	"io/ioutil"
	"os"
	"testing"
)

var metadataBackend metadata.Backend
var dataBackend data.Backend

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

	var testConfig *common.Configuration
	testConfigPath := os.Getenv("PLIKD_CONFIG")
	if testConfigPath != "" {
		fmt.Println("loading test config : " + testConfigPath)
		testConfig, err = common.LoadConfiguration(testConfigPath)
		if err != nil {
			fmt.Printf("Unable to load test configuration : %s\n", err)
			os.Exit(1)
		}
		fmt.Println(testConfig.MetadataBackend)
		utils.Dump(testConfig.MetadataBackendConfig)
	} else {
		testConfig = common.NewConfiguration()
		testConfig.MetadataBackend = "testing"
		testConfig.DataBackend = "testing"
		if os.Getenv("metadata_backend") != "" {
			testConfig.MetadataBackend = os.Getenv("metadata_backend")
			if os.Getenv("metadata_backend_config") != "" {
				var metadataBackendConfig = make(map[string]interface{})
				err = json.Unmarshal([]byte(os.Getenv("metadata_backend_config")), &metadataBackendConfig)
				if err != nil {
					fmt.Printf("Unable to unserialize metadata_backend_config : %s\n", err)
					os.Exit(1)
				}
			}
		}
		if os.Getenv("data_backend") != "" {
			testConfig.DataBackend = os.Getenv("data_backend")
			if os.Getenv("data_backend_config") != "" {
				var dataBackendConfig = make(map[string]interface{})
				err = json.Unmarshal([]byte(os.Getenv("data_backend_config")), &dataBackendConfig)
				if err != nil {
					fmt.Printf("Unable to unserialize data_backend_config : %s\n", err)
					os.Exit(1)
				}
			}
		}
	}

	// Setup metadata backend
	switch testConfig.MetadataBackend {
	case "bolt":
		dir, err := ioutil.TempDir("", "pliktest_bolt_")
		if err != nil {
			fmt.Printf("Unable to setup bolt metadata backend : %s\n", err)
			os.Exit(1)
		}

		cleanMetadata = func() {
			err = os.RemoveAll(dir)
			if err != nil {
				fmt.Println(err)
			}
		}

		metadataBackend, err = bolt.NewBackend(&bolt.Config{Path: dir + "/plik.db"})
		if err != nil {
			fmt.Printf("Unable to setup bolt metadata backend : %s\n", err)
			os.Exit(1)
		}
		fmt.Println("running tests with bold metadata backend")
	case "mongo":
		mongoConfig := mongo.NewConfig(testConfig.MetadataBackendConfig)
		metadataBackend, err = mongo.NewBackend(mongoConfig)
		if err != nil {
			fmt.Printf("Unable to setup bolt metadata backend : %s\n", err)
			os.Exit(1)
		}
		fmt.Println("running tests with mongo metadata backend")
	case "testing":
		metadataBackend = metadata_test.NewBackend()
	default:
		fmt.Printf("Invalid metadata backend : %s\n", testConfig.MetadataBackend)
		os.Exit(1)
	}

	// Setup data backend
	switch testConfig.DataBackend {
	case "file":
		dir, err := ioutil.TempDir("", "pliktest_file_")
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
	case "weedfs":
		weedfsConfig := weedfs.NewConfig(testConfig.DataBackendConfig)
		dataBackend = weedfs.NewBackend(weedfsConfig)
		fmt.Println("running tests with weedfs data backend")
	case "testing":
		dataBackend = data_test.NewBackend()
	default:
		fmt.Printf("Invalid metadata backend : %s\n", testConfig.DataBackend)
		os.Exit(1)
	}

	// Run tests
	code = m.Run()
	os.Exit(code)
}

func newPlikServerAndClient() (ps *server.PlikServer, pc *Client) {
	config := common.NewConfiguration()
	config.ListenAddress = "127.0.0.1"
	config.ListenPort = common.APIMockServerDefaultPort
	config.AutoClean(false)
	_ = config.Initialize()
	ps = server.NewPlikServer(config)
	ps.WithMetadataBackend(metadataBackend)
	ps.WithDataBackend(dataBackend)
	pc = NewClient(config.GetServerURL().String())
	return ps, pc
}

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
