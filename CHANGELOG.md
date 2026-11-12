# Changelog

## Unreleased

### Changed

- Ordering commands are disabled while the new storefront integration is pending. They exit 9 without sending a cart or order request. `doctor` checks current account prerequisites and always exits 10 in this build.
- Raw token login now requires an expected email and a token for the new backend. Use `ZONASUL_BFF_TOKEN`; the old `ZONASUL_TOKEN` VTEX cookie is ignored.

### Fixed

- Authenticate through the current storefront, verify the account identity before saving a session, and keep its secrets separate from legacy VTEX credentials.
- Use storefront email-login codes and password refresh. Reject token imports whose identity does not match the requested email.
- Clear and verify the cart after acknowledged checkout so the next order does not reuse its lines; cleanup failure exits 9 with instructions not to repeat checkout (legacy VTEX implementation).
- Route legacy VTEX API calls and OAuth callbacks through the Zona Sul account host after the public site moved to a Next.js frontend.
