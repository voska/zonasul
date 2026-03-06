# Zonasul CLI Polish Design

**Date:** 2026-03-06
**Status:** Approved

## Goal

Bring the zonasul CLI to the same level of polish as qbo-cli and amadeus-cli.

## Changes

### 1. Code Restructure

Split `cmd/zonasul/main.go` (613 lines) into:

```
cmd/zonasul/main.go          # ~45 lines: Kong parse + error handler + version
internal/cmd/
  root.go                    # CLI struct, Globals, NewGlobals()
  auth.go                    # AuthCmd (login/status/logout)
  search.go                  # SearchCmd
  cart.go                    # CartCmd (show/add/remove/clear)
  delivery.go                # DeliveryCmd (windows)
  checkout.go                # CheckoutCmd
  orders.go                  # OrdersCmd
  agent.go                   # AgentCmd (exit-codes)
internal/errfmt/
  errors.go                  # Typed Error struct with exit codes (replaces exitcode pkg)
```

Each command file has a Kong struct + `Run(g *Globals) error`. Globals holds ctx, config, output opts, version. Main.go just parses and dispatches.

### 2. Infrastructure Files

| File | Content |
|------|---------|
| `README.md` | Badges, description, install (Homebrew/Go/binary), quick start, output modes, commands, dev guide |
| `LICENSE` | MIT, Copyright 2026 Matt Voska |
| `Makefile` | build, test, lint, vet, fmt, install — version via LDFLAGS |
| `.goreleaser.yaml` | Linux/Darwin/Windows x amd64/arm64, Homebrew tap, checksums, changelog |
| `.github/workflows/ci.yml` | Build + test + lint on push to main/master + PRs |
| `.github/workflows/release.yml` | GoReleaser on v* tags |
| `.golangci.yml` | Strict linter config |

### 3. Version Injection

`var version = "dev"` in main.go, injected via `-ldflags "-s -w -X main.version=$(VERSION)"`.

### 4. errfmt Package

Replace `internal/exitcode/` with `internal/errfmt/`:
- Typed `Error` struct with `Code`, `Message`, `Detail`
- Constructor helpers: `Auth()`, `NotFound()`, `Empty()`, `RateLimit()`, etc.
- Same exit code values (0-7)

### 5. Schema Command

`zonasul schema --json` — CLI introspection (commands, flags, version) for agent self-discovery.

### 6. Rich Output

Add `muesli/termenv` for colored stderr messages. Keep stdout clean.

### 7. Skills Directory

Move `.claude/skills/zonasul-groceries/` to `skills/zonasul/`.

## Out of Scope

- `--dry-run`, `--select`, `--results-only`
- `site/` landing page
- `.github/workflows/pages.yml`
