package hiprost

import (
	"net"
	"testing"

	"github.com/TheCount/hiprost/backend/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// testServerAddr is the address the test server is listening on.
const testServerAddr = "localhost:59863"

// backendTest describes a backend test.
type backendTest struct {
	// Name is the name of the backend test.
	Name string

	// Func is the test function.
	Func func(t *testing.T, client HiprostClient)
}

// backendTests is the list of backend tests to perform for each backend.
var backendTests = [...]backendTest{
	// FIXME
}

// runBackendTestSuite runs generic tests with the specified backend.
func runBackendTestSuite(t *testing.T, backend common.Interface) {
	// create server
	srv := grpc.NewServer()
	defer srv.Stop()
	if err := RegisterNewHiprostServer(srv, backend); err != nil {
		t.Fatalf("unable to register new Hiprost server: %s", err)
	}
	listener, err := net.Listen("tcp", testServerAddr)
	if err != nil {
		t.Fatalf("unable to listen on '%s': %s", testServerAddr, err)
	}
	go func() {
		if err := srv.Serve(listener); err != nil {
			t.Errorf("unable to serve requests: %s", err)
		}
	}()
	// create client
	conn, err := grpc.Dial(testServerAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("unable to create client connection: %s", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("closing client connection: %s", err)
		}
	}()
	ctx, cancel := getContext(t)
	for {
		connState := conn.GetState()
		if connState == connectivity.Ready {
			break
		}
		if !conn.WaitForStateChange(ctx, connState) {
			t.Fatal("unable to connect to server")
		}
	}
	cancel()
	client := NewHiprostClient(conn)
	// run tests
	for _, test := range backendTests {
		t.Run(test.Name, func(t *testing.T) {
			test.Func(t, client)
		})
	}
}
