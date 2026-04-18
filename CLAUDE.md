# vk-investment-middleend

workflow-version: v0.1.0

## Stack

- Language: Go
- Framework: Gin
- Type: Middleend (BFF)
- Pattern: Server-Driven UI (SDUI)

## Commands

- `make run` — Start the server
- `make build` — Build the binary
- `make test` — Run tests (JSON output)
- `make lint` — Run linter

## Structure

- `cmd/server/` — Application entry point
- `internal/server/` — HTTP server, routes, screen handlers
- `internal/components/` — SDUI component types and constructors
- `internal/<screen_name>/` — Per-screen handlers and builders
- `internal/shared/` — Cross-cutting concerns
- `spec/` — Project specification (canonical; source of truth post-implementation)
- `docs/superpowers/specs/` — Brainstorm design docs (per layer, dated)
- `docs/superpowers/plans/` — Implementation plans (per layer, dated)

## Workflow — Spec-Driven Development (SDD)

Each screen is decomposed into **layers** (portfolio has 6, assets has 2). Every layer goes through this pipeline end-to-end before the next layer starts:

1. **Brainstorm** → design doc at `docs/superpowers/specs/YYYY-MM-DD-<screen>-layer<N>-design.md`. Captures intent, scope, tree shape, endpoints, i18n, error handling, acceptance criteria. Committed as `docs(spec): ...`.
2. **Plan** → task-by-task implementation plan at `docs/superpowers/plans/YYYY-MM-DD-<screen>-layer<N>.md`. TDD with failing-first tests, exact code per step, exact commit messages. Committed as `docs(plan): ...`.
3. **Implement** → Go code in `internal/<screen>/` following the plan. One commit per task (feat / test / chore as appropriate). Runs via `subagent-driven-development` with two-stage review (spec compliance → code quality) per task.
4. **Canonical spec** → `spec/screens/<screen>/NN-<layer>.md`. Written or updated at the close of the layer. This is the **source of truth** for the shipped contract; the design doc may drift but this file must match code behavior.
5. **Index** → `spec/spec.md` points at the canonical spec directory / file (not at the design doc).

Artifacts produced per layer: 1 design doc + 1 plan + 1 canonical spec + N implementation commits. Each layer gets its own cycle. The design doc captures the *intent*; the canonical spec captures the *contract*.

**Rules that save rework:**

- Write the canonical spec before merging a layer. If it doesn't exist in `spec/screens/<screen>/`, the layer isn't done.
- `spec/spec.md` only references canonical specs. It should never link to a file under `docs/superpowers/`.
- Kill + restart the middleend on `:8082` after code changes (see `/Users/vadimkent/.claude/projects/.../memory/feedback_restart_middleend.md`).
