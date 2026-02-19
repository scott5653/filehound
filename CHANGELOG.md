# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-20

### Added

- Initial release
- `scan` command with filters:
  - Content regex matching
  - Path regex matching
  - Entropy detection for secrets
  - MIME type filtering
  - Extension filtering
  - Glob pattern matching
  - Size filtering
  - Modification time filtering
  - Empty file detection
  - Parallel scanning with configurable workers
  - JSON/CSV/table output formats
- `rename` command with pattern templates:
  - `{{name}}` - filename without extension
  - `{{ext}}` - file extension
  - `{{size}}` - file size
  - `{{sha1:N}}` - SHA1 hash prefix
  - `{{sha256:N}}` - SHA256 hash prefix
  - Dry-run mode for preview
  - Force overwrite option
- Cross-platform support (Linux, macOS, Windows)
- Default exclusions for common directories (.git, node_modules, etc.)
