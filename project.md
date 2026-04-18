# vk-investment-middleend

Middleend (BFF) for the VK Investment tracker. Go + Gin, Server-Driven UI (SDUI).

## Commands

- `make run` — start the server
- `make build` — build the binary
- `make test` — run tests
- `make lint` — run linter
- `./cli run` — restart the dev server on `:8082` (kill existing listener first)

## Structure

- `cmd/server/` — entry point
- `internal/server/` — HTTP server, routes
- `internal/components/` — SDUI component types and constructors
- `internal/<screen>/` — per-screen handlers + builders
- `internal/shared/` — cross-cutting concerns
- `spec/` — **project specs (source of truth)**

## Workflow — SDD (Spec-Driven Development)

Each screen is broken into **layers**. One layer at a time: spec → implement.

**Canonical specs live in `spec/screens/<screen>/NN-<layer>.md`.** That file is the contract. `spec/spec.md` is the index; it only points at files under `spec/`.

Per layer: update the spec in `spec/`, then implement the code in `internal/<screen>/`, then the spec stays in sync with what shipped.

Anything outside `spec/` (e.g. `docs/superpowers/`) is **plugin-generated transient output** — not maintained, can be ignored or deleted.

## Rules for working on this repo

- **Ask before making changes.** No surprise refactors, no off-topic tangents, no "while I'm in here" cleanup. If something looks off, raise it first; don't fix it on your own.
- **Restart the middleend after code changes.** Kill the existing listener on `:8082` and run `./cli run` in background.
- **Commit messages** follow Conventional Commits (`feat(scope): ...`, `docs(spec): ...`, etc.). No Claude Code co-author trailer unless asked.
- **Terse by default.** Deliver what's asked, no unsolicited explanation or summary.
