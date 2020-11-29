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

// testPutNilObject tests PutObject with a nil object.
func testPutNilObject(t *testing.T, client HiprostClient) {
	ctx, cancel := getContext(t)
	defer cancel()
	if _, err := client.PutObject(ctx, &PutObjectRequest{
		Address: testAddress,
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatal(
			"expected invalid argument error when calling PutObject with nil object")
	}
}

// testPutGetObject tests whether the result of GetObject matches the object
// sent with PutObject.
func testPutGetObject(t *testing.T, client HiprostClient) {
	ctx, cancel := getContext(t)
	defer cancel()
	putResp, err := client.PutObject(ctx, &PutObjectRequest{
		Address: testAddress,
		Object:  testObject,
	})
	if err != nil {
		t.Fatalf("put test object GRPC error: %s", err)
	}
	if putResp.Error != nil {
		t.Fatalf("unable to put test object: %s", putResp.Error)
	}
	getResp, err := client.GetObject(ctx, &GetObjectRequest{
		Address: testAddress,
	})
	if err != nil {
		t.Fatalf("get test object GRPC error: %s", err)
	}
	if getResp.Error != nil {
		t.Fatalf("unable to get test object: %s", getResp.Error)
	}
	if !objectEqual(testObject, getResp.Object) {
		t.Errorf("expected to get object %s, got %s", testObject, getResp.Object)
	}
}
