package domain

import (
	"errors"
	"github.com/cerebellum-network/cere-ddc-sdk-go/core/pkg/crypto"
	"github.com/cerebellum-network/cere-ddc-sdk-go/model/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type StateSignature struct {
	Signature     []byte
	PublicKey     []byte
	Scheme        crypto.SchemeName
	MultiHashType uint64 //TODO: move to crypto
	Timestamp     uint64
}

var _ Protobufable = (*StateSignature)(nil)

func (ss *StateSignature) ToProto() *pb.StateSignature {
	return &pb.StateSignature{
		Value:         ss.Signature,
		Signer:        ss.PublicKey,
		Scheme:        string(ss.Scheme),
		MultiHashType: ss.MultiHashType,
		Timestamp:     ss.Timestamp,
	}
}

func (ss *StateSignature) ToDomain(pbSignature *pb.StateSignature) {
	ss.Scheme, _ = crypto.RedSchemeNameFromString(pbSignature.Scheme)
	ss.Signature = pbSignature.Value
	ss.PublicKey = pbSignature.Signer
	ss.Scheme = crypto.SchemeName(pbSignature.Scheme)
	ss.MultiHashType = pbSignature.MultiHashType
	ss.Timestamp = pbSignature.Timestamp
}

func (ss *StateSignature) MarshalProto() ([]byte, error) {
	return proto.Marshal(ss.ToProto())
}

func (ss *StateSignature) UnmarshalProto(signatureAsBytes []byte) error {
	signature := &pb.StateSignature{}
	if err := proto.Unmarshal(signatureAsBytes, signature); err != nil {
		return err
	}

	ss.ToDomain(signature)
	return nil
}

func (ss *StateSignature) DecodedSignature() ([]byte, error) {
	if len(ss.Signature) > 64 {
		signature, err := decodeHex(ss.Signature)
		if err != nil {
			return nil, errors.New("unable to decode hex signature")
		}

		return signature, nil
	}

	return ss.Signature, nil
}

func (ss *StateSignature) DecodedPublicKey() ([]byte, error) {
	if len(ss.PublicKey) > 32 {
		signer, err := decodeHex(ss.PublicKey)
		if err != nil {
			return nil, errors.New("unable to decode hex public key")
		}

		return signer, nil
	}

	return ss.PublicKey, nil
}

func (ss *StateSignature) verify(body []byte) bool {
	pubKey, err := decodeHex(ss.PublicKey)
	if err != nil {
		return false
	}
	sig, err := decodeHex(ss.Signature)
	if err != nil {
		return false
	}
	v, err := crypto.Verify(ss.Scheme, pubKey, body, sig)
	if err != nil {
		log.WithError(err).Warning("Failed validate state signature")
		return false
	}
	return v
}
