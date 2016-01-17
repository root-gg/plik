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

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
)

// Result object
type Result struct {
	Message string      `json:"message"`
	Value   interface{} `json:"value"`
}

// NewResult create a new Result instance
func NewResult(message string, value interface{}) (r *Result) {
	r = new(Result)
	r.Message = message
	r.Value = value
	return
}

// ToJSON serialize result object to JSON
func (result *Result) ToJSON() []byte {
	j, err := utils.ToJson(result)
	if err != nil {
		msg := fmt.Sprintf("Unable to serialize result %s to json : %s", result.Message, err)
		Logger().Warning(msg)
		return []byte("{message:\"" + msg + "\"}")
	}

	return j
}

// ToJSONString is the same as ToJson but it returns
// a string instead of a byte array
func (result *Result) ToJSONString() string {
	return string(result.ToJSON())
}
