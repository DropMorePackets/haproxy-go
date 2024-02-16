package main

import (
	"context"
	"log"

	"github.com/dropmorepackets/haproxy-go/peers"
	"github.com/dropmorepackets/haproxy-go/peers/sticktable"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	err := peers.ListenAndServe(":21000", peers.HandlerFunc(func(_ context.Context, u *sticktable.EntryUpdate) {
		log.Println(u.String())
	}))
	if err != nil {
		log.Fatal(err)
	}
}
