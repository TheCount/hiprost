package hiprost

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/TheCount/hiprost/backend/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// server implements HiprostServer.
type server struct {
	UnimplementedHiprostServer

	// backend is the storage backend used by this server.
	backend common.Interface
}

// PutObject implements HiprostServer.PutObject.
func (s *server) PutObject(ctx context.Context, req *PutObjectRequest) (
	*PutObjectResponse, error,
) {
	if req.Address == nil {
		return nil, status.Error(codes.InvalidArgument, "address missing")
	}
	if req.Object == nil {
		return nil, status.Error(codes.InvalidArgument, "object missing")
	}
	address := req.Address.AsCommon()
	object := req.Object.AsCommon()
	result := &PutObjectResponse{}
	// Calculate TTL
	var ttl time.Duration
	exp := req.Expiration
	if exp != nil {
		switch x := exp.Type.(type) {
		case nil:
			return nil, status.Error(codes.InvalidArgument, "ttl type missing")
		case *Expiration_Ttl:
			ttl = x.Ttl.AsDuration()
		case *Expiration_ExpiresAt:
			ttl = x.ExpiresAt.AsTime().Sub(time.Now())
		default:
			return nil, status.Error(codes.Unimplemented, "unknown expiry type")
		}
		if ttl < 0 {
			result.Error = &Error{
				Type: Error_ALREADY_EXPIRED,
				Msg:  "object has already expired",
			}
		}
	}
	// Handle CAS case
	if req.OldObject != nil {
		ok, err := s.backend.CompareAndSwapObject(ctx, address,
			req.OldObject.AsCommon(), object, ttl)
		if err != nil {
			result.Error = backendError(err)
			return result, nil
		}
		if ok {
			return result, nil
		}
		if !req.NoCreate {
			ok, err = s.backend.CreateObject(ctx, address, object, ttl)
			if err != nil {
				result.Error = backendError(err)
				return result, nil
			}
			if ok {
				result.Created = true
				return result, nil
			}
		}
		result.Error = casFailure
		return result, nil
	}
	// Handle standard case
	if req.NoCreate || object.Type == "" {
		ok, err := s.backend.UpdateObject(ctx, address, object, ttl)
		if err != nil {
			result.Error = backendError(err)
			return result, nil
		}
		if !ok {
			if req.NoCreate {
				result.Error = objectDoesNotExist
			} else {
				result.Error = noInferrableType
			}
		}
		return result, nil
	}
	created, err := s.backend.StoreObject(ctx, address, object, ttl)
	if err != nil {
		result.Error = backendError(err)
		return result, nil
	}
	result.Created = created
	return result, nil
}

// GetObject implements HiprostServer.GetObject.
func (s *server) GetObject(ctx context.Context, req *GetObjectRequest) (
	*GetObjectResponse, error,
) {
	if req.Address == nil {
		return nil, status.Error(codes.InvalidArgument, "address missing")
	}
	address := req.Address.AsCommon()
	result := &GetObjectResponse{}
	obj, err := s.backend.LoadObject(ctx, address)
	if err != nil {
		result.Error = backendError(err)
		return result, nil
	}
	if obj.Data == nil {
		if !req.MissingOk {
			result.Error = objectDoesNotExist
		}
		return result, nil
	}
	result.Object = &Object{
		Type: obj.Type,
		Data: obj.Data,
	}
	return result, nil
}

// DeleteObject implements HiprostServer.DeleteObject.
func (s *server) DeleteObject(ctx context.Context, req *DeleteObjectRequest) (
	*DeleteObjectResponse, error,
) {
	if req.Address == nil {
		return nil, status.Error(codes.InvalidArgument, "address missing")
	}
	address := req.Address.AsCommon()
	result := &DeleteObjectResponse{}
	// Handle CAS case
	if req.OldObject != nil {
		ok, err := s.backend.CompareAndSwapObject(ctx, address,
			req.OldObject.AsCommon(), common.Object{}, 0)
		if err != nil {
			result.Error = backendError(err)
			return result, nil
		}
		if !ok {
			obj, err := s.backend.LoadObject(ctx, address)
			if err != nil {
				result.Error = backendError(err)
				return result, nil
			}
			if obj.Data == nil {
				result.Missing = true
				if req.FailIfMissing {
					result.Error = objectDoesNotExist
				}
			} else {
				result.Error = casFailure
			}
		}
		return result, nil
	}
	// Normal case
	ok, err := s.backend.DeleteObject(ctx, address)
	if err != nil {
		result.Error = backendError(err)
		return result, nil
	}
	if !ok {
		result.Missing = true
		if req.FailIfMissing {
			result.Error = objectDoesNotExist
		}
	}
	return result, nil
}

// ListObjects implements HiprostServer.ListObjects.
func (s *server) ListObjects(ctx context.Context, req *ListObjectsRequest) (
	*ListObjectsResponse, error,
) {
	if req.Hierarchy == nil {
		return nil, status.Error(codes.InvalidArgument, "hierarchy missing")
	}
	baseAddr := req.Hierarchy.AsCommon()
	result := &ListObjectsResponse{}
	addresses, err := s.backend.ListObjects(ctx, baseAddr)
	if err != nil {
		result.Error = backendError(err)
		return result, nil
	}
	result.Addresses = make([]*Address, len(addresses))
	for i := range addresses {
		result.Addresses[i] = &Address{
			Components: addresses[i],
		}
	}
	return result, nil
}

// WatchObjects implements HiprostServer.WatchObjects.
func (s *server) WatchObjects(
	req *WatchObjectsRequest, stream Hiprost_WatchObjectsServer,
) error {
	if len(req.Hierarchies) == 0 {
		return status.Error(codes.InvalidArgument, "hierarchies missing")
	}
	// Sort hierarchies lexicographically and check for duplicates.
	hierarchies := make([]common.Address, len(req.Hierarchies))
	for i := range hierarchies {
		hierarchies[i] = req.Hierarchies[i].AsCommon()
	}
	sort.Slice(hierarchies, func(i, j int) bool {
		return hierarchies[i].Less(hierarchies[j])
	})
	for i := 0; i < len(hierarchies)-1; i++ {
		if hierarchies[i].IsPrefixOf(hierarchies[i+1]) {
			return status.Errorf(codes.InvalidArgument,
				"hierarchy '%s' is a prefix of '%s'", hierarchies[i], hierarchies[i+1])
		}
	}
	// Heuristic for update channel buffer size.
	bufsize := 10 * len(hierarchies)
	if bufsize > 1000 {
		bufsize = 1000
	}
	updateChan := make(chan common.Update, bufsize)
	defer close(updateChan)
	// Request updates
	ctx := stream.Context()
	result := &WatchObjectsResponse{}
	for _, h := range hierarchies {
		if err := s.backend.WatchObjects(
			ctx, h, req.Interrogate, updateChan,
		); err != nil {
			result.Error = backendError(err)
			return stream.Send(result)
		}
	}
	var previousAddress common.Address
	seenTypes := make(map[string]string)
	for {
		select {
		case update, ok := <-updateChan:
			if !ok {
				result.Error = &Error{
					Type: Error_BACKEND,
					Msg:  "backend overloaded",
				}
				result.Address = nil
				result.Object = nil
				result.Created = false
				return stream.Send(result)
			}
			if update.Err != nil {
				result.Error = backendError(update.Err)
				result.Address = nil
				result.Object = nil
				result.Created = false
				return stream.Send(result)
			}
			if req.ChangedOnly && !update.IsChanged() {
				continue
			}
			if previousAddress.Equal(update.Address) {
				result.Address = nil
			} else {
				previousAddress = update.Address
				result.Address = &Address{
					Components: update.Address,
				}
			}
			result.Created = update.IsCreated()
			if update.IsDeleted() {
				result.Object = nil
			} else {
				result.Object = &Object{
					Data: update.Object.Data,
				}
				if update.Object.Type != seenTypes[update.Address.String()] {
					result.Object.Type = update.Object.Type
					seenTypes[update.Address.String()] = update.Object.Type
				}
			}
			if err := stream.Send(result); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Sync implements HiprostServer.Sync.
func (s *server) Sync(
	ctx context.Context, _ *SyncRequest,
) (*SyncResponse, error) {
	result := &SyncResponse{}
	if err := s.backend.Sync(ctx); err != nil {
		result.Error = backendError(err)
	}
	return result, nil
}

// NewHiprostServer creates a new GRPC Hiprost server with the specified
// storage backend.
func NewHiprostServer(backend common.Interface) (HiprostServer, error) {
	if backend == nil {
		return nil, errors.New("backend is nil")
	}
	return &server{
		backend: backend,
	}, nil
}

// RegisterNewHiprostServer creates a new GRPC Hiprost server with the specified
// backend and registers it with the specified registrar (typically an instance
// of *grpc.Server).
func RegisterNewHiprostServer(
	s grpc.ServiceRegistrar, backend common.Interface,
) error {
	if s == nil {
		return errors.New("service registrar is nil")
	}
	server, err := NewHiprostServer(backend)
	if err != nil {
		return err
	}
	RegisterHiprostServer(s, server)
	return nil
}
