package common

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/root-gg/utils"
)

// Base62Charset is the character set for Base62 encoding.
// Used by GenerateRandomID and token checksum encoding.
const Base62Charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateRandomID generates a cryptographically random Base62 string of the specified length.
//
// It uses crypto/rand.Read (CSPRNG) with rejection sampling to avoid modulo bias:
//   - A random byte can hold values 0–255 (256 possible values)
//   - Our charset has 62 characters, and 256 is not evenly divisible by 62
//   - 256 % 62 = 8, so a naive b%62 would make the first 8 characters ~25% more likely
//   - We reject bytes ≥ 248 (the largest multiple of 62 ≤ 256), giving perfectly uniform output
//   - Rejection rate is only 8/256 ≈ 3.1%, so the loop almost never needs a second pass
func GenerateRandomID(length int) string {
	if length == 0 {
		return ""
	}

	result := make([]byte, length)

	// Double-size buffer to account for rejected bytes (~3.1% rejection rate)
	raw := make([]byte, length*2)

	filled := 0
	for filled < length {
		if _, err := rand.Read(raw); err != nil {
			panic(fmt.Sprintf("failed to generate random ID: %s", err))
		}
		for _, b := range raw {
			if b < 248 { // Reject bytes 248–255 to avoid modulo bias
				result[filled] = Base62Charset[b%62]
				filled++
				if filled == length {
					break
				}
			}
		}
	}

	return string(result)
}

// Base32Charset is similar to Crockford's Base32 specification to avoid similar-looking characters.
const Base32Charset = "0123456789abcdefghjkmnpqrstvwxyz"

// GenerateB32ID generates a random upload ID of the specified length
// It uses crypto/rand.Read but without rejection sampling since 256 is already completely divisible by 32
func GenerateB32ID(length int) string {
	if length == 0 {
		return ""
	}

	result := make([]byte, length)

	if _, err := rand.Read(result); err != nil {
		panic(fmt.Sprintf("failed to generate upload ID: %s", err))
	}
	for i := range length {
		result[i] = Base32Charset[result[i]%32]
	}

	return string(result)
}

// IsPlikWebapp checks if the request comes from the Plik web application
func IsPlikWebapp(req *http.Request) bool {
	return req.Header.Get("X-ClientApp") == "web_client"
}

// Ensure HTTPError implements error
var _ error = (*HTTPError)(nil)

// HTTPError allows to return an error and a HTTP status code
type HTTPError struct {
	Message    string
	Err        error
	StatusCode int
}

// NewHTTPError return a new HTTPError
func NewHTTPError(message string, err error, code int) HTTPError {
	return HTTPError{message, err, code}
}

// Error return the error
func (e HTTPError) Error() string {
	return e.String()
}

func (e HTTPError) String() string {
	if e.Err != nil {
		return fmt.Sprintf("%s : %s", e.Message, e.Err)
	}
	return e.Message
}

// StripPrefix returns a handler that serves HTTP requests
// removing the given prefix from the request URL's Path
// It differs from http.StripPrefix by defaulting to "/" and not ""
func StripPrefix(prefix string, handler http.Handler) http.Handler {
	if prefix == "" || prefix == "/" {
		return handler
	}
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		// Relative paths to javascript, css, ... imports won't work without a trailing slash
		if req.URL.Path == prefix {
			http.Redirect(resp, req, prefix+"/", http.StatusMovedPermanently)
			return
		}
		if p := strings.TrimPrefix(req.URL.Path, prefix); len(p) < len(req.URL.Path) {
			req.URL.Path = p
		} else {
			http.NotFound(resp, req)
			return
		}
		if !strings.HasPrefix(req.URL.Path, "/") {
			req.URL.Path = "/" + req.URL.Path
		}
		handler.ServeHTTP(resp, req)
	})
}

// SanitizeFilenameForDisposition strips characters that could break a
// Content-Disposition header value: double quotes, CR, LF, null bytes,
// and Unicode BiDi override characters that can spoof file extensions.
// It also truncates excessively long names to 1024 characters.
func SanitizeFilenameForDisposition(name string) string {
	r := strings.NewReplacer(
		`"`, "",
		"\r", "",
		"\n", "",
		"\x00", "",
		// Unicode BiDi overrides — can make "evil\u202Efdp.exe" appear as "evil exe.pdf"
		"\u202A", "", // LRE  (Left-to-Right Embedding)
		"\u202B", "", // RLE  (Right-to-Left Embedding)
		"\u202C", "", // PDF  (Pop Directional Formatting)
		"\u202D", "", // LRO  (Left-to-Right Override)
		"\u202E", "", // RLO  (Right-to-Left Override)
		"\u2066", "", // LRI  (Left-to-Right Isolate)
		"\u2067", "", // RLI  (Right-to-Left Isolate)
		"\u2068", "", // FSI  (First Strong Isolate)
		"\u2069", "", // PDI  (Pop Directional Isolate)
	)
	name = r.Replace(name)
	if len(name) > 1024 {
		name = name[:1024]
	}
	return name
}

// EncodeAuthBasicHeader return the base64 version of "login:password"
func EncodeAuthBasicHeader(login string, password string) (value string) {
	return base64.StdEncoding.EncodeToString([]byte(login + ":" + password))
}

// WriteJSONResponse serialize the response to json and write it to the HTTP response body
func WriteJSONResponse(resp http.ResponseWriter, obj any) {
	json, err := utils.ToJson(obj)
	if err != nil {
		panic(fmt.Errorf("unable to serialize json response : %s", err))
	}

	_, _ = resp.Write(json)
}

// AskConfirmation from process input
func AskConfirmation(defaultValue bool) (bool, error) {
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		if err.Error() == "unexpected newline" {
			return defaultValue, nil
		}
		return false, err
	}

	if strings.HasPrefix(strings.ToLower(input), "y") {
		return true, nil
	} else if strings.HasPrefix(strings.ToLower(input), "n") {
		return false, nil
	} else {
		return defaultValue, nil
	}
}

// HumanDuration displays duration with days and years
func HumanDuration(d time.Duration) string {
	var minus bool
	if d < 0 {
		minus = true
		d = -d
	}

	years := d / (365 * 24 * time.Hour)
	d = d % (365 * 24 * time.Hour)
	days := d / (24 * time.Hour)
	d = d % (24 * time.Hour)
	hours := d / (time.Hour)
	d = d % (time.Hour)
	minutes := d / (time.Minute)
	d = d % (time.Minute)
	seconds := d / (time.Second)
	d = d % (time.Second)

	str := ""

	if minus {
		str += "-"
	}

	if years > 0 {
		str += fmt.Sprintf("%dy", years)
	}

	if days > 0 {
		str += fmt.Sprintf("%dd", days)
	}

	if hours > 0 {
		str += fmt.Sprintf("%dh", hours)
	}

	if minutes > 0 {
		str += fmt.Sprintf("%dm", minutes)
	}

	if seconds > 0 {
		str += fmt.Sprintf("%ds", seconds)
	}

	if str == "" || d > 0 {
		str += fmt.Sprintf("%s", d)
	}

	return str
}

// LookupBinary resolves the path to an external binary.
// If configured is a valid executable path, it is returned as-is.
// Otherwise, it falls back to exec.LookPath(name) to search $PATH.
func LookupBinary(configured string, name string) (string, error) {
	if _, err := os.Stat(configured); err == nil {
		return configured, nil
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s not found at %s and not in $PATH, please install or set the path in ~/.plikrc", name, configured)
	}
	return path, nil
}
