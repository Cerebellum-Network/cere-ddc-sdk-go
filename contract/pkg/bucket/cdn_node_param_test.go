package bucket

import (
	"reflect"
	"testing"
)

func TestReadCDNNodeParams(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		wantP   CDNNodeParams
		wantErr bool
	}{
		{
			name: "Positive test",
			args: args{s: `{"url":"http://localhost:8080", "publicKey": "NodePublicKey", "size":1, "location":"US"}`},
			wantP: CDNNodeParams{
				Url:       "http://localhost:8080",
				Size:      1,
				Location:  "US",
				PublicKey: "NodePublicKey",
			},
			wantErr: false,
		},
		{
			name: "Positive test with size as string",
			args: args{s: `{"url":"http://localhost:8080", "publicKey": "NodePublicKey", "size":"1", "location":"US"}`},
			wantP: CDNNodeParams{
				Url:       "http://localhost:8080",
				Size:      1,
				Location:  "US",
				PublicKey: "NodePublicKey",
			},
			wantErr: false,
		},
		{
			name:    "Wrong structure",
			args:    args{s: `{"}`},
			wantP:   CDNNodeParams{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotP, err := ReadCDNNodeParams(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadCDNNodeParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotP, tt.wantP) {
				t.Errorf("ReadCDNNodeParams() gotP = %v, want %v", gotP, tt.wantP)
			}
		})
	}
}
