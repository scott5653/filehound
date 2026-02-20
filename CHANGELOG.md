# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-02-20

### Added

- **Extensible Source System**: New `internal/source` package with pluggable sources:
  - `Source` interface for unified file listing and reading
  - Auto-detection of source type from path (local, S3, Git)
  - Scheme-based registration (`s3://`, `git://`)
- **S3 Source Plugin**:
  - Scan S3 buckets with `s3://bucket/prefix` paths
  - Support for custom regions and S3-compatible endpoints
  - Configurable credentials via options
  - `--s3-region` and `--s3-endpoint` flags
- **Git Source Plugin**:
  - Scan Git repositories in working tree mode or full history mode
  - Branch filtering with `--git-branch` flag
  - Commit date filtering with `--git-since` flag
  - `--git-mode working|full` flag
- **Unified CLI**: Same filters work across all source types

### Dependencies

- Added `github.com/aws/aws-sdk-go-v2` for S3 support
- Added `github.com/go-git/go-git/v5` for Git support

## [0.2.0] - 2026-02-20

### Added

- `hash` command for file hashing and duplicate detection:
  - SHA1, SHA256, SHA512 algorithm support
  - `--duplicates` flag to find and group duplicate files
  - JSON/CSV/table output formats
  - Integration with glob, extension, and size filters
- `internal/hash` package:
  - Concurrent file hashing
  - Duplicate detection by content hash
  - Configurable buffer size for performance
- Interactive progress display for `scan` command:
  - Animated progress bar with gradient colors
  - Real-time file count and error statistics
  - Elapsed time display
  - Press 'q' to cancel scan
  - Enable with `--progress` or `-p` flag
- `internal/tui` package with bubbletea integration

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
