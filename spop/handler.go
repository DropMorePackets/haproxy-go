package spop

import (
	"context"
	"github.com/fionera/haproxy-go/pkg/encoding"
)

type Handler interface {
	HandleSPOE(context.Context, *encoding.ActionWriter, *encoding.Message)
}

type HandlerFunc func(context.Context, *encoding.ActionWriter, *encoding.Message)

func (h HandlerFunc) HandleSPOE(ctx context.Context, w *encoding.ActionWriter, m *encoding.Message) {
	h(ctx, w, m)
}
