# AXONE Prolog changelog

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
