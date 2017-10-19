package utils

import "encoding/json"

func ToJson(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func ToJsonString(data interface{}) (string, error) {
	json, err := ToJson(data)
	return string(json), err
}
