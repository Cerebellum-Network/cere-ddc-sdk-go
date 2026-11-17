package cnsapi

import "github.com/cerebellum-network/cere-ddc-sdk-go/auth/pb"

// SetSignature sets the record's signature. It makes *Record a
// pb.SignableMessage, so auth.SignMessage and auth.VerifySignature work on it.
func (r *Record) SetSignature(s *pb.Signature) { r.Signature = s }

var _ pb.SignableMessage = (*Record)(nil)
