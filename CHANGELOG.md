# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Go 1.27 module baseline
- Corporate masking: `CorporateMasker`, `CorporateMaskRules`, `WithCorporateMasking`, JWT/AWS/PEM/IBAN/Luhn auto-detect, fingerprints
- Live level control: `ParseLevel`, `SetLevelString`, `LevelHTTPHandler`, `ListenLevelHTTP`, `WatchLevelEnv`
- Context recovery of slogx config from `slog.Default()` when installed via `SetupDefault`
- `Corporate()` preset alias; production presets enable corporate masking by default

### Changed

- Mask key matching is case/separator-insensitive with nested and suffix lookup
- Email/phone/card redaction shapes tightened for corporate logs
