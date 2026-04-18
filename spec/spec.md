# VK Investment Middleend — Spec

## Overview

**Problem**: The VK Investment Tracker backend exposes a resource-oriented REST API. The legacy web frontend is a React SPA tightly coupled to those endpoints, which prevents UI evolution without redeploys and duplicates effort across platforms.

**Solution**: A middleend (BFF) that consumes the backend REST API and serves SDUI component trees to a new frontend. One server description of pages and navigation, rendered identically on web, web_mobile, Android, and iOS. Stateless; no new business logic.

**Project type**: middleend

**Target users**: End users of the investment tracker, through the new SDUI frontends.

## Goals

- Single source of UI truth for all platforms.
- Functional parity with the legacy frontend (portfolio, assets, trades, snapshots, import, analysis).
- UI evolution without frontend redeploys.
- Transparent JWT auth passthrough and SSE proxy for the AI analysis stream.

## Non-Goals

- Replacing or modifying the backend.
- Persistent storage in the middleend (caching only where required).
- Adding business logic beyond composition and adaptation.

## Spec Index

| Spec | Description |
|---|---|
| [Shell](shell.md) | App shell: navigation slots, platform adaptation |
| [API Contract](api.md) | Middleend endpoints (shell, screens, actions) — TBD |
| [SDUI Custom Components](sdui-custom-components.md) | Custom SDUI components (line_chart, etc.) |
| [Security](security.md) | JWT validation, login/register proxy, auth response extension |
| [Error Handling](errors.md) | Error categories and behavior — TBD |
| [Acceptance Criteria](acceptance.md) | Testable completion criteria — TBD |

Screens (one file per screen, added as SDD progresses):

| Screen | File |
|---|---|
| Login | [`screens/login.md`](screens/login.md) |
| Portfolio | [`screens/portfolio/`](screens/portfolio/) — decomposed into 6 layers; layer 1 live in [`01-positions.md`](screens/portfolio/01-positions.md) |
| Assets | [`screens/assets/`](screens/assets/) — decomposed into 2 layers; both live: [`01-list.md`](screens/assets/01-list.md), [`02-mutations.md`](screens/assets/02-mutations.md) |
| Trades | `screens/trades.md` — TBD |
| Snapshots | `screens/snapshots.md` — TBD |
| Import | `screens/import.md` — TBD |
| Analysis | `screens/analysis.md` — TBD |
