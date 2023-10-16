# Benchmarks

Benchmarks comparing the different SPOP libraries. Currently only raw decoding and no proper e2e flow.

Criteo Library is replaced with a slightly modified fork by Babiel
https://github.com/babiel/haproxy-spoe-go

---

Tested and compared via benchstat (thx to https://www.rodolfocarvalho.net/blog/go-test-bench-pipe-to-benchstat/)

    go test -run='^$' -bench=. -benchmem -count=5 | tee >(benchstat /dev/stdin)

---

```
goos: linux
goarch: amd64
pkg: github.com/dropmorepackets/haproxy-go/spop/benchmarks
cpu: AMD EPYC 7502P 32-Core Processor
BenchmarkCriteo-48             	 5874009	       245.8 ns/op	     263 B/op	      12 allocs/op
BenchmarkCriteo-48             	 4504574	       266.3 ns/op	     263 B/op	      12 allocs/op
BenchmarkCriteo-48             	 4511300	       272.8 ns/op	     263 B/op	      12 allocs/op
BenchmarkCriteo-48             	 4336767	       279.2 ns/op	     263 B/op	      12 allocs/op
BenchmarkCriteo-48             	 4241575	       267.2 ns/op	     263 B/op	      12 allocs/op
BenchmarkCriteo-48             	 4719711	       274.6 ns/op	     263 B/op	      12 allocs/op
BenchmarkCriteo-48             	 4419110	       255.4 ns/op	     263 B/op	      12 allocs/op
BenchmarkCriteo-48             	 5013790	       270.8 ns/op	     263 B/op	      12 allocs/op
BenchmarkCriteo-48             	 4283295	       267.7 ns/op	     263 B/op	      12 allocs/op
BenchmarkCriteo-48             	 4446008	       270.4 ns/op	     263 B/op	      12 allocs/op
BenchmarkNegasus-48            	 1668440	       725.3 ns/op	     755 B/op	      18 allocs/op
BenchmarkNegasus-48            	 1583863	       763.6 ns/op	     755 B/op	      18 allocs/op
BenchmarkNegasus-48            	 1592184	       730.7 ns/op	     755 B/op	      18 allocs/op
BenchmarkNegasus-48            	 1579813	       755.2 ns/op	     755 B/op	      18 allocs/op
BenchmarkNegasus-48            	 1626435	       731.5 ns/op	     755 B/op	      18 allocs/op
BenchmarkNegasus-48            	 1656385	       751.8 ns/op	     755 B/op	      18 allocs/op
BenchmarkNegasus-48            	 1610750	       735.7 ns/op	     755 B/op	      18 allocs/op
BenchmarkNegasus-48            	 1632219	       750.6 ns/op	     755 B/op	      18 allocs/op
BenchmarkNegasus-48            	 1685029	       709.3 ns/op	     755 B/op	      18 allocs/op
BenchmarkNegasus-48            	 1649761	       730.2 ns/op	     755 B/op	      18 allocs/op
BenchmarkDropMorePackets-48    	120675940	        10.50 ns/op	       0 B/op	       0 allocs/op
BenchmarkDropMorePackets-48    	93222517	        16.45 ns/op	       0 B/op	       0 allocs/op
BenchmarkDropMorePackets-48    	100000000	        14.19 ns/op	       0 B/op	       0 allocs/op
BenchmarkDropMorePackets-48    	93988230	        11.41 ns/op	       0 B/op	       0 allocs/op
BenchmarkDropMorePackets-48    	97593783	        13.23 ns/op	       0 B/op	       0 allocs/op
BenchmarkDropMorePackets-48    	79098175	        16.19 ns/op	       0 B/op	       0 allocs/op
BenchmarkDropMorePackets-48    	95429886	        13.68 ns/op	       0 B/op	       0 allocs/op
BenchmarkDropMorePackets-48    	119408089	        13.65 ns/op	       0 B/op	       0 allocs/op
BenchmarkDropMorePackets-48    	80430522	        17.22 ns/op	       0 B/op	       0 allocs/op
BenchmarkDropMorePackets-48    	111808652	        11.97 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/dropmorepackets/haproxy-go/spop/benchmarks	51.642s
goos: linux
goarch: amd64
pkg: github.com/dropmorepackets/haproxy-go/spop/benchmarks
cpu: AMD EPYC 7502P 32-Core Processor
                   │  /dev/stdin  │
                   │    sec/op    │
Criteo-48            269.0n ±  5%
Negasus-48           733.6n ±  3%
DropMorePackets-48   13.66n ± 20%
geomean              139.2n

                   │  /dev/stdin  │
                   │     B/op     │
Criteo-48            263.0 ± 0%
Negasus-48           755.0 ± 0%
DropMorePackets-48   0.000 ± 0%
geomean                         ¹
¹ summaries must be >0 to compute geomean

                   │  /dev/stdin  │
                   │  allocs/op   │
Criteo-48            12.00 ± 0%
Negasus-48           18.00 ± 0%
DropMorePackets-48   0.000 ± 0%
geomean                         ¹
¹ summaries must be >0 to compute geomean
```

