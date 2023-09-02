//go:build e2e

package peers

import (
	"fmt"
	"github.com/fionera/haproxy-go/pkg/testutil"
	"log"
	"net/http"
	"testing"
)

func TestE2E(t *testing.T) {
	var success bool
	a := Peer{Handler: HandlerFunc(func(u *EntryUpdate) {
		success = true
		log.Println(u)
	})}

	// create the listener synchronously to prevent a race
	l := testutil.TCPListener(t)
	// ignore errors as the listener will be closed by t.Cleanup
	go a.Serve(l)

	cfg := testutil.HAProxyConfig{
		FrontendPort: fmt.Sprintf("%d", testutil.TCPPort(t)),
		CustomFrontendConfig: `
	http-request track-sc0 src table st_src_global
    http-request track-sc2 req.hdr(Host) table st_be_name
`,
		CustomConfig: `
backend st_be_name
    stick-table  type string  size 1m  expire 10m  store http_req_rate(10s) peers mypeers

backend st_src_global
	stick-table type ip size 1m expire 10m store http_req_rate(10s),conn_rate(10s),bytes_in_rate(10s) peers mypeers
`,
		PeerAddr: l.Addr().String(),
	}

	t.Run("foo", testutil.WithHAProxy(cfg, func(t *testing.T) {
		for i := 0; i < 10; i++ {
			_, err := http.Get("http://127.0.0.1:" + cfg.FrontendPort)
			if err != nil {
				t.Fatal(err)
			}
		}

		if !success {
			t.Fail()
		}
	}))
}
