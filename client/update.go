package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/kardianos/osext"
	"github.com/root-gg/utils"

	"github.com/root-gg/plik/plik"
	"github.com/root-gg/plik/server/common"
)

func update(client *plik.Client, updateFlag bool) (err error) {
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

	buildInfo, err := client.GetServerVersion()
	if err != nil {
		return fmt.Errorf("Unable to get server version : %s", err)
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
		return fmt.Errorf("Server does not offer a %s-%s client", runtime.GOOS, runtime.GOARCH)

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
	if ok, err := common.AskConfirmation(true); err != nil || !ok {
		if err != nil {
			return fmt.Errorf("Unable to ask for confirmation : %s", err)
		}
		if updateFlag {
			os.Exit(0)
		}
		return nil
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
				return err
			}
			var req *http.Request
			req, err = http.NewRequest("GET", URL.String(), nil)
			if err != nil {
				return fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
			}

			resp, err := client.MakeRequest(req)
			if err != nil {
				return fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != 200 {
				return fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
			}

			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("Unable to get release notes for version %s : %s", release.Name, err)
			}

			// Ask to display the release notes
			fmt.Printf("Do you want to browse the release notes of version %s ? [Y/n] ", release.Name)
			if ok, err := common.AskConfirmation(true); err != nil || !ok {
				if err != nil {
					return fmt.Errorf("Unable to ask for confirmation : %s", err)
				}
				continue
			}

			// Display the release notes
			releaseDate := time.Unix(release.Date, 0).Format("Mon Jan 2 2006 15:04")
			fmt.Printf("Plik %s has been released %s\n\n", release.Name, releaseDate)
			fmt.Println(string(body))

			// Let user review the last release notes and ask to confirm update
			if release.Name == newVersion {
				fmt.Printf("\nUpdate Plik client from %s to %s ? [Y/n] ", currentVersion, newVersion)
				if ok, err := common.AskConfirmation(true); err != nil || !ok {
					if err != nil {
						return fmt.Errorf("Unable to ask for confirmation : %s", err)
					}
					if updateFlag {
						os.Exit(0)
					}
					return nil
				}
				break
			}
		}
	}

	// Download new client
	tmpPath := filepath.Dir(path) + "/" + "." + filepath.Base(path) + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	URL, err := url.Parse(downloadURL)
	if err != nil {
		return fmt.Errorf("Unable to download client : %s", err)

	}
	req, err := http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		return fmt.Errorf("Unable to download client : %s", err)
	}
	resp, err := client.MakeRequest(req)
	if err != nil {
		return fmt.Errorf("Unable to download client : %s", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Unable to download client : %s", resp.Status)
	}
	defer resp.Body.Close()
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("Unable to download client : %s", err)
	}
	err = tmpFile.Close()
	if err != nil {
		return fmt.Errorf("Unable to download client : %s", err)
	}

	// Check download integrity
	downloadMD5, err := utils.FileMd5sum(tmpPath)
	if err != nil {
		return fmt.Errorf("Unable to download client : %s", err)
	}
	if downloadMD5 != newMD5 {
		return fmt.Errorf("Unable to download client : md5sum %s does not match %s", downloadMD5, newMD5)
	}

	// Replace old client
	err = os.Rename(tmpPath, path)
	if err != nil {
		return fmt.Errorf("Unable to replace client : %s", err)
	}

	if newVersion != "" {
		fmt.Printf("Plik client successfully updated to %s\n", newVersion)
	} else {
		fmt.Printf("Plik client successfully updated\n")
	}

	return
}
