# Changelog

## [0.1.0](https://github.com/DropMorePackets/haproxy-go/compare/v0.0.8...v0.1.0) (2026-01-31)


### Features

* add KV unmarshal feature with struct tag support ([2b5523a](https://github.com/DropMorePackets/haproxy-go/commit/2b5523a8b13327c33da01a956ba72525fcc7e519))

## [0.0.8](https://github.com/DropMorePackets/haproxy-go/compare/v0.0.7...v0.0.8) (2026-01-31)


### Bug Fixes

* eliminate string allocations in protocol handlers ([7c4d43b](https://github.com/DropMorePackets/haproxy-go/commit/7c4d43b1918925b95081ed403aff050c61a10a92))
* optimize NameEquals to avoid string allocation ([45c05fa](https://github.com/DropMorePackets/haproxy-go/commit/45c05fa25892ae57ab1f2d8e4fb9c8f98abea41e))
* **spop:** use io.ReadFull to prevent partial frame reads ([c1f732a](https://github.com/DropMorePackets/haproxy-go/commit/c1f732ad879a8d0a9e531a25fc88912b98428cfd))


### Performance Improvements

* cache length to avoid repeated len() calls in comparison functions ([022f7bf](https://github.com/DropMorePackets/haproxy-go/commit/022f7bf823b6ae0ff8de24a3ad930ccae5bd6b99))

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
