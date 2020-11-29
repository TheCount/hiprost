package hiprost

import (
	"testing"

	"google.golang.org/grpc"
)

// TestNewNilServer tests creation of a Hiprost server with a nil backend.
func TestNewNilServer(t *testing.T) {
	if _, err := NewHiprostServer(nil); err == nil {
		t.Error("expected error when creating server with nil backend")
	}
}

// TestRegisterNilServer tests registering a Hiprost server with nil arguments.
func TestRegisterNilServer(t *testing.T) {
	if err := RegisterNewHiprostServer(nil, nil); err == nil {
		t.Error("expected error when registering nil backend with nil registrar")
	}
	srv := grpc.NewServer()
	if err := RegisterNewHiprostServer(srv, nil); err == nil {
		t.Error(
			"expected error when registering nil backend with non-nil registrar")
	}
}
