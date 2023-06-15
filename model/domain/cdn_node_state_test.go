package domain

import (
	"reflect"
	"testing"
)

func TestNodeState_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		r       NodeState
		want    []byte
		wantErr bool
	}{
		{
			name:    "Marshal Grey",
			r:       Grey,
			want:    []byte(`"Grey"`),
			wantErr: false,
		},
		{
			name:    "Marshal Green",
			r:       Green,
			want:    []byte(`"Green"`),
			wantErr: false,
		},
		{
			name:    "Marshal Blue",
			r:       Blue,
			want:    []byte(`"Blue"`),
			wantErr: false,
		},
		{
			name:    "Marshal Red",
			r:       Red,
			want:    []byte(`"Red"`),
			wantErr: false,
		},
		{
			name:    "Marshal NA",
			r:       NA,
			want:    []byte(`"NA"`),
			wantErr: false,
		},
		{
			name:    "Marshal Invalid",
			r:       NodeState(100),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeState_String(t *testing.T) {
	tests := []struct {
		name string
		r    NodeState
		want string
	}{
		{
			name: "String Grey",
			r:    Grey,
			want: "Grey",
		},
		{
			name: "String Green",
			r:    Green,
			want: "Green",
		},
		{
			name: "String Blue",
			r:    Blue,
			want: "Blue",
		},
		{
			name: "String Red",
			r:    Red,
			want: "Red",
		},
		{
			name: "String NA",
			r:    NA,
			want: "NA",
		},
		{
			name: "String Invalid",
			r:    NodeState(100),
			want: "Invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeState_UnmarshalJSON(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		r       NodeState
		args    args
		wantErr bool
	}{
		{
			name:    "Unmarshal Grey",
			r:       Grey,
			args:    args{data: []byte(`"Grey"`)},
			wantErr: false,
		},
		{
			name:    "Unmarshal Green",
			r:       Green,
			args:    args{data: []byte(`"Green"`)},
			wantErr: false,
		},
		{
			name:    "Unmarshal Blue",
			r:       Blue,
			args:    args{data: []byte(`"Blue"`)},
			wantErr: false,
		},
		{
			name:    "Unmarshal Red",
			r:       Red,
			args:    args{data: []byte(`"Red"`)},
			wantErr: false,
		},
		{
			name:    "Unmarshal NA",
			r:       NA,
			args:    args{data: []byte(`"NA"`)},
			wantErr: false,
		},
		{
			name:    "Unmarshal Invalid",
			r:       NodeState(100),
			args:    args{data: []byte(`"Invalid"`)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.r.UnmarshalJSON(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeState_Validate(t *testing.T) {
	tests := []struct {
		name    string
		r       NodeState
		wantErr bool
	}{
		{
			name:    "Validate Grey",
			r:       Grey,
			wantErr: false,
		},
		{
			name:    "Validate Green",
			r:       Green,
			wantErr: false,
		},
		{
			name:    "Validate Blue",
			r:       Blue,
			wantErr: false,
		},
		{
			name:    "Validate Red",
			r:       Red,
			wantErr: false,
		},
		{
			name:    "Validate NA",
			r:       NA,
			wantErr: false,
		},
		{
			name:    "Validate Invalid",
			r:       NodeState(100),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
