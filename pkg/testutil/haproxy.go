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
    bind unix@{{ .FrontendSocket }} accept-proxy

{{- if .EngineConfigFile }}
    filter spoe engine e2e config {{ .EngineConfigFile }}
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

func (cfg HAProxyConfig) Run(tb testing.TB) string {
	tb.Helper()

	if cfg.EngineConfig == "" {
		cfg.EngineConfig = haproxyEngineConfig
	}

	if cfg.BackendConfig == "" {
		cfg.BackendConfig = `
http-request return status 200 content-type "text/plain" string "Hello World!\n"
`
	}

	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("haproxy_%s", cfg.FrontendPort))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	type tmplCfg struct {
		HAProxyConfig

		StatsSocket      string
		FrontendSocket   string
		InstanceID       string
		LocalPeerAddr    string
		EngineConfigFile string
	}
	var tcfg tmplCfg
	tcfg.HAProxyConfig = cfg
	tcfg.InstanceID = fmt.Sprintf("instance_%s", cfg.FrontendPort)
	tcfg.LocalPeerAddr = fmt.Sprintf("127.0.0.1:%d", TCPPort(tb))
	tcfg.StatsSocket = fmt.Sprintf("%s/stats.sock", tmpDir)
	tcfg.FrontendSocket = fmt.Sprintf("%s/frontend.sock", tmpDir)

	if cfg.EngineAddr != "" {
		engineConfigFile := TempFile(tb, "e2e.cfg", cfg.EngineConfig)
		tcfg.EngineConfigFile = engineConfigFile
	}

	haproxyConfig := mustExecuteTemplate(tb, haproxyConfigTemplate, tcfg)
	haproxyConfigFile := TempFile(tb, "haproxy.cfg", haproxyConfig)

	tb.Cleanup(func() {
		if tb.Failed() {
			tb.Logf("HAProxy Config: \n%s", haproxyConfig)
		}
	})

	RunProcess(tb, "haproxy", []string{"-f", haproxyConfigFile, "-L", tcfg.InstanceID})

	c := make(chan bool)
	defer close(c)

	go func() {
		for {
			l, err := net.Dial("unix", tcfg.StatsSocket)
			if err != nil {
				continue
			}
			l.Close()

			l, err = net.Dial("unix", tcfg.FrontendSocket)
			if err != nil {
				continue
			}
			l.Close()

			// if we were able to connect, exit and let the test run
			break
		}
		c <- true
	}()

	select {
	case <-time.After(3 * time.Second):
		tb.Fatal("timeout while waiting for haproxy")
	case <-c:
	}

	return tcfg.FrontendSocket
}

func mustExecuteTemplate(tb testing.TB, text string, data any) string {
	tb.Helper()
	tmpl, err := template.New("").Parse(text)
	if err != nil {
		tb.Fatal(err)
	}

	var tmplBuf bytes.Buffer
	if err := tmpl.Execute(&tmplBuf, data); err != nil {
		tb.Fatal(err)
	}

	return tmplBuf.String()
}
