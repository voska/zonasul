---
name: zonasul-groceries
description: >-
  Order groceries from Zona Sul supermarket (zonasul.com.br) in Rio de Janeiro via CLI.
  Use when the user wants to search products, build a grocery list, manage a cart,
  plan meals from a recipe, or place a delivery order.
disable-model-invocation: true
argument-hint: "[grocery list or recipe]"
allowed-tools: Bash, Read
---

# Zona Sul Grocery Ordering

Order groceries from Zona Sul supermarket in Rio de Janeiro using the `zonasul` CLI. Supports search, cart management, local lists and favorites, delivery scheduling, and saved-card checkout.

First-time setup: see [SETUP.md](SETUP.md). Debugging: see [TROUBLESHOOTING.md](TROUBLESHOOTING.md).

## Prerequisites Check

Before anything, verify the CLI is installed, auth is valid, and the account can check out:

```bash
zonasul auth status --json
zonasul doctor --json
```

`auth status` exits 0 whether or not there is a valid session; read `status`. If it is `"expired"` or `"unauthenticated"`, the user needs to log in -- see [SETUP.md](SETUP.md). A password session renews itself on the next authenticated command.

`doctor` exits 0 only when the session is verified and the account has exactly one default delivery address and a saved credit card. Otherwise it exits 10 and names what is missing.

## Authentication

```bash
# Email/password (recommended -- the session renews itself)
zonasul auth login --email user@example.com --password pass

# Or via env vars
ZONASUL_EMAIL=user@example.com ZONASUL_PASSWORD=pass zonasul auth login --no-input

# Or an emailed one-time code (needs a new code when the session expires)
zonasul auth code send --email user@example.com
zonasul auth code verify --code 123456

# Or an existing storefront token; the email is checked against it
zonasul auth login --token <token> --email user@example.com
```

## Ordering Workflow

Money in JSON output is integer centavos (`1469` is R$14,69).

### Step 1: Search for Products

```bash
zonasul search "banana" --limit 5
zonasul search "feijao preto" --limit 5 --json
```

Search in Portuguese. Each result's `sku` is an opaque `product:<UUID>` handle: pass it whole to `product` and `cart add`.

`productId`, `skuId` and `reference` are different identifiers and no command takes them. Numeric SKUs from the old storefront are rejected with exit 2.

`price` is one pack. For `unit: "kg"` a pack weighs `unitMultiplier` kg and `pricePerUnit` is the price per kg. `available` is `null`: stock is unknown until the checkout preview.

### Step 2: Add Items to Cart

```bash
zonasul cart add product:<UUID> [--qty N]
zonasul cart add product:<UUID> --qty 3             # 3 packs
zonasul cart add product:<UUID> --qty 3 --dry-run   # show the planned write, change nothing
```

`--qty` is whole packs. A weight item is sent as packs × pack weight in kg, so `--qty 2` of a 1.2 kg pack orders 2.4 kg.

Adding a product that is already in the cart increases that line. Use `cart update` to set an exact quantity.

`list order`, `fav order` and `cart reorder` are disabled on the current storefront and exit 9 without sending anything.

### Step 3: Review Cart

```bash
zonasul cart show               # human-readable
zonasul cart show --json        # structured JSON
zonasul cart update 2 --qty 4   # set the item at index 2 to 4 packs; 0 removes
zonasul cart remove 2           # remove item at index 2
zonasul cart clear              # empty the cart
```

Every cart write is read back from the store. Exit 9 with `inspect cart before retrying` means the write could not be confirmed: run `cart show` before repeating anything.

In JSON, `quantity` is packs. `quantity: null` with `quantityVerified: false` means the line could not be expressed as whole packs; read `measuredQuantity`.

### Step 4: Check Delivery Windows

```bash
zonasul delivery windows --json
```

Lists the store's current windows for the open cart: `index`, `date`, `startTime`, `endTime`, `price`. Window indexes are positions in this list, so read it again right before checkout.

### Step 5: Place Order

Payment is a credit card saved on the account. There is no Pix, cash or CVV; `--cvv` is rejected.

Preview first. It is read-only:
```bash
zonasul checkout payments --json                                         # saved card IDs, masked numbers
zonasul checkout --window 0 --payment <saved-card-id> --dry-run --json
```

The preview prints `ready`, `blockers`, the selected `window` and totals. Pass `--window` every time: without it the CLI falls back to the window the cart already carries. The preview exits 9 when something blocks checkout: an empty cart, no delivery address on the cart, no saved card or an unknown `--payment` ID, no window, an unavailable item, or an items total under the store minimum (R$100 unless the cart's policy sets another).

Place the order:
```bash
zonasul checkout --window 0 --payment <saved-card-id> --confirm --json
```

`--confirm`, `--payment` and `--window` must be on the command line; their `ZONASUL_*` environment variables are ignored.

A confirmed checkout that reaches payment always exits 11 with `"status":"submitted_unverified"`, including when the order went through. **Never run checkout again after exit 11.** The CLI cannot see the card issuer's decision, so no exit code, `paymentResponseReceived` or order status proves the charge. Check it with the issuer or in the store account.

Any other non-zero exit from `--confirm` means no payment was sent.

Before it books the window, `--confirm` records the order ID under `~/.config/zonasul/payment-attempts/`. Once a confirmation has recorded the attempt, it cannot confirm that order again, including after a failed window update. A later run may report an earlier preflight blocker; if it reaches the record check, it exits 11 with `checkout attempt already recorded` without sending anything. Never delete, move or edit the record to force a retry: read the order with `zonasul orders <order-uuid>` and establish what happened. The records protect only invocations that share this state directory.

### Step 6: Verify Order

```bash
zonasul orders <order-uuid> --json
```

On 0.9.0, pass checkout's `cartId`, the submitted order UUID. The printed `orderId` can be the next empty cart. The order read returns `statusCode`, `value` and items; none proves issuer settlement. Listing order history is not available.

## Lists and Favorites

Named lists of product handles stored in `~/.config/zonasul/lists.json`:

```bash
zonasul list show                               # all lists
zonasul list show diarista                      # items in list
zonasul list add diarista product:<UUID>        # add a product
zonasul list remove diarista product:<UUID>     # remove a product
zonasul list delete diarista                    # delete list
```

Lists are local and hold whatever was saved. Numeric entries from the old storefront no longer identify a product; add the product again by its handle. To order a list, `cart add` each handle.

`fav` is shorthand for the `favorites` list. Zona Sul has no server-side
wishlist, so favorites are stored locally on this machine:

```bash
zonasul fav add product:<UUID>     # add to favorites
zonasul fav show                   # show favorites
```

## Recipe-Based Ordering

When the user gives a recipe or meal plan:

1. Identify ingredients in Portuguese grocery terms
2. Search each with `zonasul search`
3. Pick sensible defaults: cheapest for staples, mid-range for key ingredients
4. Present options to the user with prices
5. Check total against R$100 minimum -- suggest additions if under
6. Add all items and place order

Common Portuguese grocery terms:

| English | Portuguese |
|---------|-----------|
| Onion | Cebola |
| Garlic | Alho |
| Tomato | Tomate |
| Olive oil | Azeite |
| Black beans | Feijao preto |
| Rice | Arroz |
| Ground beef | Carne moida |
| Chicken breast | Peito de frango |
| Butter | Manteiga |
| Eggs | Ovos |
| Flour (wheat) | Farinha de trigo |
| Milk | Leite |
| Paper towels | Papel toalha |
| Garbage bags | Saco de lixo |

## Environment Variables

| Variable | Effect |
|----------|--------|
| `ZONASUL_JSON=1` | Force JSON output |
| `ZONASUL_PLAIN=1` | Force plain text output |
| `ZONASUL_NO_INPUT=1` | Disable interactive prompts |
| `ZONASUL_EMAIL` | Email for credential login |
| `ZONASUL_PASSWORD` | Password for credential login |
| `ZONASUL_BFF_TOKEN` | Storefront token for `auth login --token`; `ZONASUL_TOKEN` is ignored |
| `ZONASUL_CVV` | Ignored on BFF; never supply a CVV |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error |
| 2 | Usage/invalid args |
| 3 | No results / no open cart |
| 4 | Auth required (token expired) |
| 5 | Not found |
| 6 | Permission denied |
| 7 | Rate limited |
| 8 | Retryable (transient error) |
| 9 | Pre-submit business-rule refusal or unavailable command; read the error |
| 10 | Configuration error |
| 11 | Checkout attempt recorded, settlement unverified; never repeat checkout |
| 80 | Command line did not parse (unknown flag or command, missing argument); help is printed |
