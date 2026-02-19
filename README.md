# FileHound

[![Go Report Card](https://goreportcard.com/badge/github.com/ripkitten-co/filehound)](https://goreportcard.com/report/github.com/ripkitten-co/filehound)
[![Go Reference](https://pkg.go.dev/badge/github.com/ripkitten-co/filehound.svg)](https://pkg.go.dev/github.com/ripkitten-co/filehound)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Blazing fast file hunter.** 10x faster than `find + rg` on huge directories.

## Install

```bash
# Go install
go install github.com/ripkitten-co/filehound@latest

# Or download binary from releases
curl -sSL https://raw.githubusercontent.com/ripkitten-co/filehound/main/install.sh | sh
```

## Quick Start

```bash
# Hunt secrets by regex + entropy
filehound scan . --regex "(?i)(key|pass|secret|token)" --entropy 6.0

# Find large files modified in last 24 hours
filehound scan /logs --size ">100MB" --modified "<24h" --output json

# Find all Go files
filehound scan . --ext .go

# Batch rename with hash
filehound rename ./photos --glob "*.jpg" --pattern "img_{{sha1:8}}{{ext}}" --dry-run
```

## Commands

### `scan` - Hunt files by criteria

```bash
filehound scan [PATH...] [flags]
```

| Flag | Description | Example |
|------|-------------|---------|
| `-r, --regex` | Regex pattern in file content | `--regex "(?i)password"` |
| `--regex-path` | Regex pattern in file path | `--regex-path "node_modules"` |
| `--entropy` | Min entropy threshold (0-8) | `--entropy 7.5` |
| `--mime` | MIME types to match | `--mime image/png,text/plain` |
| `--ext` | File extensions | `--ext .go,.ts,.js` |
| `-g, --glob` | Glob pattern for filename | `--glob "*.log"` |
| `--size` | Size filter | `--size ">1MB"` |
| `--modified` | Modification time filter | `--modified "<7d"` |
| `--exclude` | Additional dirs to exclude | `--exclude ".cache,tmp"` |
| `-w, --workers` | Parallel workers | `--workers 16` |
| `--empty` | Match only empty files | `--empty` |
| `--follow` | Follow symbolic links | `--follow` |
| `-o, --output` | Output format: table, json, csv | `--output json` |
| `--out-file` | Write output to file | `--out-file results.json` |

### `rename` - Batch rename files

```bash
filehound rename [PATH...] --pattern TEMPLATE [flags]
```

| Flag | Description | Example |
|------|-------------|---------|
| `-p, --pattern` | Rename template (required) | `--pattern "img_{{sha1:8}}{{ext}}"` |
| `--dry-run` | Preview changes | `--dry-run` |
| `-g, --glob` | Glob pattern | `--glob "*.jpg"` |
| `--ext` | File extensions | `--ext .jpg,.png` |
| `--size` | Size filter | `--size ">1MB"` |
| `--force` | Overwrite existing files | `--force` |

#### Template Variables

| Variable | Description |
|----------|-------------|
| `{{name}}` | Original filename without extension |
| `{{ext}}` | File extension (including dot) |
| `{{size}}` | File size in bytes |
| `{{sha1:N}}` | First N chars of SHA1 hash (default: 8) |
| `{{sha256:N}}` | First N chars of SHA256 hash (default: 8) |

## Examples

### Hunt Secrets

```bash
# Find potential API keys and secrets
filehound scan . \
  --regex "(?i)(api_key|secret|token|password)\s*[:=]" \
  --entropy 6.0 \
  --ext .env,.yml,.yaml,.json,.conf \
  --output json | jq
```

### Find Duplicates

```bash
# Find files with same content (rename with hash)
filehound scan . --glob "*.jpg" --output json | \
  jq -r '.Path' | \
  xargs -I{} filehound rename {} --pattern "{{sha1:12}}{{ext}}" --dry-run
```

### Clean Up Logs

```bash
# Find large log files older than 30 days
filehound scan /var/log \
  --ext .log \
  --size ">100MB" \
  --modified ">30d"
```

## Benchmarks

Tested on Intel i9-13900F, Windows 11, SSD.

| Operation | 100 files | 1000 files |
|-----------|-----------|------------|
| Scan | 166 µs | 1.0 ms |
| Extension match | 51 ns | - |
| Size filter | 0.14 ns | - |
| Glob match | 72 ns | - |

## Performance Tips

1. **More workers** for larger directories: `--workers 16`
2. **Exclude directories** you don't need: `--exclude "dist,build,vendor"`
3. **Use filters early** to reduce I/O: `--ext .go --size ">1KB"` before `--regex`
4. **JSON output** is faster than table for scripting: `--output json`

## Why Go?

- **Performance**: Native binary, no runtime overhead
- **Cross-platform**: Single binary for Linux, macOS, Windows
- **Static linking**: No dependencies, works everywhere
- **Fast builds**: Develop and iterate quickly

## License

MIT License - see [LICENSE](LICENSE)
