package hiprost

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testPutNilAddress tests PutObject with a nil address.
func testPutNilAddress(t *testing.T, client HiprostClient) {
	ctx, cancel := getContext(t)
	defer cancel()
	if _, err := client.PutObject(ctx, &PutObjectRequest{
		Object: testObject,
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatal(
			"expected invalid argument error when calling PutObject with nil address")
	}
}
