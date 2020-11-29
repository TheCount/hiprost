package badger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/TheCount/hiprost/backend/common"
	badger "github.com/dgraph-io/badger/v2"
)

const (
	// averageRunGC is the average duration between badger GC runs.
	averageRunGC = 10 * time.Minute

	// runGCFuzz is the maximum random deviation from the average GC run interval.
	runGCFuzz = 5 * time.Minute
)

// getRandomDuration draws a random duration in the interval
// averageRunGC ± runGCFuzz.
func getRandomDuration() time.Duration {
	limit := 2 * int64(runGCFuzz)
	fuzz := time.Duration(rand.Int63n(limit)) - runGCFuzz
	return averageRunGC + fuzz
}

// T implements the Hiprost common backend interface and the
// io.Closer interface.
type T struct {
	// db is the badger database backing the storage.
	db *badger.DB

	// done is a channel which is closed when Close is called for the first time.
	done chan struct{}
}

var (
	_ common.Interface = &T{}
	_ io.Closer        = &T{}
)

// Open opens the badger database at the specified directory. The directory
// must exist. Use an empty directory for a new database.
// The backend must be properly closed once you're done using it.
func Open(dir string) (*T, error) {
	db, err := badger.Open(badger.DefaultOptions(dir))
	if err != nil {
		return nil, fmt.Errorf("open badger database: %w", err)
	}
	result := &T{
		db:   db,
		done: make(chan struct{}),
	}
	go result.runGC()
	return result, nil
}

// Close implements io.Closer.Close. Must be called when concluding the use
// of this backend.
func (t *T) Close() error {
	if err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = errors.New("badger database already closed")
			}
		}()
		close(t.done)
		return
	}(); err != nil {
		return err
	}
	if err := t.db.Close(); err != nil {
		return fmt.Errorf("close badger database: %w", err)
	}
	return nil
}

// runGC runs the badger GC at random intervals.
func (t *T) runGC() {
	timer := time.NewTimer(getRandomDuration())
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			t.db.RunValueLogGC(0.5)
			timer.Reset(getRandomDuration())
		case <-t.done:
			return
		}
	}
}

// CompareAndSwapObject implements common.Interface.CompareAndSwapObject.
func (t *T) CompareAndSwapObject(
	ctx context.Context, addr common.Address, old, new common.Object,
) (bool, error) {
	key := []byte(addr.String())
	tx := t.db.NewTransaction(true)
	defer tx.Discard()
	item, err := tx.Get(key)
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var stored common.Object
	if err = item.Value(func(val []byte) error {
		return stored.Decode(val)
	}); err != nil {
		return false, err
	}
	if !stored.Equal(old) {
		return false, nil
	}
	// Delete if new data is nil.
	if new.Data == nil {
		if err = tx.Delete(key); err != nil {
			return false, err
		}
		if err = tx.Commit(); err != nil {
			return false, err
		}
		return true, nil
	}
	// Store new object
	if !stored.Equal(new) {
		new.Flags = common.FlagChanged
	}
	val := new.Encode(nil)
	if err = tx.Set(key, val); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// CreateObject implements common.Interface.CreateObject.
func (t *T) CreateObject(
	ctx context.Context, addr common.Address, obj common.Object,
) (bool, error) {
	obj.Flags = common.FlagCreated | common.FlagChanged
	key := []byte(addr.String())
	tx := t.db.NewTransaction(true)
	defer tx.Discard()
	if _, err := tx.Get(key); err != badger.ErrKeyNotFound {
		return false, err
	}
	val := obj.Encode(nil)
	if err := tx.Set(key, val); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// DeleteObject implements common.Interface.DeleteObject.
func (t *T) DeleteObject(
	ctx context.Context, addr common.Address,
) (bool, error) {
	key := []byte(addr.String())
	tx := t.db.NewTransaction(true)
	defer tx.Discard()
	if _, err := tx.Get(key); err == badger.ErrKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := tx.Delete(key); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// ListObjects implements common.Interface.ListObjects.
func (t *T) ListObjects(
	ctx context.Context, baseAddr common.Address,
) ([]common.Address, error) {
	result := make([]common.Address, 0)
	key := []byte(baseAddr.String())
	tx := t.db.NewTransaction(false)
	defer tx.Discard()
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		it := tx.NewIterator(badger.IteratorOptions{
			Prefix: key,
		})
		for ; it.Valid(); it.Next() {
			item := it.Item()
			addr, err := common.NewAddressFromBytes(item.Key())
			if err != nil {
				errChan <- fmt.Errorf("invalid address '%s': %w", item.Key(), err)
				return
			}
			result = append(result, addr)
		}
	}()
	select {
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// LoadObject implements common.Interface.LoadObject.
func (t *T) LoadObject(
	ctx context.Context, addr common.Address,
) (result common.Object, err error) {
	key := []byte(addr.String())
	tx := t.db.NewTransaction(false)
	defer tx.Discard()
	item, err := tx.Get(key)
	if err == badger.ErrKeyNotFound {
		return result, nil
	}
	if err = item.Value(func(val []byte) error {
		return result.Decode(val)
	}); err != nil {
		err = fmt.Errorf("decode object: %w", err)
	}
	return
}

// StoreObject implements common.Interface.StoreObject.
func (t *T) StoreObject(
	ctx context.Context, addr common.Address, obj common.Object,
) (created bool, err error) {
	key := []byte(addr.String())
	tx := t.db.NewTransaction(true)
	defer tx.Discard()
	item, err := tx.Get(key)
	if err == badger.ErrKeyNotFound {
		created = true
		obj.Flags = common.FlagCreated | common.FlagChanged
	} else if err != nil {
		return false, err
	} else {
		var stored common.Object
		if err = item.Value(func(val []byte) error {
			return stored.Decode(val)
		}); err != nil {
			return false, err
		}
		if !stored.Equal(obj) {
			obj.Flags = common.FlagChanged
		}
	}
	if err = tx.Set(key, obj.Encode(nil)); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return
}

// UpdateObject implements common.Interface.UpdateObject.
func (t *T) UpdateObject(
	ctx context.Context, addr common.Address, obj common.Object,
) (bool, error) {
	key := []byte(addr.String())
	tx := t.db.NewTransaction(true)
	defer tx.Discard()
	if _, err := tx.Get(key); err == badger.ErrKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := tx.Set(key, obj.Encode(nil)); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// WatchObjects implements common.Interface.WatchObjects.
// FIXME: old object versions are not relayed correctly yet (would require
// option to keep more versions), will heuristically assume that version 1 means
// created and nil data means deleted.
func (t *T) WatchObjects(
	ctx context.Context, baseAddr common.Address,
	sendInitial bool, updateCh chan<- common.Update,
) error {
	if updateCh == nil {
		return errors.New("nil update channel")
	}
	key := []byte(baseAddr.String())
	// send updates
	go t.sendUpdates(ctx, key, sendInitial, updateCh)
	return nil
}

// sendUpdates sends updates to the specified update channel.
// Should be called in a separate goroutine.
func (t *T) sendUpdates(
	ctx context.Context, key []byte,
	sendInitial bool, updateCh chan<- common.Update,
) {
	tx := t.db.NewTransaction(false)
	defer tx.Discard()
	// send initial values
	var seen map[string]struct{} // tracks sent initial keys
	var mx sync.Mutex            // controls access to seen
	if sendInitial {
		seen = make(map[string]struct{})
		go t.sendInitial(ctx, key, updateCh, &seen, &mx)
	}
	err := t.db.Subscribe(ctx, func(kvList *badger.KVList) error {
		for _, kv := range kvList.GetKv() {
			mx.Lock()
			if seen != nil {
				seen[string(kv.GetKey())] = struct{}{}
			}
			mx.Unlock()
			addr, err := common.NewAddressFromBytes(kv.GetKey())
			if err != nil {
				return fmt.Errorf("new address from bytes: %w", err)
			}
			var obj common.Object
			if kv.Value != nil {
				if err = obj.Decode(kv.Value); err != nil {
					return fmt.Errorf("decode object: %w", err)
				}
			}
			if err = t.sendUpdate(updateCh, addr, obj); err != nil {
				return fmt.Errorf("send update: %w", err)
			}
		}
		return nil
	}, key)
	t.sendError(updateCh, err)
}

// sendInitial sends out objects under prefix once if they have not been seen
// yet (and sent in parallel by t.sendUpdates).
func (t *T) sendInitial(
	ctx context.Context, prefix []byte, updateCh chan<- common.Update,
	seen *map[string]struct{}, seenMx *sync.Mutex,
) {
	defer func() {
		// kill tracker once we're done
		seenMx.Lock()
		*seen = nil
		seenMx.Unlock()
	}()
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		errChan <- t.db.View(func(tx *badger.Txn) error {
			it := tx.NewIterator(badger.IteratorOptions{
				Prefix: prefix,
			})
			for ; it.Valid(); it.Next() {
				item := it.Item()
				addr, err := common.NewAddressFromBytes(item.Key())
				if err != nil {
					return fmt.Errorf("invalid address '%s': %w", item.Key(), err)
				}
				var obj common.Object
				if err = item.Value(func(val []byte) error {
					return obj.Decode(val)
				}); err != nil {
					return fmt.Errorf("decode object '%s': %w", item.Key(), err)
				}
				if err = func() error {
					seenMx.Lock()
					defer seenMx.Unlock()
					if _, ok := (*seen)[string(item.Key())]; !ok {
						if err = t.sendUpdate(updateCh, addr, obj); err != nil {
							return fmt.Errorf("send object '%s': %w", item.Key(), err)
						}
					}
					(*seen)[string(item.Key())] = struct{}{}
					return nil
				}(); err != nil {
					return err
				}
			}
			return nil
		})
	}()
	select {
	case err := <-errChan:
		if err != nil {
			t.sendError(updateCh, err)
			return
		}
	case <-ctx.Done():
		t.sendError(updateCh, ctx.Err())
		return
	}
}

// sendUpdate sends an update to the specified update channel.
func (t *T) sendUpdate(
	updateCh chan<- common.Update, addr common.Address, obj common.Object,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error sending update: %v", r)
		}
	}()
	updateCh <- common.Update{
		Address: addr,
		Object:  obj,
	}
	return nil
}

// sendError sends an update with the specified error to the specified
// update channel.
func (t *T) sendError(updateCh chan<- common.Update, err error) (rerr error) {
	defer func() {
		if r := recover(); r != nil {
			rerr = fmt.Errorf("error sending error: %v", r)
		}
	}()
	updateCh <- common.Update{
		Err: err,
	}
	return nil
}
