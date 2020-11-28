// Package common defines the common Hiprost backend interface.
package common

import (
	"context"
)

// Interface is the common backend interface.
type Interface interface {
	// CompareAndSwapObject compares the object at addr with old. If they are
	// equal, the old object is replaced with new and swapped == true is returned.
	// If the data field of new is nil, the object is deleted instead.
	// If they are not equal, no operation is performed and swapped == false is
	// returned.
	CompareAndSwapObject(ctx context.Context, addr Address,
		old, new Object) (swapped bool, err error)

	// CreateObject creates the specified object at address if it does not already
	// exist. If the object is created, created == true is returned. If an object
	// already exists at address, no operation is performed and created == false
	// is returned.
	CreateObject(ctx context.Context,
		addr Address, obj Object) (created bool, err error)

	// DeleteObject deletes the object at the specified address. If the object
	// exists, deleted == true is returned. If no object exists at address,
	// no operation is performed and deleted == false is returned.
	DeleteObject(ctx context.Context, addr Address) (deleted bool, err error)

	// ListObjects returns the list of object addresses with valid objects
	// starting with baseAddr (possibly including baseAddr). If no objects exist
	// under baseAddr, the retunred list is empty.
	ListObjects(ctx context.Context, baseAddr Address) ([]Address, error)

	// LoadObject returns the specified object at address. If no object exists at
	// address, the returned object will have its Data field set to nil.
	LoadObject(ctx context.Context, addr Address) (Object, error)

	// StoreObject behaves like CreateObject if no object exists at the specified
	// address. Otherwise, StoreObject behaves like UpdateObject. StoreObject
	// returns whether the store resulted in the creation of a new object.
	StoreObject(ctx context.Context,
		addr Address, obj Object) (created bool, err error)

	// UpdateObject updates the object at address, but only if it already exists.
	// If the object is updated, updated == true is returned. If no object exists
	// at the specified address, no operation is performed and updated == false
	// is returned.
	UpdateObject(ctx context.Context,
		addr Address, obj Object) (updated bool, err error)

	// WatchObjects sends updates for the objects for whose addresses baseAddr is
	// a prefix. The updates are sent to the specified update channel.
	// If sendInitial is true, the inital values of objects will be sent in the
	// first update. Implementors must be prepared for the case that updateCh is
	// closed by another goroutine, for example if the same channel is passed to
	// multiple calls to WatchObjects.
	WatchObjects(ctx context.Context, baseAddr Address, sendInitial bool,
		updateCh chan<- Update) error
}