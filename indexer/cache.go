package indexer

import (
	"bytes"
	"container/list"
	"sync"
	"time"
)

// entry is one cached answer about one bucket.
//
// Definitive denials are cached too. A cached denial is safe under every
// condition a cached grant is not: serving a stale "no" during an outage
// denies, which is the direction this client is allowed to be wrong in. It also
// keeps a flood of lookups for a nonexistent bucket off the index.
type entry struct {
	info BucketInfo
	// deny is non-nil when the index answered with a definitive denial.
	deny error
	// fetchedAt is when the answer was read from the index.
	fetchedAt time.Time
}

// result renders the entry as the client's return values.
//
// BucketInfo.Owner and every delegate key are byte slices, and the entry they
// come from lives in a cache shared by every goroutine using this client.
// Returning them by reference would hand every caller an alias of those
// buffers: a single caller that reordered or truncated the bytes in place would
// silently change which account this client reports as the owner of that bucket
// — or who may act for them — for every later lookup, until the entry aged
// out. That is a cache poisoning primitive reachable from any consumer bug. The
// copies cost 32 bytes each and remove the class.
func (e entry) result() (BucketInfo, error) {
	if e.deny != nil {
		return BucketInfo{}, e.deny
	}
	info := e.info
	info.Owner = bytes.Clone(e.info.Owner)
	info.Delegates = cloneDelegates(e.info.Delegates)

	return info, nil
}

// delegateEntry is one cached answer about one owner's delegates.
//
// There is no deny field, and that is a property of the operation, not an
// omission: this lookup has no definitive denial. An owner the index has never
// heard of has no honored delegates, and that is an empty list and a nil error,
// so every error is an availability failure and none is ever cached.
type delegateEntry struct {
	delegates [][]byte
	// fetchedAt is when the answer was read from the index.
	fetchedAt time.Time
}

// result deep-copies the delegate keys for the same reason entry.result copies
// the owner: the slices live in a cache shared by every goroutine using this
// client, and handing them out by reference would make one consumer's bug a
// cache poisoning primitive — one mutated key would change who may act for an
// owner until the entry aged out.
func (e delegateEntry) result() [][]byte {
	return cloneDelegates(e.delegates)
}

// cache is a bounded LRU over cached answers. It stores no expiry of its own:
// freshness is a policy decision made by the client against the two TTLs, and
// an entry past its TTL is still useful as grace-window material.
//
// It is generic over the key because the client keeps two of them under one
// policy — bucket answers keyed by bucket id, delegate answers keyed by owner —
// and a second LRU implementation to change one key type would be a copy this
// file would have to keep honest forever.
type cache[K comparable, V any] struct {
	mu       sync.Mutex
	capacity int
	entries  map[K]*list.Element
	order    *list.List // front = most recently used
}

type cacheItem[K comparable, V any] struct {
	key   K
	value V
}

func newCache[K comparable, V any](capacity int) *cache[K, V] {
	if capacity < 1 {
		capacity = 1
	}
	return &cache[K, V]{
		capacity: capacity,
		entries:  make(map[K]*list.Element, capacity),
		order:    list.New(),
	}
}

func (c *cache[K, V]) get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}
	c.order.MoveToFront(element)

	return element.Value.(*cacheItem[K, V]).value, true
}

func (c *cache[K, V]) put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.entries[key]; ok {
		element.Value.(*cacheItem[K, V]).value = value
		c.order.MoveToFront(element)
		return
	}

	c.entries[key] = c.order.PushFront(&cacheItem[K, V]{key: key, value: value})

	for c.order.Len() > c.capacity {
		oldest := c.order.Back()
		if oldest == nil {
			return
		}
		c.order.Remove(oldest)
		delete(c.entries, oldest.Value.(*cacheItem[K, V]).key)
	}
}

func (c *cache[K, V]) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.order.Len()
}
