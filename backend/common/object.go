package common

import (
	"bytes"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// ObjectFlags describe object flags.
type ObjectFlags uint

// Object flags
const (
	// FlagCreated means that this is the first object to be stored at the
	// address, or that a previously deleted object has been restored.
	FlagCreated ObjectFlags = 1 << iota

	// FlagDeleted means that the object has been deleted.
	FlagDeleted

	// FlagChanged means that the object has changed.
	FlagChanged
)

// Object describes an object.
type Object struct {
	// Type is the object type. Can be omitted if it can be inferred.
	Type string

	// Data is the object data.
	Data []byte

	// Flags are object flags.
	// They are ignored for objects handed from the server frontend to a backend.
	Flags ObjectFlags
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
	dst = protowire.AppendVarint(dst, uint64(o.Flags))
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
	f, flen := protowire.ConsumeVarint(in[tlen+dlen:])
	if flen < 0 {
		return fmt.Errorf("parse flags: %w", protowire.ParseError(flen))
	}
	if len(in) != tlen+dlen+flen {
		return fmt.Errorf("unused extra bytes (want: %d, have: %d)",
			tlen+dlen+flen, len(in))
	}
	o.Type = t
	o.Data = d
	o.Flags = ObjectFlags(f)
	return nil
}
