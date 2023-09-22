# Benchmarks

Benchmarks comparing the different SPOP libraries. Currently only raw decoding and no proper e2e flow.

Criteo Library is replaced with a slightly modified fork by Babiel
https://github.com/babiel/haproxy-spoe-go

---

Tested and compared via benchstat (thx to https://www.rodolfocarvalho.net/blog/go-test-bench-pipe-to-benchstat/)

    go test -run='^$' -bench=. -benchmem -count=5 | tee >(benchstat /dev/stdin)

Tested on a MacBookPro18,4 with a M1 Max:
```
goos: darwin
goarch: arm64
pkg: github.com/fionera/haproxy-go/spop/benchmarks
BenchmarkCriteo-10      11582493               118.7 ns/op           264 B/op         12 allocs/op
BenchmarkCriteo-10      10842691               114.7 ns/op           263 B/op         12 allocs/op
BenchmarkCriteo-10      10350162               112.8 ns/op           263 B/op         12 allocs/op
BenchmarkCriteo-10      10490358               111.8 ns/op           264 B/op         12 allocs/op
BenchmarkCriteo-10       9671280               123.9 ns/op           264 B/op         12 allocs/op
BenchmarkNegasus-10      3686066               337.9 ns/op           752 B/op         18 allocs/op
BenchmarkNegasus-10      3627370               343.7 ns/op           752 B/op         18 allocs/op
BenchmarkNegasus-10      3602414               336.8 ns/op           752 B/op         18 allocs/op
BenchmarkNegasus-10      3396264               351.0 ns/op           752 B/op         18 allocs/op
BenchmarkNegasus-10      3401060               348.7 ns/op           752 B/op         18 allocs/op
BenchmarkFionera-10     68821312                27.22 ns/op            0 B/op          0 allocs/op
BenchmarkFionera-10     59779068                22.84 ns/op            0 B/op          0 allocs/op
BenchmarkFionera-10     65963814                22.08 ns/op            0 B/op          0 allocs/op
BenchmarkFionera-10     89442937                31.96 ns/op            0 B/op          0 allocs/op
BenchmarkFionera-10     88215834                18.13 ns/op            0 B/op          0 allocs/op
PASS
ok      github.com/fionera/haproxy-go/spop/benchmarks   23.882s
goos: darwin
goarch: arm64
pkg: github.com/fionera/haproxy-go/spop/benchmarks
           │  /dev/stdin  │
           │    sec/op    │
Criteo-10    114.7n ± ∞ ¹
Negasus-10   343.7n ± ∞ ¹
Fionera-10   22.84n ± ∞ ¹
geomean      96.56n
¹ need >= 6 samples for confidence interval at level 0.95

           │ /dev/stdin  │
           │    B/op     │
Criteo-10    264.0 ± ∞ ¹
Negasus-10   752.0 ± ∞ ¹
Fionera-10   0.000 ± ∞ ¹
geomean                ²
¹ need >= 6 samples for confidence interval at level 0.95
² summaries must be >0 to compute geomean

           │ /dev/stdin  │
           │  allocs/op  │
Criteo-10    12.00 ± ∞ ¹
Negasus-10   18.00 ± ∞ ¹
Fionera-10   0.000 ± ∞ ¹
geomean                ²
¹ need >= 6 samples for confidence interval at level 0.95
² summaries must be >0 to compute geomean
```

