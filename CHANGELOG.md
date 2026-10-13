# AXONE Prolog changelog

## [4.0.0](https://github.com/axone-protocol/prolog/compare/v3.2.0...v4.0.0) (2026-10-13)


### ⚠ BREAKING CHANGES

* **engine:** move Go placeholder conversion out of the VM
* **engine:** enforce immutable VM ownership for streams
* **engine:** scope variable identities to each VM and remove ResetEnv
* **engine:** scope execution state to each VM

### Features

* **engine:** add deterministic term-cell metering ([c30ae9f](https://github.com/axone-protocol/prolog/commit/c30ae9f31902c85f9ece40e8d11f371b8aca0380))
* **engine:** scope execution state to each VM ([a9b0f1c](https://github.com/axone-protocol/prolog/commit/a9b0f1c4765d43b92c75180ed3638a3fc5809c79))
* **engine:** scope variable identities to each VM and remove ResetEnv ([91e9b23](https://github.com/axone-protocol/prolog/commit/91e9b23d6e00f257278a84f03b4a116d761d4867))


### Bug Fixes

* **engine:** enforce immutable VM ownership for streams ([882b819](https://github.com/axone-protocol/prolog/commit/882b8196dc0fc05c0acaaef863f8bc02b2d69e26))
* **engine:** fix stream close lifecycle ([f78b4f0](https://github.com/axone-protocol/prolog/commit/f78b4f0882441ba155a50f63db23295184df74ab))
* **engine:** fix write mode file truncation ([07a7af5](https://github.com/axone-protocol/prolog/commit/07a7af5bae99f8286008c7132226fef980c32c61))
* **engine:** handle negative makeSlice lengths ([7ab0b75](https://github.com/axone-protocol/prolog/commit/7ab0b75407d3a47cb50a49dc8d36bee386014ee1))
* **engine:** roll back open/4 file and alias on option parse failure ([b4bfba3](https://github.com/axone-protocol/prolog/commit/b4bfba3ea0e735910e52f6dddc80c538b4027b46))


### Code Refactoring

* **engine:** move Go placeholder conversion out of the VM ([73dfacf](https://github.com/axone-protocol/prolog/commit/73dfacfd2fb214765bd014cfb6004d5d7e592292))

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
