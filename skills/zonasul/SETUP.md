# First-Time Setup

## 1. Install

```bash
brew install voska/tap/zonasul
```

Or `go install github.com/voska/zonasul/cmd/zonasul@latest`. Requires Go 1.25+.

## 2. Create a Zona Sul Account

If the user doesn't have an account:
1. Open https://www.zonasul.com.br
2. Click the person icon > "Cadastre-se"
3. Register with CPF, email, name, phone, and delivery address
4. Zona Sul delivers to select neighborhoods in Rio de Janeiro

## 3. Authenticate

The recommended method stores a session that renews itself:

```bash
zonasul auth login --email you@example.com --password yourpass
```

This logs in to the current storefront, checks that the session belongs to that email, and stores the session, with the password used to renew it, as one entry in the system keychain.

Alternative methods:

```bash
# Via environment variables (CI/headless)
ZONASUL_EMAIL=you@example.com ZONASUL_PASSWORD=yourpass zonasul auth login --no-input

# Via an emailed one-time code (no automatic renewal)
zonasul auth code send --email you@example.com
zonasul auth code verify --code 123456

# Via an existing storefront token (manual, no automatic renewal)
zonasul auth login --token <token> --email you@example.com

# Interactive (prompts for email/password)
zonasul auth login
```

Verify auth:
```bash
zonasul auth status
```

## 4. Set Up Delivery Address and Saved Credit Card

A cart opened by the CLI delivers to the account's default address, and checkout charges a credit card saved on the account. A cart that was already open keeps the address it was opened with. Set both up in the browser at https://www.zonasul.com.br:

1. Save one delivery address and make it the default. The CLI refuses to open a cart when the account has no default address or more than one.
2. Save a credit card on the account.

The CLI never asks for a card number or a CVV. It lists saved cards by ID and masked number:

```bash
zonasul checkout payments --json
```

Then check that everything is in place:

```bash
zonasul doctor
```

`doctor` exits 0 when the session, the default address and a saved card all pass, and exits 10 naming what is missing otherwise.

## 5. Headless / SSH

On headless macOS (e.g., SSH), the keychain must be unlocked first:

```bash
security unlock-keychain -p '<user-password>' ~/Library/Keychains/login.keychain-db
```

After unlocking, all zonasul commands work normally over SSH.
