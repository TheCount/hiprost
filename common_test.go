package hiprost

import (
	"context"
	"testing"
)

var (
	// testAddr is a generic test address.
	testAddr = &Address{
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
