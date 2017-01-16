package yubigo

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	dvorakToQwerty = strings.NewReplacer(
		"j", "c", "x", "b", "e", "d", ".", "e", "u", "f", "i", "g", "d", "h", "c", "i",
		"h", "j", "t", "k", "n", "l", "b", "n", "p", "r", "y", "t", "g", "u", "k", "v",
		"J", "C", "X", "B", "E", "D", ".", "E", "U", "F", "I", "G", "D", "H", "C", "I",
		"H", "J", "T", "K", "N", "L", "B", "N", "P", "R", "Y", "T", "G", "U", "K", "V")
	matchDvorak     = regexp.MustCompile(`^[jxe.uidchtnbpygkJXE.UIDCHTNBPYGK]{32,48}$`)
	matchQwerty     = regexp.MustCompile(`^[cbdefghijklnrtuvCBDEFGHIJKLNRTUV]{32,48}$`)
	signatureUrlFix = regexp.MustCompile(`\+`)
)

// Package variable used to override the http client used for communication 
// with Yubico. If nil the standard http.Client will be used - if overriding
// you need to ensure the transport options are set. 
var HTTPClient *http.Client = nil

// Parse and verify the given OTP string into prefix (identity) and ciphertext.
// Function returns a non-nil error when given OTP is not in valid format.
// NOTE: This function does NOT verify if the OTP is correct and unused/unique.
func ParseOTP(otp string) (prefix string, ciphertext string, err error) {
	if len(otp) < 32 || len(otp) > 48 {
		err = errors.New("OTP has wrong length.")
		return
	}

	// When otp matches dvorak-otp, then translate to qwerty.
	if matchDvorak.MatchString(otp) {
		otp = dvorakToQwerty.Replace(otp)
	}

	// Verify that otp matches qwerty expectations
	if !matchQwerty.MatchString(otp) {
		err = errors.New("Given string is not a valid Yubikey OTP. It contains invalid characters and/or the length is wrong.")
		return
	}

	l := len(otp)
	prefix = otp[0 : l-32]
	ciphertext = otp[l-32 : l]
	return
}

type YubiAuth struct {
	id                string
	key               []byte
	apiServerList     []string
	protocol          string
	verifyCertificate bool
	workers           []*verifyWorker
	use               sync.Mutex
	debug             bool
}

type verifyWorker struct {
	ya        *YubiAuth         // YubiAuth this worker belongs to
	id        int               // Worker id
	client    *http.Client      // http client standing by ready for work
	apiServer string            // API server URL
	work      chan *workRequest // Channel on which the worker receives work
	stop      chan bool         // Channel for stop signal
}

type workRequest struct {
	paramString *string
	resultChan  chan *workResult
}

type workResult struct {
	response     *http.Response
	requestQuery string
	err          error // indicates a failing server/network. This doesn't mean the OTP is invalid.
}

func (vw *verifyWorker) process() {
	if vw.ya.debug {
		log.Printf("worker[%d]: Started.\n", vw.id)
	}
	for {
		select {
		case w := <-vw.work:

			// Create url
			url := vw.ya.protocol + vw.apiServer + *w.paramString

			if vw.ya.debug {
				log.Printf("worker[%d]: Have work. Requesting: %s\n", vw.id, url)
			}

			// Create request
			request, err := http.NewRequest("GET", url, nil)
			if err != nil {
				w.resultChan <- &workResult{
					response:     nil,
					requestQuery: url,
					err:          fmt.Errorf("Could not create http request. Error: %s\n", err),
				}
				continue
			}
			request.Header.Add("User-Agent", "github.com/GeertJohan/yubigo")

			// Call server
			response, err := vw.client.Do(request)

			// If we received an error from the client, return that (wrapped) on the channel.
			if err != nil {
				w.resultChan <- &workResult{
					response:     nil,
					requestQuery: url,
					err:          fmt.Errorf("Http client error: %s\n", err),
				}
				if vw.ya.debug {
					log.Printf("worker[%d]: Http client error: %s", vw.id, err)
				}
				continue
			}

			// It seems everything is ok! return the response (wrapped) on the channel.
			if vw.ya.debug {
				log.Printf("worker[%d] Received result from api server. Sending on channel.", vw.id)
			}
			w.resultChan <- &workResult{
				response:     response,
				requestQuery: url,
				err:          nil,
			}
			continue
		case <-vw.stop:
			if vw.ya.debug {
				log.Printf("worker[%d]: received stop signal.\n", vw.id)
			}
			return
		}
	}
}

// Create a yubiAuth instance with given API-id and API-key.
// Returns an error when the key could not be base64 decoded.
// To use yubigo with the Yubico Web Service (default api servers), create an API id+key here: https://upgrade.yubico.com/getapikey/
// Debugging is disabled. For debugging: use NewYubiAuthDebug(..)
func NewYubiAuth(id string, key string) (auth *YubiAuth, err error) {
	return NewYubiAuthDebug(id, key, false)
}

// Create a yubiAuth instance for given API-id and API-key.
// Has third parameter `debug`. When debug is true this YubiAuth instance will spam the console with logging messages.
// Returns an error when the key could not be base64 decoded.
// To use yubigo with the Yubico Web Service (default api servers), create an API id+key here: https://upgrade.yubico.com/getapikey/
func NewYubiAuthDebug(id string, key string, debug bool) (auth *YubiAuth, err error) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		err = fmt.Errorf("Given key seems to be invalid. Could not base64_decode. Error: %s\n", err)
		return
	}

	if debug {
		log.Printf("NewYubiAuthDebug: Given key is base64 decodable. Creating new YubiAuth instance with api id '%s'.\n", id)
	}

	auth = &YubiAuth{
		id:  id,
		key: keyBytes,

		apiServerList: []string{"api.yubico.com/wsapi/2.0/verify",
			"api2.yubico.com/wsapi/2.0/verify",
			"api3.yubico.com/wsapi/2.0/verify",
			"api4.yubico.com/wsapi/2.0/verify",
			"api5.yubico.com/wsapi/2.0/verify"},

		protocol:          "https://",
		verifyCertificate: true,

		debug: debug,
	}

	if debug {
		log.Printf("NewYubiAuthDebug: Using yubico web servers: %#v\n", auth.apiServerList)
		log.Println("NewYubiAuthDebug: Going to build workers.")
	}

	// Build workers
	auth.buildWorkers()

	// All done :)
	return
}

// Stops existing workers and creates new ones.
func (ya *YubiAuth) buildWorkers() {
	// Unexported (internal) method, so no locking.

	// create tls config
	tlsConfig := &tls.Config{}
	if !ya.verifyCertificate {
		tlsConfig.InsecureSkipVerify = true
	}

	// stop all existing workers
	for _, worker := range ya.workers {
		worker.stop <- true
	}

	// create new (empty) slice with exact capacity
	ya.workers = make([]*verifyWorker, 0, len(ya.apiServerList))

	// start new workers. One for each apiServerString
	for id, apiServer := range ya.apiServerList {
		// create worker instance with new http.Client instance
		worker := &verifyWorker{
			ya: ya,
			id: id,
			apiServer: apiServer + "?",
			work:      make(chan *workRequest),
			stop:      make(chan bool),
		}

		if HTTPClient == nil {
			worker.client = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsConfig,
				},
			}
		} else {
			worker.client = HTTPClient
		}

		ya.workers = append(ya.workers, worker)

		// start worker process in new goroutine
		go worker.process()
	}
}

// Use this method to specify a list of servers for verification.
// Each server string should contain host + path. 
// Example: "api.yubico.com/wsapi/2.0/verify".
func (ya *YubiAuth) SetApiServerList(urls ...string) {
	// Lock
	ya.use.Lock()
	defer ya.use.Unlock()

	// save setting
	ya.apiServerList = urls

	// rebuild workers (api server url's have changed)
	ya.buildWorkers()
}

// Retrieve the the ist of servers that are being used for verification.
func (ya *YubiAuth) GetApiServerList() []string {
	return ya.apiServerList
}

// Enable or disable the use of https
func (ya *YubiAuth) UseHttps(useHttps bool) {
	// Lock
	ya.use.Lock()
	defer ya.use.Unlock()

	// change setting
	if useHttps {
		ya.protocol = "https://"
	} else {
		ya.protocol = "http://"
	}

	// no need to rebuild workers, they re-read ya.protocol on each request.
}

// Enable or disable https certificate verification
// Disable this at your own risk.
func (ya *YubiAuth) HttpsVerifyCertificate(verifyCertificate bool) {
	// Lock
	ya.use.Lock()
	defer ya.use.Unlock()

	// save setting
	ya.verifyCertificate = verifyCertificate

	// rebuild workers (client has to be changed)
	ya.buildWorkers()
}

// The verify method calls the API with given OTP and returns if the OTP is valid or not.
// This method will return an error if something unexpected happens
// If no error was returned, the returned 'ok bool' indicates if the OTP is valid
// if the 'ok bool' is true, additional informtion can be found in the returned YubiResponse object
func (ya *YubiAuth) Verify(otp string) (yr *YubiResponse, ok bool, err error) {
	// Lock
	ya.use.Lock()
	defer ya.use.Unlock()

	// check the OTP
	_, _, err = ParseOTP(otp)
	if err != nil {
		return nil, false, err
	}

	// create slice to store parameters for this verification request
	paramSlice := make([]string, 0)
	paramSlice = append(paramSlice, "id="+ya.id)
	paramSlice = append(paramSlice, "otp="+otp)

	// Create 40 characters nonce
	rand.Seed(time.Now().UnixNano())
	k := make([]rune, 40)
	for i := 0; i < 40; i++ {
		c := rand.Intn(35)
		if c < 10 {
			c += 48 // numbers (0-9) (0+48 == 48 == '0', 9+48 == 57 == '9')
		} else {
			c += 87 // lower case alphabets (a-z) (10+87 == 97 == 'a', 35+87 == 122 = 'z')
		}
		k[i] = rune(c)
	}
	nonce := string(k)
	paramSlice = append(paramSlice, "nonce="+nonce)

	// These settings are hardcoded in the library for now.
	//++ TODO(GeertJohan): add these values to the yubiAuth object and create getters/setters
	// paramSlice = append(paramSlice, "timestamp=1")
	paramSlice = append(paramSlice, "sl=secure")

	//++ TODO(GeertJohan): Add timeout support?
	//++ //paramSlice = append(paramSlice, "timeout=")

	// sort the slice
	sort.Strings(paramSlice)

	// create parameter string
	paramString := strings.Join(paramSlice, "&")

	// generate signature
	if len(ya.key) > 0 {
		hmacenc := hmac.New(sha1.New, ya.key)
		_, err := hmacenc.Write([]byte(paramString))
		if err != nil {
			return nil, false, fmt.Errorf("Could not calculate signature. Error: %s\n", err)
		}
		signature := base64.StdEncoding.EncodeToString(hmacenc.Sum([]byte{}))
		signature = signatureUrlFix.ReplaceAllString(signature, `%2B`)
		paramString = paramString + "&h=" + signature
	}

	// create result channel, buffersize equals the amount of workers.
	resultChan := make(chan *workResult, len(ya.workers))

	// create workRequest instance
	wr := &workRequest{
		paramString: &paramString,
		resultChan:  resultChan,
	}

	// send workRequest to each worker
	for _, worker := range ya.workers {
		worker.work <- wr
	}

	// count the errors so we can handle when all servers fail (network fail for instance)
	errCount := 0

	// local result var, will contain the first result we have
	var result *workResult

	// keep looping until we have a good result
	for {
		// listen for result from a worker
		result = <-resultChan

		// check for error
		if result.err != nil {
			// increment error counter
			errCount++

			if ya.debug {
				// debug logging
				log.Printf("A server (%s) gave error back: %s\n", result.requestQuery, result.err)
			}

			if errCount == len(ya.apiServerList) {
				// All workers are done, there's nothing left to try. we return an error.
				return nil, false, errors.New("None of the servers responded properly.")
			}

			// we have an error, but not all workers responded yet, so lets wait for the next result.
			continue
		}

		// create a yubiResult from the workers response.
		yr, err = newYubiResponse(result)
		if err != nil {
			return nil, false, err
		}

		// Check for "REPLAYED_REQUEST" result.
		if status, _ := yr.resultParameters["status"]; status == "REPLAYED_REQUEST" {
			// The result status is "REPLAYED_REQUEST".
			// This means that the server for this request got sync with an other server before our request.
			// Lets wait for the result from the other server.
			// See: http://forum.yubico.com/viewtopic.php?f=3&t=701

			// increment error counter
			errCount++

			if ya.debug {
				// debug logging
				log.Println("Got replayed request: ", result.response.Body)
			}

			if errCount == len(ya.apiServerList) {
				// All workers are done, there' is nothing left to try. We return an error.
				return nil, false, errors.New("None of the servers responded properly.")
			}

			// We have a replayed request, but not all workers responded yet, so lets wait for the next result.
			continue
		}

		// No error or REPLAYED_REQUEST. Seems like we have a proper result.
		break
	}

	// check status
	status, ok := yr.resultParameters["status"]
	if !ok || status != "OK" {
		switch status {
		case "BAD_OTP":
			return yr, false, nil
		case "REPLAYED_OTP":
			return yr, false, errors.New("The OTP is valid, but has been used before. If you receive this error, you might be the victim of a man-in-the-middle attack.")
		case "BAD_SIGNATURE":
			return yr, false, errors.New("Signature verification at the api server failed. The used id/key combination could be invalid or is not activated (yet).")
		case "NO_SUCH_CLIENT":
			return yr, false, errors.New("The api server does not accept the given id. It might be invalid or is not activated (yet).")
		case "OPERATION_NOT_ALLOWED":
			return yr, false, errors.New("The api server does not allow the given api id to verify OTPs.")
		case "BACKEND_ERROR":
			return yr, false, errors.New("The api server seems to be broken. Please contact the api servers system administration (yubico servers? contact yubico).")
		case "NOT_ENOUGH_ANSWERS":
			return yr, false, errors.New("The api server could not get requested number of syncs during before timeout")
		case "REPLAYED_REQUEST":
			panic("Unexpected. This status should've been catched in the worker response loop.")
			return yr, false, errors.New("The api server has seen this unique request before. If you receive this error, you might be the victim of a man-in-the-middle attack.")
		default:
			return yr, false, fmt.Errorf("Unknown status parameter (%s) sent by api server.", status)
		}
	}

	// check otp
	otpCheck, ok := yr.resultParameters["otp"]
	if !ok || otp != otpCheck {
		return nil, false, errors.New("Could not validate otp value from server response.")
	}

	// check nonce
	nonceCheck, ok := yr.resultParameters["nonce"]
	if !ok || nonce != nonceCheck {
		return nil, false, errors.New("Could not validate nonce value from server response.")
	}

	// check attached signature with remake of that signature, if key is actually in use.
	if len(ya.key) > 0 {
		receivedSignature, ok := yr.resultParameters["h"]
		if !ok || len(receivedSignature) == 0 {
			return nil, false, errors.New("No signature hash was attached by the api server, we do expect one though. This might be a hacking attempt.")
		}

		// create a slice with the same size-1 as the parameters map (we're leaving the hash itself out of it's replica calculation)
		receivedValuesSlice := make([]string, 0, len(yr.resultParameters)-1)
		for key, value := range yr.resultParameters {
			if key != "h" {
				receivedValuesSlice = append(receivedValuesSlice, key+"="+value)
			}
		}
		sort.Strings(receivedValuesSlice)
		receivedValuesString := strings.Join(receivedValuesSlice, "&")
		hmacenc := hmac.New(sha1.New, ya.key)
		_, err := hmacenc.Write([]byte(receivedValuesString))
		if err != nil {
			return nil, false, fmt.Errorf("Could not calculate signature replica. Error: %s\n", err)
		}
		recievedSignatureReplica := base64.StdEncoding.EncodeToString(hmacenc.Sum([]byte{}))

		if receivedSignature != recievedSignatureReplica {
			return nil, false, errors.New("The received signature hash is not valid. This might be a hacking attempt.")
		}
	}

	// we're done!
	yr.validOTP = true
	return yr, true, nil

}

// Contains details about yubikey OTP verification.
type YubiResponse struct {
	requestQuery     string
	resultParameters map[string]string
	validOTP         bool
}

func newYubiResponse(result *workResult) (*YubiResponse, error) {
	bodyReader := bufio.NewReader(result.response.Body)
	yr := &YubiResponse{}
	yr.resultParameters = make(map[string]string)
	yr.requestQuery = result.requestQuery
	for {
		// read through the response lines
		line, err := bodyReader.ReadString('\n')

		// handle error, which at one point should be an expected io.EOF (end of file)
		if err != nil {
			if err == io.EOF {
				break // successfully done with reading lines, lets break this for loop
			}
			return nil, fmt.Errorf("Could not read result body from the server. Error: %s\n", err)
		}

		// parse result lines, split on first '=', trim \n and \r
		keyvalue := strings.SplitN(line, "=", 2)
		if len(keyvalue) == 2 {
			yr.resultParameters[keyvalue[0]] = strings.Trim(keyvalue[1], "\n\r")
		}
	}
	return yr, nil
}

// Returns wether the verification was successful
func (yr *YubiResponse) IsValidOTP() bool {
	return yr.validOTP
}

// Get the requestQuery that was used during verification.
func (yr *YubiResponse) GetRequestQuery() string {
	return yr.requestQuery
}

// Retrieve a parameter from the api's response
func (yr *YubiResponse) GetResultParameter(key string) (value string) {
	value, ok := yr.resultParameters[key]
	if !ok {
		value = ""
	}
	return value
}
