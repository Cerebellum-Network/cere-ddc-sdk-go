package indexer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachePutGet(t *testing.T) {
	c := newCache[uint64, entry](4)
	now := time.Now()

	c.put(1, entry{info: BucketInfo{BucketID: 1, HasBalance: true}, fetchedAt: now})

	got, ok := c.get(1)
	require.True(t, ok)
	assert.Equal(t, now, got.fetchedAt)

	info, err := got.result()
	require.NoError(t, err)
	assert.True(t, info.HasBalance)

	_, ok = c.get(2)
	assert.False(t, ok)
}

func TestCacheOverwrite(t *testing.T) {
	c := newCache[uint64, entry](4)

	c.put(1, entry{info: BucketInfo{BucketID: 1, HasBalance: true}})
	c.put(1, entry{info: BucketInfo{BucketID: 1, HasBalance: false}})

	assert.Equal(t, 1, c.len())

	got, ok := c.get(1)
	require.True(t, ok)
	info, err := got.result()
	require.NoError(t, err)
	assert.False(t, info.HasBalance)
}

func TestCacheDenialEntry(t *testing.T) {
	c := newCache[uint64, entry](4)
	c.put(9, entry{deny: ErrBucketNotFound})

	got, ok := c.get(9)
	require.True(t, ok)

	_, err := got.result()
	assert.ErrorIs(t, err, ErrBucketNotFound)
}

func TestCacheEvictsLeastRecentlyUsed(t *testing.T) {
	c := newCache[uint64, entry](2)

	c.put(1, entry{info: BucketInfo{BucketID: 1}})
	c.put(2, entry{info: BucketInfo{BucketID: 2}})

	// Touch 1 so 2 becomes the eviction candidate.
	_, ok := c.get(1)
	require.True(t, ok)

	c.put(3, entry{info: BucketInfo{BucketID: 3}})

	assert.Equal(t, 2, c.len())
	_, ok = c.get(1)
	assert.True(t, ok, "recently used entry must survive")
	_, ok = c.get(3)
	assert.True(t, ok, "newest entry must survive")
	_, ok = c.get(2)
	assert.False(t, ok, "least recently used entry must be evicted")
}

func TestCacheCapacityFloor(t *testing.T) {
	c := newCache[uint64, entry](0)
	c.put(1, entry{info: BucketInfo{BucketID: 1}})
	c.put(2, entry{info: BucketInfo{BucketID: 2}})

	assert.Equal(t, 1, c.len())
}

// TestCacheResultDoesNotAliasStoredOwner pins the defensive copy in
// entry.result(). The owner is a byte slice living in a cache shared by every
// caller of one Client; handing it out by reference would let a single caller
// that mutated it in place change which account this client reports as the owner
// of that bucket for every later lookup.
func TestCacheResultDoesNotAliasStoredOwner(t *testing.T) {
	c := newCache[uint64, entry](4)
	owner := testOwner(0xAA)
	c.put(1, entry{
		info:      BucketInfo{BucketID: 1, Owner: owner, HasBalance: true},
		fetchedAt: time.Now(),
	})

	stored, ok := c.get(1)
	require.True(t, ok)
	first, err := stored.result()
	require.NoError(t, err)

	// A caller scribbling on what it was handed must not reach the cache.
	first.Owner[0] ^= 0xff

	stored, ok = c.get(1)
	require.True(t, ok)
	second, err := stored.result()
	require.NoError(t, err)

	assert.Equal(t, testOwner(0xAA), second.Owner,
		"the cached owner must survive a caller mutating an earlier result")
	assert.Equal(t, testOwner(0xAA), owner,
		"and the slice originally handed to put must be intact too")
}
