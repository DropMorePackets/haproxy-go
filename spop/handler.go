package spop

import (
	"github.com/fionera/haproxy-go/pkg/encoding"
)

type Handler interface {
	HandleSPOE(*encoding.ActionWriter, *encoding.Message)
}

type HandlerFunc func(*encoding.ActionWriter, *encoding.Message)

func (h HandlerFunc) HandleSPOE(w *encoding.ActionWriter, m *encoding.Message) {
	h(w, m)
}
