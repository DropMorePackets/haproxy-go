//go:build tools

package tools

import (
	_ "golang.org/x/tools/cmd/stringer"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
