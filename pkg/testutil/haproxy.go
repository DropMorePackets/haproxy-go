package testutil

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"testing"
	"text/template"
	"time"
)

const haproxyConfigTemplate = `
global
	log stdout format short daemon
    stats socket {{ .StatsSocket }} mode 660 level admin expose-fd listeners
    stats timeout 30s

defaults
    log global
    option httplog
	timeout connect 1s
	timeout server 5s
	timeout client 5s

{{ .CustomConfig }}

frontend test
    mode http
    bind 127.0.0.1:{{ .FrontendPort }}

{{- if .EngineConfig }}
    filter spoe engine e2e config {{ .EngineConfig }}
{{ end -}}

    {{ .CustomFrontendConfig }}

    use_backend backend

backend backend
    mode http

    {{ .CustomBackendConfig }}

    {{ .BackendConfig }}

{{ if .EngineAddr -}}
backend e2e-spoa
    mode tcp
    server e2e {{ .EngineAddr }}
{{ end }}

{{ if .PeerAddr -}}
peers mypeers
	peer {{ .InstanceID }} {{ .LocalPeerAddr }}
	peer go_client {{ .PeerAddr }}
{{ end }}

`

const haproxyEngineConfig = `
[e2e]
spoe-agent e2e-agent
    messages e2e-req e2e-res
    option var-prefix e2e
    option set-on-error error
    timeout hello      100ms
    timeout idle       10s
    timeout processing 500ms
    use-backend e2e-spoa
    log global

spoe-message e2e-req
    args id=unique-id src-ip=src method=method path=path query=query version=req.ver headers=req.hdrs body=req.body
    event on-frontend-http-request

spoe-message e2e-res
    args id=unique-id version=res.ver status=status headers=res.hdrs body=res.body
    event on-http-response
`

type HAProxyConfig struct {
	EngineAddr           string
	PeerAddr             string
	EngineConfig         string
	FrontendPort         string
	CustomFrontendConfig string
	BackendConfig        string
	CustomBackendConfig  string
	CustomConfig         string
}

func mustExecuteTemplate(t *testing.T, text string, data any) string {
	tmpl, err := template.New("").Parse(text)
	if err != nil {
		t.Fatal(err)
	}

	var tmplBuf bytes.Buffer
	if err := tmpl.Execute(&tmplBuf, data); err != nil {
		t.Fatal(err)
	}

	return tmplBuf.String()
}

func WithHAProxy(cfg HAProxyConfig, f func(t *testing.T)) func(t *testing.T) {
	return func(t *testing.T) {
		if cfg.EngineConfig == "" {
			cfg.EngineConfig = haproxyEngineConfig
		}

		if cfg.BackendConfig == "" {
			cfg.BackendConfig = `
http-request return status 200 content-type "text/plain" string "Hello World!\n"
`
		}

		type tmplCfg struct {
			HAProxyConfig


			StatsSocket   string
			InstanceID    string
			LocalPeerAddr string
		}
		var tcfg tmplCfg
		tcfg.HAProxyConfig = cfg
		tcfg.InstanceID = fmt.Sprintf("instance_%s", cfg.FrontendPort)
		tcfg.LocalPeerAddr = fmt.Sprintf("127.0.0.1:%d", TCPPort(t))
		tcfg.StatsSocket = fmt.Sprintf("%s/stats%s.sock", os.TempDir(), tcfg.FrontendPort)

		if cfg.EngineAddr != "" {
			engineConfigFile := TempFile(t, "e2e.cfg", cfg.EngineConfig)
			tcfg.EngineConfig = engineConfigFile
			defer os.Remove(engineConfigFile)
		}

		haproxyConfig := mustExecuteTemplate(t, haproxyConfigTemplate, tcfg)
		haproxyConfigFile := TempFile(t, "haproxy.cfg", haproxyConfig)
		defer os.Remove(haproxyConfigFile)

		defer func() {
			if t.Failed() {
				t.Logf("HAProxy Config: \n%s", haproxyConfig)
			}
		}()

		WithProcess("haproxy", []string{"-f", haproxyConfigFile, "-L", tcfg.InstanceID}, func(t *testing.T) {
			waitOrTimeout(t, time.Second*3, func() {
				for {
					l, err := net.Dial("unix", tcfg.StatsSocket)
					if err != nil {
						continue
					}
					l.Close()

					// if we were able to connect, exit and let the test run
					break
				}
			})

			f(t)
		})(t)
	}
}

func waitOrTimeout(t *testing.T, d time.Duration, f func()) {
	c := make(chan bool)
	defer close(c)

	go func() {
		f()
		c <- true
	}()

	select {
	case <-time.After(d):
		t.Fatal("timeout while waiting for haproxy")
	case <-c:
	}
}
