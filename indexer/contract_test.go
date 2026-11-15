package indexer

import (
	"crypto/ed25519"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucketPath(t *testing.T) {
	assert.Equal(t, "/v1/buckets/1337", BucketPath(1337))
	assert.True(t, strings.HasPrefix(BucketPath(1), BucketsPath))
}

func TestEncodeDecodeOwner(t *testing.T) {
	owner := make(ed25519.PublicKey, ed25519.PublicKeySize)
	for i := range owner {
		owner[i] = byte(i)
	}

	encoded := EncodeOwner(owner)
	assert.True(t, strings.HasPrefix(encoded, "0x"))
	assert.Len(t, encoded, 2+2*ed25519.PublicKeySize)

	decoded, err := DecodeOwner(encoded)
	require.NoError(t, err)
	assert.Equal(t, owner, decoded)
}

func TestDecodeOwnerRejectsMalformed(t *testing.T) {
	for _, tt := range []struct {
		name    string
		encoded string
	}{
		{name: "not hex", encoded: "0xzz"},
		{name: "too short", encoded: "0x0102"},
		{name: "too long", encoded: EncodeOwner(make([]byte, 33))},
		{name: "empty", encoded: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeOwner(tt.encoded)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidResponse)
		})
	}
}

func TestBucketResponseDecode(t *testing.T) {
	owner := EncodeOwner(make([]byte, ed25519.PublicKeySize))

	t.Run("valid", func(t *testing.T) {
		info, err := BucketResponse{
			ContractVersion: ContractVersion,
			BucketID:        7,
			Owner:           owner,
			HasBalance:      true,
			IsRemoved:       false,
		}.Decode(7)
		require.NoError(t, err)
		assert.Equal(t, uint64(7), info.BucketID)
		assert.True(t, info.HasBalance)
		assert.Len(t, info.Owner, ed25519.PublicKeySize)
	})

	t.Run("wrong contract version", func(t *testing.T) {
		_, err := BucketResponse{ContractVersion: ContractVersion + 1, BucketID: 7, Owner: owner}.Decode(7)
		assert.ErrorIs(t, err, ErrInvalidResponse)
	})

	t.Run("wrong bucket", func(t *testing.T) {
		_, err := BucketResponse{ContractVersion: ContractVersion, BucketID: 8, Owner: owner}.Decode(7)
		assert.ErrorIs(t, err, ErrInvalidResponse)
	})
}

func TestErrorForCode(t *testing.T) {
	assert.ErrorIs(t, ErrorForCode(CodeBucketNotFound, ""), ErrBucketNotFound)
	assert.ErrorIs(t, ErrorForCode(CodeAccountNotFound, "no ledger"), ErrAccountNotFound)
	assert.ErrorIs(t, ErrorForCode(CodeInvalidBucketID, ""), ErrBucketNotFound)
	assert.ErrorIs(t, ErrorForCode(CodeUnavailable, ""), ErrUnavailable)

	// An unknown code must never read as a denial: a future implementation
	// growing a code cannot be allowed to silently mean "no such bucket".
	assert.ErrorIs(t, ErrorForCode("something_new", ""), ErrUnavailable)
	assert.Contains(t, ErrorForCode(CodeAccountNotFound, "no ledger").Error(), "no ledger")
}

// ---------------------------------------------------------------------------
// owner listing
// ---------------------------------------------------------------------------

// TestBucketsRequestPathIsTheFilteredCollection pins the literal target. If
// it moves, every deployed client gets a 404 — and a 404 on this operation is
// indistinguishable from a node that predates it, so the breakage reads as "your
// node is old" instead of "the path changed".
//
// It also pins the SHAPE: the listing is the bucket collection with a required
// filter, not a subresource of an /owners root. There is no owner resource here,
// and inventing a collection root to carry one child is a resource the contract
// does not have.
func TestBucketsRequestPathIsTheFilteredCollection(t *testing.T) {
	owner := make(ed25519.PublicKey, ed25519.PublicKeySize)

	assert.Equal(t, "/v1/buckets?owner="+EncodeOwner(owner),
		BucketsRequestPath(owner, BucketsQuery{}))
	assert.NotContains(t, BucketsRequestPath(owner, BucketsQuery{}), "/owners")

	// The item lookup keeps its own path, which is why filtering the collection
	// cannot change what it returns.
	assert.Equal(t, "/v1/buckets/42", BucketPath(42))
}

// TestBucketsRequestPathIsDeterministic guards the client's coalescing key.
// Two callers asking the same question must render the same string, or they each
// issue their own request; two callers asking different questions must not, or one
// gets the other's page.
func TestBucketsRequestPathIsDeterministic(t *testing.T) {
	owner := make(ed25519.PublicKey, ed25519.PublicKeySize)
	full := BucketsQuery{After: 7, Limit: 10, IncludeRemoved: true}

	assert.Equal(t, BucketsRequestPath(owner, full), BucketsRequestPath(owner, full))
	assert.Equal(t, "after=7&includeRemoved=true&limit=10&owner="+EncodeOwner(owner),
		strings.SplitN(BucketsRequestPath(owner, full), "?", 2)[1])

	// Defaults are omitted, so a caller that spells "give me the default" as a
	// zero value shares a key with one that says nothing at all. Owner is always
	// present — it is required — so the query string never disappears entirely.
	assert.Equal(t, "owner="+EncodeOwner(owner),
		strings.SplitN(BucketsRequestPath(owner, BucketsQuery{}), "?", 2)[1])

	for _, tt := range []struct {
		name  string
		query BucketsQuery
	}{
		{name: "after", query: BucketsQuery{After: 1}},
		{name: "limit", query: BucketsQuery{Limit: 1}},
		{name: "includeRemoved", query: BucketsQuery{IncludeRemoved: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEqual(t, BucketsRequestPath(owner, BucketsQuery{}),
				BucketsRequestPath(owner, tt.query),
				"a different question must not share a coalescing key with the default one")
		})
	}
}

func testListOwner(b byte) ed25519.PublicKey {
	owner := make(ed25519.PublicKey, ed25519.PublicKeySize)
	for i := range owner {
		owner[i] = b
	}

	return owner
}

func testListElement(bucketID uint64, owner ed25519.PublicKey) BucketResponse {
	return BucketResponse{
		ContractVersion: ContractVersion,
		BucketID:        bucketID,
		Owner:           EncodeOwner(owner),
		HasBalance:      true,
	}
}

func TestBucketsResponseDecode(t *testing.T) {
	owner := testListOwner(0x11)

	t.Run("valid page", func(t *testing.T) {
		list, err := BucketsResponse{
			ContractVersion: ContractVersion,
			Owner:           EncodeOwner(owner),
			Buckets: []BucketResponse{
				testListElement(4, owner),
				testListElement(9, owner),
			},
			NextAfter: 9,
		}.Decode(owner)
		require.NoError(t, err)

		assert.Equal(t, []uint64{4, 9}, []uint64{list.Buckets[0].BucketID, list.Buckets[1].BucketID})
		assert.True(t, list.HasMore())
		assert.Equal(t, owner, ed25519.PublicKey(list.Owner))
	})

	// An account that owns nothing is the single most important non-error in this
	// contract: if it ever became an error, a brand-new developer's first login
	// would be indistinguishable from a broken index.
	t.Run("empty page is not an error", func(t *testing.T) {
		list, err := BucketsResponse{
			ContractVersion: ContractVersion,
			Owner:           EncodeOwner(owner),
			Buckets:         []BucketResponse{},
		}.Decode(owner)
		require.NoError(t, err)
		assert.Empty(t, list.Buckets)
		assert.False(t, list.HasMore())
	})
}

func TestBucketsResponseDecodeRejects(t *testing.T) {
	owner := testListOwner(0x11)
	other := testListOwner(0x22)

	for _, tt := range []struct {
		name string
		// why says what breaks in production if this stops being rejected.
		why      string
		response BucketsResponse
	}{
		{
			name: "wrong contract version",
			why:  "fields would be read under the wrong meaning",
			response: BucketsResponse{
				ContractVersion: ContractVersion + 1,
				Owner:           EncodeOwner(owner),
			},
		},
		{
			name: "envelope answers about another account",
			why:  "another account's page would be shown as this user's",
			response: BucketsResponse{
				ContractVersion: ContractVersion,
				Owner:           EncodeOwner(other),
			},
		},
		{
			name: "element owned by another account",
			why:  "an implementation that filtered on the wrong field would leak bucket ids across accounts",
			response: BucketsResponse{
				ContractVersion: ContractVersion,
				Owner:           EncodeOwner(owner),
				Buckets:         []BucketResponse{testListElement(1, other)},
			},
		},
		{
			name: "ids not ascending",
			why:  "paging with the last id seen would silently skip every smaller id",
			response: BucketsResponse{
				ContractVersion: ContractVersion,
				Owner:           EncodeOwner(owner),
				Buckets:         []BucketResponse{testListElement(9, owner), testListElement(4, owner)},
			},
		},
		{
			name: "duplicate id",
			why:  "a duplicated bucket would be rendered twice and counted twice",
			response: BucketsResponse{
				ContractVersion: ContractVersion,
				Owner:           EncodeOwner(owner),
				Buckets:         []BucketResponse{testListElement(4, owner), testListElement(4, owner)},
			},
		},
		{
			name: "bucket id zero",
			why:  "zero is never a bucket, and it is also the end-of-pages marker",
			response: BucketsResponse{
				ContractVersion: ContractVersion,
				Owner:           EncodeOwner(owner),
				Buckets:         []BucketResponse{testListElement(0, owner)},
			},
		},
		{
			name: "nextAfter is not the last id",
			why:  "any other cursor value can skip buckets between the pages",
			response: BucketsResponse{
				ContractVersion: ContractVersion,
				Owner:           EncodeOwner(owner),
				Buckets:         []BucketResponse{testListElement(4, owner)},
				NextAfter:       99,
			},
		},
		{
			name: "nextAfter on an empty page",
			why:  "a cursor into nothing would make a caller page forever",
			response: BucketsResponse{
				ContractVersion: ContractVersion,
				Owner:           EncodeOwner(owner),
				NextAfter:       4,
			},
		},
		{
			name: "element with a mismatched contract version",
			why:  "the envelope's version would vouch for elements it does not describe",
			response: BucketsResponse{
				ContractVersion: ContractVersion,
				Owner:           EncodeOwner(owner),
				Buckets: []BucketResponse{{
					ContractVersion: ContractVersion + 1,
					BucketID:        4,
					Owner:           EncodeOwner(owner),
				}},
			},
		},
		{
			name: "unparseable owner",
			why:  "a non-key would be carried as an account id",
			response: BucketsResponse{
				ContractVersion: ContractVersion,
				Owner:           "0xnothex",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.response.Decode(owner)
			require.Error(t, err, tt.why)
			assert.ErrorIs(t, err, ErrInvalidResponse, tt.why)
		})
	}
}

// TestErrorForCodeNeverDeniesTheOwnerListing pins that the listing's request-side
// codes stay availability failures. If either mapped onto a denial, a caller could
// read "your request was malformed" as "you own nothing".
func TestErrorForCodeNeverDeniesTheOwnerListing(t *testing.T) {
	for _, code := range []string{CodeInvalidOwner, CodeInvalidPagination} {
		t.Run(code, func(t *testing.T) {
			err := ErrorForCode(code, "bad request")
			assert.ErrorIs(t, err, ErrUnavailable)
			assert.NotErrorIs(t, err, ErrBucketNotFound)
			assert.NotErrorIs(t, err, ErrAccountNotFound)
		})
	}
}

// TestOwnerBucketsCloneDoesNotAlias is the unit form of the aliasing hazard: every
// caller sharing one coalesced fetch is handed the same page, so one of them
// mutating its own copy in place must not change anyone else's.
func TestOwnerBucketsCloneDoesNotAlias(t *testing.T) {
	owner := testListOwner(0x11)

	list, err := BucketsResponse{
		ContractVersion: ContractVersion,
		Owner:           EncodeOwner(owner),
		Buckets:         []BucketResponse{testListElement(4, owner)},
	}.Decode(owner)
	require.NoError(t, err)

	cloned := list.clone()
	cloned.Owner[0] ^= 0xff
	cloned.Buckets[0].Owner[0] ^= 0xff
	cloned.Buckets[0].BucketID = 77

	assert.Equal(t, owner, ed25519.PublicKey(list.Owner))
	assert.Equal(t, owner, ed25519.PublicKey(list.Buckets[0].Owner))
	assert.Equal(t, uint64(4), list.Buckets[0].BucketID)
}

// TestOwnerBucketsCloneKeepsAnEmptyPageEmpty: an absent page and an empty page mean
// the same thing here, and cloning must not silently turn one into a non-nil slice
// a caller would then append to.
func TestOwnerBucketsCloneKeepsAnEmptyPageEmpty(t *testing.T) {
	cloned := OwnerBuckets{Owner: testListOwner(0x11)}.clone()
	assert.Nil(t, cloned.Buckets)
	assert.False(t, cloned.HasMore())
}

// ---------------------------------------------------------------------------
// delegates
// ---------------------------------------------------------------------------

// TestBucketResponseDecodeToleratesAbsentDelegates is the rollout shape: every
// pre-upgrade server omits the key, and an absent delegates list must decode as
// empty — empty means no delegates, and no delegates means denied. Requiring the
// key would take every old server offline for consumers that had never asked
// about delegation.
func TestBucketResponseDecodeToleratesAbsentDelegates(t *testing.T) {
	// v1 body without "delegates" — the shape every pre-upgrade server emits.
	var r BucketResponse
	if err := json.Unmarshal([]byte(`{"contractVersion":1,"bucketId":7,"owner":"0x`+strings.Repeat("11", 32)+`","hasBalance":true,"isRemoved":false}`), &r); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(r.Delegates) != 0 {
		t.Fatalf("absent delegates must decode empty, got %v", r.Delegates)
	}

	info, err := r.Decode(7)
	require.NoError(t, err)
	assert.Empty(t, info.Delegates, "an absent key must survive Decode as no delegates")
}

// TestBucketResponseDecodeDelegates: a body carrying the delegates array decodes
// into raw keys, in the order the index holds them. Order is preserved rather
// than canonicalised so the decoded answer stays comparable with what the index
// serves.
func TestBucketResponseDecodeDelegates(t *testing.T) {
	first := testListOwner(0xAA)
	second := testListOwner(0xBB)

	body := `{"contractVersion":1,"bucketId":7,"owner":"0x` + strings.Repeat("11", 32) + `",` +
		`"hasBalance":true,"isRemoved":false,` +
		`"delegates":["` + EncodeOwner(first) + `","` + EncodeOwner(second) + `"]}`

	var r BucketResponse
	require.NoError(t, json.Unmarshal([]byte(body), &r))

	info, err := r.Decode(7)
	require.NoError(t, err)
	require.Len(t, info.Delegates, 2)
	assert.Equal(t, first, ed25519.PublicKey(info.Delegates[0]), "decodes in order")
	assert.Equal(t, second, ed25519.PublicKey(info.Delegates[1]))
}

// TestBucketResponseDecodeRejectsMalformedDelegate: a delegate entry that is not
// a 32-byte hex key fails the whole decode rather than being dropped. Dropping
// would deny rather than allow, but a malformed key means the body is not from
// this contract, and a partial answer from a foreign shape is worse than none.
func TestBucketResponseDecodeRejectsMalformedDelegate(t *testing.T) {
	owner := testListOwner(0x11)

	_, err := BucketResponse{
		ContractVersion: ContractVersion,
		BucketID:        7,
		Owner:           EncodeOwner(owner),
		Delegates:       []string{EncodeOwner(testListOwner(0xAA)), "0xnothex"},
	}.Decode(7)
	assert.ErrorIs(t, err, ErrInvalidResponse)
}

// TestDelegatesRequestPathIsOwnerScoped pins the literal target, for the same
// reason the listing's path is pinned: a client and an implementation that
// disagree here see a 404, and on this operation a 404 looks exactly like a node
// too old to serve it. It also pins the SHAPE — a ?owner= filter on a top-level
// collection, not a subresource of /v1/buckets — because the question "who may
// act for this owner" is asked before any bucket exists to hang it off.
func TestDelegatesRequestPathIsOwnerScoped(t *testing.T) {
	owner := testListOwner(0x11)

	assert.Equal(t, "/v1/delegates?owner="+EncodeOwner(owner), DelegatesRequestPath(owner))
	assert.Equal(t, DelegatesRequestPath(owner), DelegatesRequestPath(owner),
		"the coalescing key must be deterministic")
}

// TestDelegatesResponseRoundTrip walks the owner-scoped answer end to end at the
// contract level: a well-formed body decodes into raw keys in order, the owner
// echo is enforced, and an absent delegates key tolerates as empty (a v1
// implementation that predates the field). The request-side half — a missing
// owner is a 400 CodeInvalidOwner — is pinned against the real handler in the
// router package.
func TestDelegatesResponseRoundTrip(t *testing.T) {
	owner := testListOwner(0x11)
	first := testListOwner(0xAA)
	second := testListOwner(0xBB)

	encoded, err := json.Marshal(DelegatesResponse{
		ContractVersion: ContractVersion,
		Owner:           EncodeOwner(owner),
		Delegates:       []string{EncodeOwner(first), EncodeOwner(second)},
	})
	require.NoError(t, err)

	var decoded DelegatesResponse
	require.NoError(t, json.Unmarshal(encoded, &decoded))

	delegates, err := decoded.Decode(owner)
	require.NoError(t, err)
	require.Len(t, delegates, 2)
	assert.Equal(t, first, ed25519.PublicKey(delegates[0]))
	assert.Equal(t, second, ed25519.PublicKey(delegates[1]))

	t.Run("absent delegates decodes empty", func(t *testing.T) {
		var decoded DelegatesResponse
		require.NoError(t, json.Unmarshal([]byte(`{"contractVersion":1,"owner":"`+EncodeOwner(owner)+`"}`), &decoded))

		delegates, err := decoded.Decode(owner)
		require.NoError(t, err, "no delegates is an answer, not an error")
		assert.Empty(t, delegates)
	})

	t.Run("owner echo mismatch is rejected", func(t *testing.T) {
		_, err := DelegatesResponse{
			ContractVersion: ContractVersion,
			Owner:           EncodeOwner(testListOwner(0x22)),
		}.Decode(owner)
		assert.ErrorIs(t, err, ErrInvalidResponse,
			"another account's delegates must not be attributed to this one")
	})

	t.Run("wrong contract version is rejected", func(t *testing.T) {
		_, err := DelegatesResponse{
			ContractVersion: ContractVersion + 1,
			Owner:           EncodeOwner(owner),
		}.Decode(owner)
		assert.ErrorIs(t, err, ErrInvalidResponse)
	})

	t.Run("malformed delegate hex is rejected", func(t *testing.T) {
		_, err := DelegatesResponse{
			ContractVersion: ContractVersion,
			Owner:           EncodeOwner(owner),
			Delegates:       []string{"0x0102"},
		}.Decode(owner)
		assert.ErrorIs(t, err, ErrInvalidResponse)
	})
}
