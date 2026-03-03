// push is an example that demonstrates how to push stick table entries
// to HAProxy over an existing peer connection. When HAProxy connects to
// this peer, the handler uses WriterFromContext to obtain a Writer and
// sends a table definition followed by entry updates.
package main

import (
	"context"
	"log"
	"net/netip"

	"github.com/dropmorepackets/haproxy-go/peers"
	"github.com/dropmorepackets/haproxy-go/peers/sticktable"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	err := peers.ListenAndServe(":21000", peers.HandlerFunc(func(ctx context.Context, u *sticktable.EntryUpdate) {
		log.Println("received:", u.String())

		// Get the writer for this connection to push entries back.
		w := peers.WriterFromContext(ctx)

		// Define the stick table we want to push to.
		// Matches: stick-table type ip size 200k expire 5m store gpc0 peers local-peers
		tableDef := &sticktable.Definition{
			StickTableID: 0,
			Name:         "my_blocklist",
			KeyType:      sticktable.KeyTypeIPv4Address,
			KeyLength:    4,
			DataTypes: []sticktable.DataTypeDefinition{
				{DataType: sticktable.DataTypeGPC0},
			},
			Expiry: 300000, // 5 minutes in ms
		}

		if err := w.SendTableDefinition(tableDef); err != nil {
			log.Printf("error sending table definition: %v", err)
			return
		}

		// Push an entry marking an IP as blocked (gpc0 = 1).
		key := sticktable.IPv4AddressKey(netip.MustParseAddr("10.0.0.1"))
		gpc0 := sticktable.UnsignedIntegerData(1)
		entry := &sticktable.EntryUpdate{
			StickTable: tableDef,
			Key:        &key,
			Data:       []sticktable.MapData{&gpc0},
		}

		if err := w.SendEntry(entry); err != nil {
			log.Printf("error sending entry: %v", err)
			return
		}

		log.Println("pushed blocklist entry for 10.0.0.1")
	}))
	if err != nil {
		log.Fatal(err)
	}
}
