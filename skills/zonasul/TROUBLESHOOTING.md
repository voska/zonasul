# Troubleshooting

## "auth required" (exit code 4)

There is no saved session, or it expired and could not be renewed. Re-run `zonasul auth login` with email/password. If credentials are already stored, check that the keychain is accessible (see SSH section below).

## "BFF login rejected" or "BFF authentication rejected"

The store refused the credentials or the saved session. Check the email and password by logging in at https://www.zonasul.com.br, then run `zonasul auth login` again. An emailed code (`zonasul auth code send --email <email>`, then `zonasul auth code verify --code <code>`) also works.

## "could not store Zona Sul session in OS keyring" (exit code 10)

The macOS keychain is locked. Over SSH, unlock it first:

```bash
security unlock-keychain -p '<user-password>' ~/Library/Keychains/login.keychain-db
```

## Exit code 9

Exit 9 is a refusal before anything was paid. Read the message:

- `command is disabled during the storefront migration`: `cart reorder`, `list order`, `fav order`, `delivery simulate` and subscriptions are not available on the current storefront. Nothing was sent.
- `inspect cart before retrying`: a cart write could not be confirmed by readback. Run `zonasul cart show --json` before repeating anything; `cart add` is additive.
- `checkout is not ready; see blockers`: the preview found a blocker. `below_minimum` means the items total is under the store minimum (R$100 unless the cart's policy sets another); add more items. `item_unavailable` means a cart line is out of stock.
- `delivery window readback did not match; payment was not sent`: no payment was sent, but the attempt is already recorded for this order, so it cannot be confirmed again: a later `--confirm` that gets past the preflight blockers exits 11. Read the order with `zonasul orders <order-uuid> --json` before deciding anything.

## Exit code 11

A payment was attempted and its settlement is unverified. This is the result of every confirmed checkout that reaches payment, including one that went through. Do not run checkout again. Record `cartId` as the submitted order UUID, then run `zonasul orders <cartId> --json`. In 0.9.0, checkout's `orderId` can identify the next empty cart. Its empty items or status 1 do not mean the submitted order failed. Never rebuild the order or repeat checkout because of that read. Confirm the charge with the card issuer or in the store account.

`checkout attempt already recorded` is the same exit code from a different cause: this run sent nothing, because an earlier `--confirm` of the same order left a record under `~/.config/zonasul/payment-attempts/`. The record stays after a failed window update or payment request, and the CLI has no command that clears it. Read the order and establish what the earlier run did; do not delete the record to force a retry.

## "could not persist checkout attempt" (exit code 10)

The CLI could not write its record under `~/.config/zonasul/payment-attempts/`, so it booked no window and sent no payment. Check that the directory is writable. A partly written record still counts as an attempt for that order.

## "use product:<UUID> from search" (exit code 2)

`product` and `cart add` take only the `sku` value from a current `zonasul search` result. Numeric SKUs from the old storefront, `productId`, `skuId` and `reference` are not accepted.

## Search returns unexpected results

Search in Portuguese. Try singular forms and common brand names. Example: "leite" not "milk".

## Cart add fails

Check the handle against a fresh search: `zonasul search "product name" --json`. Search shows `available: null` because stock is not known there; an out-of-stock item surfaces as `item_unavailable` in the checkout preview.

## List or fav commands error

Lists are stored in `~/.config/zonasul/lists.json`. If the file is corrupted, delete it and rebuild:

```bash
rm ~/.config/zonasul/lists.json
zonasul list add mylist product:<UUID>
```

## Delivery windows empty

`delivery windows` needs an open cart and exits 3 without one; add an item first. Exit 10 from `delivery windows` means the cart has no delivery method; choose delivery for it on zonasul.com.br. Exit 10 from `cart add` means the account has no default delivery address, or more than one; see [SETUP.md](SETUP.md) step 4.

## Checkout cannot find a card

`no_saved_card` means the account has no saved credit card; see [SETUP.md](SETUP.md) step 4. `select_saved_card` means `--payment` is not one of the IDs from `zonasul checkout payments --json`. Pix and cash are not supported.

## Order history

`zonasul orders` without an ID exits 2. Pass the submitted order's UUID: `zonasul orders <order-uuid>`. On 0.9.0 that is checkout's `cartId`; the printed `orderId` can be the next empty cart.

## Technical Notes

- Prices are integer centavos in JSON and `--plain` output (879 = R$8.79) and displayed as reais in the default output
- Backend: the storefront's own API at `www.zonasul.com.br/api/bff`. The session is saved only after the store confirms it belongs to the requested email
- Cart writes send the complete item list and read the order back
- Weight items are sent in kg (packs × pack weight)
- Opening a new cart binds it to the default delivery address
- Checkout records the attempt, books the delivery window, reads it back, submits the saved card once and never retries a payment
