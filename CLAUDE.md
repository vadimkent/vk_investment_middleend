# vk-investment-middleend

Canonical project doc: [`project.md`](project.md). Read it first.

## Stack
Go · Gin · SDUI middleend (BFF).

## Commands
- `make run` · `make build` · `make test` · `make lint`
- `./cli run` — restart dev server on `:8082` (kill existing first)

## Source of truth

**`spec/` is the project spec directory — the canonical source of truth.**
- Screen contracts: `spec/screens/<screen>/NN-<layer>.md`
- Screen index: `spec/spec.md` (only links files inside `spec/`)

**`docs/superpowers/` is plugin-generated transient output.** Not maintained. Do not rely on it and do not add references to it from `spec/`.

## Workflow — SDD
Per screen: decompose into layers. For each layer: write the canonical spec in `spec/screens/<screen>/`, then implement the code in `internal/<screen>/`. The canonical spec must match shipped behavior.

## Rules

- **Ask before making any change.** No surprise refactors, no off-topic tangents, no unrequested cleanup. If you see something off, raise it and wait.
- **Restart the middleend after code changes** (see `./cli run`).
- **Commit messages** use Conventional Commits. No Claude Code co-author trailer unless explicitly requested.
- **Terse responses.** Deliver what's asked, no unsolicited analysis or summary.
