package hiprost

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testGetNilAddress tests GetObject with a nil address.
func testGetNilAddress(t *testing.T, client HiprostClient) {
	ctx, cancel := getContext(t)
	defer cancel()
	if _, err := client.GetObject(
		ctx, &GetObjectRequest{},
	); status.Code(err) != codes.InvalidArgument {
		t.Fatal(
			"expected invalid argument error when calling GetObject with nil address")
	}
}

// testGetMissingObject tests getting a missing object.
func testGetMissingObject(t *testing.T, client HiprostClient) {
	ctx, cancel := getContext(t)
	defer cancel()
	resp, err := client.GetObject(ctx, &GetObjectRequest{
		Address: missingAddress,
	})
	if err != nil {
		t.Fatalf("get missing object GRPC error: %s", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error getting missing object")
	}
	if resp.Error.Type != Error_NOT_FOUND {
		t.Fatalf("expected NOT_FOUND getting missing object, got %s", resp.Error)
	}
	// try again with missing_ok
	resp, err = client.GetObject(ctx, &GetObjectRequest{
		Address:   missingAddress,
		MissingOk: true,
	})
	if err != nil {
		t.Fatalf("get missing object GRPC error: %s", err)
	}
	if resp.Error != nil {
		t.Errorf("got error after permissive getting missing object: %s",
			resp.Error)
	}
	if resp.Object != nil {
		t.Fatalf("found missing object: %s", resp.Object)
	}
}
