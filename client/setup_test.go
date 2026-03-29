package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	data_test "github.com/root-gg/plik/server/data/testing"
	"github.com/root-gg/plik/server/metadata"
	"github.com/root-gg/plik/server/server"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/plik"
)

// testServerURL holds the ephemeral server URL, set by TestMain.
var testServerURL string

// testServer holds the server instance for teardown.
var testServer *server.PlikServer

// serverLogBuf captures server log output for troubleshooting test failures.
var serverLogBuf syncBuffer

// syncBuffer is a goroutine-safe bytes.Buffer for capturing server logs.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *syncBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

// Reset clears the buffer and returns the previous contents.
func (sb *syncBuffer) Reset() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	s := sb.buf.String()
	sb.buf.Reset()
	return s
}

// dumpServerLogs registers a t.Cleanup that dumps captured server logs on test failure.
// Uses fmt.Fprintf(os.Stderr) instead of t.Logf because panics cause re-panic
// which swallows buffered t.Log output.
func dumpServerLogs(t *testing.T) {
	t.Helper()
	serverLogBuf.Reset() // clear logs from previous tests
	t.Cleanup(func() {
		if t.Failed() {
			if logs := serverLogBuf.Reset(); logs != "" {
				fmt.Fprintf(os.Stderr, "\n=== SERVER LOGS (%s) ===\n%s=== END SERVER LOGS ===\n", t.Name(), logs)
			}
		}
	})
}

func TestMain(m *testing.M) {
	code := 0
	defer func() {
		os.Exit(code)
	}()

	config := common.NewConfiguration()
	config.ListenAddress = "127.0.0.1"
	config.ListenPort = 0 // Ephemeral port
	config.AutoClean(false)
	config.FeatureExtendTTL = common.FeatureEnabled
	config.LogOutput = &serverLogBuf // Redirect server logs to buffer for test failure output

	_ = config.Initialize(nil)

	ps := server.NewPlikServer(config)

	metadataBackendConfig := &metadata.Config{
		Driver:           "sqlite3",
		ConnectionString: filepath.Join(os.TempDir(), fmt.Sprintf("plik.client.test.%d.db", os.Getpid())),
		EraseFirst:       true,
	}

	metadataBackend, err := metadata.NewBackend(metadataBackendConfig, config.NewLogger())
	if err != nil {
		fmt.Printf("Unable to setup metadata backend: %s\n", err)
		code = 1
		return
	}
	ps.WithMetadataBackend(metadataBackend)

	var dataBackend data.Backend = data_test.NewBackend()
	ps.WithDataBackend(dataBackend)

	err = ps.Start()
	if err != nil {
		fmt.Printf("Unable to start server: %s\n", err)
		code = 1
		return
	}

	err = common.CheckHTTPServer(ps.GetConfig().ListenPort)
	if err != nil {
		fmt.Printf("Server not ready: %s\n", err)
		code = 1
		return
	}

	testServerURL = ps.GetConfig().GetServerURL().String()
	testServer = ps

	code = m.Run()

	_ = ps.ShutdownNow()
}

// ---------- Helpers ----------

// requireBinary fails the test immediately if the named binary is not in PATH.
func requireBinary(t *testing.T, name string) {
	t.Helper()
	_, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf("required binary %q not found in PATH", name)
	}
}

// newTestConfig creates a CliConfig pointing at the test server.
func newTestConfig() *CliConfig {
	config := NewUploadConfig()
	config.URL = testServerURL
	config.Quiet = true // suppress progress bars in tests
	return config
}

// cliResult holds captured output from a CLI run.
type cliResult struct {
	Stdout string // CLI stdout output
	Stderr string // CLI stderr (debug output, errors)
}

// buildCLI creates a PlikCLI wired to output buffers, ready to Run().
// Returns the cli, client, and the output buffers. On UnmarshalArgs failure
// it returns a non-nil error immediately.
func buildCLI(t *testing.T, config *CliConfig, opts map[string]any) (*PlikCLI, *plik.Client, *bytes.Buffer, *bytes.Buffer, error) {
	t.Helper()

	arguments := makeOpts()
	maps.Copy(arguments, opts)

	if err := config.UnmarshalArgs(arguments); err != nil {
		return nil, nil, nil, nil, err
	}

	cli := NewPlikCLI(config, arguments)
	client := config.NewClient("plik_test")

	var outBuf, errBuf bytes.Buffer
	cli.Stdout = &outBuf
	cli.Stderr = &errBuf

	return cli, client, &outBuf, &errBuf, nil
}

// runCLI creates a PlikCLI and runs it, capturing stdout and stderr.
// opts override the defaults from makeOpts(); filePaths are set via config.
func runCLI(t *testing.T, config *CliConfig, opts map[string]any) cliResult {
	t.Helper()
	dumpServerLogs(t)

	cli, client, outBuf, errBuf, err := buildCLI(t, config, opts)
	if err != nil {
		t.Fatalf("UnmarshalArgs failed: %s", err)
	}

	if err := cli.Run(client); err != nil {
		fmt.Fprintf(os.Stderr, "\n=== CLIENT OUTPUT (%s) ===\nStdout:\n%s\nStderr:\n%s\n=== END CLIENT OUTPUT ===\n", t.Name(), outBuf.String(), errBuf.String())
		t.Fatalf("cli.Run() failed: %s", err)
	}

	return cliResult{Stdout: outBuf.String(), Stderr: errBuf.String()}
}

// runCLIExpectError is like runCLI but expects an error from Run().
func runCLIExpectError(t *testing.T, config *CliConfig, opts map[string]any) (cliResult, error) {
	t.Helper()
	dumpServerLogs(t)

	cli, client, outBuf, errBuf, err := buildCLI(t, config, opts)
	if err != nil {
		// UnmarshalArgs error is acceptable as an expected error path
		return cliResult{}, err
	}

	runErr := cli.Run(client)
	return cliResult{Stdout: outBuf.String(), Stderr: errBuf.String()}, runErr
}

// runInfo runs cli.info() and captures stdout.
func runInfo(t *testing.T, config *CliConfig) string {
	t.Helper()
	dumpServerLogs(t)

	cli := NewPlikCLI(config, makeOpts())
	client := config.NewClient("plik_test")

	var buf bytes.Buffer
	cli.Stdout = &buf

	err := cli.info(client)
	if err != nil {
		t.Fatalf("cli.info() failed: %s", err)
	}
	return buf.String()
}

// createTestFile creates a file in dir with the given name and content.
// Returns the absolute path.
func createTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("unable to create test file: %s", err)
	}
	return path
}

// createTestDir creates a subdirectory in dir.
func createTestDir(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		t.Fatalf("unable to create test dir: %s", err)
	}
	return path
}

// getUploadMetadata fetches upload metadata from the server API.
func getUploadMetadata(t *testing.T, output string) map[string]any {
	t.Helper()

	// Extract upload ID from output URL (format: .../upload/UPLOAD_ID/...)
	// or from quiet mode URL (format: .../file/UPLOAD_ID/FILE_ID/FILENAME)
	uploadID := extractUploadID(t, output)

	resp, err := http.Get(testServerURL + "/upload/" + uploadID)
	if err != nil {
		t.Fatalf("unable to fetch upload metadata: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d fetching upload %s", resp.StatusCode, uploadID)
	}

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatalf("unable to decode upload metadata: %s", err)
	}
	return result
}

// getUploadMetadataAuth fetches upload metadata with HTTP basic auth.
func getUploadMetadataAuth(t *testing.T, output, login, password string) map[string]any {
	t.Helper()

	uploadID := extractUploadID(t, output)

	req, err := http.NewRequest("GET", testServerURL+"/upload/"+uploadID, nil)
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}
	req.SetBasicAuth(login, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unable to fetch upload metadata: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d fetching upload (auth)", resp.StatusCode)
	}

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatalf("unable to decode upload metadata: %s", err)
	}
	return result
}

// extractUploadID extracts the upload ID from CLI output.
func extractUploadID(t *testing.T, output string) string {
	t.Helper()

	// Try quiet mode URL: SERVER/file/UPLOAD_ID/FILE_ID/FILENAME
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, testServerURL+"/file/") {
			parts := strings.Split(line, "/file/")
			if len(parts) >= 2 {
				idParts := strings.SplitN(parts[1], "/", 2)
				return idParts[0]
			}
		}
		// Try upload URL: SERVER/?id=UPLOAD_ID
		if strings.HasPrefix(line, testServerURL) && strings.Contains(line, "?id=") {
			parts := strings.Split(line, "?id=")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1])
			}
		}
		// Try curl output: curl "SERVER/file/..."
		if strings.Contains(line, "curl") && strings.Contains(line, testServerURL+"/file/") {
			urlPart := line[strings.Index(line, testServerURL):]
			parts := strings.Split(urlPart, "/file/")
			if len(parts) >= 2 {
				idParts := strings.SplitN(parts[1], "/", 2)
				return idParts[0]
			}
		}
	}

	t.Fatalf("unable to extract upload ID from output:\n%s", output)
	return ""
}

// downloadFileBytes downloads a file from the given URL and returns raw bytes.
func downloadFileBytes(t *testing.T, fileURL string) []byte {
	t.Helper()

	resp, err := http.Get(fileURL)
	if err != nil {
		t.Fatalf("unable to download file: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d for %s", resp.StatusCode, fileURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable to read response body: %s", err)
	}
	return body
}

// downloadFileContent downloads a file from the given URL and returns its content as a string.
func downloadFileContent(t *testing.T, fileURL string) string {
	t.Helper()
	return string(downloadFileBytes(t, fileURL))
}

// extractFileURLFromOutput extracts a single file download URL from CLI output.
// Fails if zero or more than one URL is found.
func extractFileURLFromOutput(t *testing.T, output string) string {
	t.Helper()
	urls := extractAllFileURLsFromOutput(t, output)
	require.Len(t, urls, 1, "expected exactly 1 file URL in output")
	return urls[0]
}

// extractAllFileURLsFromOutput extracts all file download URLs from CLI output.
func extractAllFileURLsFromOutput(t *testing.T, output string) []string {
	t.Helper()

	var urls []string
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, testServerURL+"/file/") {
			urls = append(urls, line)
		}
		if strings.Contains(line, "curl") && strings.Contains(line, "/file/") {
			start := strings.Index(line, `"`)
			end := strings.LastIndex(line, `"`)
			if start >= 0 && end > start {
				urls = append(urls, line[start+1:end])
			}
		}
	}

	if len(urls) == 0 {
		t.Fatalf("unable to extract any file URLs from output:\n%s", output)
	}
	return urls
}

// testContent is the default specimen content for test files.
const testContent = `Lorem ipsum dolor sit amet, eu munere invenire est, in vel liber salutatus.
Cu eum ullum constituto theophrastus, te eam nihil ignota iudicabit.
At vix clita aliquam docendi. Ex eum utroque dignissim theophrastus.
Erat vulputate intellegebat an nam, te reque atomorum molestiae eos.
Sed electram dignissim reformidans ut. In vim graeco torquatos pertinacia.`

// findExtractedFile searches for a file by name under a directory tree.
// Used after extracting archives that preserve full absolute paths.
func findExtractedFile(t *testing.T, dir, name string) string {
	t.Helper()
	var found string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == name {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		t.Fatalf("error walking directory %s: %s", dir, err)
	}
	if found == "" {
		t.Fatalf("file %q not found under %s", name, dir)
	}
	return found
}

// downloadAndExtractTar downloads a tar.gz archive and extracts it to dlDir.
func downloadAndExtractTar(t *testing.T, fileURL, dlDir, archiveName string) {
	t.Helper()

	archiveData := downloadFileBytes(t, fileURL)
	archivePath := filepath.Join(dlDir, archiveName)
	require.NoError(t, os.WriteFile(archivePath, archiveData, 0644))

	cmd := exec.Command("tar", "xzf", archivePath, "-C", dlDir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "tar extract failed: %s", string(out))
}

// downloadAndExtractTarBz2 downloads a tar.bz2 archive and extracts it to dlDir.
func downloadAndExtractTarBz2(t *testing.T, fileURL, dlDir, archiveName string) {
	t.Helper()

	archiveData := downloadFileBytes(t, fileURL)
	archivePath := filepath.Join(dlDir, archiveName)
	require.NoError(t, os.WriteFile(archivePath, archiveData, 0644))

	cmd := exec.Command("tar", "xjf", archivePath, "-C", dlDir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "tar extract failed: %s", string(out))
}

// downloadAndExtractZip downloads a zip archive and extracts it to dlDir.
func downloadAndExtractZip(t *testing.T, fileURL, dlDir, archiveName string) {
	t.Helper()

	archiveData := downloadFileBytes(t, fileURL)
	archivePath := filepath.Join(dlDir, archiveName)
	require.NoError(t, os.WriteFile(archivePath, archiveData, 0644))

	cmd := exec.Command("unzip", archivePath, "-d", dlDir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "unzip failed: %s", string(out))
}

// requireFileNotExtracted verifies that a file with the given name does not
// exist anywhere under dir. Used to verify archive exclusion options.
func requireFileNotExtracted(t *testing.T, dir, name string) {
	t.Helper()
	var found bool
	filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() && info.Name() == name {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	require.False(t, found, "file %q should not exist under %s", name, dir)
}
