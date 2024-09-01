# CHANGELOG

## v1.0.0

January 19, 2024

This release will include the following changes:
- Add an entry expiration in the cache
- Add GC-optimized cache implementation
- Change implementation of the bucket table in the cache.

### BREAKING CHANGES
- Add expiration policy. [\#27](https://github.com/scalalang2/golang-fifo/pull/27) by @scalalang2

### FEATURES
- Add expiration policy. [\#27](https://github.com/scalalang2/golang-fifo/pull/27) by @scalalang2

### IMPROVEMENTS
- Bump up go version to 1.22. [\#26](https://github.com/scalalang2/golang-fifo/pull/26) by @scalalang2

### BUG FIXES
- Fix a race condition issue in the SIEVE cache. [\#23](https://github.com/scalalang2/golang-fifo/pull/23) by @scalalang2