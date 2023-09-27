package crypto

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

const (
	content               = "Hello world!"
	address               = "5FJDBC3jJbWX48PyfpRCo7pKsFwSy4Mzj5t39PfXixD5jMgy"
	addressForCereNetwork = "6Sk8H6YZv61EvqFYYCyVKw1Rd5eydFvEzNXcXbP9HDLw7E4W"
	pubKeyHex             = "0x8f01969eb5244d853cc9c6ad73c46d8a1a091842c414cabd2377531f0832635f"
)

func TestCreateScheme(t *testing.T) {
	type args struct {
		schemeName SchemeName
		seed       string
	}
	tests := []struct {
		name    string
		args    args
		want    Scheme
		wantErr error
	}{
		{
			"Wrong scheme",
			args{
				schemeName: "SuperDuperCrupt",
				seed:       "0x0029ffc486837f4d7159837fdcdffef0c4283e4ae77af25a4ea1d76ab38bbb5a",
			},
			nil,
			ErrSchemeNotExist,
		},
		{
			"SR25519",
			args{
				schemeName: "sr25519",
				seed:       "0x0029ffc486837f4d7159837fdcdffef0c4283e4ae77af25a4ea1d76ab38bbb5a",
			},
			func(seed string) Scheme {
				s, _ := createSr25519SchemeFromString(seed)
				return s
			}("0x0029ffc486837f4d7159837fdcdffef0c4283e4ae77af25a4ea1d76ab38bbb5a"),
			nil,
		},
		{
			"Default",
			args{
				schemeName: "",
				seed:       "0x0029ffc486837f4d7159837fdcdffef0c4283e4ae77af25a4ea1d76ab38bbb5a",
			},
			func(seed string) Scheme {
				s, _ := createSr25519SchemeFromString(seed)
				return s
			}("0x0029ffc486837f4d7159837fdcdffef0c4283e4ae77af25a4ea1d76ab38bbb5a"),
			nil,
		},
		{
			"ED25519",
			args{
				schemeName: "ed25519",
				seed:       "0x0029ffc486837f4d7159837fdcdffef0c4283e4ae77af25a4ea1d76ab38bbb5a",
			},
			func(seed string) Scheme {
				s, _ := createEd25519SchemeFromString(seed)
				return s
			}("0x0029ffc486837f4d7159837fdcdffef0c4283e4ae77af25a4ea1d76ab38bbb5a"),
			nil,
		},
		{
			"Secp256k1",
			args{
				schemeName: "secp256k1",
				seed:       "0x0029ffc486837f4d7159837fdcdffef0c4283e4ae77af25a4ea1d76ab38bbb5a",
			},
			func(seed string) Scheme {
				pk, _ := hex.DecodeString(strings.TrimPrefix(seed, "0x"))
				s, _ := createSecp256k1Scheme(pk)
				return s
			}("0x0029ffc486837f4d7159837fdcdffef0c4283e4ae77af25a4ea1d76ab38bbb5a"),
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateScheme(tt.args.schemeName, tt.args.seed)
			assert.Equal(t, tt.wantErr, err, "CreateScheme(%v, %v)", tt.args.schemeName, tt.args.seed)
			assert.Equalf(t, tt.want, got, "CreateScheme(%v, %v)", tt.args.schemeName, tt.args.seed)
		})
	}
}
