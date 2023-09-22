package main

import (
	"context"
	"github.com/fionera/haproxy-go/pkg/encoding"
	"github.com/fionera/haproxy-go/spop"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go http.ListenAndServe(":9001", nil)

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Fatal(spop.ListenAndServe(":9000", spop.HandlerFunc(HandleSPOE)))
}

func HandleSPOE(_ context.Context, w *encoding.ActionWriter, m *encoding.Message) {
	k := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(k)

	for m.KV.Next(k) {
		if k.NameEquals("headers") {
			err := w.SetStringBytes(encoding.VarScopeTransaction, "body", k.ValueBytes())
			if err != nil {
				log.Printf("err: %v", err)
			}
		}
	}

	if m.KV.Error() != nil {
		log.Println(m.KV.Error())
	}
}
