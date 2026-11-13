# Changelog

## Unreleased

### Changed

- Migrate catalog, cart, delivery windows and saved-credit-card checkout to the current BFF. Use `product:<UUID>` catalog handles; legacy numeric SKUs and `sku.reference` are not interchangeable. Local bulk ordering, reordering, delivery simulation and subscriptions remain disabled.
- Preserve integer pack quantities and centavo pack prices; convert weight cart updates to kg and verify the full list after each mutation. Read back delivery scheduling before payment.
- Make checkout previews read-only. `doctor` passes only with a verified session, delivery address and saved credit card. Payment submission requires an explicit saved-card ID and `--confirm`; no Pix/cash fallback or CVV.
- Report any attempted payment as `submitted_unverified` with additive exit 11 until issuer settlement can be verified. Never retry a payment automatically or claim settlement from status alone.
- Raw token login now requires an expected email and a token for the new backend. Use `ZONASUL_BFF_TOKEN`; the old `ZONASUL_TOKEN` VTEX cookie is ignored.

### Fixed

- Keep checkout's `orderId`, order readback and recovery guidance tied to the submitted order. Version 0.9.0 reported the next empty cart as `orderId`; use `cartId` for order IDs recorded from 0.9.0.
- Authenticate through the current storefront, verify the account identity before saving a session, and keep its secrets separate from legacy VTEX credentials.
- Use storefront email-login codes and password refresh. Reject token imports whose identity does not match the requested email.
- Clear and verify the cart after acknowledged checkout so the next order does not reuse its lines; cleanup failure exits 9 with instructions not to repeat checkout (legacy VTEX implementation).
- Route legacy VTEX API calls and OAuth callbacks through the Zona Sul account host after the public site moved to a Next.js frontend.
