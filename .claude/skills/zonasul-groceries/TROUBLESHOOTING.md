# Troubleshooting

## "auth required" (exit code 4)

Token expired. Re-run `./zonasul auth login` and paste a fresh JWT from the browser.

## "min-order" (exit code 6)

Cart total is under R$100. Add more items.

## Search returns unexpected results

Search in Portuguese. Try singular forms and common brand names.

## Cart add fails

Check that the SKU exists and is in stock: `./zonasul search "product name" --json`

## Delivery windows empty

The address or shipping SLA isn't set. This usually resolves when checkout is run (it sets the address automatically).

## Checkout "place order" error

The CLI needs the browser's orderFormId in `~/.config/zonasul/config.json` for saved card access. See [SETUP.md](SETUP.md) step 4.

## Technical Notes

- Prices are in centavos internally (879 = R$8.79) but displayed as reais
- VTEX session: the CLI sets `shippingOption: "Entrega Zona Sul"` via `/api/sessions`
- Credit card payment goes through `zonasul.vtexpayments.com.br` gateway with `validationCode` field for CVV
- The gateway callback at `/api/checkout/pub/gatewayCallback/{orderGroup}` is polled until 200/204
- Persisted GraphQL hashes may change if Zona Sul updates their VTEX apps
