package utils

import (
	"encoding/json"
	"fmt"
)

func Dump(data interface{}) {
	fmt.Println(Sdump(data))
}

func Sdump(data interface{}) string {
	if json, err := json.MarshalIndent(data, "", "  "); err == nil {
		return string(json)
	} else {
		return fmt.Sprintf("Unable to json data %v : %s", data, err)
	}
}
