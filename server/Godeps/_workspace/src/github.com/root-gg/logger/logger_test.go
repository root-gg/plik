package logger

import (
	"bytes"
	"fmt"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

var logMessage string = "This is a log message\n"

func TestNew(t *testing.T) {
	logger := NewLogger()
	if logger.MinLevel != MinLevel {
		t.Errorf("Invalid timer default level %s instead of %s", logger.MinLevel, MinLevel)
	}
	logger.Log(INFO, logMessage)
}

func TestLogger(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(0)
	logger.Log(INFO, logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != logMessage {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), logMessage)
	}
}

func TestAutoNewLine(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(0)
	logger.Log(INFO, "This is a log message")
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != logMessage {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), logMessage)
	}
}

func TestPrefix(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	prefix := "prefix"
	logger := NewLogger().SetOutput(buffer).SetFlags(0).SetPrefix(prefix)
	expected := fmt.Sprintf("[%s] %s", prefix, logMessage)
	logger.Info(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
}

func TestDateFormat(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(Fdate).SetDateFormat("01/02/2006")
	expected := fmt.Sprintf("[%s] %s", time.Now().Format("01/02/2006"), logMessage)
	logger.Info(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
}

func TestShortFile(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(FshortFile)
	file, line, _ := utils.GetCaller(1)
	expected := fmt.Sprintf("[%s:%d] %s", path.Base(file), line+2, logMessage)
	logger.Info(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
}

func TestLongFile(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(FlongFile)
	file, line, _ := utils.GetCaller(1)
	expected := fmt.Sprintf("[%s:%d] %s", file, line+2, logMessage)
	logger.Info(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
}

func TestShortFunction(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(FshortFunction)
	expected := fmt.Sprintf("[%s] %s", "TestShortFunction", logMessage)
	logger.Info(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
}

func TestLongFunction(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(FlongFunction)
	expected := fmt.Sprintf("[%s] %s", "github.com/root-gg/logger.TestLongFunction", logMessage)
	logger.Info(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
}

func TestFileAndFunction(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(FshortFile | FshortFunction)
	file, line, _ := utils.GetCaller(1)
	expected := fmt.Sprintf("[%s:%d TestFileAndFunction] %s", path.Base(file), line+2, logMessage)
	logger.Info(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
}

func TestCallDepth(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(FshortFunction).SetCallDepth(1)
	expected := fmt.Sprintf("[%s] %s", "Log", logMessage)
	logger.Info(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
}

func TestDebug(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(Flevel).SetMinLevel(DEBUG)
	expected := fmt.Sprintf("[%s] %s", levels[DEBUG], logMessage)
	logger.Debug(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
	buffer.Reset()
	logger.Debugf("%s", logMessage)
	output, err = ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
	logIf := logger.LogIf(DEBUG)
	if logIf != true {
		t.Errorf("Invalid LogIf %t instead of %t", logIf, true)
	}
}

func TestInfo(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(Flevel).SetMinLevel(INFO)
	expected := fmt.Sprintf("[%s] %s", levels[INFO], logMessage)
	logger.Info(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
	buffer.Reset()
	logger.Infof("%s", logMessage)
	output, err = ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
	logIf := logger.LogIf(INFO)
	if logIf != true {
		t.Errorf("Invalid LogIf %t instead of %t", logIf, true)
	}
}

func TestWarning(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(Flevel).SetMinLevel(WARNING)
	expected := fmt.Sprintf("[%s] %s", levels[WARNING], logMessage)
	logger.Warning(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
	buffer.Reset()
	logger.Warningf("%s", logMessage)
	output, err = ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
	logIf := logger.LogIf(WARNING)
	if logIf != true {
		t.Errorf("Invalid LogIf %t instead of %t", logIf, true)
	}
}

func TestCritical(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(Flevel).SetMinLevel(CRITICAL)
	expected := fmt.Sprintf("[%s] %s", levels[CRITICAL], logMessage)
	logger.Critical(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
	buffer.Reset()
	logger.Criticalf("%s", logMessage)
	output, err = ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
	logIf := logger.LogIf(CRITICAL)
	if logIf != true {
		t.Errorf("Invalid LogIf %t instead of %t", logIf, true)
	}
}

func TestFatal(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(Flevel).SetMinLevel(FATAL)
	expected := fmt.Sprintf("[%s] %s", levels[FATAL], logMessage)
	var exitcode int = 0
	exiter = func(code int) {
		exitcode = code
	}
	logger.Fatal(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
	if exitcode != 1 {
		t.Errorf("Invalid exit code %d instead %d", exitcode, 1)
	}
	exitcode = 0
	buffer.Reset()
	logger.Fatalf("%s", logMessage)
	output, err = ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
	if exitcode != 1 {
		t.Errorf("Invalid exit code %d instead %d", exitcode, 1)
	}
	logIf := logger.LogIf(FATAL)
	if logIf != true {
		t.Errorf("Invalid LogIf %t instead of %t", logIf, true)
	}
}

func TestFixedSizeLevel(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(Flevel | FfixedSizeLevel)
	expected := fmt.Sprintf("[%-8s] %s", levels[INFO], logMessage)
	logger.Info(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
}

func TestMinLevel(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetMinLevel(FATAL)
	buffer.Reset()
	logger.Debug(logMessage)
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if len(output) > 0 {
		t.Errorf("Invalid logger output when level < MinLevel")
	}
	logIf := logger.LogIf(DEBUG)
	if logIf != false {
		t.Errorf("Invalid LogIf %t instead of %t", logIf, false)
	}
	buffer.Reset()
	logger.Info(logMessage)
	output, err = ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if len(output) > 0 {
		t.Errorf("Invalid logger output when level < MinLevel")
	}
	logIf = logger.LogIf(INFO)
	if logIf != false {
		t.Errorf("Invalid LogIf %t instead of %t", logIf, false)
	}
	buffer.Reset()
	logger.Warning(logMessage)
	output, err = ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if len(output) > 0 {
		t.Errorf("Invalid logger output when level < MinLevel")
	}
	logIf = logger.LogIf(WARNING)
	if logIf != false {
		t.Errorf("Invalid LogIf %t instead of %t", logIf, false)
	}
	buffer.Reset()
	logger.Critical(logMessage)
	output, err = ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if len(output) > 0 {
		t.Errorf("Invalid logger output when level < MinLevel")
	}
	logIf = logger.LogIf(CRITICAL)
	if logIf != false {
		t.Errorf("Invalid LogIf %t instead of %t", logIf, false)
	}
	logIf = logger.LogIf(FATAL)
	if logIf != true {
		t.Errorf("Invalid LogIf %t instead of %t", logIf, true)
	}
}

func TestMinLevelFromString(t *testing.T) {
	logger := NewLogger()
	logger.SetMinLevelFromString("DEBUG")
	if logger.MinLevel != DEBUG {
		t.Errorf("Invalid min level %s instead of %s", logger.MinLevel, DEBUG)
	}
	logger.SetMinLevelFromString("INVALID")
	if logger.MinLevel != DEBUG {
		t.Errorf("Invalid min level %s instead of %s", logger.MinLevel, DEBUG)
	}
	logger.SetMinLevelFromString("INFO")
	if logger.MinLevel != INFO {
		t.Errorf("Invalid min level %s instead of %s", logger.MinLevel, INFO)
	}
	logger.SetMinLevelFromString("WARNING")
	if logger.MinLevel != WARNING {
		t.Errorf("Invalid min level %s instead of %s", logger.MinLevel, WARNING)
	}
	logger.SetMinLevelFromString("CRITICAL")
	if logger.MinLevel != CRITICAL {
		t.Errorf("Invalid min level %s instead of %s", logger.MinLevel, CRITICAL)
	}
	logger.SetMinLevelFromString("FATAL")
	if logger.MinLevel != FATAL {
		t.Errorf("Invalid min level %s instead of %s", logger.MinLevel, FATAL)
	}
}

func TestError(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Errorf("Unable to open %s : %s", os.DevNull, err)
	}
	logger := NewLogger().SetOutput(devNull)
	err = logger.EWarning("Oops!")
	if err.Error() != "Oops!" {
		t.Errorf("Invalid error message \"%s\" instead of \"%s\"", err.Error(), "Oops!")
	}
	err = logger.EWarningf("Oops : %s", "it's broken")
	if err.Error() != "Oops : it's broken" {
		t.Errorf("Invalid error message \"%s\" instead of \"%s\"", err.Error(), "Oops : it's broken")
	}
	err = logger.ECritical("Oops!")
	if err.Error() != "Oops!" {
		t.Errorf("Invalid error message \"%s\" instead of \"%s\"", err.Error(), "Oops!")
	}
	err = logger.ECriticalf("Oops : %s", "it's broken")
	if err.Error() != "Oops : it's broken" {
		t.Errorf("Invalid error message \"%s\" instead of \"%s\"", err.Error(), "Oops : it's broken")
	}
	err = logger.Error(DEBUG, "Oops!")
	if err.Error() != "Oops!" {
		t.Errorf("Invalid error message \"%s\" instead of \"%s\"", err.Error(), "Oops!")
	}
	err = logger.Errorf(DEBUG, "Oops : %s", "it's broken")
	if err.Error() != "Oops : it's broken" {
		t.Errorf("Invalid error message \"%s\" instead of \"%s\"", err.Error(), "Oops : it's broken")
	}
}

func TestCopy(t *testing.T) {
	logger1 := NewLogger().SetPrefix("logger1")
	logger2 := logger1.Copy().SetPrefix("logger2")
	if logger1.Prefix != "logger1" {
		t.Errorf("Invalid logger prefix %t instead of %t", logger1.Prefix, "logger1")
	}
	if logger2.Prefix != "logger2" {
		t.Errorf("Invalid logger prefix %t instead of %t", logger2.Prefix, "logger2")
	}
}

type TestData struct {
	Foo string
}

func TestDump(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewLogger().SetOutput(buffer).SetFlags(0)
	logger.Dump(INFO, TestData{"bar"})
	expected := "{\n  \"Foo\": \"bar\"\n}\n"
	output, err := ioutil.ReadAll(buffer)
	if err != nil {
		t.Errorf("Unable to read logger output : %s", err)
	}
	if string(output) != expected {
		t.Errorf("Invalid log message \"%s\" instead of \"%s\"", string(output), expected)
	}
}
