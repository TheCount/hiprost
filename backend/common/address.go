package common

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Address describes an object address.
type Address []string

// Less performs a lexicographic comparison to see whether this address compares
// smaller than other.
func (a Address) Less(other Address) bool {
	for i := range a {
		if i >= len(other) || a[i] > other[i] {
			return false
		}
		if a[i] < other[i] {
			return true
		}
	}
	return false
}

// IsPrefixOf checks whether this address is a prefix of other. If both
// addresses are equal, true is returned.
func (a Address) IsPrefixOf(other Address) bool {
	if len(a) > len(other) {
		return false
	}
	for i := range a {
		if a[i] != other[i] {
			return false
		}
	}
	return true
}

// Equal reports whether this address and other are equal.
func (a Address) Equal(other Address) bool {
	if len(a) != len(other) {
		return false
	}
	for i := range a {
		if a[i] != other[i] {
			return false
		}
	}
	return true
}

// String renders this address as a string.
// The string can be used as the path component in an URL.
func (a Address) String() string {
	var result strings.Builder
	for _, comp := range a {
		result.WriteByte('/')
		result.WriteString(url.PathEscape(comp))
	}
	return result.String()
}

// NewAddressFromBytes creates a new address from the specified raw bytes.
// Invariant: addr.Equal(NewAddressFromBytes([]byte(addr.String()))).
func NewAddressFromBytes(raw []byte) (Address, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	if raw[0] != '/' {
		return nil, errors.New("encoded non-empty address must start with '/'")
	}
	raw = raw[1:]
	parts := bytes.Split(raw, []byte{'/'})
	result := make([]string, len(parts))
	for i := range result {
		component, err := url.PathUnescape(string(parts[i]))
		if err != nil {
			return nil, fmt.Errorf("unescape component %d: %w", i+1, err)
		}
		result[i] = component
	}
	return result, nil
}
