package hiprost

import (
	"bytes"
	"context"
	"testing"
)

var (
	// testAddress is a generic test address.
	testAddress = &Address{
		Components: []string{"test", "123"},
	}

	// testObject is a generic test object.
	testObject = &Object{
		Type: "testType",
		Data: []byte("testData"),
	}
)

// getContext obtains a context for the specified test.
func getContext(t *testing.T) (context.Context, context.CancelFunc) {
	deadline, ok := t.Deadline()
	if !ok {
		return context.Background(), func() {}
	}
	return context.WithDeadline(context.Background(), deadline)
}

// objectEqual checks whether the two objects are equal.
func objectEqual(o1, o2 *Object) bool {
	return o1.Type == o2.Type && bytes.Equal(o1.Data, o2.Data)
}
