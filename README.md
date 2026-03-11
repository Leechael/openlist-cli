# openlist-cli

Friendly command-line client for the OpenList API.

It supports both:
- high-level commands for common workflows
- low-level spec-driven commands for full API coverage

## Install

```bash
go install github.com/openlist/openlist-cli/cmd/openlist-cli@latest
```

Or build from source:

```bash
make build
```

## Configuration

You can configure the CLI with environment variables or local config.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `OPENLIST_BASE_URL` | `http://localhost:5244` | OpenList server base URL |
| `OPENLIST_TOKEN` | _(empty)_ | API token sent as the raw `Authorization` header |
| `OPENLIST_CLI_CONFIG` | platform default | Override config file path |

### Local config

```bash
openlist-cli config set --base-url http://localhost:5244
openlist-cli config set --token 'your-fixed-admin-token'
openlist-cli config show
```

Default config path:
- macOS/Linux: `~/.config/openlist-cli/config.json`

Priority order:
1. command-line flags
2. environment variables
3. local config file

## Authentication model

OpenList supports two token styles in practice:

1. **Login JWT** from `/api/auth/login/hash`
   - expires according to server-side `TOKEN_EXPIRES_IN`
2. **Fixed admin token** from server settings
   - does not expire automatically
   - becomes invalid only if rotated/reset on the server

This CLI sends the token as the raw `Authorization` header value to match the server implementation.

## Usage

```bash
openlist-cli <command> [flags]
```

All commands support:
- `--json` for machine-readable output
- `--plain` for stable plain-text output
- `--jq` only together with `--json`

## Friendly commands

### Auth

```bash
openlist-cli auth login --username admin --password 'secret'
openlist-cli auth token
openlist-cli auth whoami
openlist-cli auth logout
```

- `auth login` uses `/api/auth/login/hash`
- plain passwords are hashed with SHA256 before sending
- `auth token` prints the effective token, similar to `gh auth token`

### Config

```bash
openlist-cli config show
openlist-cli config set --base-url http://localhost:5244
openlist-cli config set --token 'your-token'
openlist-cli config clear
```

### File system

```bash
# List a directory
openlist-cli fs ls /
openlist-cli fs ls .

# Get file or directory metadata
openlist-cli fs stat /Movies/file.mkv

# Print a tree (default depth: 3)
openlist-cli fs tree /Movies
openlist-cli fs tree /Movies --depth 5

# Search
openlist-cli fs search --parent / --keywords movie

# Build OpenList download URLs
openlist-cli fs download-url /Movies/file.mkv
openlist-cli fs download-url /Movies/file.mkv --proxy
openlist-cli fs download-url /Movies/file.mkv --raw-url
```

### Shares

```bash
openlist-cli share ls
openlist-cli share url abcdef123456
openlist-cli share url abcdef123456 dir/file.txt --archive
```

## Low-level commands

These are useful when you want direct access to spec operations.

```bash
# List all OpenAPI operations
openlist-cli list-ops --plain

# Call any operation by operationId
openlist-cli call fsList --body '{"path":"/"}' --json
openlist-cli call ping --json

# Build non-JSON route URLs
openlist-cli route direct-url --path '/Movies/file.mkv' --sign 'BASE64:0'
openlist-cli route proxy-url --path '/Movies/file.mkv' --sign 'BASE64:0'
openlist-cli route archive-url --archive-path '/archive.zip' --inner 'a.txt' --sign 'SIG'
openlist-cli route share-url --sharing-id abcdef123456

# Fetch a URL or OpenList-relative route
openlist-cli fetch --url '/d/Movies/file.mkv?sign=BASE64:0' --output ./file.mkv
```

## Examples

Use a fixed token from environment variables:

```bash
export OPENLIST_BASE_URL='http://localhost:5244'
export OPENLIST_TOKEN='your-fixed-admin-token'

openlist-cli auth token
openlist-cli auth whoami
openlist-cli fs ls --path /
```

Use login-based auth and save the returned JWT locally:

```bash
openlist-cli auth login --username admin --password 'secret'
openlist-cli auth whoami
```

## Development

```bash
make test         # unit tests
make bdd          # BDD/smoke tests
make ci           # fmt check + vet + tests + build
make build        # local binary
make cross-build  # darwin/linux amd64/arm64
make clean        # remove build artifacts
```

## Project layout

- `cmd/openlist-cli` - entrypoint
- `internal/app` - CLI implementation
- `internal/spec` - embedded OpenAPI spec and operation index
- `docs/download-and-url-routes.md` - non-JSON route reference
- `docs/openapi/` - generated OpenAPI summary/plan/test matrix
- `.github/workflows/` - CI and release workflows
