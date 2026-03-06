# First-Time Setup

## 1. Build the CLI

```bash
make build
```

Requires Go 1.25+. If using [mise](https://mise.jdx.dev/), Go is managed automatically.

## 2. Create a Zona Sul Account

If the user doesn't have an account:
1. Open https://www.zonasul.com.br in a browser
2. Click the person icon > "Cadastre-se"
3. Register with CPF, email, name, phone, and delivery address
4. Zona Sul delivers to select neighborhoods in Rio de Janeiro (Leblon, Ipanema, Copacabana, Botafogo, etc.)

## 3. Authenticate

Zona Sul uses a custom OAuth provider — classic email/password login is disabled on their VTEX backend. The CLI stores a JWT token in macOS Keychain.

```bash
./zonasul auth login
```

Choose option 1 (paste JWT from browser):
1. Open https://www.zonasul.com.br and log in normally
2. Open DevTools > Application > Cookies > `www.zonasul.com.br`
3. Copy the value of `VtexIdclientAutCookie_zonasul`
4. Paste into the CLI prompt

The token lasts 24 hours. Re-run `./zonasul auth login` when it expires.

## 4. Set Up Saved Credit Card

The CLI supports credit card checkout with a saved card. To save a card:
1. Open https://www.zonasul.com.br/checkout/#/payment in the browser (with items in cart)
2. Select "Cartao de credito" and fill in your card details
3. Check "Salvar este cartao de forma segura para proximas compras"
4. Complete one order through the browser to save the card

Then configure the CLI to use the browser's orderForm (which has the saved card):
1. In the browser, open DevTools > Application > Cookies
2. Find `checkout.vtex.com` cookie > copy the `__ofid=` value (the orderFormId)
3. Save it to the CLI config:

```bash
echo '{"orderFormId":"PASTE_ORDER_FORM_ID_HERE"}' > ~/.config/zonasul/config.json
```

## 5. CVV Access

For credit card checkout, the CVV is required each time. Options:

- **Flag**: `./zonasul checkout --cvv XXX --confirm`
- **Environment variable**: `ZONASUL_CVV=XXX`

Never store your CVV in plaintext config files.
