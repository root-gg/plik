/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015> Copyright holders list can be found in AUTHORS file
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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFile(t *testing.T) {
	file := NewFile()
	require.NotNil(t, file, "invalid file")
	require.NotZero(t, file.ID, "invalid file id")
}

func TestFileGenerateID(t *testing.T) {
	file := &File{}
	file.GenerateID()
	require.NotEqual(t, "", file.ID, "missing file id")
}

func TestFileSanitize(t *testing.T) {
	file := &File{}
	file.BackendDetails = make(map[string]interface{})
	file.BackendDetails["key"] = "value"
	file.Sanitize()
	require.Nil(t, file.BackendDetails, "invalid backend details")
}
