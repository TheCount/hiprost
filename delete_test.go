package hiprost

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testDeleteNilAddress tests a deletion reuest with a nil address.
func testDeleteNilAddress(t *testing.T, client HiprostClient) {
	ctx, cancel := getContext(t)
	defer cancel()
	if _, err := client.DeleteObject(
		ctx, &DeleteObjectRequest{},
	); status.Code(err) != codes.InvalidArgument {
		t.Fatal("expected invalid argument error when calling DeleteObject " +
			"with nil address")
	}
}

// testDeleteMissingObject tests deleting a nonexistent object.
func testDeleteMissingObject(t *testing.T, client HiprostClient) {
	ctx, cancel := getContext(t)
	defer cancel()
	resp, err := client.DeleteObject(ctx, &DeleteObjectRequest{
		Address:       missingAddress,
		FailIfMissing: true,
	})
	if err != nil {
		t.Fatalf("delete missing object GRPC error: %s", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error when deleting missing object with fail_if_missing")
	}
	if resp.Error.Type != Error_NOT_FOUND {
		t.Errorf(
			"expected NOT_FOUND when deleting missing object with fail_if_missing, "+
				"got %s", resp.Error,
		)
	}
	if !resp.Missing {
		t.Error("expected missing=true in response when deleting missing object")
	}
	// try again without fail_if_missing
	resp, err = client.DeleteObject(ctx, &DeleteObjectRequest{
		Address: missingAddress,
	})
	if err != nil {
		t.Fatalf("delete missing object GRPC error: %s", err)
	}
	if resp.Error != nil {
		t.Errorf("error deleting missing object: %s", resp.Error)
	}
	if !resp.Missing {
		t.Error("expected missing=true in response when deleting missing object")
	}
}

// testDeleteAfterPut tests deleting an existing object, checking that it is
// indeed gone.
func testDeleteAfterPut(t *testing.T, client HiprostClient) {
	ctx, cancel := getContext(t)
	defer cancel()
	// make sure object is present
	pResp, err := client.PutObject(ctx, &PutObjectRequest{
		Address: testAddress,
		Object:  testObject,
	})
	if err != nil || pResp.Error != nil {
		t.Fatal("error ensuring object existence")
	}
	// delete object
	dResp, err := client.DeleteObject(ctx, &DeleteObjectRequest{
		Address:       testAddress,
		FailIfMissing: true,
	})
	if err != nil {
		t.Fatalf("delete existing object GRPC error: %s", err)
	}
	if dResp.Error != nil {
		t.Fatalf("delete existing object error: %s", dResp.Error)
	}
	if dResp.Missing {
		t.Error("delete existing object should not set missing")
	}
	// make dure object is gone
	gResp, err := client.GetObject(ctx, &GetObjectRequest{
		Address:   testAddress,
		MissingOk: true,
	})
	if err != nil || gResp.Error != nil {
		t.Fatal("check deleted object error")
	}
	if gResp.Object != nil {
		t.Fatalf("object should have been deleted but found %s", gResp.Object)
	}
}
