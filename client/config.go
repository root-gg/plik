/*
 * Charles-Antoine Mathieu <charles-antoine.mathieu@ovh.net>
 */

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	homedir "github.com/mitchellh/go-homedir"
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
	Yubikey        bool
	Login          string
	Password       string
	TTL            int
	AutoUpdate     bool
	Token          string

	filePaths        []string
	filenameOverride string
	yubikeyToken     string
}

// NewUploadConfig construct a new configuration with default values
func NewUploadConfig() (config *CliConfig) {
	config = new(CliConfig)
	config.Debug = false
	config.Quiet = false
	config.URL = "http://127.0.0.1:8080"
	config.OneShot = false
	config.Removable = false
	config.Stream = false
	config.Secure = false
	config.Archive = false
	config.ArchiveMethod = "tar"
	config.ArchiveOptions = make(map[string]interface{})
	config.ArchiveOptions["Tar"] = "/bin/tar"
	config.ArchiveOptions["Compress"] = "gzip"
	config.ArchiveOptions["Options"] = ""
	config.SecureMethod = "openssl"
	config.SecureOptions = make(map[string]interface{})
	config.SecureOptions["Openssl"] = "/usr/bin/openssl"
	config.SecureOptions["Cipher"] = "aes-256-cbc"
	config.SecureOptions["Options"] = "-md sha256"
	config.DownloadBinary = "curl"
	config.Comments = ""
	config.Yubikey = false
	config.Login = ""
	config.Password = ""
	config.TTL = 86400 * 30
	config.AutoUpdate = false
	config.Token = ""
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

// LoadConfig creates a new default configuration and override it with .plikrc fike.
// If .plikrc does not exist, ask domain, and create a new one in user HOMEDIR
func LoadConfig() (config *CliConfig, err error) {
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

	// Check if quiet mode ( you'll have to pass --server flag )
	for _, arg := range os.Args[1:] {
		if arg == "-q" || arg == "--quiet" {
			return config, nil
		}
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
	resp, err := http.Head(config.URL)
	if err != nil {
		return nil, err
	}

	finalURL := resp.Request.URL.String()
	if finalURL != "" && finalURL != config.URL {
		fmt.Printf("We have been redirected to : %s\n", finalURL)
		fmt.Printf("Replace current url (%s) with the new one ? [Y/n] ", config.URL)

		input := "y"
		fmt.Scanln(&input)

		if strings.HasPrefix(strings.ToLower(input), "y") {
			config.URL = strings.TrimSuffix(finalURL, "/")
		}
	}

	// Enable client updates ?
	fmt.Println("Do you want to enable client auto update ? [Y/n] ")
	input := "y"
	fmt.Scanln(&input)
	if strings.HasPrefix(strings.ToLower(input), "y") {
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

	f.Write(buf.Bytes())
	f.Close()

	fmt.Println("Plik client settings successfully saved to " + path)
	return config, nil
}

// UnmarshalArgs turns command line arguments into upload settings
// Command line arguments override config file settings
func (config *CliConfig) UnmarshalArgs(arguments map[string]interface{}) (err error) {

	// Handle flags
	if arguments["--version"].(bool) {
		fmt.Printf("Plik client %s\n", common.GetBuildInfo())
		os.Exit(0)
	}
	if arguments["--debug"].(bool) {
		config.Debug = true
	}
	if arguments["--quiet"].(bool) {
		config.Quiet = true
	}

	// Plik server url
	if arguments["--server"] != nil && arguments["--server"].(string) != "" {
		config.URL = arguments["--server"].(string)
	}

	// Paths
	if _, ok := arguments["FILE"].([]string); ok {
		config.filePaths = arguments["FILE"].([]string)
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
	if arguments["--name"] != nil && arguments["--name"].(string) != "" {
		config.filenameOverride = arguments["--name"].(string)
	}

	// Upload options
	if arguments["--oneshot"].(bool) {
		config.OneShot = true
	}
	if arguments["--removable"].(bool) {
		config.Removable = true
	}

	if arguments["--stream"].(bool) {
		config.Stream = true
	}

	if arguments["--comments"] != nil && arguments["--comments"].(string) != "" {
		config.Comments = arguments["--comments"].(string)
	}

	// Configure upload expire date
	if arguments["--ttl"] != nil && arguments["--ttl"].(string) != "" {
		ttlStr := arguments["--ttl"].(string)
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
			return fmt.Errorf("Invalid TTL %s", arguments["--ttl"].(string))
		}
		config.TTL = ttl * mul
	}

	// Enable archive mode ?
	if arguments["-a"].(bool) || arguments["--archive"] != nil || config.Archive {
		config.Archive = true

		if arguments["--archive"] != nil && arguments["--archive"] != "" {
			config.ArchiveMethod = arguments["--archive"].(string)
		}
	}

	// Enable secure mode ?
	if arguments["--not-secure"].(bool) {
		config.Secure = false
	} else if arguments["-s"].(bool) || arguments["--secure"] != nil || config.Secure {
		config.Secure = true
		if arguments["--secure"] != nil && arguments["--secure"].(string) != "" {
			config.SecureMethod = arguments["--secure"].(string)
		}
	}

	// Enable password protection ?
	if arguments["-p"].(bool) {
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
	} else if arguments["--password"] != nil && arguments["--password"].(string) != "" {
		credentials := arguments["--password"].(string)
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

	// Enable Yubikey protection ?
	if config.Yubikey || arguments["--yubikey"].(bool) {
		fmt.Printf("Yubikey token : ")
		_, err := fmt.Scanln(&config.yubikeyToken)
		if err != nil {
			return fmt.Errorf("Unable to get yubikey token : %s", err)
		}
	}

	// Override upload token ?
	if arguments["--token"] != nil && arguments["--token"].(string) != "" {
		config.Token = arguments["--token"].(string)
	}

	return
}
