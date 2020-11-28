package common

import (
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
