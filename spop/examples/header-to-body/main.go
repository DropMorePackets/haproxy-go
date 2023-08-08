package main

import (
	"github.com/fionera/haproxy-go/pkg/newenc"
	"github.com/fionera/haproxy-go/pkg/stream"
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

func HandleSPOE(w *newenc.ActionWriter, m *stream.Message) {
	k := stream.AcquireKVEntry()
	defer stream.ReleaseKVEntry(k)

	for m.KV().Next(k) {
		if k.NameEquals("headers") {
			err := w.SetStringBytes(newenc.VarScopeTransaction, "body", k.ValueBytes())
			if err != nil {
				log.Printf("err: %v", err)
			}
		}
	}

	if m.KV().Error() != nil {
		log.Println(m.KV().Error())
	}
}
