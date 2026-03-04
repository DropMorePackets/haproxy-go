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
cpu: AMD EPYC 9555P 64-Core Processor               
BenchmarkCriteo-128                      5435854               584.0 ns/op           264 B/op         12 allocs/op
BenchmarkCriteo-128                      5856099               233.0 ns/op           263 B/op         12 allocs/op
BenchmarkCriteo-128                      5507936               236.3 ns/op           263 B/op         12 allocs/op
BenchmarkCriteo-128                      5513528               234.0 ns/op           263 B/op         12 allocs/op
BenchmarkCriteo-128                      6009813               233.6 ns/op           263 B/op         12 allocs/op
BenchmarkNegasus-128                     1682955               709.7 ns/op           755 B/op         18 allocs/op
BenchmarkNegasus-128                     1690485               702.2 ns/op           755 B/op         18 allocs/op
BenchmarkNegasus-128                     1653942               707.2 ns/op           755 B/op         18 allocs/op
BenchmarkNegasus-128                     1736671               702.9 ns/op           755 B/op         18 allocs/op
BenchmarkNegasus-128                     1763883               704.7 ns/op           755 B/op         18 allocs/op
BenchmarkDropMorePackets-128            409480474                3.257 ns/op           0 B/op          0 allocs/op
BenchmarkDropMorePackets-128            373124017                3.617 ns/op           0 B/op          0 allocs/op
BenchmarkDropMorePackets-128            574863938                1.791 ns/op           0 B/op          0 allocs/op
BenchmarkDropMorePackets-128            742134968                5.611 ns/op           0 B/op          0 allocs/op
BenchmarkDropMorePackets-128            576501963                2.403 ns/op           0 B/op          0 allocs/op
PASS
ok      github.com/dropmorepackets/haproxy-go/spop/benchmarks   30.927s
goos: linux
goarch: amd64
pkg: github.com/dropmorepackets/haproxy-go/spop/benchmarks
cpu: AMD EPYC 9555P 64-Core Processor               
                    │  /dev/stdin  │
                    │    sec/op    │
Criteo-128            234.0n ± ∞ ¹
Negasus-128           704.7n ± ∞ ¹
DropMorePackets-128   3.257n ± ∞ ¹
geomean               81.29n
¹ need >= 6 samples for confidence interval at level 0.95

                    │ /dev/stdin  │
                    │    B/op     │
Criteo-128            263.0 ± ∞ ¹
Negasus-128           755.0 ± ∞ ¹
DropMorePackets-128   0.000 ± ∞ ¹
geomean                         ²
¹ need >= 6 samples for confidence interval at level 0.95
² summaries must be >0 to compute geomean

                    │ /dev/stdin  │
                    │  allocs/op  │
Criteo-128            12.00 ± ∞ ¹
Negasus-128           18.00 ± ∞ ¹
DropMorePackets-128   0.000 ± ∞ ¹
geomean                         ²
¹ need >= 6 samples for confidence interval at level 0.95
² summaries must be >0 to compute geomean
```

