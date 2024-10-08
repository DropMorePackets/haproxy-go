package encoding

import (
	"fmt"
	"testing"

	"github.com/dropmorepackets/haproxy-go/pkg/testutil"
)

func TestKVWriter(t *testing.T) {
	buf := make([]byte, 16386)

	const exampleKey, exampleValue = "key", "value"
	testutil.WithoutAllocations(t, func() {
		aw := NewKVWriter(buf, 0)

		if err := aw.SetString(exampleKey, exampleValue); err != nil {
			t.Error(err)
		}

		buf = aw.Bytes()
	})

	const expectedValue = "036b6579080576616c7565"
	if s := fmt.Sprintf("%x", buf); s != expectedValue {
		t.Errorf("result doesnt match golden string: %s != %s", expectedValue, s)
	}
}
