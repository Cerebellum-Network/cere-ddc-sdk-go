package pkg

import (
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	"github.com/vedhavyas/go-subkey"
	"github.com/vedhavyas/go-subkey/sr25519"
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
