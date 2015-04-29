package common

type File struct {
	Id             string                 `json:"id" bson:"fileId"`
	Name           string                 `json:"fileName" bson:"fileName"`
	Md5            string                 `json:"fileMd5" bson:"fileMd5"`
	Status         string                 `json:"status" bson:"status"`
	Type           string                 `json:"fileType" bson:"fileType"`
	UploadDate     int64                  `json:"fileUploadDate" bson:"fileUploadDate"`
	CurrentSize    int64                  `json:"fileSize" bson:"fileSize"`
	BackendDetails map[string]interface{} `json:"backendDetails,omitempty" bson:"backendDetails"`
}

func NewFile() (file *File) {
	file = new(File)
	file.Id = GenerateRandomId(16)
	return
}

func (file *File) Sanitize() {
	file.BackendDetails = nil
}
