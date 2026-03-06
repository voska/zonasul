# Zona Sul CLI

Go CLI for ordering groceries from Zona Sul supermarket (zonasul.com.br) in Rio de Janeiro. Designed for AI agent consumption following steipete CLI patterns (gogcli, bird, imsg).

## Project Structure

```
cmd/zonasul/       # CLI entry point
internal/
  vtex/            # VTEX API client (auth, search, cart, checkout)
  config/          # Config management (~/.config/zonasul/)
  keyring/         # macOS Keychain integration for JWT storage
  outfmt/          # Triple output mode (human/plain/json)
docs/              # API research, design docs
```

## Tech Stack

- **Language**: Go (single static binary)
- **CLI framework**: Kong (struct-tag-based parser)
- **Platform**: VTEX IO (Zona Sul's ecommerce backend)
- **Auth**: VTEX ID JWT stored in macOS Keychain

## Key Context

- Zona Sul's VTEX account name is `zonasul`, seller ID is `zonasulzsa`
- Search uses persisted GraphQL queries (not REST catalog API)
- Add-to-cart uses GraphQL mutation via `/_v/private/graphql/v1`
- Checkout uses standard VTEX REST API (`/api/checkout/pub/orderForm/...`)
- Auth cookie is `VtexIdclientAutCookie_zonasul` (JWT, 24h TTL, HttpOnly)
- Prices from VTEX are in centavos (879 = R$8.79)
- Delivery address: R. das Laranjeiras, 100 Apto 200, Laranjeiras, Rio de Janeiro, CEP 22240-003
- GraphQL query hashes and full API details are in `docs/zonasul-api-research.md`

## CLI Design Patterns

Follow steipete's agent-friendly CLI conventions:

- **Triple output**: `--json` for agents, `--plain` for piping, human-readable default
- **Stable exit codes**: 0=success, 1=error, 2=usage, 3=empty, 4=auth-required, 5=not-found, 6=min-order, 7=rate-limited
- **Stderr for humans, stdout for machines**: progress/warnings to stderr, data to stdout
- **Env var overrides**: `ZONASUL_JSON=1`, `ZONASUL_ACCOUNT` for config
- **`--no-input` flag**: disable interactive prompts for headless operation
- **`--confirm` safety gate**: required flag to actually place orders

## Commands

```
zonasul auth login/status/logout
zonasul search <query> [--limit N] [--json]
zonasul cart [add <sku> --qty N | remove <index> | clear]
zonasul delivery windows [--json]
zonasul checkout [--window N] [--payment N] [--confirm]
```

## Development

```sh
pnpm install   # N/A - this is Go
go build -o zonasul ./cmd/zonasul
go test ./...
```

## Important

- Never hardcode credentials or tokens in source
- The `docs/zonasul-api-research.md` file contains the full API surface captured from live network traffic - reference it for endpoint details, query hashes, and response shapes
- Persisted GraphQL hashes may change if Zona Sul updates their VTEX apps - if search or cart operations break, re-capture the hashes from the live site
