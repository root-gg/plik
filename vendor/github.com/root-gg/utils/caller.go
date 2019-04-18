package utils

import (
	"runtime"
	"strings"
)

func GetCaller(depth int) (file string, line int, function string) {
	pc, file, line, ok := runtime.Caller(depth)
	if ok {
		function = runtime.FuncForPC(pc).Name()
	}
	return
}

func ParseFunction(fct string) (pkg string, function string) {
	i := strings.LastIndex(fct, ".")
	if i > 0 {
		pkg = fct[:i]
		function = fct[i+1:]
	}
	return
}
