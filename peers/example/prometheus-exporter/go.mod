module github.com/dropmorepackets/haproxy-go/peers/example/prometheus-exporter

go 1.21

replace github.com/dropmorepackets/haproxy-go => ../../../

require (
	github.com/dropmorepackets/haproxy-go v0.0.2
	github.com/prometheus/client_golang v1.17.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
