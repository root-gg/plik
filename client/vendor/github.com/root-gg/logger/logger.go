package logger

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/root-gg/utils"
	"io"
	"os"
	"path"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARNING
	CRITICAL
	FATAL
)

var levels = []string{"DEBUG", "INFO", "WARNING", "CRITICAL", "FATAL"}

const (
	Fdate = 1 << iota
	Flevel ; FfixedSizeLevel
	FshortFile
	FlongFile
	FshortFunction
	FlongFunction
	Fdefault = Fdate | Flevel | FshortFile | FshortFunction
)

type Logger struct {
	MinLevel   Level
	Prefix     string
	Flags      int
	CallDepth  int
	DateFormat string
	Output     io.Writer
	lock       sync.RWMutex
}

var MinLevel = INFO

func NewLogger() (logger *Logger) {
	logger = new(Logger)
	logger.MinLevel = MinLevel
	logger.Prefix = ""
	logger.Flags = Fdefault
	logger.DateFormat = "01/02/2006 15:04:05"
	logger.Output = os.Stdout
	logger.CallDepth = 3
	return
}

func (logger *Logger) SetMinLevel(level Level) *Logger {
	logger.MinLevel = level
	return logger
}

func (logger *Logger) SetMinLevelFromString(level string) *Logger {
	for i := 0; i < len(levels); i++ {
		if levels[i] == level {
			logger.SetMinLevel(Level(i))
		}
	}
	return logger
}

func (logger *Logger) SetPrefix(prefix string) *Logger {
	logger.Prefix = prefix
	return logger
}

func (logger *Logger) SetFlags(flags int) *Logger {
	logger.Flags = flags
	return logger
}

func (logger *Logger) SetDateFormat(format string) *Logger {
	logger.DateFormat = format
	return logger
}

func (logger *Logger) SetOutput(output io.Writer) *Logger {
	logger.Output = output
	return logger
}

func (logger *Logger) SetCallDepth(depth int) *Logger {
	logger.CallDepth = depth
	return logger
}

func (logger *Logger) Copy() (copy *Logger) {
	copy = new(Logger)
	*copy = *logger
	return
}

func (logger *Logger) LogIf(level Level) bool {
	return level >= logger.MinLevel
}

func (logger *Logger) Debug(message string) {
	logger.Log(DEBUG, message)
}

func (logger *Logger) Debugf(format string, values ...interface{}) {
	logger.Log(DEBUG, fmt.Sprintf(format, values...))
}

func (logger *Logger) Info(message string) {
	logger.Log(INFO, message)
}

func (logger *Logger) Infof(format string, values ...interface{}) {
	logger.Log(INFO, fmt.Sprintf(format, values...))
}

func (logger *Logger) Warning(message string) {
	logger.Log(WARNING, message)
}

func (logger *Logger) Warningf(format string, values ...interface{}) {
	logger.Log(WARNING, fmt.Sprintf(format, values...))
}

func (logger *Logger) EWarning(message string) (err error) {
	err = errors.New(message)
	logger.Log(WARNING, message)
	return
}

func (logger *Logger) EWarningf(format string, values ...interface{}) (err error) {
	err = errors.New(fmt.Sprintf(format, values...))
	logger.Error(WARNING, err.Error())
	return
}

func (logger *Logger) Critical(message string) {
	logger.Log(CRITICAL, message)
}

func (logger *Logger) Criticalf(format string, values ...interface{}) {
	logger.Log(CRITICAL, fmt.Sprintf(format, values...))
}

func (logger *Logger) ECritical(message string) (err error) {
	err = errors.New(message)
	logger.Log(CRITICAL, message)
	return
}

func (logger *Logger) ECriticalf(format string, values ...interface{}) (err error) {
	err = errors.New(fmt.Sprintf(format, values...))
	logger.Error(CRITICAL, err.Error())
	return
}

func (logger *Logger) Fatal(message string) {
	logger.Log(FATAL, message)
}

func (logger *Logger) Fatalf(format string, values ...interface{}) {
	logger.Log(FATAL, fmt.Sprintf(format, values...))
}

func (logger *Logger) Error(level Level, message string) (err error) {
	err = errors.New(message)
	logger.Log(level, message)
	return
}

func (logger *Logger) Errorf(level Level, format string, values ...interface{}) (err error) {
	err = errors.New(fmt.Sprintf(format, values...))
	logger.Error(level, err.Error())
	return
}

func (logger *Logger) Dump(level Level, data interface{}) {
	logger.Log(level, utils.Sdump(data))
}

func (logger *Logger) Log(level Level, message string) {
	if level >= logger.MinLevel {
		str := bytes.NewBufferString("")
		if logger.Flags&Fdate != 0 {
			str.WriteString(fmt.Sprintf("[%s]", time.Now().Format(logger.DateFormat)))
		}
		if logger.Flags&Flevel != 0 {
			if logger.Flags&FfixedSizeLevel != 0 {
				str.WriteString(fmt.Sprintf("[%-8s]", levels[level]))
			} else {
				str.WriteString(fmt.Sprintf("[%s]", levels[level]))
			}
		}
		if logger.Flags&(FshortFile|FlongFile|FshortFunction|FlongFunction) != 0 {
			file, line, function := utils.GetCaller(logger.CallDepth)
			str.WriteString("[")
			if logger.Flags&FlongFile != 0 {
				str.WriteString(fmt.Sprintf("%s:%d", file, line))
			} else if logger.Flags&FshortFile != 0 {
				str.WriteString(fmt.Sprintf("%s:%d", path.Base(file), line))
			}
			if ((logger.Flags & (FshortFile | FlongFile)) != 0) && (logger.Flags&(FshortFunction|FlongFunction) != 0) {
				str.WriteString(" ")
			}
			if logger.Flags&FlongFunction != 0 {
				str.WriteString(fmt.Sprintf("%s", function))
			} else if logger.Flags&FshortFunction != 0 {
				_, function = utils.ParseFunction(function)
				str.WriteString(fmt.Sprintf("%s", function))
			}
			str.WriteString("]")
		}
		if len(logger.Prefix) > 0 {
			str.WriteString(fmt.Sprintf("[%s]", logger.Prefix))
		}
		if len(str.String()) > 0 {
			str.WriteString(" ")
		}
		str.WriteString(fmt.Sprintf("%s", message))
		if len(message) > 0 && message[len(message)-1] != '\n' {
			str.WriteString("\n")
		}
		logger.lock.Lock()
		defer logger.lock.Unlock()
		str.WriteTo(logger.Output)
		if level == FATAL {
			exiter(1)
		}
	}
}

var exiter func(code int) = func(code int) {
	os.Exit(code)
}
