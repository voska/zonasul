# Zona Sul CLI

Go CLI for ordering groceries from Zona Sul supermarket (zonasul.com.br) in Rio de Janeiro.

Zona Sul left VTEX IO for its own storefront. Login, catalog, cart, delivery windows,
saved-card checkout and the single-order read now talk to that storefront's BFF
(`https://www.zonasul.com.br/api/bff`) from code in this repo, under `internal/`.

The shared library **`github.com/voska/vtexkit`**, which also powers the Frescatto CLI,
still supplies the command tree, output and error formatting, money types, and the local
`list`/`fav` and `schema` commands. Its VTEX auth, cart and checkout code is not on any
ordering path here; `store info` is the one command that still reads the legacy VTEX
descriptor.

## Project Structure

```
internal/bff/        # BFF client: login, catalog, cart, windows, card checkout
internal/command/    # Command handlers and routing; disables what the BFF path lacks
store.go             # Legacy VTEX descriptor; still names the store and keyring service
oauth.go             # Dormant store.OAuthDriver for the old VTEX login
cmd/zonasul/         # Entry point: routes to internal/command, then to vtexkit
skills/zonasul/      # Claude Code agent skill
site/ docs/          # Landing page, API research
```

## The legacy VTEX descriptor

`store.go` and `oauth.go` describe the old VTEX storefront. No ordering command uses
them now; do not read this section as how login or checkout works today.

Three facts about the VTEX store could not be discovered from its API:

- **MinOrder R$100** — enforced, but appears nowhere in `storePreferencesData`; it
  surfaces only as a checkout error string.
- **Custom OAuth** — classic password auth is disabled on this account, so login is
  driven through `autenticacao.zonasul.com.br` (a Laravel app) and the OAuth redirect
  chain back to a VTEX JWT. This breaks first if they change that app.
- **Quirks** — the payment gateway rejects cards without a ClearSale device fingerprint
  (Cielo code 59), and needs the `gatewayCallback` poll to settle a payment.

Everything else — payment systems, seller IDs, delivery SLAs, the login method — is
discovered at runtime.

## Favorites and lists

`fav` and `list` are local, stored in `~/.config/zonasul/lists.json`; there is no
server-side wishlist to read. They do not check that an entry identifies a product, so new entries
should be `product:<UUID>` handles. Numeric entries saved against the VTEX catalog
identify nothing on the current storefront. `list order` and `fav order` are disabled.

## Search

`search` and `product` read the BFF catalog (`/products/search`, `/products/<id>`).
The `sku` field carries an opaque `product:<UUID>` handle. `skuId` is the BFF's separate
cart-line identifier and `reference` is not the old numeric SKU; neither is accepted in
place of the handle.

## Compatibility

v0.5.0 is public. The binary name, `~/.config/zonasul/`, keyring service `zonasul-cli`
and exit codes 0–10 are unchanged. `agent exit-codes` still works as a hidden alias for
the new `exit-codes`.

The BFF migration changed the rest of the surface: commands take `product:<UUID>` handles
instead of numeric SKUs, checkout pays with a saved card ID only, `orders` reads one order
by UUID, exit 11 (`submitted_unverified`) was added, and `cart reorder`, `list order`,
`fav order`, `delivery simulate` and `subs` exit 9 without sending anything.

## Development

```sh
make build   # bin/zonasul
make test
make ci
```

## Important

- Never commit credentials or tokens.
- `checkout --confirm` is the only path that places a real order. Without it, checkout
  is a read-only preview. `--confirm`, `--payment` and `--window` are read from the
  command line only; their `ZONASUL_*` environment variables are ignored.
- Before its first checkout write, `--confirm` records the order ID under
  `~/.config/zonasul/payment-attempts/`. Once a confirmation has recorded the attempt, it
  cannot confirm that order again, including after a failed window update. A later run may
  report an earlier preflight blocker; if it reaches the record check, it exits 11 without
  sending anything. The record is never removed by the CLI and covers only invocations
  sharing that state directory.
- Exit code 9 is now the generic `domain_error`, not specifically "below minimum".
- A payment attempt always exits 11 and is never retried. The CLI cannot see the issuer's
  decision, so nothing in its output may be reported as settlement.
