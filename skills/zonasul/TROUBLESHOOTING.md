# Troubleshooting

## "auth required" (exit code 4)

Token expired and auto-refresh failed. Re-run `zonasul auth login` with email/password. If credentials are already stored, check that the keychain is accessible (see SSH section below).

## "credential login failed" during auto-refresh

Zona Sul may have changed their custom OAuth app at `autenticacao.zonasul.com.br`. Try logging in through the browser and using `--token` as a fallback.

## "failed to store token in keychain" (exit code 10)

The macOS keychain is locked. Over SSH, unlock it first:

```bash
security unlock-keychain -p '<user-password>' ~/Library/Keychains/login.keychain-db
```

## "min-order" (exit code 9)

Cart total is under R$100. Add more items. The exit code was corrected from 6 to 9.

## Search returns unexpected results

Search in Portuguese. Try singular forms and common brand names. Example: "leite" not "milk".

## Cart add fails

Check that the SKU exists and is in stock: `zonasul search "product name" --json`. Out-of-stock items show `available: 0`.

## List or fav commands error

Lists are stored in `~/.config/zonasul/lists.json`. If the file is corrupted, delete it and rebuild:

```bash
rm ~/.config/zonasul/lists.json
zonasul list add mylist 12345
```

## Delivery windows empty

The address or shipping SLA isn't set. This usually resolves when checkout is run (it sets the address automatically).

## Checkout "place order" error

The CLI needs the browser's orderFormId in `~/.config/zonasul/config.json` for saved card access. See [SETUP.md](SETUP.md) step 4.

## Cart reorder fails

The order detail API may not return SKUs for very old orders. Try with a recent order ID: `zonasul cart reorder <order-id>`.

## Technical Notes

- Prices are in centavos internally (879 = R$8.79) but displayed as reais
- Auth flow: VTEX classic auth is disabled; the CLI drives `autenticacao.zonasul.com.br/api/login` (a Laravel app), then follows the OAuth redirect chain to extract the JWT from the `accountAuthCookieValue` query parameter
- VTEX session: the CLI sets `shippingOption: "Entrega Zona Sul"` via `/api/sessions`
- Credit card payment goes through `zonasul.vtexpayments.com.br` gateway
- The gateway callback at `/api/checkout/pub/gatewayCallback/{orderGroup}` is polled until 200/204
- Persisted GraphQL hashes may change if Zona Sul updates their VTEX apps
