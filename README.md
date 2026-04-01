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

$ zonasul list order diarista
Added SKU 39908
Added SKU 135
Added SKU 12274
Added SKU 33297

$ zonasul checkout --window 0 --payment credit --cvv 123 --confirm
Order placed! ID: 1621763420899
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
# Login with email and password (credentials stored for auto-refresh)
zonasul auth login --email you@example.com --password yourpass

# Search products (use Portuguese)
zonasul search "feijao preto" --json

# Add items by SKU
zonasul cart add 33277 --qty 3

# Build reusable lists
zonasul list add weekly 33277
zonasul list add weekly 6180
zonasul list order weekly          # add all to cart

# Favorites shorthand
zonasul fav add 33277
zonasul fav order

# Reorder from last order
zonasul cart reorder

# View cart
zonasul cart --json

# List delivery windows
zonasul delivery windows --json

# Place order (Pix payment, window 0)
zonasul checkout --window 0 --confirm

# Place order with credit card
zonasul checkout --window 0 --payment credit --cvv 123 --confirm
```

## Authentication

Three ways to log in:

```bash
# Email and password (recommended — enables auto-refresh)
zonasul auth login --email you@example.com --password yourpass

# Environment variables (for CI/headless)
ZONASUL_EMAIL=you@example.com ZONASUL_PASSWORD=yourpass zonasul auth login

# Manual JWT token (from browser DevTools)
zonasul auth login --token <VtexIdclientAutCookie_zonasul>
```

Email/password login stores credentials locally (`~/.config/zonasul/credentials.json` + keychain) so the CLI can auto-refresh expired tokens. You log in once and every command just works.

## Lists and Favorites

Named SKU lists live in `~/.config/zonasul/lists.json`:

```bash
zonasul list show                  # show all lists
zonasul list show diarista         # show items in a list
zonasul list add diarista 39908    # add SKU to list
zonasul list remove diarista 39908 # remove SKU from list
zonasul list order diarista        # add all to cart
zonasul list delete diarista       # delete entire list
```

`fav` is a shorthand for the `favorites` list:

```bash
zonasul fav add 18868              # add to favorites
zonasul fav                        # show favorites
zonasul fav order                  # add all favorites to cart
```

## Reorder

Re-add items from a previous order:

```bash
zonasul cart reorder               # from most recent order
zonasul cart reorder <order-id>    # from a specific order
```

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
| `auth login\|status\|logout` | Authentication (email/password, token, or OAuth) |
| `search <query>` | Search products |
| `cart [show\|add\|remove\|clear\|reorder]` | Cart management |
| `list [show\|add\|remove\|order\|delete]` | Named SKU lists |
| `fav [show\|add\|remove\|order]` | Favorites (shorthand for `list favorites`) |
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

## Config Files

All config lives in `~/.config/zonasul/`:

| File | Contents |
|------|----------|
| `config.json` | Address, orderFormId |
| `credentials.json` | Email (password in keychain) |
| `lists.json` | Named SKU lists and favorites |

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
