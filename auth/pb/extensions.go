package pb

import (
	"errors"

	"google.golang.org/protobuf/proto"
)

var (
	// ErrSignatureEmpty is returned when verifying a message that carries no
	// signature.
	ErrSignatureEmpty = errors.New("signature is empty")
	// ErrSignatureNotEmpty is returned when signing a message that is already
	// signed.
	ErrSignatureNotEmpty = errors.New("signature is not empty")
)

// SignableMessage is a message whose Signature field covers the rest of it. The
// signature is computed over the message marshaled with Signature unset.
type SignableMessage interface {
	proto.Message
	GetSignature() *Signature
	SetSignature(s *Signature)
}

// SetSignature sets the token's signature. It makes *AuthToken a SignableMessage.
func (t *AuthToken) SetSignature(s *Signature) { t.Signature = s }

var _ SignableMessage = (*AuthToken)(nil)
