package redis

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/TheCount/hiprost/backend/common"
	redis "github.com/go-redis/redis/v8"
)

// T implements the Hiprost common backend interface and the io.Closer
// interface.
type T struct {
	// client is the redis client.
	client *redis.Client
}

var (
	_ common.Interface = &T{}
	_ io.Closer        = &T{}
)

// Options describes the redis client options.
type Options = redis.Options

// Connect connects to a redis server and returns a new redis HiProst backend.
func Connect(opts *Options) (*T, error) {
	client := redis.NewClient(opts)
	return &T{
		client: client,
	}, nil
}

// Close implements io.Closer.Close. Must be called when concluding the use of
// this backend.
func (t *T) Close() error {
	return t.client.Close()
}

// publish publishes the specified value with the specified key.
func (t *T) publish(ctx context.Context, key, val string) error {
	return t.client.Publish(ctx, key, val).Err()
}

// CompareAndSwapObject implements common.Interface.CompareAndSwapObject.
func (t *T) CompareAndSwapObject(
	ctx context.Context, addr common.Address, old, new common.Object,
	ttl time.Duration,
) (swapped bool, rerr error) {
	if ttl < 0 {
		return false, errors.New("negative ttl")
	}
	key := addr.String()
	var val string
	rerr = t.client.Watch(ctx, func(tx *redis.Tx) error {
		item, err := tx.Get(ctx, key).Result()
		if err == redis.Nil {
			return nil
		}
		if err != nil {
			return err
		}
		var stored common.Object
		if err = stored.Decode([]byte(item)); err != nil {
			return err
		}
		if !stored.Equal(old) {
			return nil
		}
		// Delete if new data is nil
		if new.Data == nil {
			if _, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				return tx.Del(ctx, key).Err()
			}); err != nil {
				return err
			}
			swapped = true
			return nil
		}
		// Store new object
		if !stored.Equal(new) {
			new.Flags = common.FlagChanged
		}
		val = string(new.Encode(nil))
		if _, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			return pipe.Set(ctx, key, val, ttl).Err()
		}); err != nil {
			return err
		}
		swapped = true
		return nil
	}, key)
	if rerr == nil && swapped {
		rerr = t.publish(ctx, key, val)
	}
	return
}

// CreateObject implements common.Interface.CreateObject.
func (t *T) CreateObject(
	ctx context.Context, addr common.Address, obj common.Object,
	ttl time.Duration,
) (created bool, rerr error) {
	if ttl < 0 {
		return false, errors.New("negative ttl")
	}
	obj.Flags = common.FlagCreated | common.FlagChanged
	key := addr.String()
	var val string
	rerr = t.client.Watch(ctx, func(tx *redis.Tx) error {
		if err := tx.Get(ctx, key).Err(); err != redis.Nil {
			return err
		}
		val = string(obj.Encode(nil))
		if _, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			return pipe.Set(ctx, key, val, ttl).Err()
		}); err != nil {
			return err
		}
		created = true
		return nil
	}, key)
	if rerr == nil && created {
		rerr = t.publish(ctx, key, val)
	}
	return
}

// DeleteObject implements common.Interface.DeleteObject.
func (t *T) DeleteObject(
	ctx context.Context, addr common.Address,
) (deleted bool, rerr error) {
	key := addr.String()
	rerr = t.client.Watch(ctx, func(tx *redis.Tx) error {
		if err := tx.Get(ctx, key).Err(); err == redis.Nil {
			return nil
		} else if err != nil {
			return err
		}
		if _, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			return pipe.Del(ctx, key).Err()
		}); err != nil {
			return err
		}
		deleted = true
		return nil
	}, key)
	if rerr == nil && deleted {
		rerr = t.publish(ctx, key, "")
	}
	return
}

// ListObjects implements common.Interface.ListObjects.
func (t *T) ListObjects(
	ctx context.Context, baseAddr common.Address,
) ([]common.Address, error) {
	result := make([]common.Address, 0)
	pattern := baseAddr.String() + "*"
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = t.client.Scan(ctx, cursor, pattern, 0).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan '%s' at %d: %w", pattern, cursor, err)
		}
		for _, key := range keys {
			addr, err := common.NewAddressFromBytes([]byte(key))
			if err != nil {
				return nil, fmt.Errorf("convert redis key '%s' to address: %w",
					key, err)
			}
			result = append(result, addr)
		}
		if cursor == 0 {
			break
		}
	}
	return result, nil
}

// LoadObject implements common.Interface.LoadObject.
func (t *T) LoadObject(
	ctx context.Context, addr common.Address,
) (result common.Object, err error) {
	key := addr.String()
	item, err := t.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if err = result.Decode([]byte(item)); err != nil {
		err = fmt.Errorf("decode object: %w", err)
	}
	return
}

// StoreObject implements common.Interface.StoreObject.
func (t *T) StoreObject(
	ctx context.Context, addr common.Address, obj common.Object,
	ttl time.Duration,
) (created bool, rerr error) {
	if ttl < 0 {
		return false, errors.New("negative ttl")
	}
	key := addr.String()
	var val string
	rerr = t.client.Watch(ctx, func(tx *redis.Tx) error {
		item, err := tx.Get(ctx, key).Result()
		if err == redis.Nil {
			created = true
			obj.Flags = common.FlagCreated | common.FlagChanged
		} else if err != nil {
			return err
		} else {
			var stored common.Object
			if err = stored.Decode([]byte(item)); err != nil {
				return fmt.Errorf("decode object: %w", err)
			}
			if !stored.Equal(obj) {
				obj.Flags = common.FlagChanged
			}
		}
		val = string(obj.Encode(nil))
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			return pipe.Set(ctx, key, val, ttl).Err()
		})
		return err
	}, key)
	if rerr == nil {
		rerr = t.publish(ctx, key, val)
	}
	return
}

// UpdateObject implements common.Interface.UpdateObject.
func (t *T) UpdateObject(
	ctx context.Context, addr common.Address, obj common.Object,
	ttl time.Duration,
) (updated bool, rerr error) {
	if ttl < 0 {
		return false, errors.New("negative ttl")
	}
	key := addr.String()
	var val string
	rerr = t.client.Watch(ctx, func(tx *redis.Tx) error {
		if err := tx.Get(ctx, key).Err(); err == redis.Nil {
			return nil
		} else if err != nil {
			return err
		}
		val = string(obj.Encode(nil))
		var err error
		if _, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			return pipe.Set(ctx, key, val, ttl).Err()
		}); err == nil {
			updated = true
		}
		return err
	}, key)
	if rerr == nil {
		rerr = t.publish(ctx, key, val)
	}
	return
}

// WatchObjects implements common.Interface.WatchObjects.
func (t *T) WatchObjects(
	ctx context.Context, baseAddr common.Address,
	sendInitial bool, updateCh chan<- common.Update,
) error {
	if updateCh == nil {
		return errors.New("nil update channel")
	}
	key := baseAddr.String() + "*"
	// send updates
	go t.sendUpdates(ctx, key, sendInitial, updateCh)
	return nil
}

// sendUpdates sends updates to the specified update channel.
// Should be called in a separate goroutine.
func (t *T) sendUpdates(
	ctx context.Context, key string,
	sendInitial bool, updateCh chan<- common.Update,
) {
	// send initial values
	var seen map[string]struct{} // tracks sent initial keys
	var mx sync.Mutex            // controls access to seen
	if sendInitial {
		seen = make(map[string]struct{})
		go t.sendInitial(ctx, key, updateCh, &seen, &mx)
	}
	pubsub := t.client.PSubscribe(ctx, key)
	defer pubsub.Close()
	ch := pubsub.Channel()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			mx.Lock()
			if seen != nil {
				seen[msg.Channel] = struct{}{}
			}
			mx.Unlock()
			addr, err := common.NewAddressFromBytes([]byte(msg.Channel))
			if err != nil {
				t.sendError(updateCh, err)
				return
			}
			var obj common.Object
			if msg.Payload != "" {
				if err = obj.Decode([]byte(msg.Payload)); err != nil {
					t.sendError(updateCh, fmt.Errorf("decode object: %w", err))
					return
				}
			}
			if err = t.sendUpdate(updateCh, addr, obj); err != nil {
				t.sendError(updateCh, fmt.Errorf("send update: %w", err))
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// sendInitial sends out objects under prefix once if they have not been seen
// yet (and sent in parallel by t.sendUpdates).
func (t *T) sendInitial(
	ctx context.Context, pattern string, updateCh chan<- common.Update,
	seen *map[string]struct{}, seenMx *sync.Mutex,
) {
	defer func() {
		// kill tracker once we're done
		seenMx.Lock()
		*seen = nil
		seenMx.Unlock()
	}()
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = t.client.Scan(ctx, cursor, pattern, 0).Result()
		if err != nil {
			t.sendError(updateCh,
				fmt.Errorf("redis scan '%s' at %d: %w", pattern, cursor, err))
			return
		}
		for _, key := range keys {
			addr, err := common.NewAddressFromBytes([]byte(key))
			if err != nil {
				t.sendError(updateCh,
					fmt.Errorf("convert redis key '%s' to address: %w", key, err))
				return
			}
			item, err := t.client.Get(ctx, key).Result()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				t.sendError(updateCh, err)
				return
			}
			var obj common.Object
			if err = obj.Decode([]byte(item)); err != nil {
				t.sendError(updateCh, fmt.Errorf("decode object: %w", err))
				return
			}
			if err = func() error {
				seenMx.Lock()
				defer seenMx.Unlock()
				if _, ok := (*seen)[key]; !ok {
					if err = t.sendUpdate(updateCh, addr, obj); err != nil {
						return fmt.Errorf("send object '%s': %w", key, err)
					}
				}
				return nil
			}(); err != nil {
				t.sendError(updateCh, err)
				return
			}
		}
		if cursor == 0 {
			break
		}
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

// Sync implements common.Interface.Sync.
func (t *T) Sync(ctx context.Context) error {
	return nil // in-memory
}
