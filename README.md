# Zona Sul CLI

A CLI for [Zona Sul](https://www.zonasul.com.br) supermarket in Rio de Janeiro.

Zona Sul moved to a new storefront. Authentication, catalog, cart, delivery
windows and saved-credit-card checkout now use its BFF API.

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
Password sessions refresh through the new backend when an authenticated command
needs a session.

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
| `doctor` | Checks identity, delivery address and saved credit card; exit 10 when a prerequisite is missing |
| `search`, `product` | Current BFF catalog |
| `cart show/add/update/remove/clear` | BFF cart; mutations verify the complete item list by readback |
| `delivery windows` | Current cart's live window grade |
| `checkout payments` | Saved credit card IDs and masked numbers |
| `checkout`, `checkout --dry-run` | Read-only payment-step preview; exit 9 with blockers when not ready |
| `checkout --confirm` | Explicit saved-card submission; see the settlement limitation below |
| `orders <UUID>` | Read one BFF order; status alone does not prove issuer settlement |
| `list show/add/remove/delete`, `fav show/add/remove` | Local lists |
| `store info` | Legacy catalog descriptor |
| `schema`, `exit-codes`, `agent exit-codes` | Command and exit-code reference |
| `cart reorder`, `delivery simulate`, subscriptions, `list order`, `fav order` | Disabled, exit 9 |

## Cart and checkout

Search returns an opaque `product:<UUID>` handle in the existing string `sku`
field. Use that handle with `product` and `cart add`. The additional `skuId` is
the BFF's distinct cart-line identifier. Numeric legacy SKU IDs and BFF
`sku.reference` values cannot be substituted for either identifier. Existing
local lists need remapping before ordering.

```sh
zonasul search banana --json
zonasul product product:<UUID> --json
zonasul cart add product:<UUID> --qty 2 --dry-run
zonasul cart show --json
zonasul cart update 0 --qty 3 --dry-run
zonasul delivery windows --json
zonasul checkout --window 0 --dry-run --json
zonasul checkout payments --json
```

`--qty` and cart `quantity` remain integer **packs**. Weight items are sent as
kg (`packs × arithmeticFactor`); cart output adds `measuredQuantity` and
`unitOfMeasure`. Prices remain integer centavos per pack; catalog `pricePerUnit`
is the BFF selling price per kg for weight items. Window `price` remains integer
centavos, and `start`/`end` include the store's UTC−03 offset. Search cannot
verify regionalized stock and reports `availabilityVerified:false`; check the
cart before checkout. Unknown catalog `available` and non-integral cart pack
`quantity` are `null`; `quantityVerified:false` preserves the measured quantity
for inspection. Choose the window from the current grade; availability
and delivery charges can change.

All previews and cart dry-runs leave the cart/window/payment unchanged. A new
cart uses the saved default address and scheduled delivery policy. The BFF's
two-step regionalization can leave an empty extra order; the CLI uses and
re-reads the second returned order ID.

Submission requires `--confirm --payment <saved-card-id>` and a valid delivery
window. `--confirm --dry-run` is rejected. This backend uses the saved credit
card and does not accept `--cvv`; Pix and cash are unsupported. There is no
automatic retry after a payment attempt.

**Settlement limitation:** the migrated BFF issuer-approval response has not
been verified. An acknowledged submission prints `submitted_unverified`,
`settlementVerified:false`, and exits 11 even if the order status is 300
(PaymentApproved). Read the order and independently verify the issuer evidence;
never repeat checkout to resolve an uncertain payment. This release does not
claim a settled purchase from an HTTP acknowledgement or order status alone.

## Install

The BFF migration is available from the release binaries once tagged; package
managers follow the release workflow.

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
| 11 | Payment attempted, settlement unverified; never retry checkout |

## Development

```sh
make build
make ci  # format, lint, vet, race tests, build
```

The shared `github.com/voska/vtexkit` dependency continues to supply the command
schema, output formatting and local-list operations. The current storefront
integration is implemented in `internal/bff` and `internal/command`.

## License

MIT
