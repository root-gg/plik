package cursor

// CustomType is an interface that custom types need to implement
// in order to allow pagination over fields inside custom types.
type CustomType interface {
	// GetCustomTypeValue returns the value corresponding to the meta attribute inside the custom type.
	GetCustomTypeValue(meta interface{}) (interface{}, error)
}
