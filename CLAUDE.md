# Zona Sul CLI

Go CLI for ordering groceries from Zona Sul supermarket (zonasul.com.br) in Rio de Janeiro.

All logic lives in the shared library **`github.com/voska/vtexkit`**, which also powers the
Frescatto CLI. This repo holds the store descriptor and Zona Sul's custom OAuth driver.

## Project Structure

```
store.go             # The Zona Sul descriptor
oauth.go             # store.OAuthDriver for their custom auth app
cmd/zonasul/         # Entry point, ~15 lines
skills/zonasul/      # Claude Code agent skill
site/ docs/          # Landing page, API research
```

## Why this descriptor is bigger than Frescatto's

Three facts about Zona Sul cannot be discovered from the API:

- **MinOrder R$100** — enforced, but appears nowhere in `storePreferencesData`; it
  surfaces only as a checkout error string.
- **Custom OAuth** — classic password auth is disabled on this account, so login is
  driven through `autenticacao.zonasul.com.br` (a Laravel app) and the OAuth redirect
  chain back to a VTEX JWT. This breaks first if they change that app.
- **Quirks** — the payment gateway rejects cards without a ClearSale device fingerprint
  (Cielo code 59), and needs the `gatewayCallback` poll to settle a payment.

Everything else — payment systems, seller IDs, delivery SLAs, the login method — is
discovered at runtime.

## Search

Uses Intelligent Search REST with catalog REST fallback. This replaced the persisted
GraphQL query used through v0.5.0, whose pinned SHA-256 hash VTEX rotates on every
`search-graphql` release (it went stale once already, commit `b76608c`). REST returns
identical results and cannot go stale.

## Compatibility

v0.5.0 is public. The binary name, command surface, `~/.config/zonasul/`, keyring service
`zonasul-cli`, and every exit code are unchanged. `agent exit-codes` still works as a
hidden alias for the new `exit-codes`.

## Development

```sh
make build   # bin/zonasul
make test
make ci
```

vtexkit resolves through a `replace` to `../vtexkit` until it is published.

## Important

- Never commit credentials or tokens.
- `checkout --confirm` is the only path that places a real order.
- Exit code 9 is now the generic `domain_error`, not specifically "below minimum".
