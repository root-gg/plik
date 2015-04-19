package utils

import (
	"encoding/json"
	"fmt"
	"log"
)

type Result struct {
	Message string      `json:"message"`
	Value   interface{} `json:"value"`
}

func NewResult(message string, value interface{}) (r *Result) {
	r = new(Result)
	r.Message = message
	r.Value = value
	return
}

func ToJson(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func ToJsonString(data interface{}) (string, error) {
	json, err := ToJson(data)
	return string(json), err
}

func (result *Result) ToJson() []byte {
	if j, err := ToJson(result); err == nil {
		return j
	} else {
		msg := fmt.Sprintf("Unable to serialize result %s to json : %s", result.Message, err)
		log.Println(msg)
		return []byte("{message:\"" + msg + "\"}")
	}
}

func (result *Result) ToJsonString() string {
	return string(result.ToJson())
}

//
//func (result *Result) WriteToResponse(w http.ResponseWriter) {
//	if j, err := result.toJson(); err == nil {
//		w.Write(byte(j))
//	} else {
//		log.Println("Unable to serialize result %s to json : %s ", result.Message, err)
//	}
//}
