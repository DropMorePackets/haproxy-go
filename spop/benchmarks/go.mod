module github.com/dropmorepackets/haproxy-go/spop/benchmarks

go 1.21.0

replace github.com/dropmorepackets/haproxy-go => ../../

// Replace to allow benchmarking
replace github.com/criteo/haproxy-spoe-go => github.com/babiel/haproxy-spoe-go v1.0.7-0.20220317153857-9119f3323ea8

require (
	github.com/criteo/haproxy-spoe-go v0.0.0
	github.com/dropmorepackets/haproxy-go v0.0.0
	github.com/negasus/haproxy-spoe-go v1.0.4
)

require (
	github.com/libp2p/go-buffer-pool v0.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.7.0 // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	golang.org/x/sys v0.11.0 // indirect
)
