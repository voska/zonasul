# zonasul — Zona Sul Supermarket CLI

[![CI](https://github.com/voska/zonasul/actions/workflows/ci.yml/badge.svg)](https://github.com/voska/zonasul/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/voska/zonasul)](https://github.com/voska/zonasul/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/voska/zonasul)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

CLI for ordering groceries from [Zona Sul](https://www.zonasul.com.br) supermarket in Rio de Janeiro. Designed for AI agents following [steipete CLI patterns](https://steipete.me/posts/agent-friendly-cli-tools). Data goes to stdout (parseable), hints/progress to stderr.

```bash
$ zonasul search "banana" --limit 3
1    Banana Prata                                       SKU:33277   R$8.79
2    Banana Prata Organica 800g                         SKU:6180    R$10.99
3    Banana Nanica                                      SKU:33278   R$5.49

$ zonasul cart add 33277 --qty 3
Added 33277 to cart.

$ zonasul delivery windows --json
[{"index":0,"start":"2026-03-06T14:01:00Z","end":"2026-03-06T16:00:59Z","price":700}]

$ zonasul checkout --window 0 --cvv 123 --confirm
Order placed! ID: v-1234567890
```

Run `zonasul --help` for the full command tree, or `zonasul schema --json` for machine-readable introspection.

## Install

**Homebrew** (macOS / Linux):

```bash
brew install voska/tap/zonasul
```

**Go**:

```bash
go install github.com/voska/zonasul/cmd/zonasul@latest
```

**Binary**: download from [Releases](https://github.com/voska/zonasul/releases).

## Quick Start

```bash
# Login (paste JWT from browser DevTools)
zonasul auth login

# Search products (use Portuguese)
zonasul search "feijao preto" --json

# Add items by SKU
zonasul cart add 33277 --qty 3

# View cart
zonasul cart --json

# List delivery windows
zonasul delivery windows --json

# Place order (Pix payment, window 0)
zonasul checkout --window 0 --confirm

# Place order with credit card
zonasul checkout --window 0 --payment credit --cvv 123 --confirm
```

## Getting Credentials

Zona Sul uses VTEX custom OAuth — there's no API key. You authenticate by pasting a JWT token from the browser:

1. Open [zonasul.com.br](https://www.zonasul.com.br) and log in
2. Open DevTools > Application > Cookies > `www.zonasul.com.br`
3. Copy the value of `VtexIdclientAutCookie_zonasul`
4. Run `zonasul auth login` and paste the token

The token lasts 24 hours. Re-run `zonasul auth login` when it expires.

## Agent Skill

Install as a [Claude Code skill](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/skills) for AI-assisted grocery ordering:

```bash
npx skills add -g voska/zonasul
```

## Output Modes

| Flag | Description |
|------|-------------|
| (default) | Colored human-readable output, hints on stderr |
| `--json` | Structured JSON to stdout |
| `--plain` | Plain text for piping |

Environment variable overrides: `ZONASUL_JSON=1`, `ZONASUL_PLAIN=1`, `ZONASUL_NO_INPUT=1`.

## Commands

| Command | Description |
|---------|-------------|
| `auth login\|status\|logout` | Authentication |
| `search <query>` | Search products |
| `cart [show\|add\|remove\|clear]` | Cart management |
| `delivery windows` | List delivery time slots |
| `checkout` | Place an order (`--confirm` required) |
| `orders` | List recent orders |
| `agent exit-codes` | Exit code reference |
| `schema` | CLI command tree as JSON |

All commands support `--json`, `--plain`, and `--no-input`. Run `zonasul agent exit-codes` for the full exit code reference.

## Exit Codes

| Code | Name | Meaning |
|------|------|---------|
| 0 | success | Operation completed |
| 1 | error | General error |
| 2 | usage | Invalid arguments |
| 3 | empty | No results found |
| 4 | auth_required | Not logged in or token expired |
| 5 | not_found | Resource not found |
| 6 | forbidden | Permission denied |
| 7 | rate_limited | Too many requests |
| 8 | retryable | Transient error, safe to retry |
| 9 | min_order | Cart below R$100 minimum |
| 10 | config_error | Configuration error |

## Development

```bash
make build    # Build to bin/zonasul
make test     # Run tests with race detector
make lint     # Run linter
make vet      # Run go vet
make fmt      # Format code
```

## License

MIT
