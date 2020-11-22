package common

import "bytes"

// Object describes an object.
type Object struct {
	// Type is the object type. Can be omitted if it can be inferred.
	Type string

	// Data is the object data.
	Data []byte
}

// Equal reports whether this object is equal to other.
// Two objects are equal if they have the same type and identical data.
// If the type of other is missing, the types of o and other are assumed to
// be equal, but not vice versa.
func (o Object) Equal(other Object) bool {
	if other.Type != "" {
		if o.Type != other.Type {
			return false
		}
	}
	return bytes.Equal(o.Data, other.Data)
}
