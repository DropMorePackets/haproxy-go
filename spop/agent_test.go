package spop

import "testing"

func TestAgentConfiguredMaxFrameSize(t *testing.T) {
	tests := []struct {
		name    string
		value   uint32
		want    uint32
		wantErr bool
	}{
		{name: "default", want: DefaultMaxFrameSize},
		{name: "protocol minimum", value: minFrameSize, want: minFrameSize},
		{name: "large frame", value: 262140, want: 262140},
		{name: "below protocol minimum", value: minFrameSize - 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Agent{MaxFrameSize: tt.value}
			got, err := a.configuredMaxFrameSize()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("configured max frame size: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}
