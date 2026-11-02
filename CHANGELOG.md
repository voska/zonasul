# Changelog

## Unreleased

### Fixed

- Clear and verify the cart after acknowledged checkout so the next order does not reuse its lines; cleanup failure exits 9 with instructions not to repeat checkout.
- Route VTEX API calls and OAuth callbacks through the Zona Sul account host after the public site moved to a Next.js frontend.
