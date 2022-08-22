package pkg

import (
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	"github.com/vedhavyas/go-subkey"
	"github.com/vedhavyas/go-subkey/sr25519"
	"golang.org/x/crypto/blake2b"
)

const substrateNetwork = uint8(42)

func KeyringPairFromSecret(seedOrPhrase string) (signature.KeyringPair, error) {
	scheme := sr25519.Scheme{}
	kyr, err := subkey.DeriveKeyPair(scheme, seedOrPhrase)
	if err != nil {
		return signature.KeyringPair{}, err
	}

	ss58Address, err := kyr.SS58Address(substrateNetwork)
	if err != nil {
		return signature.KeyringPair{}, err
	}

	var pk = kyr.Public()

	return signature.KeyringPair{
		URI:       seedOrPhrase,
		Address:   ss58Address,
		PublicKey: pk,
	}, nil
}

func Sign(data []byte, privateKeyURI string) ([]byte, error) {
	if len(data) > 256 {
		h := blake2b.Sum256(data)
		data = h[:]
	}

	scheme := sr25519.Scheme{}
	kyr, err := subkey.DeriveKeyPair(scheme, privateKeyURI)
	if err != nil {
		return nil, err
	}

	s, err := kyr.Sign(data)
	if err != nil {
		return nil, err
	}

	return s, nil
}
