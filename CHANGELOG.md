# AXONE Prolog changelog

## [3.2.0](https://github.com/axone-protocol/prolog/compare/v3.1.0...v3.2.0) (2026-03-03)


### Features

* **engine:** add VM metering ([11d1ec5](https://github.com/axone-protocol/prolog/commit/11d1ec539920bfff48733bb2cf279a3a40aa0b2d))

## [3.1.0](https://github.com/axone-protocol/prolog/compare/v3.0.0...v3.1.0) (2026-02-12)


### Features

* **engine:** expose VM.LoadedSources() ([baa7d26](https://github.com/axone-protocol/prolog/commit/baa7d2686469f47825607383d3f9d4421fc81463))

## [3.0.0](https://github.com/axone-protocol/prolog/compare/v2.1.1...v3.0.0) (2026-02-10)


### ⚠ BREAKING CHANGES

* **engine:** restore halt/1 behavior with error signaling for VM execution
* **engine:** route open/3 and open/4 through VM.FS with OpenFileFS support

### Features

* **engine:** add read_write mode for bidirectional file I/O ([0fed4e0](https://github.com/axone-protocol/prolog/commit/0fed4e0d2c332635d97a93d53d76bfc62b08eacb))


### Bug Fixes

* **engine:** restore halt/1 behavior with error signaling for VM execution ([cb318a7](https://github.com/axone-protocol/prolog/commit/cb318a7194afb884b884ab96d4ab2099513814d0))


### Code Refactoring

* **engine:** route open/3 and open/4 through VM.FS with OpenFileFS support ([469300f](https://github.com/axone-protocol/prolog/commit/469300faefb36efb2b9af8c46c752cccb02bb8ab))
