---
name: zonasul-groceries
description: >-
  Order groceries from Zona Sul supermarket (zonasul.com.br) in Rio de Janeiro via CLI.
  Use when the user wants to search products, build a grocery list, manage a cart,
  plan meals from a recipe, reorder previous orders, or place a delivery order.
disable-model-invocation: true
argument-hint: "[grocery list or recipe]"
allowed-tools: Bash, Read
---

# Zona Sul Grocery Ordering

Order groceries from Zona Sul supermarket in Rio de Janeiro using the `zonasul` CLI. Supports search, cart management, lists, favorites, reorder, delivery scheduling, and checkout.

First-time setup: see [SETUP.md](SETUP.md). Debugging: see [TROUBLESHOOTING.md](TROUBLESHOOTING.md).

## Prerequisites Check

Before anything, verify the CLI is installed and auth is valid:

```bash
zonasul auth status --json
```

If auth returns `"status":"expired"` or `"status":"unauthenticated"`, the user needs to log in -- see [SETUP.md](SETUP.md). If credentials are stored, expired tokens auto-refresh on next command.

## Authentication

```bash
# Email/password (recommended -- stores credentials for auto-refresh)
zonasul auth login --email user@example.com --password pass

# Or via env vars
ZONASUL_EMAIL=user@example.com ZONASUL_PASSWORD=pass zonasul auth login

# Or paste JWT directly
zonasul auth login --token <jwt>
```

Once logged in with email/password, the CLI auto-refreshes expired tokens. No manual re-auth needed.

## Ordering Workflow

### Step 1: Search for Products

```bash
zonasul search "banana" --limit 5
zonasul search "feijao preto" --limit 5 --json
```

Search in Portuguese. Results include SKU ID, name, price, and availability.

### Step 2: Add Items to Cart

```bash
zonasul cart add <SKU> [--qty N]
zonasul cart add 33277 --qty 3    # Banana Prata x3
```

Or add from a saved list:

```bash
zonasul list order diarista        # add all items in "diarista" list
zonasul fav order                  # add all favorites
zonasul cart reorder               # re-add items from most recent order
zonasul cart reorder <order-id>    # re-add from specific order
```

### Step 3: Review Cart

```bash
zonasul cart              # human-readable
zonasul cart --json       # structured JSON
zonasul cart clear        # empty the cart
zonasul cart remove 2     # remove item at index 2
```

### Step 4: Check Delivery Windows

```bash
zonasul delivery windows
```

Wide 5-hour windows are free, tight 2-hour windows cost R$7.00.

### Step 5: Place Order

Preview first (no `--confirm`):
```bash
zonasul checkout --window 0
```

Place the order:
```bash
zonasul checkout --window 0 --confirm                              # Pix (default)
zonasul checkout --window 0 --payment credit --cvv XXX --confirm   # credit card
```

Minimum order is R$100.

### Step 6: Verify Order

```bash
zonasul orders
zonasul orders --json
```

## Lists and Favorites

Named SKU lists stored in `~/.config/zonasul/lists.json`:

```bash
zonasul list show                  # all lists
zonasul list show diarista         # items in list
zonasul list add diarista 39908    # add SKU
zonasul list remove diarista 39908 # remove SKU
zonasul list order diarista        # add all to cart
zonasul list order diarista --qty 2 # all with qty 2
zonasul list delete diarista       # delete list
```

`fav` is shorthand for the `favorites` list. Zona Sul has no server-side
wishlist, so favorites are stored locally on this machine:

```bash
zonasul fav add 18868              # add to favorites
zonasul fav                        # show favorites
zonasul fav order                  # add all to cart
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
| `ZONASUL_TOKEN` | JWT token (skip credential login) |
| `ZONASUL_CVV` | Credit card CVV |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error |
| 2 | Usage/invalid args |
| 3 | No results / empty cart |
| 4 | Auth required (token expired) |
| 5 | Not found |
| 6 | Permission denied |
| 7 | Rate limited |
| 8 | Retryable (transient error) |
| 9 | Below R$100 minimum |
| 10 | Configuration error |
