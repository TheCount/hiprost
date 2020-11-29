package hiprost

import (
	"context"
	"testing"
)

// testObject is a generic test object.
var testObject = &Object{
	Type: "testType",
	Data: []byte("testData"),
}

// getContext obtains a context for the specified test.
func getContext(t *testing.T) (context.Context, context.CancelFunc) {
	deadline, ok := t.Deadline()
	if !ok {
		return context.Background(), func() {}
	}
	return context.WithDeadline(context.Background(), deadline)
}
