package hiprost

import (
	"os"
	"testing"

	"github.com/TheCount/hiprost/backend/badger"
)

const (
	// badgerDBDir is the database directory to use for the badger test.
	badgerDBDir = "badger_test_db"
)

// TestBadger performs the server test suite with a badger backend.
func TestBadger(t *testing.T) {
	os.RemoveAll(badgerDBDir)
	if err := os.MkdirAll(badgerDBDir, 0700); err != nil {
		t.Fatalf("create badger DB test directory: %s", err)
	}
	if !testing.Verbose() {
		defer os.RemoveAll(badgerDBDir)
	}
	backend, err := badger.Open(badgerDBDir)
	if err != nil {
		t.Fatalf("unable to create badger DB: %s", err)
	}
	defer func() {
		if err := backend.Close(); err != nil {
			t.Errorf("error closing badger backend: %s", err)
		}
	}()
	t.Log("running test suite for badger backend")
	runBackendTestSuite(t, backend)
}
