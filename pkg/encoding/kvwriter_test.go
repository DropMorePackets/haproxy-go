package encoding

import (
	"fmt"
	"github.com/fionera/haproxy-go/pkg/testutil"
	"testing"
)

func TestKVWriter(t *testing.T) {
	buf := make([]byte, 16386)

	const exampleKey, exampleValue = "key", "value"
	testutil.WithoutAllocations(func(t *testing.T) {
		aw := NewKVWriter(buf, 0)

		if err := aw.SetString(exampleKey, exampleValue); err != nil {
			t.Error(err)
		}

		buf = aw.Bytes()
	})(t)

	const expectedValue = "036b6579080576616c7565"
	if s := fmt.Sprintf("%x", buf); s != expectedValue {
		t.Errorf("result doesnt match golden string: %s != %s", expectedValue, s)
	}
}
