# Zona Sul CLI

A CLI for [Zona Sul](https://www.zonasul.com.br) supermarket in Rio de Janeiro.

**Ordering is disabled in this build.** Zona Sul moved to a new storefront. This
build restores authentication; cart and checkout migration is still pending.

## Authentication

Password login uses the current storefront backend, verifies the live account
identity, and saves the session in the OS keyring:

```sh
zonasul auth login --no-input  # ZONASUL_EMAIL and ZONASUL_PASSWORD
zonasul auth status --json
```

Email-code login is also available:

```sh
zonasul auth code send --email user@example.com
zonasul auth code verify --code 123456
```

A successful send means the store accepted the request; check the mailbox to
verify delivery. Email-code sessions require another code when they expire.
Password sessions can refresh through the new backend. In this build, `doctor`
is the command that uses automatic refresh.

`auth login --token <token> --email user@example.com` accepts a token for the new
backend only and verifies its identity before saving it. Its environment variable
is `ZONASUL_BFF_TOKEN`; the old `ZONASUL_TOKEN` VTEX cookie is ignored. New sessions
are stored separately from VTEX secrets and are never sent to VTEX.

`auth logout` clears both the new storefront session and legacy VTEX credentials.

## Available commands

| Command | Availability |
| --- | --- |
| `auth login`, `auth code send`, `auth code verify` | Current storefront |
| `auth status`, `auth logout` | Current and local session state |
| `doctor` | Reads account address/payment prerequisites; always exits 10 because CLI checkout is unavailable |
| `search`, `product` | Legacy VTEX catalog only |
| `list show/add/remove/delete`, `fav show/add/remove` | Local lists |
| `store info` | Legacy catalog descriptor |
| `schema`, `exit-codes`, `agent exit-codes` | Command and exit-code reference |
| Cart, delivery, checkout, orders, subscriptions, `list order`, `fav order` | Disabled, exit 9; no cart or order request is sent |

New storefront product IDs are UUIDs. Legacy catalog SKUs and `sku.reference` on
the new storefront are different identifiers. Existing lists need remapping
before they can be ordered. Legacy catalog freshness is not verified against the
new storefront's regionalized prices or stock. Saved payment methods and delivery
addresses also need to be checked on the migrated account.

## Install

The authentication migration described above is unreleased. Package managers
currently install the previous VTEX release.

```sh
brew install voska/tap/zonasul
```

Other options are [release binaries](https://github.com/voska/zonasul/releases),
`go install github.com/voska/zonasul/cmd/zonasul@latest`, or Scoop:

```powershell
scoop bucket add voska https://github.com/voska/scoop-bucket
scoop install zonasul
```

## Output and exit codes

Data goes to stdout; hints and errors go to stderr. Use `--json` for structured
output, `--plain` for tab-separated output, `--quiet` for bare values, and
`--select` or `--fields` to project fields. `--no-input` disables prompts and is
automatic when stdin is not a terminal.

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | General error |
| 2 | Usage error |
| 3 | Empty result |
| 4 | Authentication required or rejected |
| 5 | Resource not found |
| 6 | Access forbidden |
| 7 | Rate limited |
| 8 | Transient request failure |
| 9 | Store rule or unavailable migration command; read the error before retrying |
| 10 | Configuration error or not ready to order |

## Development

```sh
make build
make ci  # format, lint, vet, race tests, build
```

The shared `github.com/voska/vtexkit` dependency continues to supply the command
schema, output formatting, legacy catalog reads and local-list operations. The
current storefront's authentication is implemented in `internal/bff` and
`internal/command`.

## License

MIT
