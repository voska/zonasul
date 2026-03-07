# Zona Sul API Research

Captured 2026-03-03 via Chrome DevTools MCP on zonasul.com.br.

## Platform

Zona Sul runs on **VTEX IO** (not legacy VTEX CMS).

- Account name: `zonasul`
- Storefront domain: `www.zonasul.com.br`
- VTEX stable domain: `zonasul.vtexcommercestable.com.br`
- Asset CDN: `zonasul.vtexassets.com`
- Workspace: `master`
- Binding ID: `0a362f40-93e7-42c4-90c5-a3946de77fb3`
- Seller ID: `zonasulzsa`

Hortifruti (`hortifrutibr`) and Natural da Terra (`naturalterra`) also run on VTEX (FastStore frontend), so the same API patterns apply if we expand later.

## Authentication

### Login Flow

Zona Sul uses a **custom auth domain**: `autenticacao.zonasul.com.br/login`.

Login is a two-step form:
1. Enter CPF/CNPJ or email → click OK
2. Enter password → click Continue

This redirects back to `www.zonasul.com.br` with auth cookies set.

### Auth Cookies

| Cookie | Purpose | Scope |
|--------|---------|-------|
| `VtexIdclientAutCookie_zonasul` | JWT auth token (ES256). 24h TTL. Primary credential. | HttpOnly, sent automatically |
| `VtexIdclientAutCookie_{bindingId}` | Same JWT, duplicate under binding ID key | HttpOnly |
| `checkout.vtex.com` | Contains `__ofid={orderFormId}` - links session to cart | Accessible via JS |
| `vtex_session` | Session JWT for VTEX session management | HttpOnly |
| `vtex_segment` | Segment data (channel, region, currency) | Accessible via JS |

The JWT payload:
```json
{
  "sub": "user@example.com",
  "account": "zonasul",
  "audience": "webstore",
  "exp": 1772646950,          // 24h from login
  "type": "user",
  "userId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "iss": "token-emitter"
}
```

### VTEX Auth API Endpoints (for programmatic login)

```
GET  /api/vtexid/pub/authentication/start?scope=zonasul
POST /api/vtexid/pub/authentication/classic/validate
     Body: { "login": "{email}", "password": "{password}", "authenticationToken": "{from start}" }
     → Sets VtexIdclientAutCookie_zonasul
```

Also supports OTP (access key) flow via `/accesskey/send` and `/accesskey/validate`.

### Authenticated User Check

```
GET /api/vtexid/pub/authenticated/user
→ { "userId": "...", "user": "user@example.com", "account": "zonasul", "audience": "webstore" }
```

## Product Search

The storefront uses **VTEX Intelligent Search via persisted GraphQL queries**, not the legacy REST catalog API.

### Primary Search Endpoint

```
GET /_v/segment/graphql/v1?workspace=master&maxAge=short&appsEtag=remove
    &domain=store&locale=pt-BR
    &__bindingId=0a362f40-93e7-42c4-90c5-a3946de77fb3
    &operationName=productSearchV3
    &variables={}
    &extensions={"persistedQuery":{"version":1,"sha256Hash":"31d3fa494df1fc41efef6d16dd96a96e6911b8aed7a037868699a1f3f4d365de","sender":"vtex.store-resources@0.x","provider":"vtex.search-graphql@0.x"},"variables":"{base64-encoded JSON}"}
```

The `variables` field is base64-encoded JSON:
```json
{
  "hideUnavailableItems": true,
  "skusFilter": "ALL",
  "simulationBehavior": "default",
  "installmentCriteria": "MAX_WITHOUT_INTEREST",
  "productOriginVtex": true,
  "map": "ft",
  "query": "banana",
  "orderBy": "OrderByScoreDESC",
  "from": 0,
  "to": 47,
  "selectedFacets": [{"key": "ft", "value": "banana"}],
  "fullText": "banana",
  "facetsBehavior": "Static",
  "categoryTreeBehavior": "default",
  "withFacets": false,
  "variant": "null-null"
}
```

No auth required for search.

### Response Shape (per product)

```json
{
  "productId": "6196",
  "productName": "Banana Prata Orgânica 800g",
  "items": [
    {
      "itemId": "6180",              // This is the SKU ID used for add-to-cart
      "name": "Banana Prata Orgânica 800g",
      "nameComplete": "Banana Prata Orgânica 800g",
      "ean": "7369",
      "sellers": [
        {
          "sellerId": "1",
          "sellerName": "Super Mercado Zona Sul S/A",
          "commertialOffer": {
            "Price": 10.99,
            "ListPrice": 10.99,
            "AvailableQuantity": 99999
          }
        }
      ],
      "unitMultiplier": 0.8,
      "measurementUnit": "kg"
    }
  ]
}
```

A search for "banana" returned 48 products.

### Autocomplete

```
GET /_v/segment/graphql/v1?operationName=autocompleteSearchSuggestions
    sha256Hash: 069177eb2c038ccb948b55ca406e13189adcb5addcb00c25a8400450d20e0108
    sender: vtex.store-resources@0.x
    provider: vtex.search-graphql@0.x
```

Also:
```
GET /_v/segment/graphql/v1?operationName=productSuggestions
    sha256Hash: 3eca26a431d4646a8bbce2644b78d3ca734bf8b4ba46afe4269621b64b0fb67d
```

## Delivery Mode Selection

Before adding items, the site prompts for delivery mode selection. Four modes available for CEP 22240-003:

| Mode | Description | Min Order | Speed | Catalog |
|------|-------------|-----------|-------|---------|
| **AGENDADA** | Scheduled delivery | R$100 | Pick date/time | Full catalog |
| **+ RÁPIDO** | Fast delivery | R$17.90 | ~2h15 avg | Reduced |
| **JÁ** | Instant | R$17.90 | Up to 30 min | Express only (creates new cart!) |
| **RETIRADA EM LOJA** | Store pickup | Free | Up to 2h | Reduced |

AGENDADA is the target mode for automation (full catalog, schedulable).

The delivery mode selection is stored in the VTEX session (via `/api/sessions` POST with `public.shippingOption`).

## Add to Cart

Uses a **GraphQL persisted mutation**, not the REST API:

```
POST /_v/private/graphql/v1?workspace=master&maxAge=long&appsEtag=remove
     &domain=store&locale=pt-BR
     &__bindingId=0a362f40-93e7-42c4-90c5-a3946de77fb3

Body:
{
  "operationName": "addToCart",
  "variables": {},
  "extensions": {
    "persistedQuery": {
      "version": 1,
      "sha256Hash": "a63161354718146c4282079551df81aaa8fa3d59584520cf5ea1c278fac0db33",
      "sender": "vtex.checkout-resources@0.x",
      "provider": "vtex.checkout-graphql@0.x"
    },
    "variables": "{base64 encoded}"
  }
}
```

Variables (base64-encoded):
```json
{
  "items": [
    {
      "id": 6180,
      "index": 0,
      "quantity": 1,
      "seller": "zonasulzsa",
      "options": []
    }
  ],
  "marketingData": {}
}
```

Requires `VtexIdclientAutCookie_zonasul` cookie for auth.

### Response

```json
{
  "data": {
    "addToCart": {
      "items": [
        {
          "id": "6180",
          "productId": "6196",
          "name": "Banana Prata Orgânica 800g",
          "quantity": 1,
          "price": 1099,
          "sellingPrice": 879,
          "seller": "zonasulzsa"
        }
      ],
      "totalizers": [
        {"id": "Items", "name": "Total dos Itens", "value": 879},
        {"id": "Shipping", "name": "Total do Frete", "value": 1790}
      ]
    }
  }
}
```

Prices are in centavos (879 = R$8.79).

## Checkout (REST API)

After navigating to checkout (`/checkout/#/shipping`), the standard VTEX Checkout REST API is used.

### Get Order Form (complete cart + checkout state)

```
GET /api/checkout/pub/orderForm
→ Returns the full orderForm with items, addresses, shipping options, payment methods
```

### Order Form Key Fields

**Items:**
```json
{
  "id": "6180",
  "name": "Banana Prata Orgânica 800g",
  "quantity": 1,
  "price": 1099,
  "seller": "zonasulzsa"
}
```

**Saved Address:**
```json
{
  "street": "Rua das Laranjeiras",
  "number": "100",
  "complement": "Apto 200",
  "neighborhood": "Laranjeiras",
  "postalCode": "22240-003",
  "city": "Rio de Janeiro",
  "state": "RJ"
}
```

**Delivery Windows (AGENDADA mode):**
```json
[
  {"startDateUtc": "2026-03-04T14:01:00+00:00", "endDateUtc": "2026-03-04T16:00:59+00:00", "price": 700},
  {"startDateUtc": "2026-03-04T13:00:00+00:00", "endDateUtc": "2026-03-04T18:00:59+00:00", "price": 0},
  {"startDateUtc": "2026-03-04T18:01:00+00:00", "endDateUtc": "2026-03-04T20:00:59+00:00", "price": 700}
]
```

Window prices in centavos. Price 0 = free delivery (wider 5h window). Price 700 = R$7.00 (narrower 2h window).

**Payment Methods:**

| ID | Name | Group |
|----|------|-------|
| 1 | American Express | creditCardPaymentGroup |
| 2 | Visa | creditCardPaymentGroup |
| 3 | Diners | creditCardPaymentGroup |
| 4 | Mastercard | creditCardPaymentGroup |
| 9 | Elo | creditCardPaymentGroup |
| 125 | Pix | instantPaymentPaymentGroup |
| 201 | Dinheiro | custom201PaymentGroupPaymentGroup |
| 401 | VR | customPrivate_401PaymentGroup |
| 403 | Alelo | customPrivate_403PaymentGroup |
| 404 | Ticket Alimentação | customPrivate_404PaymentGroup |
| 851 | Transfero Checkout | Transfero CheckoutPaymentGroup |

### Checkout Steps (REST endpoints)

```
POST /api/checkout/pub/orderForm/{orderFormId}/attachments/shippingData
POST /api/checkout/pub/orderForm/{orderFormId}/attachments/paymentData
POST /api/checkout/pub/orderForm/{orderFormId}/transaction
```

## Persisted GraphQL Query Hashes

| Operation | Hash | Sender | Provider |
|-----------|------|--------|----------|
| productSearchV3 | `31d3fa494df1fc41efef6d16dd96a96e6911b8aed7a037868699a1f3f4d365de` | vtex.store-resources@0.x | vtex.search-graphql@0.x |
| facetsV2 | `f58a719cabfc9839cc0b48ab2eb46a946c4219acd45e691650eed193f3f31bdf` | vtex.store-resources@0.x | vtex.search-graphql@0.x |
| autocompleteSearchSuggestions | `069177eb2c038ccb948b55ca406e13189adcb5addcb00c25a8400450d20e0108` | vtex.store-resources@0.x | vtex.search-graphql@0.x |
| productSuggestions | `3eca26a431d4646a8bbce2644b78d3ca734bf8b4ba46afe4269621b64b0fb67d` | vtex.store-resources@0.x | vtex.search-graphql@0.x |
| topSearches | `2bc9eefece58aa409f8d03dd582af43cb9acbb052fe4aa44a185e763782da88c` | vtex.store-resources@0.x | vtex.search-graphql@0.x |
| addToCart | `a63161354718146c4282079551df81aaa8fa3d59584520cf5ea1c278fac0db33` | vtex.checkout-resources@0.x | vtex.checkout-graphql@0.x |
| fixedPrice | `44afdf7de36ab2fbb4c4ed3389736005cbfebf85d4f05914db7b529a9e5f0362` | zonasul.fixed-price@0.x | zonasul.fixed-price@0.x |

## Sessions API (Token Refresh + OrderForm Discovery)

`GET /api/sessions?items=*` returns the full session state including:
- `cookie.VtexIdclientAutCookie_zonasul.value` — the live JWT (even when HttpOnly)
- `checkout.orderFormId.value` — the current orderFormId
- `authentication.storeUserEmail.value` — logged-in user email
- `profile.firstName.value`, `profile.lastName.value`, `profile.phone.value`

The JWT has a 24h TTL (`exp - iat = 86400s`), but the VTEX session outlives the JWT. Calling the sessions API with the existing auth cookie returns a fresh JWT, enabling silent token refresh without re-login.

This also means we don't need to persist `orderFormId` in a config file — we can fetch it from the session on every run.

## Anti-Fraud (ClearSale)

Zona Sul uses **ClearSale** for anti-fraud on credit card transactions. Without a valid ClearSale session, Cielo returns code 59 ("Suspected Fraud").

### ClearSale Integration

The storefront loads `https://device.clearsale.com.br/p/fp.js` with app key `b5qhnn79ksdoeru452lt`.

Initialization:
```js
csdp('app', 'b5qhnn79ksdoeru452lt');
csdp('outputsessionid', 'session-id');  // writes UUID to <input id="session-id">
```

The SDK generates a session UUID and sends device fingerprint data via two GET requests:

1. **fp1.png** — canvas/WebGL hashes: `?bb={hash}&ba={hash}&a2={hash}&app={appKey}&sid={sessionId}`
2. **fp2.png** — device telemetry: `?aa={userAgent}&ab={lang}&ac={colorDepth}&ae={screenH}&af={screenW}&ai={tzOffset}&...&app={appKey}&sid={sessionId}`

### Payment Integration

The ClearSale session ID is passed as `deviceFingerprint` in the payment `fields` object:
```json
{
  "fields": {
    "validationCode": "123",
    "securityCode": "123",
    "accountId": "...",
    "bin": "1234",
    "deviceFingerprint": "b03451ab-fd20-8e3b-d35d-932f50144870"
  }
}
```

The `deviceInfo` query parameter on the gateway URL is a separate base64-encoded string with screen dimensions (not the ClearSale session).

## Open Questions

1. ~~**Credit card tokenization**: How does VTEX handle card data in the payment attachment?~~ **RESOLVED**: Saved cards use `accountId` + CVV. ClearSale `deviceFingerprint` required in `fields`.
2. ~~**Auth refresh**: The JWT is 24h. VTEX has a refresh token flow (`vid_rt` cookie).~~ **RESOLVED**: Sessions API returns fresh JWT. No `vid_rt` needed.
3. **Delivery mode session**: The AGENDADA selection is stored in the VTEX session. Need to determine the exact session API call to set this programmatically so we don't get prompted.
4. **Custom auth domain**: `autenticacao.zonasul.com.br` may use different API endpoints than standard VTEX ID. Need to capture the actual auth network calls (lost due to cross-domain redirect during our session).
