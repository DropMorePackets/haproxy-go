package encoding

import (
	"reflect"
	"testing"
)

func Test_encode(t *testing.T) {
	type args struct {
		val uint64
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "",
			args: args{
				val: 0x1234,
			},
			want: []byte{0xf4, 0x94, 0x01},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := make([]byte, 4)
			if n, _ := PutVarint(b, tt.args.val); !reflect.DeepEqual(b[:n], tt.want) {
				t.Errorf("encode() = %v, want %v", b[:n], tt.want)
			}

			if got, _, _ := Varint(tt.want); !reflect.DeepEqual(got, tt.args.val) {
				t.Errorf("decode() = %v, want %v", got, tt.args.val)
			}
		})
	}
}
