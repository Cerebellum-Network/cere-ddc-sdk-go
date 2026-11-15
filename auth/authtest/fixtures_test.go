package authtest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cerebellum-network/cere-ddc-sdk-go/auth"
	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/authtest"
	"github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"
)

const bucket = uint64(42)

func TestFixturesAuthorize(t *testing.T) {
	ctx := context.Background()
	owner, delegate, subject, attacker := authtest.NewKey(t), authtest.NewKey(t), authtest.NewKey(t), authtest.NewKey(t)
	authorizer := auth.New(authtest.OwnedBy(owner, bucket))

	for name, encoded := range map[string]string{
		"owner token, base58": authtest.Encode58(t, authtest.OwnerToken(t, owner, bucket)),
		"owner token, base64": authtest.Encode64(t, authtest.OwnerToken(t, owner, bucket)),
		"delegated token":     authtest.Encode58(t, authtest.DelegatedToken(t, owner, delegate, subject.Pub, bucket)),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := authorizer.Authorize(ctx, encoded, pb.Operation_COMPUTE, bucket)
			require.NoError(t, err)
		})
	}

	t.Run("forgery", func(t *testing.T) {
		_, err := authorizer.Authorize(ctx, authtest.Encode58(t, authtest.OwnerToken(t, attacker, bucket)), pb.Operation_COMPUTE, bucket)
		require.ErrorIs(t, err, auth.ErrInvalidOwnership)
	})
	t.Run("unfunded", func(t *testing.T) {
		_, err := auth.New(authtest.Unfunded(owner, bucket)).Authorize(ctx, authtest.Encode58(t, authtest.OwnerToken(t, owner, bucket)), pb.Operation_COMPUTE, bucket)
		require.ErrorIs(t, err, auth.ErrInsufficientBalance)
	})
	t.Run("index down", func(t *testing.T) {
		_, err := auth.New(authtest.Down("indexer unreachable")).Authorize(ctx, authtest.Encode58(t, authtest.OwnerToken(t, owner, bucket)), pb.Operation_COMPUTE, bucket)
		require.Equal(t, auth.KindUnavailable, auth.KindOf(err))
	})
}
