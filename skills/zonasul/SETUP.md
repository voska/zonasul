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

The recommended method stores credentials for automatic token refresh:

```bash
zonasul auth login --email you@example.com --password yourpass
```

This performs the full Zona Sul OAuth flow, stores the email in `~/.config/zonasul/credentials.json` and the password in the system keychain. The JWT (24h TTL) is also stored in the keychain and auto-refreshes when expired.

Alternative methods:

```bash
# Via environment variables (CI/headless)
ZONASUL_EMAIL=you@example.com ZONASUL_PASSWORD=yourpass zonasul auth login

# Via JWT token (manual, no auto-refresh)
zonasul auth login --token <VtexIdclientAutCookie_zonasul>

# Interactive (prompts for email/password)
zonasul auth login
```

Verify auth:
```bash
zonasul auth status
```

## 4. Set Up Saved Credit Card

To use credit card checkout, save a card through the browser first:
1. Open https://www.zonasul.com.br/checkout/#/payment with items in cart
2. Select "Cartao de credito" and fill in card details
3. Check "Salvar este cartao de forma segura para proximas compras"
4. Complete one order through the browser

Then link the browser's orderForm to the CLI:
1. In DevTools > Application > Cookies, find `checkout.vtex.com`
2. Copy the `__ofid=` value (the orderFormId)
3. Save it:

```bash
echo '{"orderFormId":"PASTE_ORDER_FORM_ID_HERE"}' > ~/.config/zonasul/config.json
```

## 5. CVV

For credit card checkout, the CVV is required each time:

```bash
zonasul checkout --cvv XXX --confirm
# or
ZONASUL_CVV=XXX zonasul checkout --confirm
```

Never store CVV in plaintext config files.

## 6. Headless / SSH

On headless macOS (e.g., SSH), the keychain must be unlocked first:

```bash
security unlock-keychain -p '<user-password>' ~/Library/Keychains/login.keychain-db
```

After unlocking, all zonasul commands work normally over SSH.
