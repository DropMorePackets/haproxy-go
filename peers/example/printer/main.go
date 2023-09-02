package main

import (
	"github.com/fionera/haproxy-go/peers"
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	err := peers.ListenAndServe(":21000", peers.HandlerFunc(func(u *peers.EntryUpdate) {
		log.Println(u.String())
	}))
	if err != nil {
		log.Fatal(err)
	}
}
