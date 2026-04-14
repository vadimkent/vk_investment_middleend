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
- `spec/` — Project specification
