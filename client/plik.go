package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/kardianos/osext"
	"github.com/olekukonko/ts"
	"github.com/root-gg/utils"

	"github.com/root-gg/plik/client/archive"
	"github.com/root-gg/plik/client/crypto"
	"github.com/root-gg/plik/plik"
	"github.com/root-gg/plik/server/common"
)

// Vars
var arguments map[string]interface{}
var config *CliConfig
var archiveBackend archive.Backend
var cryptoBackend crypto.Backend
var client *plik.Client

var err error

// Main
func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
	ts.GetSize() // ?

	// Usage /!\ INDENT THIS WITH SPACES NOT TABS /!\
	usage := `plik

Usage:
  plik [options] [FILE] ...

Options:
  -h --help                 Show this help
  -d --debug                Enable debug mode
  -q --quiet                Enable quiet mode
  -o, --oneshot             Enable OneShot ( Each file will be deleted on first download )
  -r, --removable           Enable Removable upload ( Each file can be deleted by anyone at anymoment )
  -S, --stream              Enable Streaming ( It will block until remote user starts downloading )
  -t, --ttl TTL             Time before expiration (Upload will be removed in m|h|d)
  -n, --name NAME           Set file name when piping from STDIN
  --server SERVER           Overrides plik url
  --token TOKEN             Specify an upload token
  --comments COMMENT        Set comments of the upload ( MarkDown compatible )
  -p                        Protect the upload with login and password ( be prompted )
  --password PASSWD         Protect the upload with "login:password" ( if omitted default login is "plik" )
  -a                        Archive upload using default archive params ( see ~/.plikrc )
  --archive MODE            Archive upload using the specified archive backend : tar|zip
  --compress MODE           [tar] Compression codec : gzip|bzip2|xz|lzip|lzma|lzop|compress|no
  --archive-options OPTIONS [tar|zip] Additional command line options
  -s                        Encrypt upload using the default encryption parameters ( see ~/.plikrc )
  --not-secure              Do not encrypt upload files regardless of the ~/.plikrc configurations
  --secure MODE             Encrypt upload files using the specified crypto backend : openssl|pgp
  --cipher CIPHER           [openssl] Openssl cipher to use ( see openssl help )
  --passphrase PASSPHRASE   [openssl] Passphrase or '-' to be prompted for a passphrase
  --recipient RECIPIENT     [pgp] Set recipient for pgp backend ( example : --recipient Bob )
  --secure-options OPTIONS  [openssl|pgp] Additional command line options
  --update                  Update client
  -v --version              Show client version
`
	// Parse command line arguments
	arguments, _ = docopt.Parse(usage, nil, true, "", false)

	// Load config
	config, err = LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load configuration : %s\n", err)
		os.Exit(1)
	}

	// Load arguments
	err = config.UnmarshalArgs(arguments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	if config.Debug {
		fmt.Println("Arguments : ")
		utils.Dump(arguments)
		fmt.Println("Configuration : ")
		utils.Dump(config)
	}

	client = plik.NewClient(config.URL)
	client.Debug = config.Debug
	client.ClientName = "plik_cli"

	// Check client version
	updateFlag := arguments["--update"].(bool)
	err = updateClient(updateFlag)
	if err == nil {
		if updateFlag {
			os.Exit(0)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Unable to update Plik client : \n")
		fmt.Fprintf(os.Stderr, "%s\n", err)
		if updateFlag {
			os.Exit(1)
		}
	}

	// Detect STDIN type
	// --> If from pipe : ok, doing nothing
	// --> If not from pipe, and no files in arguments : printing help
	fi, _ := os.Stdin.Stat()

	if runtime.GOOS != "windows" {
		if (fi.Mode()&os.ModeCharDevice) != 0 && len(arguments["FILE"].([]string)) == 0 {
			fmt.Println(usage)
			os.Exit(1)
		}
	} else {
		if len(arguments["FILE"].([]string)) == 0 {
			fmt.Println(usage)
			os.Exit(1)
		}
	}

	upload := client.NewUpload()
	upload.Token = config.Token
	upload.TTL = config.TTL
	upload.Stream = config.Stream
	upload.OneShot = config.OneShot
	upload.Removable = config.Removable
	upload.Comments = config.Comments
	upload.Login = config.Login
	upload.Password = config.Password

	if len(config.filePaths) == 0 {
		upload.AddFileFromReader("STDIN", bufio.NewReader(os.Stdin))
	} else {
		if config.Archive {
			archiveBackend, err = archive.NewArchiveBackend(config.ArchiveMethod, config.ArchiveOptions)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to initialize archive backend : %s", err)
				os.Exit(1)
			}

			err = archiveBackend.Configure(arguments)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to configure archive backend : %s", err)
				os.Exit(1)
			}

			reader, err := archiveBackend.Archive(config.filePaths)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to create archive : %s", err)
				os.Exit(1)
			}

			filename := archiveBackend.GetFileName(config.filePaths)
			upload.AddFileFromReader(filename, reader)
		} else {
			for _, path := range config.filePaths {
				_, err := upload.AddFileFromPath(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s : %s\n", path, err)
					os.Exit(1)
				}
			}
		}
	}

	if config.filenameOverride != "" {
		if len(upload.Files()) != 1 {
			fmt.Fprintf(os.Stderr, "Can't override filename if more than one file to upload\n")
			os.Exit(1)
		}
		upload.Files()[0].Name = config.filenameOverride
	}

	// Initialize crypto backend
	if config.Secure {
		cryptoBackend, err = crypto.NewCryptoBackend(config.SecureMethod, config.SecureOptions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to initialize crypto backend : %s", err)
			os.Exit(1)
		}
		err = cryptoBackend.Configure(arguments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to configure crypto backend : %s", err)
			os.Exit(1)
		}
	}

	// Initialize progress bar display
	var progress *Progress
	if !config.Quiet && !config.Debug {
		progress = NewProgress(upload.Files())
	}

	// Add files to upload
	for _, file := range upload.Files() {
		if config.Secure {
			file.WrapReader(func(fileReader io.ReadCloser) io.ReadCloser {
				reader, err := cryptoBackend.Encrypt(fileReader)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Unable to encrypt file :%s", err)
					os.Exit(1)
				}
				return ioutil.NopCloser(reader)
			})
		}

		if !config.Quiet && !config.Debug {
			progress.register(file)
		}
	}

	// Create upload on server
	err = upload.Create()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create upload : %s\n", err)
		os.Exit(1)
	}

	// Mon, 02 Jan 2006 15:04:05 MST
	creationDate := upload.Metadata().CreatedAt.Format(time.RFC1123)

	// Display upload url
	printf("Upload successfully created at %s : \n", creationDate)

	uploadURL, err := upload.GetURL()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get upload url %s\n", err)
		os.Exit(1)
	} else {
		printf("    %s\n\n", uploadURL)
	}

	if config.Stream && !config.Debug {
		for _, file := range upload.Files() {
			cmd, err := getFileCommand(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to get download command for file %s : %s\n", file.Name, err)
			}
			fmt.Println(cmd)
		}
		printf("\n")
	}

	if !config.Quiet && !config.Debug {
		// Nothing should be printed between this an progress.Stop()
		progress.start()
	}

	// Upload files
	_ = upload.Upload()

	if !config.Quiet && !config.Debug {
		// Finalize the progress bar display
		progress.stop()
	}

	// Display download commands
	if !config.Stream {
		printf("\nCommands : \n")
		for _, file := range upload.Files() {
			// Print file information (only url if quiet mode is enabled)
			if file.Error() != nil {
				continue
			}
			if config.Quiet {
				URL, err := file.GetURL()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Unable to get download command for file %s : %s\n", file.Name, err)
				}
				fmt.Println(URL)
			} else {
				cmd, err := getFileCommand(file)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Unable to get download command for file %s : %s\n", file.Name, err)
				}
				fmt.Println(cmd)
			}
		}
	} else {
		printf("\n")
	}
}

func getFileCommand(file *plik.File) (command string, err error) {
	// Step one - Downloading file
	switch config.DownloadBinary {
	case "wget":
		command += "wget -q -O-"
	case "curl":
		command += "curl -s"
	default:
		command += config.DownloadBinary
	}

	URL, err := file.GetURL()
	if err != nil {
		return "", err
	}
	command += fmt.Sprintf(` "%s"`, URL)

	// If Ssl
	if config.Secure {
		command += fmt.Sprintf(" | %s", cryptoBackend.Comments())
	}

	// If archive
	if config.Archive {
		if config.ArchiveMethod == "zip" {
			command += fmt.Sprintf(` > '%s'`, file.Name)
		} else {
			command += fmt.Sprintf(" | %s", archiveBackend.Comments())
		}
	} else {
		command += fmt.Sprintf(` > '%s'`, file.Name)
	}

	return
}

func updateClient(updateFlag bool) (err error) {
	// Do not check for update if AutoUpdate is not enabled
	if !updateFlag && !config.AutoUpdate {
		return
	}

	// Do not update when quiet mode is enabled
	if !updateFlag && config.Quiet {
		return
	}

	// Get client MD5SUM
	path, err := osext.Executable()
	if err != nil {
		return
	}
	currentMD5, err := utils.FileMd5sum(path)
	if err != nil {
		return
	}

	// Check server version
	currentVersion := common.GetBuildInfo().Version

	var newVersion string
	var downloadURL string
	var newMD5 string
	var buildInfo *common.BuildInfo

	var URL *url.URL
	URL, err = url.Parse(config.URL + "/version")
	if err != nil {
		err = fmt.Errorf("Unable to get server version : %s", err)
		return
	}
	var req *http.Request
	req, err = http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		err = fmt.Errorf("Unable to get server version : %s", err)
		return
	}

	resp, err := client.MakeRequest(req)
	if resp == nil {
		err = fmt.Errorf("Unable to get server version : %s", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		// >=1.1 use BuildInfo from /version

		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("Unable to get server version : %s", err)
			return
		}

		// Parse json BuildInfo object
		buildInfo = new(common.BuildInfo)
		err = json.Unmarshal(body, buildInfo)
		if err != nil {
			err = fmt.Errorf("Unable to get server version : %s", err)
			return
		}

		newVersion = buildInfo.Version
		for _, client := range buildInfo.Clients {
			if client.OS == runtime.GOOS && client.ARCH == runtime.GOARCH {
				newMD5 = client.Md5
				downloadURL = config.URL + "/" + client.Path
				break
			}
		}

		if newMD5 == "" || downloadURL == "" {
			err = fmt.Errorf("Server does not offer a %s-%s client", runtime.GOOS, runtime.GOARCH)
			return
		}
	} else if resp.StatusCode == 404 {
		// <1.1 fallback on MD5SUM file

		baseURL := config.URL + "/clients/" + runtime.GOOS + "-" + runtime.GOARCH
		var URL *url.URL
		URL, err = url.Parse(baseURL + "/MD5SUM")
		if err != nil {
			return
		}
		var req *http.Request
		req, err = http.NewRequest("GET", URL.String(), nil)
		if err != nil {
			err = fmt.Errorf("Unable to get server version : %s", err)
			return
		}

		resp, err = client.MakeRequest(req)
		if err != nil {
			err = fmt.Errorf("Unable to get server version : %s", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			err = fmt.Errorf("Unable to get server version : %s", resp.Status)
			return
		}

		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("Unable to get server version : %s", err)
			return
		}
		newMD5 = utils.Chomp(string(body))

		binary := "plik"
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		downloadURL = baseURL + "/" + binary
	} else {
		err = fmt.Errorf("Unable to get server version : %s", err)
		return
	}

	// Check if the client is up to date
	if currentMD5 == newMD5 {
		if updateFlag {
			if newVersion != "" {
				printf("Plik client %s is up to date\n", newVersion)
			} else {
				printf("Plik client is up to date\n")
			}
			os.Exit(0)
		}
		return
	}

	// Ask for permission
	if newVersion != "" {
		fmt.Printf("Update Plik client from %s to %s ? [Y/n] ", currentVersion, newVersion)
	} else {
		fmt.Printf("Update Plik client to match server version ? [Y/n] ")
	}
	if ok, _ := common.AskConfirmation(true); !ok {
		if updateFlag {
			os.Exit(0)
		}
		return
	}

	// Display release notes
	if buildInfo != nil && buildInfo.Releases != nil {

		// Find current release
		currentReleaseIndex := -1
		for i, release := range buildInfo.Releases {
			if release.Name == currentVersion {
				currentReleaseIndex = i
			}
		}

		// Find new release
		newReleaseIndex := -1
		for i, release := range buildInfo.Releases {
			if release.Name == newVersion {
				newReleaseIndex = i
			}
		}

		// Find releases between current and new version
		var releases []*common.Release
		if currentReleaseIndex > 0 && newReleaseIndex > 0 && currentReleaseIndex < newReleaseIndex {
			releases = buildInfo.Releases[currentReleaseIndex+1 : newReleaseIndex+1]
		}

		for _, release := range releases {
			// Get release notes from server
			var URL *url.URL
			URL, err = url.Parse(config.URL + "/changelog/" + release.Name)
			if err != nil {
				continue
			}
			var req *http.Request
			req, err = http.NewRequest("GET", URL.String(), nil)
			if err != nil {
				err = fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
				continue
			}

			resp, err = client.MakeRequest(req)
			if err != nil {
				err = fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				err = fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
				continue
			}

			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				err = fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
				continue
			}

			// Ask to display the release notes
			fmt.Printf("Do you want to browse the release notes of version %s ? [Y/n] ", release.Name)
			if ok, _ := common.AskConfirmation(true); !ok {
				continue
			}

			// Display the release notes
			releaseDate := time.Unix(release.Date, 0).Format("Mon Jan 2 2006 15:04")
			fmt.Printf("Plik %s has been released %s\n\n", release.Name, releaseDate)
			fmt.Println(string(body))

			// Let user review the last release notes and ask to confirm update
			if release.Name == newVersion {
				fmt.Printf("\nUpdate Plik client from %s to %s ? [Y/n] ", currentVersion, newVersion)
				if ok, _ := common.AskConfirmation(true); !ok {
					if updateFlag {
						os.Exit(0)
					}
					return
				}
				break
			}
		}
	}

	// Download new client
	tmpPath := filepath.Dir(path) + "/" + "." + filepath.Base(path) + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	URL, err = url.Parse(downloadURL)
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}
	req, err = http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}
	resp, err = client.MakeRequest(req)
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("Unable to download client : %s", resp.Status)
		return
	}
	defer resp.Body.Close()
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}
	err = tmpFile.Close()
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}

	// Check download integrity
	downloadMD5, err := utils.FileMd5sum(tmpPath)
	if err != nil {
		err = fmt.Errorf("Unable to download client : %s", err)
		return
	}
	if downloadMD5 != newMD5 {
		err = fmt.Errorf("Unable to download client : md5sum %s does not match %s", downloadMD5, newMD5)
		return
	}

	// Replace old client
	err = os.Rename(tmpPath, path)
	if err != nil {
		err = fmt.Errorf("Unable to replace client : %s", err)
		return
	}

	if newVersion != "" {
		fmt.Printf("Plik client successfully updated to %s\n", newVersion)
	} else {
		fmt.Printf("Plik client successfully updated\n")
	}

	return
}

func printf(format string, args ...interface{}) {
	if !config.Quiet {
		fmt.Printf(format, args...)
	}
}
