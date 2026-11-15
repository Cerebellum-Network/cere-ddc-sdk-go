package indexer

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The assertion that *Client satisfies authz.BucketOwnerIndex lives in
// client.go, next to the method it constrains.

// ---------------------------------------------------------------------------
// harness
// ---------------------------------------------------------------------------

func testOwner(b byte) ed25519.PublicKey {
	owner := make(ed25519.PublicKey, ed25519.PublicKeySize)
	for i := range owner {
		owner[i] = b
	}
	return owner
}

// stubIndex is a stand-in implementation of the contract whose behaviour a test
// can swap at any point, including mid-run.
type stubIndex struct {
	server   *httptest.Server
	requests atomic.Int64

	mu       sync.Mutex
	respond  func(w http.ResponseWriter, bucketID uint64)
	lastPath atomic.Value
	lastVerb atomic.Value
}

func newStubIndex(t *testing.T) *stubIndex {
	t.Helper()

	s := &stubIndex{}
	s.respond = func(w http.ResponseWriter, bucketID uint64) {
		writeBucket(w, bucketID, testOwner(0x11), true, false)
	}

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests.Add(1)
		s.lastPath.Store(r.URL.Path)
		s.lastVerb.Store(r.Method)

		idText := strings.TrimPrefix(r.URL.Path, BucketsPath+"/")
		bucketID, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			writeContractError(w, http.StatusBadRequest, CodeInvalidBucketID, "unparseable bucket id")
			return
		}

		s.mu.Lock()
		respond := s.respond
		s.mu.Unlock()

		respond(w, bucketID)
	}))
	t.Cleanup(s.server.Close)

	return s
}

func (s *stubIndex) setRespond(respond func(w http.ResponseWriter, bucketID uint64)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.respond = respond
}

func (s *stubIndex) respondWithStatus(status int) {
	s.setRespond(func(w http.ResponseWriter, _ uint64) { w.WriteHeader(status) })
}

func writeBucket(w http.ResponseWriter, bucketID uint64, owner ed25519.PublicKey, hasBalance, isRemoved bool) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(BucketResponse{
		ContractVersion: ContractVersion,
		BucketID:        bucketID,
		Owner:           EncodeOwner(owner),
		HasBalance:      hasBalance,
		IsRemoved:       isRemoved,
	})
}

func writeContractError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		ContractVersion: ContractVersion,
		Code:            code,
		Message:         message,
	})
}

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// testConfig is a configuration tuned for tests: no retry sleeping, tight
// timeouts, TTLs picked so a fake clock can step over each boundary.
// testOptions are the knobs every client test starts from: retries off so a
// failure surfaces as one attempt, and TTLs the fake clock can step over.
func testOptions() []Option {
	return []Option{
		WithTTLs(time.Hour, 10*time.Second),
		WithGraceWindow(30 * time.Second),
		WithRequestTimeout(2 * time.Second),
	}
}

func newTestClient(t *testing.T, url string, clock *fakeClock, extra ...Option) *Client {
	t.Helper()

	opts := append(testOptions(), WithClock(clock.now))
	opts = append(opts, extra...)

	client, err := New(url, opts...)
	require.NoError(t, err)

	return client
}

// ---------------------------------------------------------------------------
// happy path
// ---------------------------------------------------------------------------

func TestClientLookup(t *testing.T) {
	index := newStubIndex(t)
	client := newTestClient(t, index.server.URL, newFakeClock())

	result, err := client.Lookup(context.Background(), 7)
	require.NoError(t, err)

	assert.Equal(t, testOwner(0x11), result.Owner)
	assert.True(t, result.HasBalance)
	assert.False(t, result.Removed)
	assert.Equal(t, "/v1/buckets/7", index.lastPath.Load())
	assert.Equal(t, http.MethodGet, index.lastVerb.Load())
}

func TestClientLookupNoBalance(t *testing.T) {
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, bucketID uint64) {
		writeBucket(w, bucketID, testOwner(0x22), false, false)
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	result, err := client.Lookup(context.Background(), 3)
	require.NoError(t, err)
	assert.Equal(t, testOwner(0x22), result.Owner)
	assert.False(t, result.HasBalance, "a zero-balance owner is a valid answer, not an error")
}

func TestClientLookupRemovedBucketIsReportedNotHidden(t *testing.T) {
	// Removal is a fact the index answered with, so it travels on the result
	// rather than as an error: the authorizer is what turns it into a
	// rejection (authz rejects Removed before it looks at ownership). Flattening
	// it into "no owner" or "no balance" here would hide from the verifier the
	// difference between a removed bucket and an unfunded one.
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, bucketID uint64) {
		writeBucket(w, bucketID, testOwner(0x33), true, true)
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	result, err := client.Lookup(context.Background(), 4)
	require.NoError(t, err)
	assert.True(t, result.Removed, "removal must reach the verifier")
	assert.Equal(t, testOwner(0x33), result.Owner, "the owner is still reported truthfully")
	assert.True(t, result.HasBalance)

	info, err := client.LookupBucket(context.Background(), 4)
	require.NoError(t, err, "LookupBucket exposes the same facts in contract shape")
	assert.True(t, info.IsRemoved)
	assert.True(t, info.HasBalance)
}

func TestClientLookupZeroBucketNeverHitsIndex(t *testing.T) {
	index := newStubIndex(t)
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.Lookup(context.Background(), 0)
	assert.ErrorIs(t, err, ErrBucketNotFound)
	assert.Zero(t, index.requests.Load())
}

func TestNewRejectsUnconfiguredURL(t *testing.T) {
	_, err := New("")
	assert.ErrorIs(t, err, ErrNotConfigured,
		"a consumer with no index URL must fail at boot, not deny every request at runtime")
}

// ---------------------------------------------------------------------------
// cache TTL behaviour
// ---------------------------------------------------------------------------

func TestClientServesFromCacheWithinBalanceTTL(t *testing.T) {
	index := newStubIndex(t)
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)

	_, err := client.Lookup(context.Background(), 7)
	require.NoError(t, err)

	clock.advance(9 * time.Second) // still inside the 10s balance TTL

	_, err = client.Lookup(context.Background(), 7)
	require.NoError(t, err)

	assert.Equal(t, int64(1), index.requests.Load(), "second lookup must be served from cache")

}

func TestClientRefetchesAfterBalanceTTL(t *testing.T) {
	index := newStubIndex(t)
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)

	result, err := client.Lookup(context.Background(), 7)
	require.NoError(t, err)
	require.True(t, result.HasBalance)

	// The owner spends down to zero. Ownership has not changed, but balance
	// is exactly the fact the short TTL exists to re-read.
	index.setRespond(func(w http.ResponseWriter, bucketID uint64) {
		writeBucket(w, bucketID, testOwner(0x11), false, false)
	})

	clock.advance(11 * time.Second)

	result, err = client.Lookup(context.Background(), 7)
	require.NoError(t, err)
	assert.False(t, result.HasBalance)
	assert.Equal(t, int64(2), index.requests.Load())
}

func TestClientOwnershipTTLDoesNotForceRefetchOnItsOwn(t *testing.T) {
	// Ownership TTL is a ceiling, not a cadence: within the balance TTL no
	// request is made regardless of how long ownership is allowed to live.
	index := newStubIndex(t)
	clock := newFakeClock()

	client := newTestClient(t, index.server.URL, clock,
		WithTTLs(time.Hour, time.Minute))

	for i := 0; i < 5; i++ {
		_, err := client.Lookup(context.Background(), 7)
		require.NoError(t, err)
		clock.advance(10 * time.Second)
	}

	assert.Equal(t, int64(1), index.requests.Load())
}

func TestClientEvictsBeyondCacheSize(t *testing.T) {
	index := newStubIndex(t)
	clock := newFakeClock()

	client := newTestClient(t, index.server.URL, clock,
		WithCacheSize(2))

	for _, bucketID := range []uint64{1, 2, 3} {
		_, err := client.Lookup(context.Background(), bucketID)
		require.NoError(t, err)
	}
	require.Equal(t, int64(3), index.requests.Load())

	// Bucket 1 was evicted, so it costs a request; bucket 3 is still cached.
	_, err := client.Lookup(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(4), index.requests.Load())

	_, err = client.Lookup(context.Background(), 3)
	require.NoError(t, err)
	assert.Equal(t, int64(4), index.requests.Load())
}

// ---------------------------------------------------------------------------
// fail closed
// ---------------------------------------------------------------------------

func TestClientFailsClosedOnColdCache(t *testing.T) {
	index := newStubIndex(t)
	index.respondWithStatus(http.StatusServiceUnavailable)

	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.Lookup(context.Background(), 7)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnavailable)

}

func TestClientFailsClosedOnUnreachableIndex(t *testing.T) {
	index := newStubIndex(t)
	url := index.server.URL
	index.server.Close() // nothing is listening any more

	client := newTestClient(t, (url), newFakeClock())

	_, err := client.Lookup(context.Background(), 7)
	assert.ErrorIs(t, err, ErrUnavailable)
}

func TestClientFailsClosedOnRequestTimeout(t *testing.T) {
	index := newStubIndex(t)
	release := make(chan struct{})
	index.setRespond(func(w http.ResponseWriter, bucketID uint64) {
		<-release
		writeBucket(w, bucketID, testOwner(0x11), true, false)
	})
	t.Cleanup(func() { close(release) })

	client := newTestClient(t, index.server.URL, newFakeClock(),
		WithRequestTimeout(50*time.Millisecond))

	_, err := client.Lookup(context.Background(), 7)
	assert.ErrorIs(t, err, ErrUnavailable)
}

func TestClientFailsClosedOnGarbageResponse(t *testing.T) {
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ uint64) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("<html>gateway</html>"))
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.Lookup(context.Background(), 7)
	assert.ErrorIs(t, err, ErrInvalidResponse)
}

func TestClientRejectsWrongContractVersion(t *testing.T) {
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, bucketID uint64) {
		_ = json.NewEncoder(w).Encode(BucketResponse{
			ContractVersion: ContractVersion + 1,
			BucketID:        bucketID,
			Owner:           EncodeOwner(testOwner(0x11)),
			HasBalance:      true,
		})
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.Lookup(context.Background(), 7)
	assert.ErrorIs(t, err, ErrInvalidResponse)
}

func TestClientRejectsAnswerAboutAnotherBucket(t *testing.T) {
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ uint64) {
		writeBucket(w, 999, testOwner(0x11), true, false)
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.Lookup(context.Background(), 7)
	assert.ErrorIs(t, err, ErrInvalidResponse)
}

func TestClientTreatsAuthFailureAsUnavailable(t *testing.T) {
	// A 403 says the index will not serve us. That is an outage of our access
	// to it, not a statement that the bucket has no owner — and it must never
	// be cached as a denial.
	index := newStubIndex(t)
	index.respondWithStatus(http.StatusForbidden)
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.Lookup(context.Background(), 7)
	require.ErrorIs(t, err, ErrUnavailable)
	assert.NotErrorIs(t, err, ErrBucketNotFound)

	_, err = client.Lookup(context.Background(), 7)
	assert.ErrorIs(t, err, ErrUnavailable)
	assert.Equal(t, int64(2), index.requests.Load(), "an availability failure must not be cached")
}

// ---------------------------------------------------------------------------
// grace window
// ---------------------------------------------------------------------------

func TestClientServesGraceWindowWhileIndexIsDown(t *testing.T) {
	index := newStubIndex(t)
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)

	result, err := client.Lookup(context.Background(), 7)
	require.NoError(t, err)
	require.True(t, result.HasBalance)

	index.respondWithStatus(http.StatusInternalServerError)
	clock.advance(20 * time.Second) // past the 10s balance TTL, inside the 30s grace

	result, err = client.Lookup(context.Background(), 7)
	require.NoError(t, err, "a previously-good answer inside the grace window is still served")
	assert.Equal(t, testOwner(0x11), result.Owner)
	assert.True(t, result.HasBalance)

}

func TestClientFailsClosedPastGraceWindow(t *testing.T) {
	index := newStubIndex(t)
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)

	_, err := client.Lookup(context.Background(), 7)
	require.NoError(t, err)

	index.respondWithStatus(http.StatusInternalServerError)
	clock.advance(41 * time.Second) // balance TTL 10s + grace 30s exhausted

	_, err = client.Lookup(context.Background(), 7)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnavailable)

}

func TestClientOwnershipTTLCapsTheGraceWindow(t *testing.T) {
	// Grace is additionally bounded by the ownership TTL: an operator cannot
	// configure a grace window long enough to serve an answer older than the
	// hard ceiling on answer age.
	index := newStubIndex(t)
	clock := newFakeClock()

	client := newTestClient(t, index.server.URL, clock,
		WithTTLs(time.Minute, 10*time.Second), WithGraceWindow(time.Hour))

	_, err := client.Lookup(context.Background(), 7)
	require.NoError(t, err)

	index.respondWithStatus(http.StatusInternalServerError)

	clock.advance(30 * time.Second) // inside the ownership ceiling
	_, err = client.Lookup(context.Background(), 7)
	require.NoError(t, err)

	clock.advance(31 * time.Second) // 61s old: past the ownership ceiling
	_, err = client.Lookup(context.Background(), 7)
	assert.ErrorIs(t, err, ErrUnavailable)
}

func TestClientGraceWindowRecoversWhenIndexReturns(t *testing.T) {
	index := newStubIndex(t)
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)

	_, err := client.Lookup(context.Background(), 7)
	require.NoError(t, err)

	index.respondWithStatus(http.StatusInternalServerError)
	clock.advance(20 * time.Second)
	_, err = client.Lookup(context.Background(), 7)
	require.NoError(t, err, "served from grace")

	// The index comes back with a different answer; the fresh answer wins and
	// resets the entry's age.
	index.setRespond(func(w http.ResponseWriter, bucketID uint64) {
		writeBucket(w, bucketID, testOwner(0x11), false, false)
	})
	clock.advance(1 * time.Second)

	result, err := client.Lookup(context.Background(), 7)
	require.NoError(t, err)
	assert.False(t, result.HasBalance)

	clock.advance(1 * time.Second)
	_, err = client.Lookup(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, int64(3), index.requests.Load(),
		"initial fetch, the failed grace fetch, the recovery fetch — and then the refreshed entry is inside its balance TTL again")
}

func TestClientZeroGraceWindowNeverServesStale(t *testing.T) {
	// GraceWindow=0 is the strictest setting and must mean exactly what it says:
	// the moment the cached answer goes stale, an unreachable index denies.
	index := newStubIndex(t)
	clock := newFakeClock()

	client := newTestClient(t, index.server.URL, clock,
		WithGraceWindow(0))

	_, err := client.Lookup(context.Background(), 7)
	require.NoError(t, err)

	index.respondWithStatus(http.StatusInternalServerError)

	clock.advance(9 * time.Second) // still inside the 10s balance TTL
	_, err = client.Lookup(context.Background(), 7)
	require.NoError(t, err, "still fresh, so no fetch is attempted at all")

	clock.advance(2 * time.Second) // 11s: stale, and there is no grace to fall back on
	_, err = client.Lookup(context.Background(), 7)
	require.ErrorIs(t, err, ErrUnavailable)

}

func TestWithinGraceWindow(t *testing.T) {
	// The predicate in isolation: both bounds are strict, and neither can be
	// configured away.
	client := &Client{
		balanceTTL:   10 * time.Second,
		graceWindow:  30 * time.Second,
		ownershipTTL: time.Hour,
	}

	assert.True(t, client.withinGraceWindow(0))
	assert.True(t, client.withinGraceWindow(39*time.Second))
	assert.False(t, client.withinGraceWindow(40*time.Second), "the bound is exclusive")
	assert.False(t, client.withinGraceWindow(time.Hour))

	capped := &Client{
		balanceTTL:   10 * time.Second,
		graceWindow:  100 * time.Hour, // an operator asking for the moon
		ownershipTTL: time.Minute,
	}
	assert.True(t, capped.withinGraceWindow(59*time.Second))
	assert.False(t, capped.withinGraceWindow(time.Minute),
		"no grace window can raise the ownership ceiling")

	strict := &Client{
		balanceTTL:   10 * time.Second,
		graceWindow:  0,
		ownershipTTL: time.Hour,
	}
	assert.False(t, strict.withinGraceWindow(10*time.Second),
		"zero grace: an answer that is stale at all is not servable")
}

// ---------------------------------------------------------------------------
// definitive denials
// ---------------------------------------------------------------------------

func TestClientCachesDenials(t *testing.T) {
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ uint64) {
		writeContractError(w, http.StatusNotFound, CodeBucketNotFound, "no such bucket")
	})
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)

	_, err := client.Lookup(context.Background(), 7)
	require.ErrorIs(t, err, ErrBucketNotFound)

	_, err = client.Lookup(context.Background(), 7)
	assert.ErrorIs(t, err, ErrBucketNotFound)
	assert.Equal(t, int64(1), index.requests.Load(), "a denial is an answer and is cached like one")
}

func TestClientAccountNotFoundIsADenial(t *testing.T) {
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ uint64) {
		writeContractError(w, http.StatusNotFound, CodeAccountNotFound, "no ledger entry")
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.Lookup(context.Background(), 7)
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

func TestClientServesCachedDenialWhileIndexIsDown(t *testing.T) {
	// Serving a stale "no" during an outage denies, which is the only
	// direction this client is allowed to be wrong in.
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ uint64) {
		writeContractError(w, http.StatusNotFound, CodeBucketNotFound, "no such bucket")
	})
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)

	_, err := client.Lookup(context.Background(), 7)
	require.ErrorIs(t, err, ErrBucketNotFound)

	index.respondWithStatus(http.StatusInternalServerError)
	clock.advance(20 * time.Second)

	_, err = client.Lookup(context.Background(), 7)
	assert.ErrorIs(t, err, ErrBucketNotFound)
	assert.NotErrorIs(t, err, ErrUnavailable)
}

func TestClientTreatsADegradedCodeAsUnavailableNotADenial(t *testing.T) {
	// The status alone cannot tell "no such bucket" from "I could not read my
	// index". The contract's error CODE is what decides, and a code the client
	// does not recognise as a denial must fall to the fail-closed path — never be
	// cached as a negative, which would deny a legitimate bucket for the life of
	// the entry even after the index recovers.
	for _, code := range []string{CodeUnavailable, "some_future_code"} {
		t.Run(code, func(t *testing.T) {
			index := newStubIndex(t)
			index.setRespond(func(w http.ResponseWriter, _ uint64) {
				writeContractError(w, http.StatusNotFound, code, "index is degraded")
			})
			client := newTestClient(t, index.server.URL, newFakeClock())

			_, err := client.Lookup(context.Background(), 7)
			require.ErrorIs(t, err, ErrUnavailable)
			assert.NotErrorIs(t, err, ErrBucketNotFound)

			_, err = client.Lookup(context.Background(), 7)
			require.ErrorIs(t, err, ErrUnavailable)
			assert.Equal(t, int64(2), index.requests.Load(),
				"a degraded answer must not be cached as a denial")
		})
	}
}

func TestClientIgnoresErrorBodyFromAnotherContractVersion(t *testing.T) {
	// A body this client cannot trust falls back to what the status alone
	// supports. A 404 supports "not found", so the denial stands — but it is the
	// status that carried it, not the unreadable body.
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ uint64) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			ContractVersion: ContractVersion + 1,
			Code:            "renamed_in_v2",
		})
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.Lookup(context.Background(), 7)
	assert.ErrorIs(t, err, ErrBucketNotFound)
}

func TestClientDenialCostsOneRequest(t *testing.T) {
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ uint64) {
		writeContractError(w, http.StatusNotFound, CodeBucketNotFound, "no such bucket")
	})

	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.Lookup(context.Background(), 7)
	require.ErrorIs(t, err, ErrBucketNotFound)
	assert.Equal(t, int64(1), index.requests.Load())
}

// ---------------------------------------------------------------------------
// retries and deduplication
// ---------------------------------------------------------------------------

func TestClientDedupesConcurrentLookups(t *testing.T) {
	index := newStubIndex(t)
	release := make(chan struct{})
	index.setRespond(func(w http.ResponseWriter, bucketID uint64) {
		<-release
		writeBucket(w, bucketID, testOwner(0x11), true, false)
	})

	client := newTestClient(t, index.server.URL, newFakeClock())

	const callers = 16
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = client.Lookup(context.Background(), 7)
		}(i)
	}

	// Give the goroutines a moment to pile up on the same in-flight fetch.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	for _, err := range errs {
		assert.NoError(t, err)
	}
	assert.Equal(t, int64(1), index.requests.Load(),
		"a burst on one cold bucket must collapse into a single request")
}

func TestClientDedupesConcurrentLookupsWhileIndexIsDown(t *testing.T) {
	// Deduplication has to hold in the failure direction too. A struggling index
	// under a traffic burst is precisely when it can least afford one request per
	// caller, and it is also when every caller ends up on the slow path.
	index := newStubIndex(t)
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)

	_, err := client.Lookup(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, int64(1), index.requests.Load())

	release := make(chan struct{})
	index.setRespond(func(w http.ResponseWriter, _ uint64) {
		<-release
		w.WriteHeader(http.StatusInternalServerError)
	})

	clock.advance(20 * time.Second) // stale, but inside the 30s grace window

	const callers = 16
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = client.Lookup(context.Background(), 7)
		}(i)
	}

	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	for _, err := range errs {
		assert.NoError(t, err, "every caller is served from the grace window")
	}
	assert.Equal(t, int64(2), index.requests.Load(),
		"the initial fetch plus exactly one failed refresh shared by all 16 callers")
}

func TestClientHonoursCallerCancellation(t *testing.T) {
	index := newStubIndex(t)
	release := make(chan struct{})
	index.setRespond(func(w http.ResponseWriter, bucketID uint64) {
		<-release
		writeBucket(w, bucketID, testOwner(0x11), true, false)
	})
	t.Cleanup(func() { close(release) })

	client := newTestClient(t, index.server.URL, newFakeClock())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Lookup(ctx, 7)
	assert.ErrorIs(t, err, ErrUnavailable, "a cancelled lookup denies, it does not allow")
}

// TestClientLookupDoesNotAliasCachedOwner is the end-to-end form of
// TestCacheResultDoesNotAliasStoredOwner: two Lookups served from the same
// cached entry must not share an owner buffer, or a consumer bug becomes a cache
// poisoning primitive that silently changes who owns a bucket.
func TestClientLookupDoesNotAliasCachedOwner(t *testing.T) {
	index := newStubIndex(t)
	client := newTestClient(t, index.server.URL, newFakeClock())

	first, err := client.Lookup(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, testOwner(0x11), first.Owner)

	first.Owner[0] ^= 0xff

	second, err := client.Lookup(context.Background(), 9)
	require.NoError(t, err)
	assert.Equal(t, int64(1), index.requests.Load(), "second lookup came from cache")
	assert.Equal(t, testOwner(0x11), second.Owner,
		"a mutated earlier result must not change the cached owner")
}

// TestClient400IsNeverCachedAsADenial pins spoluyan's finding on #1000.
//
// A 400 means the client and the index disagree about the contract. It says
// nothing about the bucket. But decodeErrorBody used to map the body's error code
// onto a domain error, CodeInvalidBucketID landed on ErrBucketNotFound,
// isDefinitiveDenial called that definitive, and fetchAndStore CACHED it — so one
// round of contract drift answered "this bucket does not exist" for the whole
// cache TTL, long after the drift was fixed.
//
// A false denial that persists is worse than a transient failure: the caller fails
// closed either way, but the cached form keeps failing after the cause is gone.
func TestClient400IsNeverCachedAsADenial(t *testing.T) {
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ uint64) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"contractVersion":1,"code":"invalid_bucket_id","message":"bad id"}`))
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.Lookup(context.Background(), 7)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidResponse,
		"a 400 is contract drift, not a statement about the bucket")
	assert.NotErrorIs(t, err, ErrBucketNotFound,
		"mapping a 400 onto ErrBucketNotFound makes it cacheable as a denial")

	// The second lookup must go to the index again. If the first were cached as a
	// denial, this would be served from cache and the count would stay at 1.
	_, err = client.Lookup(context.Background(), 7)
	require.Error(t, err)
	assert.Equal(t, int64(2), index.requests.Load(),
		"a 400 must not be cached; the drift may already be fixed")
}

// TestClientSharedFetchSurvivesTheFirstCallerLeaving pins the other half.
//
// The deduped fetch used to run on the FIRST caller's context, so one caller with
// a short deadline — or one that simply walked away — failed the lookup for every
// other caller waiting on the same in-flight call, even though their own contexts
// were still live. That is backwards under exactly the burst dedupe exists to
// absorb: the more callers pile onto a cold bucket, the likelier one cancels.
func TestClientSharedFetchSurvivesTheFirstCallerLeaving(t *testing.T) {
	index := newStubIndex(t)
	arrived := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	index.setRespond(func(w http.ResponseWriter, bucketID uint64) {
		once.Do(func() { close(arrived) })
		<-release
		writeBucket(w, bucketID, testOwner(0x11), true, false)
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	// Leader starts the fetch, then abandons it.
	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderErr := make(chan error, 1)
	go func() {
		_, err := client.Lookup(leaderCtx, 7)
		leaderErr <- err
	}()
	<-arrived

	// A follower joins the same in-flight call, then the leader gives up.
	followerResult := make(chan error, 1)
	go func() {
		_, err := client.Lookup(context.Background(), 7)
		followerResult <- err
	}()
	time.Sleep(20 * time.Millisecond) // let the follower reach the wait
	cancelLeader()

	// The leader returns promptly rather than waiting out the request timeout.
	select {
	case err := <-leaderErr:
		assert.ErrorIs(t, err, ErrUnavailable, "a cancelled caller denies")
	case <-time.After(time.Second):
		t.Fatal("the leader waited on a fetch it had cancelled")
	}

	// The shared fetch kept going, so the follower still gets its answer.
	close(release)
	select {
	case err := <-followerResult:
		require.NoError(t, err,
			"the follower's own context was live; the leader leaving must not fail it")
	case <-time.After(2 * time.Second):
		t.Fatal("the follower was failed by the leader's cancellation")
	}
}

// ---------------------------------------------------------------------------
// owner listing
// ---------------------------------------------------------------------------

// stubListIndex is a stand-in implementation of the owner listing. It pages a
// configurable set of bucket ids the way a real implementation must, so the
// client's cursor handling is exercised against something that could disagree
// with it.
type stubListIndex struct {
	server   *httptest.Server
	requests atomic.Int64
	lastPath atomic.Value

	mu      sync.Mutex
	ids     []uint64
	respond func(w http.ResponseWriter, r *http.Request, owner ed25519.PublicKey)
}

func newStubListIndex(t *testing.T, ids ...uint64) *stubListIndex {
	t.Helper()

	s := &stubListIndex{ids: ids}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests.Add(1)
		s.lastPath.Store(r.URL.String())

		owner, err := DecodeOwner(r.URL.Query().Get(ParamOwner))
		if err != nil {
			writeContractError(w, http.StatusBadRequest, CodeInvalidOwner, "bad owner")
			return
		}

		s.mu.Lock()
		respond := s.respond
		s.mu.Unlock()

		if respond == nil {
			s.writePage(w, r, owner)
			return
		}
		respond(w, r, owner)
	}))
	t.Cleanup(s.server.Close)

	return s
}

func (s *stubListIndex) setRespond(respond func(w http.ResponseWriter, r *http.Request, owner ed25519.PublicKey)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.respond = respond
}

func (s *stubListIndex) setIDs(ids ...uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids = ids
}

func (s *stubListIndex) writePage(w http.ResponseWriter, r *http.Request, owner ed25519.PublicKey) {
	s.mu.Lock()
	ids := append([]uint64(nil), s.ids...)
	s.mu.Unlock()

	after, _ := strconv.ParseUint(r.URL.Query().Get(ParamAfter), 10, 64)
	limit := len(ids)
	if raw := r.URL.Query().Get(ParamLimit); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeContractError(w, http.StatusBadRequest, CodeInvalidPagination, "limit must be positive")
			return
		}
		limit = parsed
	}

	buckets := make([]BucketResponse, 0, len(ids))
	for _, id := range ids {
		if id <= after {
			continue
		}
		buckets = append(buckets, testListElement(id, owner))
	}

	var nextAfter uint64
	if len(buckets) > limit {
		buckets = buckets[:limit]
		nextAfter = buckets[len(buckets)-1].BucketID
	}

	writeOwnerBuckets(w, BucketsResponse{
		ContractVersion: ContractVersion,
		Owner:           EncodeOwner(owner),
		Buckets:         buckets,
		NextAfter:       nextAfter,
	})
}

func writeOwnerBuckets(w http.ResponseWriter, payload BucketsResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func TestClientListBuckets(t *testing.T) {
	index := newStubListIndex(t, 4, 9)
	client := newTestClient(t, index.server.URL, newFakeClock())
	owner := testOwner(0x11)

	list, err := client.ListBuckets(context.Background(), owner, BucketsQuery{})
	require.NoError(t, err)

	assert.Equal(t, owner, ed25519.PublicKey(list.Owner))
	require.Len(t, list.Buckets, 2)
	assert.Equal(t, uint64(4), list.Buckets[0].BucketID)
	assert.Equal(t, uint64(9), list.Buckets[1].BucketID)
	assert.False(t, list.HasMore())

	// The literal request target, not a re-derivation of it: a client and an
	// implementation that disagree here see a 404, which on this operation looks
	// exactly like a node too old to serve it.
	assert.Equal(t, "/v1/buckets?owner="+EncodeOwner(owner), index.lastPath.Load())
}

// TestClientListBucketsEmptyIsNotAnError is the endpoint's whole reason for
// existing. If an account that owns nothing ever came back as an error, a
// brand-new developer's first login would be indistinguishable from an outage,
// and the SDK could not tell the user which one it was.
func TestClientListBucketsEmptyIsNotAnError(t *testing.T) {
	index := newStubListIndex(t)
	client := newTestClient(t, index.server.URL, newFakeClock())

	list, err := client.ListBuckets(context.Background(), testOwner(0x11), BucketsQuery{})
	require.NoError(t, err)
	assert.Empty(t, list.Buckets)
	assert.False(t, list.HasMore())
}

// TestClientListBucketsIsNeverCached pins the caching decision.
//
// If a page were cached for the balance TTL, a bucket created a second ago would
// be missing from the answer and the caller could not tell that from "you own
// nothing" — on the user's first login, with no recovery but waiting out a TTL
// they cannot see. The clock does not move in this test: the second call must
// still reach the index.
func TestClientListBucketsIsNeverCached(t *testing.T) {
	index := newStubListIndex(t, 4)
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)
	owner := testOwner(0x11)

	first, err := client.ListBuckets(context.Background(), owner, BucketsQuery{})
	require.NoError(t, err)
	require.Len(t, first.Buckets, 1)

	index.setIDs(4, 5)

	second, err := client.ListBuckets(context.Background(), owner, BucketsQuery{})
	require.NoError(t, err)
	assert.Len(t, second.Buckets, 2, "a bucket created since the last call must be visible immediately")
	assert.Equal(t, int64(2), index.requests.Load())
}

// TestClientListBucketsDoesNotPrimeTheBucketCache pins the other half of that
// decision: the listing neither reads nor writes the per-bucket cache. Priming it
// would evict up to a page's worth of hot authorization entries from a bounded LRU
// to save a round trip the caller does not need.
func TestClientListBucketsDoesNotPrimeTheBucketCache(t *testing.T) {
	index := newStubListIndex(t, 4, 9)
	client := newTestClient(t, index.server.URL, newFakeClock())

	list, err := client.ListBuckets(context.Background(), testOwner(0x11), BucketsQuery{})
	require.NoError(t, err)
	require.Len(t, list.Buckets, 2)

	assert.Equal(t, 0, client.cache.len(), "the listing must not populate the bucket cache")
}

func TestClientListBucketsPagesWithTheCursor(t *testing.T) {
	index := newStubListIndex(t, 1, 2, 3)
	client := newTestClient(t, index.server.URL, newFakeClock())
	owner := testOwner(0x11)

	first, err := client.ListBuckets(context.Background(), owner, BucketsQuery{Limit: 2})
	require.NoError(t, err)
	require.Len(t, first.Buckets, 2)
	require.True(t, first.HasMore())
	require.Equal(t, uint64(2), first.NextAfter)

	second, err := client.ListBuckets(context.Background(), owner,
		BucketsQuery{After: first.NextAfter, Limit: 2})
	require.NoError(t, err)
	require.Len(t, second.Buckets, 1)
	assert.Equal(t, uint64(3), second.Buckets[0].BucketID)
	assert.False(t, second.HasMore(), "the last page must terminate the walk")
}

// TestClientListBucketsCoalescesConcurrentCallers: a dashboard mounting
// several components asks the same question several times at once. Without
// coalescing each mount is its own request at the index.
func TestClientListBucketsCoalescesConcurrentCallers(t *testing.T) {
	index := newStubListIndex(t, 4)
	release := make(chan struct{})
	index.setRespond(func(w http.ResponseWriter, r *http.Request, owner ed25519.PublicKey) {
		<-release
		index.writePage(w, r, owner)
	})

	client := newTestClient(t, index.server.URL, newFakeClock())

	const callers = 16
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = client.ListBuckets(context.Background(), testOwner(0x11), BucketsQuery{})
		}(i)
	}

	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	for _, err := range errs {
		assert.NoError(t, err)
	}
	assert.Equal(t, int64(1), index.requests.Load(),
		"a burst of identical enumerations must collapse into one request")
}

// TestClientListBucketsDoesNotCoalesceDifferentQueries: sharing a fetch
// across different pages would hand one caller another caller's page.
func TestClientListBucketsDoesNotCoalesceDifferentQueries(t *testing.T) {
	index := newStubListIndex(t, 1, 2, 3)
	client := newTestClient(t, index.server.URL, newFakeClock())
	owner := testOwner(0x11)

	first, err := client.ListBuckets(context.Background(), owner, BucketsQuery{Limit: 1})
	require.NoError(t, err)
	second, err := client.ListBuckets(context.Background(), owner, BucketsQuery{Limit: 1, After: 1})
	require.NoError(t, err)

	assert.Equal(t, uint64(1), first.Buckets[0].BucketID)
	assert.Equal(t, uint64(2), second.Buckets[0].BucketID)
}

// TestClientListBucketsSharedFetchSurvivesTheLeaderLeaving is the listing's
// copy of the property #1000 was fixed for: the shared fetch must not be
// cancellable by whichever caller happened to start it, and a caller that walks
// away must not be made to wait for a fetch it no longer wants.
func TestClientListBucketsSharedFetchSurvivesTheLeaderLeaving(t *testing.T) {
	index := newStubListIndex(t, 4)
	arrived := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	index.setRespond(func(w http.ResponseWriter, r *http.Request, owner ed25519.PublicKey) {
		once.Do(func() { close(arrived) })
		<-release
		index.writePage(w, r, owner)
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderErr := make(chan error, 1)
	go func() {
		_, err := client.ListBuckets(leaderCtx, testOwner(0x11), BucketsQuery{})
		leaderErr <- err
	}()
	<-arrived

	followerResult := make(chan error, 1)
	go func() {
		_, err := client.ListBuckets(context.Background(), testOwner(0x11), BucketsQuery{})
		followerResult <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancelLeader()

	select {
	case err := <-leaderErr:
		assert.ErrorIs(t, err, ErrUnavailable, "a cancelled caller gets an error, not a page")
	case <-time.After(time.Second):
		t.Fatal("the leader waited on a fetch it had cancelled")
	}

	close(release)
	select {
	case err := <-followerResult:
		require.NoError(t, err,
			"the follower's own context was live; the leader leaving must not fail it")
	case <-time.After(2 * time.Second):
		t.Fatal("the follower was failed by the leader's cancellation")
	}
}

// TestClientListBucketsSharersDoNotAliasEachOther: callers that shared one
// coalesced fetch must not share its slices, or one of them sorting or truncating
// its own result rewrites everybody else's.
func TestClientListBucketsSharersDoNotAliasEachOther(t *testing.T) {
	index := newStubListIndex(t, 4, 9)
	release := make(chan struct{})
	index.setRespond(func(w http.ResponseWriter, r *http.Request, owner ed25519.PublicKey) {
		<-release
		index.writePage(w, r, owner)
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	results := make([]OwnerBuckets, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = client.ListBuckets(
				context.Background(), testOwner(0x11), BucketsQuery{})
		}(i)
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	require.Equal(t, int64(1), index.requests.Load(), "both callers shared one fetch")

	results[0].Buckets[0].Owner[0] ^= 0xff
	results[0].Buckets[0].BucketID = 77

	assert.Equal(t, testOwner(0x11), ed25519.PublicKey(results[1].Buckets[0].Owner))
	assert.Equal(t, uint64(4), results[1].Buckets[0].BucketID)
}

// TestClientListBuckets404IsNotAnEmptyList: this operation has no
// "not found", so a 404 means the route is absent — an old node or a wrong URL.
// Flattening it into an empty page would tell a user with buckets they own none,
// which is the worst answer this contract can give.
func TestClientListBuckets404IsNotAnEmptyList(t *testing.T) {
	index := newStubListIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ *http.Request, _ ed25519.PublicKey) {
		w.WriteHeader(http.StatusNotFound)
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.ListBuckets(context.Background(), testOwner(0x11), BucketsQuery{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnavailable)
	assert.NotErrorIs(t, err, ErrBucketNotFound)
	assert.NotErrorIs(t, err, ErrAccountNotFound)
}

// TestClientListBuckets400IsNeverADenial keeps the listing on the same rule
// the single-bucket path was fixed to follow: a 400 says nothing about the
// account, its error code is never mapped, and nothing about it is remembered.
func TestClientListBuckets400IsNeverADenial(t *testing.T) {
	index := newStubListIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ *http.Request, _ ed25519.PublicKey) {
		writeContractError(w, http.StatusBadRequest, CodeInvalidPagination, "`limit` must be in [1, 1000]")
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.ListBuckets(context.Background(), testOwner(0x11), BucketsQuery{Limit: 9999})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidResponse)
	assert.NotErrorIs(t, err, ErrBucketNotFound)
	assert.Contains(t, err.Error(), "`limit` must be in [1, 1000]",
		"the body's message is kept for diagnostics even though its code is ignored")

	_, err = client.ListBuckets(context.Background(), testOwner(0x11), BucketsQuery{Limit: 9999})
	require.Error(t, err)
	assert.Equal(t, int64(2), index.requests.Load(), "a 400 must not be remembered as an answer")
}

// TestClientListBucketsRejectsMalformedOwnerLocally: a mangled key must not
// be sent and must not come back as an empty page, which would read as "this
// account owns nothing".
func TestClientListBucketsRejectsMalformedOwnerLocally(t *testing.T) {
	index := newStubListIndex(t, 4)
	client := newTestClient(t, index.server.URL, newFakeClock())

	for _, tt := range []struct {
		name  string
		owner ed25519.PublicKey
	}{
		{name: "nil", owner: nil},
		{name: "short", owner: make(ed25519.PublicKey, 31)},
		{name: "long", owner: make(ed25519.PublicKey, 33)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ListBuckets(context.Background(), tt.owner, BucketsQuery{})
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidOwner)
		})
	}
	assert.Equal(t, int64(0), index.requests.Load(), "a malformed owner never reaches the index")
}

func TestClientListBucketsFailsOnUnavailableIndex(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusTooManyRequests, http.StatusForbidden} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			index := newStubListIndex(t)
			index.setRespond(func(w http.ResponseWriter, _ *http.Request, _ ed25519.PublicKey) {
				w.WriteHeader(status)
			})
			client := newTestClient(t, index.server.URL, newFakeClock())

			_, err := client.ListBuckets(context.Background(), testOwner(0x11), BucketsQuery{})
			assert.ErrorIs(t, err, ErrUnavailable)
		})
	}
}

func TestClientListBucketsHonoursCallerCancellation(t *testing.T) {
	index := newStubListIndex(t, 4)
	release := make(chan struct{})
	index.setRespond(func(w http.ResponseWriter, r *http.Request, owner ed25519.PublicKey) {
		<-release
		index.writePage(w, r, owner)
	})
	t.Cleanup(func() { close(release) })

	client := newTestClient(t, index.server.URL, newFakeClock())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.ListBuckets(ctx, testOwner(0x11), BucketsQuery{})
	assert.ErrorIs(t, err, ErrUnavailable, "a cancelled enumeration errors rather than returning nothing")
}

// TestClientListBucketsRejectsAnotherOwnersPage: an implementation that
// filtered on the wrong field would otherwise have its buckets rendered as this
// user's.
func TestClientListBucketsRejectsAnotherOwnersPage(t *testing.T) {
	index := newStubListIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ *http.Request, _ ed25519.PublicKey) {
		other := testOwner(0x22)
		writeOwnerBuckets(w, BucketsResponse{
			ContractVersion: ContractVersion,
			Owner:           EncodeOwner(other),
			Buckets:         []BucketResponse{testListElement(4, other)},
		})
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.ListBuckets(context.Background(), testOwner(0x11), BucketsQuery{})
	assert.ErrorIs(t, err, ErrInvalidResponse)
}

// TestClientListBucketsOverlongPageFailsLoudly: the body limit must surface
// as a decode error, never as a short page. A silently truncated page would tell a
// user they own fewer buckets than they do, and nothing would log.
func TestClientListBucketsOverlongPageFailsLoudly(t *testing.T) {
	index := newStubListIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ *http.Request, owner ed25519.PublicKey) {
		buckets := make([]BucketResponse, 0, 8000)
		for id := uint64(1); id <= 8000; id++ {
			buckets = append(buckets, testListElement(id, owner))
		}
		writeOwnerBuckets(w, BucketsResponse{
			ContractVersion: ContractVersion,
			Owner:           EncodeOwner(owner),
			Buckets:         buckets,
		})
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.ListBuckets(context.Background(), testOwner(0x11), BucketsQuery{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidResponse)
}

// TestErrorBodyDetail: the diagnostic text is best-effort, and a body it cannot
// read must degrade to no detail rather than to a wrong one. It must never look at
// the error code — mapping codes is what made contract drift cacheable once.
func TestErrorBodyDetail(t *testing.T) {
	for _, tt := range []struct {
		name   string
		body   string
		expect string
	}{
		{name: "not json", body: "<html>502</html>", expect: ""},
		{
			name:   "another contract version",
			body:   `{"contractVersion":99,"code":"invalid_owner","message":"nope"}`,
			expect: "",
		},
		{
			name:   "no message",
			body:   `{"contractVersion":1,"code":"invalid_owner"}`,
			expect: "",
		},
		{
			name:   "message",
			body:   `{"contractVersion":1,"code":"invalid_owner","message":"nope"}`,
			expect: ": nope",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, errorBodyDetail([]byte(tt.body)))
		})
	}
}

// ---------------------------------------------------------------------------
// request coalescing
// ---------------------------------------------------------------------------

// These test the coalescing primitive directly, because it is now shared by both
// operations: a regression here would show up as two unrelated symptoms at the
// HTTP level, and neither would name the cause.

// TestSharedFetchCollapsesABurst: without this, a cold key under load produces one
// request per caller at the index, exactly when it can least afford them.
func TestSharedFetchCollapsesABurst(t *testing.T) {
	var shared sharedFetch[int]
	var fetches atomic.Int64
	release := make(chan struct{})

	const callers = 16
	var wg sync.WaitGroup
	results := make([]int, callers)
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = shared.do(context.Background(), "k", time.Second,
				func(context.Context) int {
					fetches.Add(1)
					<-release
					return 42
				})
		}(i)
	}

	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	assert.Equal(t, int64(1), fetches.Load())
	for i := 0; i < callers; i++ {
		require.NoError(t, errs[i])
		assert.Equal(t, 42, results[i])
	}
}

// TestSharedFetchDoesNotShareAcrossKeys: sharing across keys would hand a caller
// the answer to somebody else's question.
func TestSharedFetchDoesNotShareAcrossKeys(t *testing.T) {
	var shared sharedFetch[string]

	first, err := shared.do(context.Background(), "a", time.Second,
		func(context.Context) string { return "a" })
	require.NoError(t, err)
	second, err := shared.do(context.Background(), "b", time.Second,
		func(context.Context) string { return "b" })
	require.NoError(t, err)

	assert.Equal(t, "a", first)
	assert.Equal(t, "b", second)
}

// TestSharedFetchOutlivesItsLeader pins both halves of the fix on #1000 at the
// level they now live: the fetch must survive the leader's cancellation for the
// followers' sake, and the leader must return at once rather than waiting out a
// fetch it abandoned.
func TestSharedFetchOutlivesItsLeader(t *testing.T) {
	var shared sharedFetch[int]
	arrived := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	fetch := func(context.Context) int {
		once.Do(func() { close(arrived) })
		<-release
		return 7
	}

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderErr := make(chan error, 1)
	go func() {
		_, err := shared.do(leaderCtx, "k", time.Second, fetch)
		leaderErr <- err
	}()
	<-arrived

	follower := make(chan int, 1)
	go func() {
		value, err := shared.do(context.Background(), "k", time.Second, fetch)
		require.NoError(t, err)
		follower <- value
	}()
	time.Sleep(20 * time.Millisecond)
	cancelLeader()

	select {
	case err := <-leaderErr:
		assert.ErrorIs(t, err, ErrUnavailable)
	case <-time.After(time.Second):
		t.Fatal("the leader waited on a fetch it had cancelled")
	}

	close(release)
	select {
	case value := <-follower:
		assert.Equal(t, 7, value, "the shared fetch must finish for the followers")
	case <-time.After(2 * time.Second):
		t.Fatal("the follower was failed by the leader's cancellation")
	}
}

// TestSharedFetchForgetsFinishedCalls: a key left in the map would pin one answer
// forever, which on the caching path means an answer that never refreshes.
func TestSharedFetchForgetsFinishedCalls(t *testing.T) {
	var shared sharedFetch[int]
	var fetches atomic.Int64

	for i := 0; i < 3; i++ {
		_, err := shared.do(context.Background(), "k", time.Second,
			func(context.Context) int {
				fetches.Add(1)
				return i
			})
		require.NoError(t, err)
	}

	assert.Equal(t, int64(3), fetches.Load())

	shared.mu.Lock()
	defer shared.mu.Unlock()
	assert.Empty(t, shared.calls)
}

// ---------------------------------------------------------------------------
// owner-scoped delegates
// ---------------------------------------------------------------------------

// stubDelegatesIndex is a stand-in implementation of the owner-scoped delegates
// lookup whose answers a test can swap at any point.
type stubDelegatesIndex struct {
	server   *httptest.Server
	requests atomic.Int64
	lastPath atomic.Value

	mu        sync.Mutex
	respond   func(w http.ResponseWriter, r *http.Request, owner ed25519.PublicKey)
	delegates []ed25519.PublicKey
}

func newStubDelegatesIndex(t *testing.T, delegates ...ed25519.PublicKey) *stubDelegatesIndex {
	t.Helper()

	s := &stubDelegatesIndex{delegates: delegates}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests.Add(1)
		s.lastPath.Store(r.URL.String())

		owner, err := DecodeOwner(r.URL.Query().Get(ParamOwner))
		if err != nil {
			writeContractError(w, http.StatusBadRequest, CodeInvalidOwner, "bad owner")
			return
		}

		s.mu.Lock()
		respond := s.respond
		s.mu.Unlock()

		if respond == nil {
			s.writeDelegates(w, owner)
			return
		}
		respond(w, r, owner)
	}))
	t.Cleanup(s.server.Close)

	return s
}

func (s *stubDelegatesIndex) setRespond(respond func(w http.ResponseWriter, r *http.Request, owner ed25519.PublicKey)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.respond = respond
}

func (s *stubDelegatesIndex) setDelegates(delegates ...ed25519.PublicKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.delegates = delegates
}

func (s *stubDelegatesIndex) respondWithStatus(status int) {
	s.setRespond(func(w http.ResponseWriter, _ *http.Request, _ ed25519.PublicKey) { w.WriteHeader(status) })
}

func (s *stubDelegatesIndex) writeDelegates(w http.ResponseWriter, owner ed25519.PublicKey) {
	s.mu.Lock()
	delegates := append([]ed25519.PublicKey(nil), s.delegates...)
	s.mu.Unlock()

	encoded := make([]string, 0, len(delegates))
	for _, delegate := range delegates {
		encoded = append(encoded, EncodeOwner(delegate))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(DelegatesResponse{
		ContractVersion: ContractVersion,
		Owner:           EncodeOwner(owner),
		Delegates:       encoded,
	})
}

func TestClientLookupDelegates(t *testing.T) {
	first := testOwner(0xAA)
	second := testOwner(0xBB)
	index := newStubDelegatesIndex(t, first, second)
	client := newTestClient(t, index.server.URL, newFakeClock())
	owner := testOwner(0x11)

	delegates, err := client.LookupDelegates(context.Background(), owner)
	require.NoError(t, err)

	require.Len(t, delegates, 2)
	assert.Equal(t, first, ed25519.PublicKey(delegates[0]), "decodes in order")
	assert.Equal(t, second, ed25519.PublicKey(delegates[1]))

	// The literal request target, not a re-derivation of it.
	assert.Equal(t, "/v1/delegates?owner="+EncodeOwner(owner), index.lastPath.Load())
}

// TestClientLookupDelegatesEmptyIsAnAnswer: "no honored delegates" is the answer
// a claim-time caller polls for, so it is served and cached like any other —
// and must never be produced by flattening a failure into a list.
func TestClientLookupDelegatesEmptyIsAnAnswer(t *testing.T) {
	index := newStubDelegatesIndex(t)
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)

	delegates, err := client.LookupDelegates(context.Background(), testOwner(0x11))
	require.NoError(t, err)
	assert.Empty(t, delegates)

	delegates, err = client.LookupDelegates(context.Background(), testOwner(0x11))
	require.NoError(t, err)
	assert.Empty(t, delegates)
	assert.Equal(t, int64(1), index.requests.Load(),
		"an empty answer is an answer and is cached like one")
}

// TestClientLookupDelegatesCachesPerOwner: the cache key is the owner, so two
// owners never share an answer. A second owner must cost a second request —
// being served the first owner's cached entry would hand one account another's
// delegate set — while a repeat of the first owner is served from cache.
func TestClientLookupDelegatesCachesPerOwner(t *testing.T) {
	first := testOwner(0xAA)
	second := testOwner(0xBB)
	index := newStubDelegatesIndex(t, first, second)
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.LookupDelegates(context.Background(), testOwner(0x11))
	require.NoError(t, err)
	require.Equal(t, int64(1), index.requests.Load())

	_, err = client.LookupDelegates(context.Background(), testOwner(0x22))
	require.NoError(t, err)
	assert.Equal(t, int64(2), index.requests.Load(),
		"another owner must not be served the first owner's cached answer")

	_, err = client.LookupDelegates(context.Background(), testOwner(0x11))
	require.NoError(t, err)
	assert.Equal(t, int64(2), index.requests.Load(),
		"the first owner's entry is still fresh and serves from cache")
}

// TestClientLookupDelegatesRefetchesRevocationPastTTL: the delegates answer is
// cached on the bucket policy's short TTL, not the near-static ownership one,
// and the reason is this test — a revoked grant must stop being honored on the
// same cadence balance is re-read on.
func TestClientLookupDelegatesRefetchesRevocationPastTTL(t *testing.T) {
	granted := testOwner(0xAA)
	index := newStubDelegatesIndex(t, granted)
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)
	owner := testOwner(0x11)

	delegates, err := client.LookupDelegates(context.Background(), owner)
	require.NoError(t, err)
	require.Len(t, delegates, 1)

	clock.advance(9 * time.Second) // inside the 10s balance TTL

	delegates, err = client.LookupDelegates(context.Background(), owner)
	require.NoError(t, err)
	assert.Len(t, delegates, 1, "still fresh, served from cache")

	index.setDelegates() // the owner revokes the grant
	clock.advance(2 * time.Second)

	delegates, err = client.LookupDelegates(context.Background(), owner)
	require.NoError(t, err)
	assert.Empty(t, delegates, "a revocation past the TTL must be picked up")
	assert.Equal(t, int64(2), index.requests.Load())
}

// TestClientLookupDelegatesServesGraceWindowWhileIndexIsDown and
// TestClientLookupDelegatesFailsClosedPastGraceWindow mirror the bucket cache's
// outage policy for the owner-scoped cache: a previously-good answer stands in
// for an unreachable index only inside the grace window, and past it the lookup
// errors rather than degrade into an empty list — an outage must never read as
// a revocation.
func TestClientLookupDelegatesServesGraceWindowWhileIndexIsDown(t *testing.T) {
	granted := testOwner(0xAA)
	index := newStubDelegatesIndex(t, granted)
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)
	owner := testOwner(0x11)

	delegates, err := client.LookupDelegates(context.Background(), owner)
	require.NoError(t, err)
	require.Len(t, delegates, 1)

	index.respondWithStatus(http.StatusInternalServerError)
	clock.advance(20 * time.Second) // past the 10s balance TTL, inside the 30s grace

	delegates, err = client.LookupDelegates(context.Background(), owner)
	require.NoError(t, err, "a previously-good answer inside the grace window is still served")
	require.Len(t, delegates, 1)
}

func TestClientLookupDelegatesFailsClosedPastGraceWindow(t *testing.T) {
	index := newStubDelegatesIndex(t, testOwner(0xAA))
	clock := newFakeClock()
	client := newTestClient(t, index.server.URL, clock)
	owner := testOwner(0x11)

	_, err := client.LookupDelegates(context.Background(), owner)
	require.NoError(t, err)

	index.respondWithStatus(http.StatusInternalServerError)
	clock.advance(41 * time.Second) // balance TTL 10s + grace 30s exhausted

	_, err = client.LookupDelegates(context.Background(), owner)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnavailable)
	assert.NotErrorIs(t, err, ErrBucketNotFound)
}

// TestClientLookupDelegatesFailsClosedOnColdCache: with nothing cached, an
// unreachable index is an error and never an empty list — "no delegates" is the
// exact answer a claim-time caller polls for, so producing it from an outage
// would end a wait that has not been answered.
func TestClientLookupDelegatesFailsClosedOnColdCache(t *testing.T) {
	index := newStubDelegatesIndex(t)
	index.respondWithStatus(http.StatusServiceUnavailable)
	client := newTestClient(t, index.server.URL, newFakeClock())

	delegates, err := client.LookupDelegates(context.Background(), testOwner(0x11))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnavailable)
	assert.Nil(t, delegates)
}

// TestClientLookupDelegates404IsUnavailable: this operation has no "not found"
// any more than the owner listing does, so a 404 means the route is missing —
// a node that predates the endpoint — and must surface as an outage, never as
// an empty delegate set.
func TestClientLookupDelegates404IsUnavailable(t *testing.T) {
	index := newStubDelegatesIndex(t)
	index.respondWithStatus(http.StatusNotFound)
	client := newTestClient(t, index.server.URL, newFakeClock())

	delegates, err := client.LookupDelegates(context.Background(), testOwner(0x11))
	require.ErrorIs(t, err, ErrUnavailable)
	assert.Nil(t, delegates)
}

// TestClientLookupDelegates400IsNeverADenial: the client only ever sends a
// well-formed owner, so a 400 is contract drift. Its code is never mapped and
// nothing about it is remembered — same rule as both existing operations.
func TestClientLookupDelegates400IsNeverADenial(t *testing.T) {
	index := newStubDelegatesIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ *http.Request, _ ed25519.PublicKey) {
		writeContractError(w, http.StatusBadRequest, CodeInvalidOwner, "owner required")
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.LookupDelegates(context.Background(), testOwner(0x11))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidResponse)
	assert.NotErrorIs(t, err, ErrBucketNotFound)
	assert.NotErrorIs(t, err, ErrAccountNotFound)

	_, err = client.LookupDelegates(context.Background(), testOwner(0x11))
	require.Error(t, err)
	assert.Equal(t, int64(2), index.requests.Load(), "a 400 must not be remembered as an answer")
}

// TestClientLookupDelegatesRejectsAnotherOwnersAnswer: the owner echo is
// checked, so an implementation answering about a different account is rejected
// instead of having its delegates attributed to this one.
func TestClientLookupDelegatesRejectsAnotherOwnersAnswer(t *testing.T) {
	index := newStubDelegatesIndex(t)
	index.setRespond(func(w http.ResponseWriter, _ *http.Request, _ ed25519.PublicKey) {
		other := testOwner(0x22)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DelegatesResponse{
			ContractVersion: ContractVersion,
			Owner:           EncodeOwner(other),
			Delegates:       []string{EncodeOwner(testOwner(0xAA))},
		})
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	_, err := client.LookupDelegates(context.Background(), testOwner(0x11))
	assert.ErrorIs(t, err, ErrInvalidResponse)
}

// TestClientLookupDelegatesRejectsMalformedOwnerLocally: a mangled key must not
// be sent, and must not come back as an empty answer either.
func TestClientLookupDelegatesRejectsMalformedOwnerLocally(t *testing.T) {
	index := newStubDelegatesIndex(t, testOwner(0xAA))
	client := newTestClient(t, index.server.URL, newFakeClock())

	for _, tt := range []struct {
		name  string
		owner ed25519.PublicKey
	}{
		{name: "nil", owner: nil},
		{name: "short", owner: make(ed25519.PublicKey, 31)},
		{name: "long", owner: make(ed25519.PublicKey, 33)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.LookupDelegates(context.Background(), tt.owner)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidOwner)
		})
	}
	assert.Equal(t, int64(0), index.requests.Load(), "a malformed owner never reaches the index")
}

// TestClientLookupFillsLookupResultDelegates: the bucket-scoped lookup carries
// the owner's delegates through to the authz LookupResult, and a body without
// the field — the pre-upgrade server shape — decodes as no delegates rather
// than as an error.
func TestClientLookupFillsLookupResultDelegates(t *testing.T) {
	delegate := testOwner(0xAA)
	index := newStubIndex(t)
	index.setRespond(func(w http.ResponseWriter, bucketID uint64) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BucketResponse{
			ContractVersion: ContractVersion,
			BucketID:        bucketID,
			Owner:           EncodeOwner(testOwner(0x11)),
			HasBalance:      true,
			Delegates:       []string{EncodeOwner(delegate)},
		})
	})
	client := newTestClient(t, index.server.URL, newFakeClock())

	result, err := client.Lookup(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, result.Delegates, 1)
	assert.Equal(t, delegate, ed25519.PublicKey(result.Delegates[0]))

	// An old server omits the field: still a valid answer, delegation denied.
	plain := newStubIndex(t)
	client = newTestClient(t, plain.server.URL, newFakeClock())

	result, err = client.Lookup(context.Background(), 7)
	require.NoError(t, err)
	assert.Empty(t, result.Delegates, "an absent field decodes as no delegates")
}
