package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/docopt/docopt-go"
	"github.com/mitchellh/go-homedir"

	"github.com/root-gg/plik/plik"
	"github.com/root-gg/plik/server/common"
)

// CliConfig object
type CliConfig struct {
	Debug          bool
	Quiet          bool
	URL            string
	OneShot        bool
	Removable      bool
	Stream         bool
	Secure         bool
	SecureMethod   string
	SecureOptions  map[string]interface{}
	Archive        bool
	ArchiveMethod  string
	ArchiveOptions map[string]interface{}
	DownloadBinary string
	Comments       string
	Login          string
	Password       string
	TTL            int
	ExtendTTL      bool
	AutoUpdate     bool
	Token          string
	DisableStdin   bool
	Insecure       bool

	filePaths        []string
	filenameOverride string
}

// NewUploadConfig construct a new configuration with default values
func NewUploadConfig() (config *CliConfig) {
	config = new(CliConfig)
	config.URL = "http://127.0.0.1:8080"
	config.ArchiveMethod = "tar"
	config.ArchiveOptions = make(map[string]interface{})
	config.ArchiveOptions["Tar"] = "/bin/tar"
	config.ArchiveOptions["Compress"] = "gzip"
	config.ArchiveOptions["Options"] = ""
	config.SecureMethod = "openssl"
	config.SecureOptions = make(map[string]interface{})
	config.SecureOptions["Openssl"] = "/usr/bin/openssl"
	config.SecureOptions["Cipher"] = "aes-256-cbc"
	config.SecureOptions["Options"] = "-md sha512 -pbkdf2 -iter 120000"
	config.DownloadBinary = "curl"
	return
}

// LoadConfigFromFile load TOML config file
func LoadConfigFromFile(path string) (*CliConfig, error) {
	config := NewUploadConfig()
	if _, err := toml.DecodeFile(path, config); err != nil {
		return nil, fmt.Errorf("Failed to deserialize ~/.plickrc : %s", err)
	}

	return config, nil
}

// LoadConfig creates a new default configuration and override it with .plikrc file.
// If .plikrc does not exist, ask domain, and create a new one in user HOMEDIR
func LoadConfig(opts docopt.Opts) (config *CliConfig, err error) {
	// Load config file from environment variable
	path := os.Getenv("PLIKRC")
	if path != "" {
		_, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("Plikrc file %s not found", path)
		}
		return LoadConfigFromFile(path)
	}

	// Detect home dir
	home, err := homedir.Dir()
	if err != nil {
		home = os.Getenv("HOME")
		if home == "" {
			home = "."
		}
	}

	// Load config file from ~/.plikrc
	path = home + "/.plikrc"
	_, err = os.Stat(path)
	if err == nil {
		config, err = LoadConfigFromFile(path)
		if err == nil {
			return config, nil
		}
	} else {
		// Load global config file from /etc directory
		path = "/etc/plik/plikrc"
		_, err = os.Stat(path)
		if err == nil {
			config, err = LoadConfigFromFile(path)
			if err == nil {
				return config, nil
			}
		}
	}

	config = NewUploadConfig()

	// Bypass ~/.plikrc file creation if quiet mode and/or --server flag
	if opts["--quiet"].(bool) || (opts["--server"] != nil && opts["--server"].(string) != "") {
		return config, nil
	}

	// Config file not found. Create one.
	path = home + "/.plikrc"

	// Ask for domain
	var domain string
	fmt.Println("Please enter your plik domain [default:http://127.0.0.1:8080] : ")
	_, err = fmt.Scanf("%s", &domain)
	if err == nil {
		domain = strings.TrimRight(domain, "/")
		parsedDomain, err := url.Parse(domain)
		if err == nil {
			if parsedDomain.Scheme == "" {
				parsedDomain.Scheme = "http"
			}
			config.URL = parsedDomain.String()
		}
	}

	// Try to HEAD the site to see if we have a redirection
	client := plik.NewClient(config.URL)
	client.Insecure()
	resp, err := client.HTTPClient.Head(config.URL)
	if err != nil {
		return nil, err
	}

	finalURL := resp.Request.URL.String()
	if finalURL != "" && finalURL != config.URL {
		fmt.Printf("We have been redirected to : %s\n", finalURL)
		fmt.Printf("Replace current url (%s) with the new one ? [Y/n] ", config.URL)

		ok, err := common.AskConfirmation(true)
		if err != nil {
			return nil, fmt.Errorf("Unable to ask for confirmation : %s", err)
		}
		if ok {
			config.URL = strings.TrimSuffix(finalURL, "/")
		}
	}

	// Try to get server config to sync default values
	serverConfig, err := client.GetServerConfig()
	if err != nil {
		fmt.Printf("Unable to get server configuration : %s", err)
	} else {
		config.OneShot = common.IsFeatureDefault(serverConfig.FeatureOneShot)
		config.Removable = common.IsFeatureDefault(serverConfig.FeatureRemovable)
		config.Stream = common.IsFeatureDefault(serverConfig.FeatureStream)
		config.ExtendTTL = common.IsFeatureDefault(serverConfig.FeatureExtendTTL)

		if serverConfig.FeatureAuthentication == common.FeatureForced {
			fmt.Printf("Anonymous uploads are disabled on this server")
			fmt.Printf("Do you want to provide a user authentication token ? [Y/n] ")
			ok, err := common.AskConfirmation(true)
			if err != nil {
				return nil, fmt.Errorf("Unable to ask for confirmation : %s", err)
			}
			if ok {
				var token string
				fmt.Println("Please enter a valid user token : ")
				_, err = fmt.Scanf("%s", &token)
				if err == nil {
					config.Token = token
				}
			}
		}
	}

	// Enable client updates ?
	fmt.Println("Do you want to enable client auto update ? [Y/n] ")
	ok, err := common.AskConfirmation(true)
	if err != nil {
		return nil, fmt.Errorf("Unable to ask for confirmation : %s", err)
	}
	if ok {
		config.AutoUpdate = true
	}

	// Encode in TOML
	buf := new(bytes.Buffer)
	if err = toml.NewEncoder(buf).Encode(config); err != nil {
		return nil, fmt.Errorf("Failed to serialize ~/.plickrc : %s", err)
	}

	// Write file
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0700)
	if err != nil {
		return nil, fmt.Errorf("Failed to save ~/.plickrc : %s", err)
	}

	_, _ = f.Write(buf.Bytes())
	_ = f.Close()

	fmt.Println("Plik client settings successfully saved to " + path)
	return config, nil
}

// UnmarshalArgs turns command line arguments into upload settings
// Command line arguments override config file settings
func (config *CliConfig) UnmarshalArgs(opts docopt.Opts) (err error) {
	if opts["--debug"].(bool) {
		config.Debug = true
	}
	if opts["--quiet"].(bool) {
		config.Quiet = true
	}

	// Plik server url
	if opts["--server"] != nil && opts["--server"].(string) != "" {
		config.URL = opts["--server"].(string)
	}

	// Paths
	if _, ok := opts["FILE"].([]string); ok {
		config.filePaths = opts["FILE"].([]string)
	} else {
		return fmt.Errorf("No files specified")
	}

	for _, path := range config.filePaths {
		// Test if file exists
		fileInfo, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("File %s not found", path)
		}

		// Automatically enable archive mode is at least one file is a directory
		if fileInfo.IsDir() {
			config.Archive = true
		}
	}

	// Override file name if specified
	if opts["--name"] != nil && opts["--name"].(string) != "" {
		config.filenameOverride = opts["--name"].(string)
	}

	// Upload options
	if opts["--oneshot"].(bool) {
		config.OneShot = true
	}
	if opts["--removable"].(bool) {
		config.Removable = true
	}

	if opts["--stream"].(bool) {
		config.Stream = true
	}

	if opts["--comments"] != nil && opts["--comments"].(string) != "" {
		config.Comments = opts["--comments"].(string)
	}

	// Configure upload expire date
	if opts["--ttl"] != nil && opts["--ttl"].(string) != "" {
		ttlStr := opts["--ttl"].(string)
		mul := 1
		if string(ttlStr[len(ttlStr)-1]) == "m" {
			mul = 60
		} else if string(ttlStr[len(ttlStr)-1]) == "h" {
			mul = 3600
		} else if string(ttlStr[len(ttlStr)-1]) == "d" {
			mul = 86400
		}
		if mul != 1 {
			ttlStr = ttlStr[:len(ttlStr)-1]
		}
		ttl, err := strconv.Atoi(ttlStr)
		if err != nil {
			return fmt.Errorf("Invalid TTL %s", opts["--ttl"].(string))
		}
		config.TTL = ttl * mul
	}

	if opts["--extend-ttl"].(bool) {
		config.ExtendTTL = true
	}

	// Enable archive mode ?
	if opts["-a"].(bool) || opts["--archive"] != nil || config.Archive {
		config.Archive = true

		if opts["--archive"] != nil && opts["--archive"] != "" {
			config.ArchiveMethod = opts["--archive"].(string)
		}
	}

	// Enable secure mode ?
	if opts["--not-secure"].(bool) {
		config.Secure = false
	} else if opts["-s"].(bool) || opts["--secure"] != nil || config.Secure {
		config.Secure = true
		if opts["--secure"] != nil && opts["--secure"].(string) != "" {
			config.SecureMethod = opts["--secure"].(string)
		}
	}

	// Enable password protection ?
	if opts["-p"].(bool) {
		fmt.Printf("Login [plik]: ")
		var err error
		_, err = fmt.Scanln(&config.Login)
		if err != nil && err.Error() != "unexpected newline" {
			return fmt.Errorf("Unable to get login : %s", err)
		}
		if config.Login == "" {
			config.Login = "plik"
		}
		fmt.Printf("Password: ")
		_, err = fmt.Scanln(&config.Password)
		if err != nil {
			return fmt.Errorf("Unable to get password : %s", err)
		}
	} else if opts["--password"] != nil && opts["--password"].(string) != "" {
		credentials := opts["--password"].(string)
		sepIndex := strings.Index(credentials, ":")
		var login, password string
		if sepIndex > 0 {
			login = credentials[:sepIndex]
			password = credentials[sepIndex+1:]
		} else {
			login = "plik"
			password = credentials
		}
		config.Login = login
		config.Password = password
	}

	// Override upload token ?
	if opts["--token"] != nil && opts["--token"].(string) != "" {
		config.Token = opts["--token"].(string)
	}

	// Ask for token
	if config.Token == "-" {
		fmt.Printf("Token: ")
		var err error
		_, err = fmt.Scanln(&config.Token)
		if err != nil {
			return fmt.Errorf("Unable to get token : %s", err)
		}
	}

	if opts["--stdin"].(bool) {
		config.DisableStdin = false
	}

	return
}
