# Changelog

## [0.0.7](https://github.com/DropMorePackets/haproxy-go/compare/v0.0.6...v0.0.7) (2025-06-05)


### Bug Fixes

* **spop:** Disable pipeline and async support ([c12e722](https://github.com/DropMorePackets/haproxy-go/commit/c12e722bc2171bd585d6613d08dcecfb4accbda7))

## [0.0.6](https://github.com/DropMorePackets/haproxy-go/compare/v0.0.5...v0.0.6) (2025-05-30)

### CI
* update staticcheck config
* add release-please
* allow release-please to access issues

### Bug Fixes
* **spop:** set write and read buffer to 64K
* **spop:** don't let panics take the library or workers out
* **spop:** remove unused field lf
* **peers:** update struct alignment to be more efficient ([5a12fb3](https://github.com/DropMorePackets/haproxy-go/commit/5a12fb36a131076baf277deee278f0a8a5894a3b))
