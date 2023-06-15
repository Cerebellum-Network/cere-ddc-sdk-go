package domain

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func Test_decodeHex(t *testing.T) {
	type args struct {
		src []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Decode string",
			args: args{
				src: func() []byte {
					s := []byte("SOME STRING")
					src := make([]byte, hex.EncodedLen(len(s)))
					hex.Encode(src, s)
					return append([]byte("0x"), src...)
				}(),
			},
			want:    []byte("SOME STRING"),
			wantErr: false,
		},
		{
			name: "Empty string",
			args: args{
				src: []byte("0x"),
			},
			want:    []byte{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeHex(tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeHex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeHex() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_encodeHex(t *testing.T) {
	type args struct {
		src []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Encode string",
			args: args{
				src: []byte("SOME STRING"),
			},
			want: func() []byte {
				s := []byte("SOME STRING")
				src := make([]byte, hex.EncodedLen(len(s)))
				hex.Encode(src, s)
				return append([]byte("0x"), src...)
			}(),
		},
		{
			name: "Empty string",
			args: args{
				src: []byte{},
			},
			want: []byte("0x"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeHex(tt.args.src); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encodeHex() = %v, want %v", got, tt.want)
			}
		})
	}
}
