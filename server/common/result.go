package common

import (
	"fmt"
	"github.com/root-gg/utils"
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

func (result *Result) ToJson() []byte {
	if j, err := utils.ToJson(result); err == nil {
		return j
	} else {
		msg := fmt.Sprintf("Unable to serialize result %s to json : %s", result.Message, err)
		Log().Warning(msg)
		return []byte("{message:\"" + msg + "\"}")
	}
}

func (result *Result) ToJsonString() string {
	return string(result.ToJson())
}
