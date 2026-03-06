# zonasul CLI — Agent Reference

AI-agent-friendly CLI for ordering groceries from Zona Sul supermarket (zonasul.com.br) in Rio de Janeiro.

## Quick Start

```bash
# Login (interactive or via env vars)
zonasul auth login --email user@example.com --password secret
# or: ZONASUL_EMAIL=... ZONASUL_PASSWORD=... zonasul auth login --no-input

# Search products
zonasul search "banana" --json --limit 10

# Add to cart by SKU
zonasul cart add 6180 --qty 2

# View cart
zonasul cart --json

# List delivery windows
zonasul delivery windows --json

# Place order (Pix payment, window 0)
zonasul checkout --window 0 --payment 125 --confirm
```

## Global Flags

| Flag | Env Var | Description |
|------|---------|-------------|
| `--json` | `ZONASUL_JSON=1` | JSON output to stdout (for agents) |
| `--plain` | `ZONASUL_PLAIN=1` | Plain text output (for piping) |
| `--no-input` | `ZONASUL_NO_INPUT=1` | Disable interactive prompts |

## Commands

### `auth login`

Login with email/password. Stores JWT in macOS Keychain.

```bash
zonasul auth login --email user@example.com --password secret
```

Flags: `--email`, `--password` (also via `ZONASUL_EMAIL`, `ZONASUL_PASSWORD`)

### `auth status`

Check authentication state. Exit code 4 if not logged in or token expired.

```bash
zonasul auth status --json
# {"status":"ok","user":"user@example.com"}
```

### `auth logout`

Clear stored credentials from keychain.

### `search <query>`

Search products. No auth required.

```bash
zonasul search "leite" --limit 5 --json
```

JSON output:
```json
[
  {
    "productId": "6196",
    "sku": "6180",
    "name": "Banana Prata Orgânica 800g",
    "price": 10.99,
    "listPrice": 10.99,
    "available": 99999,
    "unit": "kg",
    "unitMultiplier": 0.8
  }
]
```

Exit code 3 if no results.

### `cart`

Show current cart contents. Requires auth.

```bash
zonasul cart --json
```

### `cart add <sku>`

Add item to cart by SKU ID (from search results).

```bash
zonasul cart add 6180 --qty 3
```

### `cart remove <index>`

Remove item by cart index (0-based).

```bash
zonasul cart remove 0
```

### `cart clear`

Remove all items from cart.

### `delivery windows`

List available delivery time slots for scheduled delivery.

```bash
zonasul delivery windows --json
```

JSON output:
```json
[
  {
    "index": 0,
    "start": "2026-03-04T14:01:00Z",
    "end": "2026-03-04T16:00:59Z",
    "price": 700
  }
]
```

Prices in centavos (700 = R$7.00, 0 = free).

### `checkout`

Place an order. Requires `--confirm` flag as safety gate.

```bash
# Preview order (no --confirm)
zonasul checkout --window 0 --payment 125

# Actually place order
zonasul checkout --window 0 --payment 125 --confirm
```

| Flag | Default | Description |
|------|---------|-------------|
| `--window N` | -1 | Delivery window index from `delivery windows` |
| `--payment N` | 125 | Payment method ID (125=Pix, 2=Visa, 4=Mastercard) |
| `--confirm` | false | Required to actually place the order |

Exit code 6 if cart total < R$100 (AGENDADA minimum).

### `agent exit-codes`

Print exit code reference table.

```bash
zonasul agent exit-codes --json
```

## Exit Codes

| Code | Name | Meaning |
|------|------|---------|
| 0 | success | Operation completed |
| 1 | error | General error |
| 2 | usage | Invalid command/arguments |
| 3 | empty-results | No results found |
| 4 | auth-required | Not logged in or token expired |
| 5 | not-found | Resource not found |
| 6 | min-order | Cart below R$100 minimum |
| 7 | rate-limited | Too many requests |

## Payment Method IDs

| ID | Name |
|----|------|
| 125 | Pix |
| 2 | Visa |
| 4 | Mastercard |
| 1 | American Express |
| 9 | Elo |
| 201 | Dinheiro (cash) |

## Typical Agent Workflow

```bash
# 1. Authenticate
zonasul auth login --no-input  # with ZONASUL_EMAIL + ZONASUL_PASSWORD

# 2. Search for items
items=$(zonasul search "banana" --json --limit 5)

# 3. Add items by SKU
zonasul cart add 6180 --qty 1

# 4. Check cart
zonasul cart --json

# 5. Pick delivery window
windows=$(zonasul delivery windows --json)

# 6. Place order
zonasul checkout --window 0 --payment 125 --confirm --json
```
