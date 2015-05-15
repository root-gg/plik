/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

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
