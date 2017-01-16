package utils

import (
	"fmt"
	"strings"
)

func BytesToString(size int) (result string) {
	switch {
	case size > (1024 * 1024 * 1024):
		result = fmt.Sprintf("%#.02f GB", float64(size)/1024/1024/1024)
	case size > (1024 * 1024):
		result = fmt.Sprintf("%#.02f MB", float64(size)/1024/1024)
	case size > 1024:
		result = fmt.Sprintf("%#.02f KB", float64(size)/1024)
	default:
		result = fmt.Sprintf("%d B", size)
	}
	result = strings.Trim(result, " ")
	return
}
