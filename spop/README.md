# A HAProxy SPOE implementation in Go

## Frame size

Agents accept SPOP frames up to 65,535 bytes by default. Set `Agent.MaxFrameSize`
when HAProxy is configured to advertise larger frames:

```go
agent := spop.Agent{
	Addr:         ":9000",
	Handler:      handler,
	MaxFrameSize: 256*1024 - 4,
}
log.Fatal(agent.ListenAndServe())
```

The configured value excludes the four-byte frame-length prefix. It is an
upper bound: each connection negotiates the lower of this value and HAProxy's
advertised `max-frame-size`. Values below 256 are invalid. Each active pooled
frame buffer is allocated at the configured size, so use the smallest ceiling
that accommodates the HAProxy configuration.

## References
https://www.haproxy.org/download/2.0/doc/SPOE.txt

## Alternative implementations
https://github.com/criteo/haproxy-spoe-go
https://github.com/negasus/haproxy-spoe-go
