package indexer

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
)

// maxResponseBytes bounds how much of a single-bucket or delegates response body
// is read. The contract's bodies are a few hundred bytes (a delegates body is
// one key per granted proxy, and an owner's proxy list is small by chain
// construction); anything larger is a wrong endpoint or a hostile one, and
// either way must not be allowed to consume memory.
const maxResponseBytes = 64 << 10

// maxListResponseBytes bounds how much of an owner-listing body is read. A page
// is a collection, so the single-answer bound above would truncate a perfectly
// legitimate response.
//
// The arithmetic: one element is ~168 bytes on the wire (a 20-digit id, a
// 66-character owner, two booleans, the echoed version), so 1 MiB holds ~6200 of
// them. An implementation configured to page larger than that must raise this
// constant — and if it does not, the symptom is a JSON decode error, which is
// loud, because a truncated body cannot parse. It is never a short page, which
// would quietly tell a user they own fewer buckets than they do.
const maxListResponseBytes = 1 << 20

// Defaults for the tunables. Overridable through the With* options.
//
// This package carries no envconfig tags on purpose: env-var naming belongs to
// the component that embeds the client, not to a shared package. A component
// reads its own {COMPONENT}_* variables and passes the values in.
const (
	defaultOwnershipTTL        = time.Hour
	defaultBalanceTTL          = 15 * time.Second
	defaultGraceWindow         = 2 * time.Minute
	defaultRequestTimeout      = 2 * time.Second
	defaultCacheSize           = 8192
	defaultMaxIdleConnsPerHost = 32
)

// Client answers bucket ownership, balance and delegate questions against an
// implementation of the contract, with an in-memory cache in front of it.
//
// It implements auth.BucketOwnerIndex — the authorizer owns that
// declaration, this package supplies one way to satisfy it. The assertion below
// fails the build if either side drifts.
type Client struct {
	baseURL    string
	httpClient *http.Client
	now        func() time.Time

	ownershipTTL   time.Duration
	balanceTTL     time.Duration
	graceWindow    time.Duration
	requestTimeout time.Duration
	cacheSize      int

	cache *cache[uint64, entry]
	// delegatesCache holds owner-scoped delegate answers under the same
	// freshness policy the bucket cache holds bucket answers. Keyed by the
	// canonical owner encoding — the same string the request path carries — so
	// two callers asking about one owner share one entry by construction.
	// cacheSize bounds it independently of the bucket cache.
	delegatesCache *cache[string, delegateEntry]

	// Coalescing is per operation because the results have nothing in common,
	// but the keys are request paths in all three, so the key spaces cannot
	// collide even conceptually.
	bucketFetches    sharedFetch[fetchOutcome]
	listFetches      sharedFetch[listOutcome]
	delegatesFetches sharedFetch[delegatesOutcome]
}

var _ auth.BucketOwnerIndex = (*Client)(nil)

// Option customises a Client at construction.
type Option func(*Client)

// WithHTTPClient supplies the HTTP client used for contract calls. Each attempt
// is bounded by the request timeout through the request context, so the client's
// own Timeout is not relied upon.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithClock replaces the clock used for cache ageing. Tests use it to advance
// past TTLs without sleeping.
func WithClock(now func() time.Time) Option {
	return func(c *Client) { c.now = now }
}

// WithTTLs sets how long an ownership answer and a balance answer stay fresh.
// Ownership is near-static; balance moves, so it wants a short window.
func WithTTLs(ownership, balance time.Duration) Option {
	return func(c *Client) { c.ownershipTTL, c.balanceTTL = ownership, balance }
}

// WithRequestTimeout bounds a single contract call.
func WithRequestTimeout(d time.Duration) Option {
	return func(c *Client) { c.requestTimeout = d }
}

// WithCacheSize bounds the number of cached buckets.
func WithCacheSize(n int) Option {
	return func(c *Client) { c.cacheSize = n }
}

// WithGraceWindow bounds how long a stale-but-previously-good answer may still
// be served while the indexer is unreachable. Past it, lookups fail closed.
func WithGraceWindow(d time.Duration) Option {
	return func(c *Client) { c.graceWindow = d }
}

// New builds a Client for the contract served at baseURL. It fails rather than
// starting with a configuration that cannot serve, so a misconfigured consumer
// dies at boot instead of denying every request at runtime.
func New(baseURL string, opts ...Option) (*Client, error) {
	c := &Client{
		baseURL:        strings.TrimSuffix(baseURL, "/"),
		now:            time.Now,
		ownershipTTL:   defaultOwnershipTTL,
		balanceTTL:     defaultBalanceTTL,
		graceWindow:    defaultGraceWindow,
		requestTimeout: defaultRequestTimeout,
		cacheSize:      defaultCacheSize,
	}
	for _, opt := range opts {
		opt(c)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	c.cache = newCache[uint64, entry](c.cacheSize)
	c.delegatesCache = newCache[string, delegateEntry](c.cacheSize)

	if c.httpClient == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.MaxIdleConnsPerHost = defaultMaxIdleConnsPerHost
		c.httpClient = &http.Client{Transport: transport, Timeout: defaultRequestTimeout}
	}

	return c, nil
}

// validate rejects a configuration that could not serve a lookup.
func (c *Client) validate() error {
	if c.baseURL == "" {
		return ErrNotConfigured
	}
	if !strings.HasPrefix(c.baseURL, "http://") && !strings.HasPrefix(c.baseURL, "https://") {
		return fmt.Errorf("%w: %q must be http:// or https://", ErrNotConfigured, c.baseURL)
	}
	for _, f := range []struct {
		name string
		d    time.Duration
	}{
		{"ownership TTL", c.ownershipTTL},
		{"balance TTL", c.balanceTTL},
	} {
		if f.d <= 0 {
			return fmt.Errorf("indexer: %s must be positive", f.name)
		}
	}
	if c.requestTimeout <= 0 {
		return fmt.Errorf("indexer: request timeout must be positive")
	}
	if c.cacheSize <= 0 {
		return fmt.Errorf("indexer: cache size must be positive")
	}
	if c.graceWindow < 0 {
		return fmt.Errorf("indexer: grace window must not be negative")
	}
	return nil
}

// Lookup implements auth.BucketOwnerIndex.
//
// Every field of the returned auth.LookupResult is a fact read from the index,
// including Removed. A removed bucket is reported as Removed=true with a nil
// error rather than being flattened into "no owner" or "no balance", because the
// contract's caller — the authorizer — is the component that decides what
// each fact denies (it rejects Removed with auth.ErrBucketRemoved before it
// even looks at ownership).
//
// Delegates rides along on the same answer: the owner's honored delegates as
// raw keys, nil when the serving implementation predates the field — which the
// consumer reads as "no delegates", the fail-closed direction.
//
// This is not a softening of the fail-closed rule. The two are different
// things: Removed=true is the index ANSWERING, and the answer denies; a
// non-nil error means the index did NOT answer, and that also denies. There is
// no return from this method that authorizes anything on its own.
func (c *Client) Lookup(ctx context.Context, bucketID uint64) (auth.LookupResult, error) {
	info, err := c.LookupBucket(ctx, bucketID)
	if err != nil {
		return auth.LookupResult{}, err
	}
	return auth.LookupResult{
		Owner:      info.Owner,
		HasBalance: info.HasBalance,
		Removed:    info.IsRemoved,
		Delegates:  info.Delegates,
	}, nil
}

// LookupBucket returns everything the contract knows about a bucket.
//
// Cache policy, in order:
//
//  1. Younger than BalanceTTL — served from cache, no network.
//  2. Older — refetched. Success refreshes the entry.
//  3. Fetch failed and the index is unreachable — a cached answer is served
//     only while it is younger than BalanceTTL+GraceWindow and younger than
//     OwnershipTTL. Otherwise the lookup fails closed.
//
// A definitive denial from the index (no such bucket, no such account) is
// returned as-is and cached; it is an answer, not an outage.
func (c *Client) LookupBucket(ctx context.Context, bucketID uint64) (BucketInfo, error) {
	if bucketID == 0 {
		return BucketInfo{}, fmt.Errorf("%w: bucket id 0 is not a bucket", ErrBucketNotFound)
	}

	if cached, ok := c.cache.get(bucketID); ok && c.now().Sub(cached.fetchedAt) < c.balanceTTL {
		return cached.result()
	}

	result := c.fetchDeduped(ctx, bucketID)
	if result.err == nil {
		return result.entry.result()
	}

	// The index did not answer. Re-read the cache: a concurrent lookup may
	// have refreshed the entry while this fetch was failing.
	if cached, ok := c.cache.get(bucketID); ok {
		age := c.now().Sub(cached.fetchedAt)
		if c.withinGraceWindow(age) {
			return cached.result()
		}
	}

	// Nothing recent enough to stand in. Deny.
	return BucketInfo{}, result.err
}

// withinGraceWindow decides whether an answer of the given age may stand in for
// an answer the index failed to give.
//
// This is the only mechanism in this package through which a stale answer is
// served, and it is reached only after a fetch has already failed — on the
// bucket path and on the delegates path alike. Both bounds must hold:
//
//   - age < BalanceTTL+GraceWindow — the operator's explicit, bounded tolerance
//     for serving an answer whose balance half may have moved.
//   - age < OwnershipTTL — the hard ceiling on the age of any answer this client
//     will serve, which no grace window can raise. Config.Validate enforces
//     BalanceTTL <= OwnershipTTL, so this bound really is a ceiling.
//
// With GraceWindow = 0 the first bound is age < BalanceTTL, which this path has
// already established is false: zero grace means the client fails closed the
// instant the cached answer goes stale. There is no configuration under which
// this function returns true for an answer of unbounded age, which is what
// "never fail open" means concretely.
func (c *Client) withinGraceWindow(age time.Duration) bool {
	return age < c.balanceTTL+c.graceWindow && age < c.ownershipTTL
}

type fetchOutcome struct {
	// entry is the answer to cache and return; valid when err is nil.
	entry entry
	// err is an availability failure. Definitive denials travel inside entry.
	err error
}

// fetchDeduped runs one fetch per bucket per burst. The rationale for how the
// coalescing handles cancellation lives on sharedFetch.do; the only thing this
// wrapper adds is the key, which is the request path so it reads the same on
// both operations.
func (c *Client) fetchDeduped(ctx context.Context, bucketID uint64) fetchOutcome {
	outcome, err := c.bucketFetches.do(ctx, BucketPath(bucketID), c.requestTimeout,
		func(fetchCtx context.Context) fetchOutcome {
			return c.fetchAndStore(fetchCtx, bucketID)
		})
	if err != nil {
		return fetchOutcome{err: err}
	}

	return outcome
}

func (c *Client) fetchAndStore(ctx context.Context, bucketID uint64) fetchOutcome {
	info, err := c.doFetch(ctx, bucketID)

	switch {
	case err == nil:
		stored := entry{info: info, fetchedAt: c.now()}
		c.cache.put(bucketID, stored)
		return fetchOutcome{entry: stored}
	case isDefinitiveDenial(err):
		stored := entry{deny: err, fetchedAt: c.now()}
		c.cache.put(bucketID, stored)
		return fetchOutcome{entry: stored}
	default:
		return fetchOutcome{err: err}
	}
}

// doFetch performs the HTTP call.
func (c *Client) doFetch(ctx context.Context, bucketID uint64) (BucketInfo, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL+BucketPath(bucketID), nil)
	if err != nil {
		return BucketInfo{}, fmt.Errorf("%w: building request: %w", ErrUnavailable, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return BucketInfo{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	return c.interpret(bucketID, resp.StatusCode, body, readErr)
}

func (c *Client) interpret(bucketID uint64, statusCode int, body []byte, readErr error) (BucketInfo, error) {
	switch {
	case statusCode == http.StatusOK:
		if readErr != nil {
			return BucketInfo{}, fmt.Errorf("%w: reading body: %w", ErrUnavailable, readErr)
		}
		var payload BucketResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return BucketInfo{}, fmt.Errorf("%w: decoding body: %w", ErrInvalidResponse, err)
		}
		info, err := payload.Decode(bucketID)
		if err != nil {
			return BucketInfo{}, err
		}
		return info, nil

	case statusCode == http.StatusNotFound:
		return BucketInfo{}, decodeErrorBody(body, fmt.Errorf("%w: status 404", ErrBucketNotFound))

	case statusCode == http.StatusBadRequest:
		// The client only ever sends decimal, non-zero bucket ids, so a 400 means
		// the two sides disagree about the contract — a client/server mismatch,
		// never a statement about the bucket.
		//
		// The error code in the body is deliberately NOT consulted here.
		// decodeErrorBody would map CodeInvalidBucketID onto ErrBucketNotFound,
		// isDefinitiveDenial would call that a definitive denial, and
		// fetchAndStore would CACHE it — so one round of contract drift would
		// answer "this bucket does not exist" for the whole cache TTL, long after
		// the drift was fixed. A false denial that persists is worse than a
		// transient failure: the caller fails closed either way, but the cached
		// form keeps failing after the cause is gone.
		return BucketInfo{}, fmt.Errorf("%w: status 400 (contract mismatch)", ErrInvalidResponse)

	case statusCode == http.StatusTooManyRequests || statusCode >= 500:
		return BucketInfo{}, fmt.Errorf("%w: status %d", ErrUnavailable, statusCode)

	default:
		// 401/403 and friends: the index is up but will not serve us. That is
		// an availability failure, not a denial of the bucket.
		return BucketInfo{}, fmt.Errorf("%w: status %d", ErrUnavailable, statusCode)
	}
}

// decodeErrorBody turns a contract error body into a typed error, falling back
// when the body is missing, unparseable or from a different contract version.
func decodeErrorBody(body []byte, fallback error) error {
	var payload ErrorResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return fallback
	}
	if payload.ContractVersion != ContractVersion || payload.Code == "" {
		return fallback
	}
	return ErrorForCode(payload.Code, payload.Message)
}

// isDefinitiveDenial reports whether the index answered "no" as opposed to
// failing to answer. Only definitive denials are cached as denials; everything
// else falls through to the grace window and then fails closed.
func isDefinitiveDenial(err error) bool {
	return errors.Is(err, ErrBucketNotFound) || errors.Is(err, ErrAccountNotFound)
}

// ---------------------------------------------------------------------------
// owner listing
// ---------------------------------------------------------------------------

type listOutcome struct {
	list OwnerBuckets
	err  error
}

// ListBuckets returns one page of the buckets an account owns.
//
// List answers are NOT cached, and the asymmetry with LookupBucket is the whole
// design:
//
//   - A stale bucket answer can only be wrong in the denying direction, and the
//     client bounds how stale it may be. That is why caching it is safe.
//   - A stale page is wrong in a direction nobody can see. A bucket created one
//     second ago is simply absent, and absence is indistinguishable from "this
//     account owns nothing" — the exact distinction this operation exists to
//     make. The user's recovery is to wait out a TTL they cannot observe, on
//     their first login, which is the worst possible moment to look broken.
//   - Nothing here would age into correctness. A cached bucket refreshes because
//     a caller asks about that bucket again; a cached page would refresh only
//     when someone asks about that owner again, and the thing that changed — a
//     new bucket — is precisely what the stale page denies knowledge of. There
//     is no invalidation signal on this side: buckets are created on chain.
//
// The cost of not caching is one round trip per enumeration, answered on the
// implementation side by a filter over data already resident in its memory.
// Enumeration is a control-plane call — a login, a dashboard render — not a
// data-path call, so paying a round trip to never show a wrong list is the right
// trade. If enumeration ever lands on a hot path, the fix is a TTL measured in
// seconds and an explicit "as of" timestamp in the response, not a silent hour.
//
// What does apply is coalescing: concurrent identical enumerations share one
// request, which flattens the burst a dashboard mounting three components
// produces. Coalescing shares an in-flight answer; unlike a cache it can never
// serve an aged one.
//
// The page is deliberately not used to prime the per-bucket cache either. It
// would save a round trip the caller does not need — every fact the single-bucket
// operation returns is already in the page — at the price of evicting up to a
// page's worth of hot authorization entries from a bounded LRU.
func (c *Client) ListBuckets(
	ctx context.Context,
	owner ed25519.PublicKey,
	query BucketsQuery,
) (OwnerBuckets, error) {
	if len(owner) != ed25519.PublicKeySize {
		return OwnerBuckets{}, fmt.Errorf(
			"%w: owner is %d bytes, want %d", ErrInvalidOwner, len(owner), ed25519.PublicKeySize)
	}

	path := BucketsRequestPath(owner, query)

	outcome, err := c.listFetches.do(ctx, path, c.requestTimeout,
		func(fetchCtx context.Context) listOutcome {
			list, listErr := c.doList(fetchCtx, owner, path)
			return listOutcome{list: list, err: listErr}
		})
	if err != nil {
		return OwnerBuckets{}, err
	}
	if outcome.err != nil {
		return OwnerBuckets{}, outcome.err
	}

	// Every caller that shared this fetch holds the same slices otherwise.
	return outcome.list.clone(), nil
}

func (c *Client) doList(ctx context.Context, owner ed25519.PublicKey, path string) (OwnerBuckets, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return OwnerBuckets{}, fmt.Errorf("%w: building request: %w", ErrUnavailable, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return OwnerBuckets{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxListResponseBytes))

	return c.interpretList(owner, resp.StatusCode, body, readErr)
}

func (c *Client) interpretList(
	owner ed25519.PublicKey,
	statusCode int,
	body []byte,
	readErr error,
) (OwnerBuckets, error) {
	switch {
	case statusCode == http.StatusOK:
		if readErr != nil {
			return OwnerBuckets{}, fmt.Errorf("%w: reading body: %w", ErrUnavailable, readErr)
		}
		var payload BucketsResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return OwnerBuckets{}, fmt.Errorf("%w: decoding body: %w", ErrInvalidResponse, err)
		}

		return payload.Decode(owner)

	case statusCode == http.StatusNotFound:
		// This operation has no "not found": an account the index has never seen
		// owns nothing, which is a 200 and an empty page. So a 404 means the
		// ROUTE is missing — an implementation that predates this operation, or a
		// base URL pointing at something else — and it must never be flattened
		// into an empty page, which would tell a user with buckets they have
		// none.
		return OwnerBuckets{}, fmt.Errorf(
			"%w: status 404 (owner listing not implemented at this url)", ErrUnavailable)

	case statusCode == http.StatusBadRequest:
		// Same rule as the single-bucket path, and for the same reason: a 400 is
		// never an answer, and the body's error CODE is never consulted. Here the
		// request may genuinely be at fault — a limit above the implementation's
		// maximum — but that is still not a statement about the account, and
		// keeping one rule instead of two is what makes the cached-false-denial
		// bug impossible to reintroduce on either path.
		return OwnerBuckets{}, fmt.Errorf(
			"%w: status 400 (request rejected)%s", ErrInvalidResponse, errorBodyDetail(body))

	case statusCode == http.StatusTooManyRequests || statusCode >= 500:
		return OwnerBuckets{}, fmt.Errorf("%w: status %d", ErrUnavailable, statusCode)

	default:
		// 401/403 and friends: the index is up but will not serve us. An
		// availability failure, not a statement about the account.
		return OwnerBuckets{}, fmt.Errorf("%w: status %d", ErrUnavailable, statusCode)
	}
}

// errorBodyDetail extracts the human-readable message from a contract error body
// and nothing else.
//
// It deliberately cannot see the error code. decodeErrorBody's code mapping is
// what turned contract drift into a cached denial once; a helper that only ever
// returns text cannot be misused the same way, however this file is edited later.
func errorBodyDetail(body []byte) string {
	var payload ErrorResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if payload.ContractVersion != ContractVersion || payload.Message == "" {
		return ""
	}

	return ": " + payload.Message
}

// ---------------------------------------------------------------------------
// owner-scoped delegates
// ---------------------------------------------------------------------------

type delegatesOutcome struct {
	// entry is the answer to cache and return; valid when err is nil.
	entry delegateEntry
	// err is an availability failure. This operation has no denial to carry.
	err error
}

// LookupDelegates returns the owner's delegates whose proxy grants are honored
// for DDC writes, as raw 32-byte keys — the accounts that may stand in for the
// owner. The bucket answer carries the same list (Lookup), but this question is
// older than the bucket: claim-time grant-wait asks it while the bucket it will
// bill does not exist yet, so it is scoped to the owner.
//
// Cache policy — LookupBucket's, keyed by owner instead of bucket id:
//
//  1. Younger than BalanceTTL — served from cache, no network.
//  2. Older — refetched. Success refreshes the entry, which is what makes a
//     revocation visible on the same cadence balance is re-read on; grants
//     change at human speed, so the short TTL is the right one for them.
//  3. Fetch failed and the index is unreachable — a cached answer is served
//     only within BalanceTTL+GraceWindow and OwnershipTTL. Past both, the
//     lookup fails closed.
//
// Failing closed means erroring, never returning an empty list. Empty is an
// answer — "no delegates" — and it is the exact answer a claim-time caller is
// polling for; producing it from an outage would end a wait the chain has not
// answered, which is an outage dressed as a revocation.
func (c *Client) LookupDelegates(ctx context.Context, owner ed25519.PublicKey) ([][]byte, error) {
	if len(owner) != ed25519.PublicKeySize {
		return nil, fmt.Errorf(
			"%w: owner is %d bytes, want %d", ErrInvalidOwner, len(owner), ed25519.PublicKeySize)
	}

	key := EncodeOwner(owner)

	if cached, ok := c.delegatesCache.get(key); ok && c.now().Sub(cached.fetchedAt) < c.balanceTTL {
		return cached.result(), nil
	}

	outcome := c.fetchDelegatesDeduped(ctx, owner, key)
	if outcome.err == nil {
		return outcome.entry.result(), nil
	}

	// The index did not answer. Re-read the cache: a concurrent lookup may
	// have refreshed the entry while this fetch was failing.
	if cached, ok := c.delegatesCache.get(key); ok {
		if c.withinGraceWindow(c.now().Sub(cached.fetchedAt)) {
			return cached.result(), nil
		}
	}

	// Nothing recent enough to stand in. Deny.
	return nil, outcome.err
}

// fetchDelegatesDeduped runs one fetch per owner per burst, keyed by the request
// path like the other operations.
func (c *Client) fetchDelegatesDeduped(
	ctx context.Context,
	owner ed25519.PublicKey,
	key string,
) delegatesOutcome {
	outcome, err := c.delegatesFetches.do(ctx, DelegatesRequestPath(owner), c.requestTimeout,
		func(fetchCtx context.Context) delegatesOutcome {
			return c.fetchAndStoreDelegates(fetchCtx, owner, key)
		})
	if err != nil {
		return delegatesOutcome{err: err}
	}

	return outcome
}

func (c *Client) fetchAndStoreDelegates(
	ctx context.Context,
	owner ed25519.PublicKey,
	key string,
) delegatesOutcome {
	delegates, err := c.doFetchDelegates(ctx, owner)
	if err != nil {
		// Nothing is cached on failure. This operation has no denial, so there
		// is no "definitive" half to store: every error is an availability
		// failure, and it is answered out of the grace window or not at all.
		return delegatesOutcome{err: err}
	}

	stored := delegateEntry{delegates: delegates, fetchedAt: c.now()}
	c.delegatesCache.put(key, stored)

	return delegatesOutcome{entry: stored}
}

// doFetchDelegates performs the HTTP call.
func (c *Client) doFetchDelegates(ctx context.Context, owner ed25519.PublicKey) ([][]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL+DelegatesRequestPath(owner), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: building request: %w", ErrUnavailable, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))

	return c.interpretDelegates(owner, resp.StatusCode, body, readErr)
}

func (c *Client) interpretDelegates(
	owner ed25519.PublicKey,
	statusCode int,
	body []byte,
	readErr error,
) ([][]byte, error) {
	switch {
	case statusCode == http.StatusOK:
		if readErr != nil {
			return nil, fmt.Errorf("%w: reading body: %w", ErrUnavailable, readErr)
		}
		var payload DelegatesResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("%w: decoding body: %w", ErrInvalidResponse, err)
		}

		return payload.Decode(owner)

	case statusCode == http.StatusNotFound:
		// This operation has no "not found", same as the owner listing: an
		// account the index has never heard of has no delegates, and that is a
		// 200 and an empty list. So a 404 means the ROUTE is missing — an
		// implementation that predates this operation, or a base URL pointing
		// at something else — and it must never be flattened into an empty
		// answer, which would end a claim-wait the chain has not settled.
		return nil, fmt.Errorf(
			"%w: status 404 (delegates lookup not implemented at this url)", ErrUnavailable)

	case statusCode == http.StatusBadRequest:
		// Same rule as the other two operations: a 400 is contract drift (the
		// client only ever sends a well-formed owner), the body's error CODE is
		// never consulted, and nothing about it is remembered.
		return nil, fmt.Errorf(
			"%w: status 400 (request rejected)%s", ErrInvalidResponse, errorBodyDetail(body))

	case statusCode == http.StatusTooManyRequests || statusCode >= 500:
		return nil, fmt.Errorf("%w: status %d", ErrUnavailable, statusCode)

	default:
		// 401/403 and friends: the index is up but will not serve us. An
		// availability failure, not a statement about the owner.
		return nil, fmt.Errorf("%w: status %d", ErrUnavailable, statusCode)
	}
}

// ---------------------------------------------------------------------------
// request coalescing
// ---------------------------------------------------------------------------
//
// Lives here rather than in its own file: sharedFetch is unexported, has exactly
// one consumer — the two fields on Client above — and a separate file for a
// helper with a single user in the same package is a split without a boundary.
// It is also not a utility: naming it one would invite unrelated things to join
// it, since "utils" constrains nothing about what belongs.

// sharedFetch coalesces concurrent identical fetches: the first caller for a key
// runs one, everyone who arrives while it is running waits on the same result.
// Without it a cold cache plus a burst of traffic for one key turns into a burst
// of identical requests at the index, which is exactly when it can least afford
// them.
//
// Keys are request paths, so "the same question" and "the same key" are the same
// statement by construction, and two different questions can never share an
// answer. R is whatever the fetch produces — the caching path carries a cache
// entry, the listing path carries a decoded page — and the zero value of
// sharedFetch is usable.
type sharedFetch[R any] struct {
	mu    sync.Mutex
	calls map[string]*fetchCall[R]
}

// fetchCall is one in-flight fetch, shared by every caller that asked the same
// question while it was running.
type fetchCall[R any] struct {
	done   chan struct{}
	result R
}

// do returns fetch's result for key, joining an identical in-flight fetch rather
// than starting a second one.
//
// The returned error is non-nil only when the CALLER's context ended before an
// answer arrived. A failure inside fetch travels in R, because only the caller
// knows what a failure means — the caching path, for one, answers some failures
// out of its cache.
//
// The shared fetch runs on a context detached from whichever caller happened to
// arrive first, bounded by timeout, and it runs in a goroutine so the leader
// waits on it exactly like a follower does.
//
// Two separate concerns, and an earlier version of this code conflated them by
// running the fetch synchronously on the leader's ctx:
//
//	the SHARED FETCH must not be cancellable by one caller. Using the leader's
//	ctx made every follower hostage to it — one caller with a short deadline,
//	or one that walked away, failed the lookup for everybody waiting on
//	call.done even though their own contexts were live. That is backwards
//	under the burst this dedupe exists to absorb: the more callers pile onto a
//	cold key, the likelier one of them cancels.
//
//	an INDIVIDUAL CALLER who cancels must return immediately, and must not be
//	made to wait out a fetch it no longer wants. Detaching the fetch without
//	also making the leader wait via select would have traded one bug for
//	another: a cancelled leader would block for the whole timeout.
//
// So the fetch is detached AND every caller — leader included — selects on
// call.done against its own ctx. A leader that walks away still leaves the fetch
// running, which is the point: it finishes for the followers it was sharing with.
//
// context.WithoutCancel keeps the leader's values (trace ids and anything a
// caller attached) while dropping its cancellation and deadline.
func (s *sharedFetch[R]) do(
	ctx context.Context,
	key string,
	timeout time.Duration,
	fetch func(context.Context) R,
) (R, error) {
	s.mu.Lock()
	if s.calls == nil {
		s.calls = make(map[string]*fetchCall[R])
	}
	if call, ok := s.calls[key]; ok {
		s.mu.Unlock()
		return call.await(ctx)
	}
	call := &fetchCall[R]{done: make(chan struct{})}
	s.calls[key] = call
	s.mu.Unlock()

	go func() {
		fetchCtx, cancelFetch := context.WithTimeout(context.WithoutCancel(ctx), timeout)
		defer cancelFetch()

		call.result = fetch(fetchCtx)

		// Removed from the map before done is closed, so a caller that arrives
		// in between starts a fresh fetch instead of joining a finished one.
		s.mu.Lock()
		delete(s.calls, key)
		s.mu.Unlock()
		close(call.done)
	}()

	return call.await(ctx)
}

func (c *fetchCall[R]) await(ctx context.Context) (R, error) {
	select {
	case <-c.done:
		return c.result, nil
	case <-ctx.Done():
		var zero R
		return zero, fmt.Errorf("%w: %w", ErrUnavailable, ctx.Err())
	}
}
