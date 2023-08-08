package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"text/template"
)

type TemplateInstance struct {
	Instances []*TemplateInstance
	Instance  string
	ID        string
}

var count = flag.Int("count", 20, "")

func main() {
	flag.Parse()

	var instances []*TemplateInstance
	for i := 0; i < *count; i++ {
		instances = append(instances, &TemplateInstance{
			Instance: fmt.Sprintf("instance%d", i),
			ID:       fmt.Sprintf("%02d", i),
		})
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, instance := range instances {
		instance.Instances = instances
		wg.Add(1)
		go startHAProxy(&wg, ctx, instance)
	}

	<-c
	cancel()
	wg.Wait()
}

func startHAProxy(wg *sync.WaitGroup, ctx context.Context, instance *TemplateInstance) {
	defer wg.Done()

	f, err := os.CreateTemp("/tmp", "haproxy")
	if err != nil {
		log.Fatal(err)
	}
	//defer os.Remove(f.Name())
	defer f.Close()

	tmpl, err := template.New("").Parse(haproxyCfgTemplate)
	if err != nil {
		log.Fatal(err)
	}

	if err := tmpl.Execute(f, instance); err != nil {
		log.Fatal(err)
	}

	log.Println("Starting ", instance.ID)
	cmd := exec.Command("haproxy", "-L", instance.Instance, "--", f.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
	if err := cmd.Process.Kill(); err != nil {
		log.Fatal(err)
	}
}

var haproxyCfgTemplate = `
global
	log stdout format short daemon
    stats socket /tmp/haproxy{{ .ID }}.sock mode 660 level admin expose-fd listeners
    stats timeout 30s

defaults
	timeout connect 1s
	timeout server 5s
	timeout client 5s

frontend http
	mode http
	log global

	bind *:80{{ .ID }}

	http-request track-sc0 src table st_src_global
    http-request track-sc2 req.hdr(Host) table st_be_name

    #tcp-request inspect-delay 1s
    #tcp-request content track-sc0 path table st_gpc


	default_backend backend_stats

#backend st_gpc
#    stick-table type string len 128 size 2k expire 1d store http_err_rate(1d) peers mycluster

backend st_be_name
    stick-table  type string  size 1m  expire 10m  store http_req_rate(10s) peers mycluster

backend st_src_global
	stick-table type ip size 1m expire 10m store http_req_rate(10s),conn_rate(10s),bytes_in_rate(10s) peers mycluster

backend backend_stats
	mode http
    #stick on payload_lv(43,1) table st_gpc
    #stick store-response payload_lv(43,1) table st_gpc
	server stats 127.0.0.1:90{{ .ID }}

frontend stats
    bind *:84{{ .ID }}
    mode http
    stats enable
    stats uri /stats
    stats refresh 10s

listen stats_listen
  bind 127.0.0.1:90{{ .ID }}
  mode http

peers mycluster
	{{ range .Instances }}
    peer {{ .Instance }} 127.0.0.1:200{{ .ID }}
	{{ end }}
	peer go_client 127.0.0.1:21000
`
