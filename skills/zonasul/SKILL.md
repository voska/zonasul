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

Order groceries from Zona Sul supermarket in Rio de Janeiro using the `zonasul` CLI. Supports search, cart management, delivery scheduling, and credit card checkout.

First-time setup: see [SETUP.md](SETUP.md). Debugging: see [TROUBLESHOOTING.md](TROUBLESHOOTING.md).

## Prerequisites Check

Before anything, verify the CLI is built and auth is valid:

```bash
make build
bin/zonasul auth status --json
```

If auth returns `"status":"expired"` or `"status":"unauthenticated"`, the user needs to re-authenticate — see [SETUP.md](SETUP.md).

## Ordering Workflow

### Step 1: Search for Products

```bash
./zonasul search "banana" --limit 5
./zonasul search "feijao preto" --limit 5 --json
```

Search in Portuguese. Results include SKU ID, name, price, and availability. Out-of-stock items are labeled.

### Step 2: Add Items to Cart

```bash
./zonasul cart add <SKU> [--qty N]
```

Examples:
```bash
./zonasul cart add 33277 --qty 3    # Banana Prata x3
./zonasul cart add 11240            # Feijao Preto 1kg
```

The seller must be `zonasulzsa`. The CLI handles this automatically.

### Step 3: Review Cart

```bash
./zonasul cart              # Human-readable
./zonasul cart --json       # Structured JSON
./zonasul cart clear        # Empty the cart
./zonasul cart remove 2     # Remove item at index 2
```

### Step 4: Check Delivery Windows

```bash
./zonasul delivery windows
```

Shows available time slots. Tight 2-hour windows cost R$7.00, wider 5-hour windows are free. Today's windows appear first.

### Step 5: Place Order

Preview first (no `--confirm`):
```bash
./zonasul checkout --window 0
```

Place the order with credit card:
```bash
./zonasul checkout --window 0 --cvv XXX --confirm
```

The `--window` index corresponds to the delivery windows listing. Minimum order is R$100 for scheduled (AGENDADA) delivery.

The CVV can be passed via flag (`--cvv XXX`) or env var (`ZONASUL_CVV`).

### Step 6: Verify Order

```bash
./zonasul orders
./zonasul orders --json
```

Order statuses: "Pagamento Aprovado" (approved), "Faturado" (invoiced/shipped).

## Recipe-Based Ordering

When the user gives a recipe or meal plan:

1. **Identify ingredients** in Portuguese grocery terms
2. **Search each** with `./zonasul search`
3. **Pick sensible defaults**: cheapest for staples, mid-range for key ingredients
4. **Present options** to the user with prices
5. **Check total** against R$100 minimum — suggest additions if under
6. **Add all items** and place order

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
| Cassava flour | Farinha de mandioca |
| Parsley | Salsinha |
| Green pepper | Pimentao verde |
| Black pepper | Pimenta do reino |
| Vinegar | Vinagre |
| Paper towels | Papel toalha |
| Garbage bags | Saco de lixo |

## Environment Variables

| Variable | Effect |
|----------|--------|
| `ZONASUL_JSON=1` | Force JSON output |
| `ZONASUL_PLAIN=1` | Force plain text output |
| `ZONASUL_NO_INPUT=1` | Disable interactive prompts |
| `ZONASUL_TOKEN` | JWT token (skip keychain) |
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
