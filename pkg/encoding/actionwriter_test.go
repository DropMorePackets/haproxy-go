package encoding

import (
	"fmt"
	"testing"

	"github.com/dropmorepackets/haproxy-go/pkg/testutil"
)

func TestActionWriter(t *testing.T) {
	buf := make([]byte, 16386)

	const exampleKey, exampleValue = "key", "value"
	testutil.WithoutAllocations(func(t *testing.T) {
		aw := NewActionWriter(buf, 0)

		if err := aw.SetString(VarScopeTransaction, exampleKey, exampleValue); err != nil {
			t.Error(err)
		}

		buf = aw.Bytes()
	})(t)

	const expectedValue = "010302036b6579080576616c7565"
	if s := fmt.Sprintf("%x", buf); s != expectedValue {
		t.Errorf("result doesnt match golden string: %s != %s", expectedValue, s)
	}
}
