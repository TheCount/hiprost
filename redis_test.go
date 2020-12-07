package hiprost

import (
	"testing"

	"github.com/TheCount/hiprost/backend/redis"
)

const (
	// redisServer is the address of the redis server used for testing.
	redisServer = "127.0.0.1:6379"
)

// TestRedis performs the server test suite with a redis backend.
func TestRedis(t *testing.T) {
	backend, err := redis.Connect(&redis.Options{
		Addr: redisServer,
	})
	if err != nil {
		t.Fatalf("unable to connect to redis server '%s': %s", redisServer, err)
	}
	defer func() {
		if err := backend.Close(); err != nil {
			t.Errorf("error closing redis backend: %s", err)
		}
	}()
	t.Log("running test suite for redis backend")
	runBackendTestSuite(t, backend)
}
