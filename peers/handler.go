package peers

import (
	"context"

	"github.com/dropmorepackets/haproxy-go/peers/sticktable"
)

type Handler interface {
	HandleUpdate(context.Context, *sticktable.EntryUpdate)
	HandleHandshake(context.Context, *Handshake)
	Close() error
}

type HandlerFunc func(context.Context, *sticktable.EntryUpdate)

func (HandlerFunc) Close() error { return nil }

func (HandlerFunc) HandleHandshake(context.Context, *Handshake) {}

func (h HandlerFunc) HandleUpdate(ctx context.Context, u *sticktable.EntryUpdate) {
	h(ctx, u)
}

var (
	_ Handler = (HandlerFunc)(nil)
)
