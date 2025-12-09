package encoding

import (
	"net/netip"
	"testing"
)

func TestKVScanner_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		target  interface{}
		wantErr bool
		check   func(t *testing.T, v interface{})
	}{
		{
			name: "basic types",
			data: func() []byte {
				buf := make([]byte, 1024)
				w := NewKVWriter(buf, 0)
				w.SetString("name", "test")
				w.SetInt32("age", 25)
				w.SetBool("active", true)
				return w.Bytes()
			}(),
			target: &struct {
				Name   string `spoe:"name"`
				Age    int32  `spoe:"age"`
				Active bool   `spoe:"active"`
			}{},
			wantErr: false,
			check: func(t *testing.T, v interface{}) {
				s := v.(*struct {
					Name   string `spoe:"name"`
					Age    int32  `spoe:"age"`
					Active bool   `spoe:"active"`
				})
				if s.Name != "test" {
					t.Errorf("Name = %q, want %q", s.Name, "test")
				}
				if s.Age != 25 {
					t.Errorf("Age = %d, want %d", s.Age, 25)
				}
				if !s.Active {
					t.Errorf("Active = %v, want %v", s.Active, true)
				}
			},
		},
		{
			name: "ip address",
			data: func() []byte {
				buf := make([]byte, 1024)
				w := NewKVWriter(buf, 0)
				addr := netip.MustParseAddr("192.168.1.1")
				w.SetAddr("ip", addr)
				return w.Bytes()
			}(),
			target: &struct {
				IP netip.Addr `spoe:"ip"`
			}{},
			wantErr: false,
			check: func(t *testing.T, v interface{}) {
				s := v.(*struct {
					IP netip.Addr `spoe:"ip"`
				})
				if s.IP.String() != "192.168.1.1" {
					t.Errorf("IP = %q, want %q", s.IP.String(), "192.168.1.1")
				}
			},
		},
		{
			name: "optional pointer field",
			data: func() []byte {
				buf := make([]byte, 1024)
				w := NewKVWriter(buf, 0)
				w.SetString("required", "value")
				// optional field not set
				return w.Bytes()
			}(),
			target: &struct {
				Required string  `spoe:"required"`
				Optional *string `spoe:"optional"`
			}{},
			wantErr: false,
			check: func(t *testing.T, v interface{}) {
				s := v.(*struct {
					Required string  `spoe:"required"`
					Optional *string `spoe:"optional"`
				})
				if s.Required != "value" {
					t.Errorf("Required = %q, want %q", s.Required, "value")
				}
				if s.Optional != nil {
					t.Errorf("Optional = %v, want nil", s.Optional)
				}
			},
		},
		{
			name: "unknown keys ignored",
			data: func() []byte {
				buf := make([]byte, 1024)
				w := NewKVWriter(buf, 0)
				w.SetString("known", "value")
				w.SetString("unknown", "ignored")
				return w.Bytes()
			}(),
			target: &struct {
				Known string `spoe:"known"`
			}{},
			wantErr: false,
			check: func(t *testing.T, v interface{}) {
				s := v.(*struct {
					Known string `spoe:"known"`
				})
				if s.Known != "value" {
					t.Errorf("Known = %q, want %q", s.Known, "value")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewKVScanner(tt.data, -1)
			err := scanner.Unmarshal(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, tt.target)
			}
		})
	}
}

func TestKVScanner_Unmarshal_Errors(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		target interface{}
	}{
		{
			name: "not a pointer",
			data: []byte{},
			target: struct {
				Name string `spoe:"name"`
			}{},
		},
		{
			name: "nil pointer",
			data: []byte{},
			target: (*struct {
				Name string `spoe:"name"`
			})(nil),
		},
		{
			name: "not a struct",
			data: []byte{},
			target: func() *int {
				v := 0
				return &v
			}(),
		},
		{
			name: "type mismatch",
			data: func() []byte {
				buf := make([]byte, 1024)
				w := NewKVWriter(buf, 0)
				w.SetString("name", "test")
				return w.Bytes()
			}(),
			target: &struct {
				Name int32 `spoe:"name"`
			}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewKVScanner(tt.data, -1)
			err := scanner.Unmarshal(tt.target)
			if err == nil {
				t.Errorf("Unmarshal() expected error, got nil")
			}
		})
	}
}

