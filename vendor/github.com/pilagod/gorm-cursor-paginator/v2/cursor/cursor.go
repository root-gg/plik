package cursor

// Cursor cursor data
type Cursor struct {
	After  *string `json:"after" query:"after"`
	Before *string `json:"before" query:"before"`
}
