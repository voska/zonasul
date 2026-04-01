# Zona Sul CLI

Go CLI for ordering groceries from Zona Sul supermarket (zonasul.com.br) in Rio de Janeiro. Designed for AI agent consumption following steipete CLI patterns (gogcli, bird, imsg).

## Project Structure

```
cmd/zonasul/       # Thin CLI entry point (~40 lines)
internal/
  cmd/             # Kong command structs (auth, search, cart, list, fav, delivery, checkout, orders, agent, schema)
  vtex/            # VTEX API client (auth, search, cart, checkout, orders)
  config/          # Config + credentials + lists (~/.config/zonasul/)
  errfmt/          # Typed errors with exit codes
  outfmt/          # Triple output mode (human/plain/json) + colored stderr
skills/zonasul/    # Claude Code agent skill
docs/              # API research, design docs
```

## Tech Stack

- **Language**: Go (single static binary)
- **CLI framework**: Kong (struct-tag-based parser)
- **Platform**: VTEX IO (Zona Sul's ecommerce backend)
- **Auth**: Credential login via custom OAuth + JWT stored in macOS Keychain, auto-refresh on expiry
- **Output**: muesli/termenv for colored stderr

## Key Context

- Zona Sul's VTEX account name is `zonasul`, seller ID is `zonasulzsa`
- Search uses persisted GraphQL queries (not REST catalog API)
- Add-to-cart uses GraphQL mutation via `/_v/private/graphql/v1`
- Checkout uses standard VTEX REST API (`/api/checkout/pub/orderForm/...`)
- Auth cookie is `VtexIdclientAutCookie_zonasul` (JWT, 24h TTL, HttpOnly)
- Auth flow: classic VTEX auth is disabled; login goes through `autenticacao.zonasul.com.br/api/login` (Laravel app) then follows OAuth redirect chain to extract JWT
- Credentials stored: email in `~/.config/zonasul/credentials.json`, password in keychain (`zonasul-cli/login-password`)
- Auto-refresh: `AuthedClient()` detects expired tokens and re-authenticates using stored credentials
- Prices from VTEX are in centavos (879 = R$8.79)
- Delivery address: R. das Laranjeiras, 100 Apto 200, Laranjeiras, Rio de Janeiro, CEP 22240-003
- GraphQL query hashes and full API details are in `docs/zonasul-api-research.md`

## CLI Design Patterns

Follow steipete's agent-friendly CLI conventions:

- **Triple output**: `--json` for agents, `--plain` for piping, human-readable default
- **Stable exit codes**: 0=ok, 1=error, 2=usage, 3=empty, 4=auth, 5=not-found, 6=forbidden, 7=rate-limited, 8=retryable, 9=min-order, 10=config
- **Stderr for humans, stdout for machines**: progress/warnings to stderr (colored via termenv), data to stdout
- **Env var overrides**: `ZONASUL_JSON=1`, `ZONASUL_PLAIN=1`, `ZONASUL_NO_INPUT=1`
- **`--no-input` flag**: disable interactive prompts for headless operation
- **`--confirm` safety gate**: required flag to actually place orders

## Commands

```
zonasul auth login [--email X --password Y | --token JWT]
zonasul auth status/logout
zonasul search <query> [--limit N] [--json]
zonasul cart [show | add <sku> --qty N | remove <index> | clear | reorder [order-id]]
zonasul list [show [name] | add <name> <sku> | remove <name> <sku> | order <name> | delete <name>]
zonasul fav [show | add <sku> | remove <sku> | order]
zonasul delivery windows [--json]
zonasul checkout [--window N] [--payment pix|credit|cash|vr|alelo|ticket] [--cvv XXX] [--confirm]
zonasul orders [--json]
zonasul agent exit-codes [--json]
zonasul schema [--json]
```

## Config Files

All config in `~/.config/zonasul/` (dir 0700, files 0600):

| File | Contents |
|------|----------|
| `config.json` | Address, orderFormId |
| `credentials.json` | Email for auto-refresh (password in keychain) |
| `lists.json` | Named SKU lists (`{"diarista": ["39908","135"], "favorites": ["18868"]}`) |

## Development

```sh
make build    # Build to bin/zonasul (version injected via LDFLAGS)
make test     # Run tests with race detector
make lint     # Run golangci-lint
make vet      # Run go vet
make fmt      # Format code
```

## Important

- Never hardcode credentials or tokens in source
- The `docs/zonasul-api-research.md` file contains the full API surface captured from live network traffic - reference it for endpoint details, query hashes, and response shapes
- Persisted GraphQL hashes may change if Zona Sul updates their VTEX apps - if search or cart operations break, re-capture the hashes from the live site
- The custom OAuth flow (`CredentialLogin` in `vtex/auth.go`) drives `autenticacao.zonasul.com.br` -- if Zona Sul changes their auth app, this will break first
