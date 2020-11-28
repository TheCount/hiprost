package common

import (
	"bytes"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

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

// Encode serialises this object, appending the serialised data to dst and
// returning it.
func (o Object) Encode(dst []byte) []byte {
	dst = protowire.AppendString(dst, o.Type)
	dst = protowire.AppendBytes(dst, o.Data)
	return dst
}

// Decode decodes the serialised input into this object.
func (o *Object) Decode(in []byte) error {
	t, tlen := protowire.ConsumeString(in)
	if tlen < 0 {
		return fmt.Errorf("parse type: %w", protowire.ParseError(tlen))
	}
	d, dlen := protowire.ConsumeBytes(in[tlen:])
	if dlen < 0 {
		return fmt.Errorf("parse data: %w", protowire.ParseError(dlen))
	}
	if len(in) != tlen+dlen {
		return fmt.Errorf("unused extra bytes (want: %d, have: %d)",
			tlen+dlen, len(in))
	}
	o.Type = t
	o.Data = d
	return nil
}
