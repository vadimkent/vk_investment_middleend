# Shared: Assets Catalog

A cross-screen helper that returns the **full list of assets** in one call, for screens that need an asset selector (filter dropdowns, form selects). Unlike the `Assets` screen client, which is paginated for display, the catalog loops through backend pages until it has every asset.

## Purpose

Screens that register or filter events on an asset (trades, snapshots, import, analysis) need the complete asset list up front — not a paginated slice. The backend `GET /v1/assets` only returns pages, so the middleend performs the paging loop and returns a flat `[]Asset` to callers.

## Consumers

| Screen | Uses catalog for |
|---|---|
| Trades | `asset_id` filter dropdown + create/edit form select |
| Snapshots | Per-asset entry rows in the snapshot wizard (future) |
| Import | Asset resolution preview (future) |
| Analysis | Asset picker for per-asset analysis (future) |

The `Assets` screen itself does **not** use the catalog — it reads one paginated page directly for display.

## Backend dependency

- `GET /v1/assets?size=100&offset=<n>&sort=ticker&order=desc` — called repeatedly, incrementing `offset` by `100` each iteration, until `offset + size >= total`. `size=100` is a fixed choice (not user-facing); it keeps the round-trip count low without approaching backend per-page limits.

The `Authorization` header from the inbound request is forwarded verbatim on every page call.

## Contract

### Input

- `context.Context`
- `authorization string` — forwarded as-is.

### Output

A slice of `Asset` objects in the order returned by the backend (i.e. `ticker DESC` across all pages). No additional sorting or filtering is applied in the helper — screens that need a different order sort the returned slice themselves.

Minimum fields required on each `Asset` for downstream screens:

| Field | Reason |
|---|---|
| `id` | Used as the `value` of select options. |
| `ticker` | Label on select options and table cells. |
| `name` | Secondary label (e.g. `AAPL — Apple Inc.`). |
| `asset_type` | Optional grouping on selects. |
| `currency` | Needed to format money fields on screens that compute totals (Trades). |
| `is_complex` | Trade form must **exclude** complex assets. |

Additional fields returned by the backend pass through untouched; callers ignore what they don't need.

### Errors

- `ErrUnauthorized` — any page call returns `401`. The caller surfaces the standard `/login` redirect.
- `ErrBackend` — network error, `5xx`, malformed JSON, or any non-`200` / non-`401` status. The caller returns `502 BACKEND_ERROR`.

If a later page fails after earlier pages succeeded, the helper returns `ErrBackend` and discards partial results. Callers never see a truncated list.

## Caching

No caching in the middleend for now. Each screen render calls the helper and gets a fresh list. This keeps the middleend stateless (per project non-goal of persistent storage). If latency becomes an issue, a short in-memory TTL cache keyed by the authenticated user can be layered on without changing the contract.

## Performance

Typical user has fewer than 100 assets → 1 backend call. Power users with several hundred → 2–5 calls, executed sequentially to keep the logic simple and the backend load predictable. Parallelisation is not worth the added complexity at this scale.

## Acceptance criteria

- [ ] Returns the **complete** asset list across all backend pages, ordered as the backend returns them (`ticker DESC`).
- [ ] Issues `GET /v1/assets?size=100&offset=<n>&sort=ticker&order=desc` repeatedly until `offset + size >= total`, forwarding `Authorization`.
- [ ] Returns `ErrUnauthorized` if any page call returns `401`.
- [ ] Returns `ErrBackend` on network error, `5xx`, malformed JSON, or if a later page fails after earlier pages succeeded (no partial results).
- [ ] Each returned `Asset` includes at minimum `id`, `ticker`, `name`, `asset_type`, `currency`, `is_complex`.
- [ ] Stateless — no caching; every call hits the backend.
