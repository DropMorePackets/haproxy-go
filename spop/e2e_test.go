//go:build e2e

package spop

import (
	"context"
	"fmt"
	"github.com/fionera/haproxy-go/pkg/encoding"
	"github.com/fionera/haproxy-go/pkg/testutil"
	"net/http"
	"testing"
	"time"
)

func TestE2E(t *testing.T) {
	tests := []E2ETest{
		{
			name: "default",
			hf:   func(_ context.Context, w *encoding.ActionWriter, m *encoding.Message) {},
			tf: func(t *testing.T, config testutil.HAProxyConfig) {
				resp, err := http.Get("http://127.0.0.1:" + config.FrontendPort)
				if err != nil {
					t.Fatal(err)
				}

				if resp.StatusCode != http.StatusOK {
					t.Fatalf("expected %d; got %d", http.StatusOK, resp.StatusCode)
				}
			},
		},
		{
			name: "status-code acl",
			hf: func(_ context.Context, w *encoding.ActionWriter, m *encoding.Message) {
				err := w.SetInt64(encoding.VarScopeTransaction, "statuscode", http.StatusUnauthorized)
				if err != nil {
					t.Fatalf("writing status-code: %v", err)
				}
			},
			tf: func(t *testing.T, config testutil.HAProxyConfig) {
				resp, err := http.Get("http://127.0.0.1:" + config.FrontendPort)
				if err != nil {
					t.Fatal(err)
				}

				if resp.StatusCode != http.StatusUnauthorized {
					t.Fatalf("expected %d; got %d", http.StatusUnauthorized, resp.StatusCode)
				}
			},
			backendCfg: "http-request return status 401 if { var(txn.e2e.statuscode) -m int eq 401 }",
		},
		{
			name: "ctx cancel on disconnect",
			hf: func(ctx context.Context, w *encoding.ActionWriter, m *encoding.Message) {
				select {
				case <-ctx.Done():
				case <-time.After(5 * time.Second):
					panic("ctx not cancelled")
				}
			},
			tf: func(t *testing.T, config testutil.HAProxyConfig) {
				resp, err := http.Get("http://127.0.0.1:" + config.FrontendPort)
				if err != nil {
					t.Fatal(err)
				}

				if resp.StatusCode != http.StatusUnauthorized {
					t.Fatalf("expected %d; got %d", http.StatusUnauthorized, resp.StatusCode)
				}
			},
			backendCfg: "http-request return status 401 if { var(txn.e2e.error) -m found }",
		},
		{
			name: "recover from panic",
			hf: func(ctx context.Context, w *encoding.ActionWriter, m *encoding.Message) {
				panic("example panic")
			},
			tf: func(t *testing.T, config testutil.HAProxyConfig) {
				resp, err := http.Get("http://127.0.0.1:" + config.FrontendPort)
				if err != nil {
					t.Fatal(err)
				}

				if resp.StatusCode != http.StatusOK {
					t.Fatalf("expected %d; got %d", http.StatusOK, resp.StatusCode)
				}
			},
		},
	}

	t.Parallel()
	for _, test := range tests {
		t.Run(test.name, withSPOP(t, test.frontendCfg, test.backendCfg, test.hf, test.tf))
	}
}

type E2ETest struct {
	name        string
	hf          HandlerFunc
	tf          func(*testing.T, testutil.HAProxyConfig)
	frontendCfg string
	backendCfg  string
}

func withSPOP(t *testing.T, frontendCfg, backendCfg string, hf HandlerFunc, f func(*testing.T, testutil.HAProxyConfig)) func(t *testing.T) {
	a := Agent{Handler: hf}

	// create the listener synchronously to prevent a race
	l := testutil.TCPListener(t)
	// ignore errors as the listener will be closed by t.Cleanup
	go a.Serve(l)

	cfg := testutil.HAProxyConfig{
		EngineAddr:           l.Addr().String(),
		FrontendPort:         fmt.Sprintf("%d", testutil.TCPPort(t)),
		CustomFrontendConfig: frontendCfg,
		CustomBackendConfig:  backendCfg,
	}

	return testutil.WithHAProxy(cfg, func(t *testing.T) {
		f(t, cfg)
	})
}
