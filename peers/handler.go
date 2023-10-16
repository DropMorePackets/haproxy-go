package peers

import "github.com/dropmorepackets/haproxy-go/peers/sticktable"

type Handler interface {
	Update(*sticktable.EntryUpdate)
}

type HandlerFunc func(*sticktable.EntryUpdate)

func (h HandlerFunc) Update(u *sticktable.EntryUpdate) {
	h(u)
}
