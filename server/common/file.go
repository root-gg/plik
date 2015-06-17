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

// File object
type File struct {
	ID             string                 `json:"id" bson:"fileId"`
	Name           string                 `json:"fileName" bson:"fileName"`
	Md5            string                 `json:"fileMd5" bson:"fileMd5"`
	Status         string                 `json:"status" bson:"status"`
	Type           string                 `json:"fileType" bson:"fileType"`
	UploadDate     int64                  `json:"fileUploadDate" bson:"fileUploadDate"`
	CurrentSize    int64                  `json:"fileSize" bson:"fileSize"`
	BackendDetails map[string]interface{} `json:"backendDetails,omitempty" bson:"backendDetails"`
	Reference      string                 `json:"reference" bson:"reference"`
}

// NewFile instantiate a new object
// and generate a random id
func NewFile() (file *File) {
	file = new(File)
	file.ID = GenerateRandomID(16)
	return
}

// GenerateID generate a new File ID
func (file *File) GenerateID() {
	file.ID = GenerateRandomID(16)
}

// Sanitize removes sensible information from
// object. Used to hide information in API.
func (file *File) Sanitize() {
	file.BackendDetails = nil
}
