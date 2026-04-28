# Import & Export Screen

Single screen at `/screens/import` that hosts two independent flows: **AI Import** (upload a broker/exchange export → AI parse → review modal with gap resolution → confirm) and **Export / Restore** (download all data as CSV, or restore from a previously exported CSV). The screen adds two reusable SDUI extensions: a `file_upload` custom component and a `loading` indicator that accepts cycling progress messages (both specified in the shared specs they extend).

---

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET`  | `/screens/import` | Full screen render. No query params. |
| `POST` | `/actions/import/analyze` | Multipart `{file, hint?}`. Calls `POST /v1/import/sessions`, blocks until BE responds. Success → `replace import-modal-slot` with the review subtree. Failure → `replace ai-import-card` with idle + error banner. Loading: full + cycling messages. |
| `POST` | `/actions/import/sessions/:id/resolve_gaps` | Body: `{resolutions:[{gap_id,value}]}`. Calls `PATCH /v1/import/sessions/:id/gaps`. Success → `replace import-modal-slot` with refreshed review. 422 → same replace + inline error banner. Loading: section over `import-modal-slot`. |
| `POST` | `/actions/import/sessions/:id/confirm` | Calls `POST /v1/import/sessions/:id/confirm`. Success → `replace import-root` with fresh tree + success snackbar. 422 → `replace import-modal-slot` with review + error banner. Loading: full. |
| `POST` | `/actions/import/sessions/:id/cancel` | Calls `DELETE /v1/import/sessions/:id`. Success or 404 → `replace import-root` + info snackbar. Loading: full. |
| `GET`  | `/actions/import/export` | Streaming proxy of `GET /v1/export`. Copies `Content-Disposition` verbatim. Action type `open_url` — browser handles the download. Missing/invalid JWT → HTTP 302 redirect to `/login`. |
| `POST` | `/actions/import/restore` | Multipart `{file}`. Calls `POST /v1/restore`. Success → `replace restore-card` with success state. Error → `replace restore-card` with idle + error banner + snackbar. Loading: section over `restore-card`. |

All endpoints except the export proxy require a valid JWT. Missing/invalid JWT → `401 {"error":"unauthorized","redirect":"/login"}`.

### Backend dependencies

- `POST /v1/import/sessions` — multipart `{file, hint?}`. Synchronous; blocks until AI finishes parsing. Returns the full session object. Timeout ≥ 90s.
- `PATCH /v1/import/sessions/:id/gaps` — body: `{resolutions:[{gap_id,value}]}`. Returns the updated session.
- `POST /v1/import/sessions/:id/confirm` — no body. Returns `{assets_created, trades_imported, snapshots_imported, warnings}`. 404 if session expired; 422 if blocking gaps remain.
- `DELETE /v1/import/sessions/:id` — no body. Returns 204. 404 treated as success (idempotent).
- `GET /v1/export` — returns `text/csv` with `Content-Disposition: attachment; filename="vk_tracker_export_<YYYY-MM-DD>.csv"`. Middleend streams through unchanged.
- `POST /v1/restore` — multipart `{file}` (CSV, up to 10 MB). Returns counts of imported/skipped rows for assets, trades, snapshots, and snapshot entries.

---

## Layout

```
screen
└── column "import-root"  (gap: lg)
    ├── header                       (title "Import & Export", no actions)
    ├── section "import-section"
    │   ├── card "ai-import-card"       (AI Import — switcheable)
    │   └── group "export-restore-group" (two sub-cards side by side; stacked on mobile)
    │       ├── card "export-card"
    │       └── card "restore-card"     (switcheable)
    └── column "import-modal-slot"   (initially empty; only AI Import uses it)
```

- Header is title-only. No HideValues toggle, no global filters.
- `ai-import-card` and `restore-card` are **switcheable cards**: replaced in full via `replace target_id="<card-id>"`. They never replace each other.
- `export-restore-group` lays two cards in a 2-column grid on desktop and a single-column stack on mobile.
- `import-modal-slot` follows the standard modal-slot pattern. Only AI Import injects into it. Restore and Export never use it.

---

## Data and business rules

### AI Import card states

`ai-import-card` cycles through two states:

**Idle** (initial render, post-confirm success, post-cancel, post-failure):
- Header: `import.ai.title` + `import.ai.description`.
- `file_upload`: `name="file"`, `accept=".csv,.tsv,.xls,.xlsx,.txt"`, `max_size_bytes=5_242_880` (5 MB), localized labels.
- `textarea`: `name="hint"`, optional, `max_length=500`, localized label and placeholder.
- Button `import.analyze` ("Analyze file"), disabled until `file_upload` has a file. Action: `submit` to `POST /actions/import/analyze`, `loading: {scope:"full", messages:[5–7 localized phrases]}`.

**Failure** (analyze returned an error):
- Same as idle, with an `error` banner above the file_upload and `prefill_filename` set from the submitted filename.
- Button stays enabled (retry path).

### Review modal (in `import-modal-slot`)

On successful analyze, the middleend returns `replace import-modal-slot` with a modal tree. The modal is `dismissible: false`. Sections in order:

1. **Banner** — `warning` variant if `gap_counts.blocking > 0` (`import.review.blocking_banner` interpolating `{n}`); `info` variant if blocking == 0 (`import.review.ready_banner`).
2. **AI Summary card** (`import.review.summary`) — `text` of `session.ai_summary`. If `session.assumptions` non-empty: a collapsible `import.review.assumptions` toggle.
3. **Issues section** (only if `gap_counts.blocking > 0`; title `import.review.issues`) — one card per blocking gap: badge with `gap.type`, description, affected rows, suggestion, `input` pre-filled with `gap.resolution`. Submit button (`import.gaps.save`) targets `POST /actions/import/sessions/:id/resolve_gaps`.
4. **Warnings section** (only if any gap has `severity=="warning"`; collapsed by default) — read-only rows, badge + description per warning.
5. **Preview section** (`import.review.preview`; always present) — three collapsible blocks (open by default):
   - **Assets** (`import.review.preview.assets`): Ticker · Name · Type · Currency · Action.
   - **Trades** (`import.review.preview.trades`): Row · Ticker · Type · Date · Qty · Price · Fees · Status.
   - **Snapshots** (`import.review.preview.snapshots`): Date · Entries count · Status.
6. **Action bar** (sticky bottom) — status badge, ghost Cancel button, primary Confirm button (disabled when `session.status != "ready"`).

### Resolve gaps

`POST /actions/import/sessions/:id/resolve_gaps` parses `resolutions[<gap_id>]` inputs, drops empty values, calls `PATCH /v1/import/sessions/:id/gaps`. Returns the updated session as a fresh review replace. On 422: same replace + error banner. On 404 (session expired): replace `import-root` + warning snackbar `import.session_expired`.

### Confirm

`POST /actions/import/sessions/:id/confirm` calls the backend, on success returns `replace import-root` with fresh tree + success snackbar interpolating `{assets}`, `{trades}`, `{snapshots}`, `{warnings}`. On 422: replace `import-modal-slot` + error banner. On 404: replace root + `import.session_expired` snackbar.

### Cancel

`POST /actions/import/sessions/:id/cancel` calls `DELETE`. On 204 or 404: replace `import-root` + info snackbar `import.cancelled`. On 5xx: snackbar error, modal stays open.

### Export card

Single static state. Header: `import.export.title` + `import.export.description`. Button `import.export.submit` with action `open_url` to `/actions/import/export`. The middleend handler streams `GET /v1/export` through to the client. Missing/invalid JWT → HTTP 302 to `/login`.

### Restore card

**Idle**: `import.restore.title` + description, `file_upload` (`name="file"`, `accept=".csv"`, `max_size_bytes=10_485_760`), button `import.restore.submit` (disabled until file), action `submit` to `POST /actions/import/restore`, `loading: section` over `restore-card`.

**Failure**: idle re-emitted, `prefill_filename` set, `error` banner above file_upload, error snackbar.

**Success**: green header (`import.restore.success_title`), 4-row table (Assets · Trades · Snapshots · Snapshot entries) with Imported and Skipped counts (right-aligned, tabular-nums), button `import.restore.try_again` whose action embeds the idle subtree in a `replace` tree (no extra round-trip).

---

## Custom components used

This screen uses the [`file_upload`](../sdui-custom-components.md#4-file_upload) custom component for both the AI Import upload form and the Restore upload form. It also uses the extended [loading indicator](../sdui-actions.md#2b-loading-indicators) object form (`{scope, messages[]}`) on the Analyze action to cycle localized progress phrases during the AI parse.

---

## i18n keys

**Screen / sections**: `import.title`, `import.ai.title`, `import.ai.description`, `import.export.title`, `import.export.description`, `import.restore.title`, `import.restore.description`.

**File upload**: `import.upload.label`, `import.upload.placeholder`, `import.upload.hint_ai`, `import.upload.hint_restore`, `import.upload.error_size` (with `{limit}`), `import.upload.error_format`, `import.upload.reattach_hint`.

**AI Import form**: `import.hint.label`, `import.hint.placeholder`, `import.analyze`.

**Loading messages**: `import.loading.analyze.1`–`import.loading.analyze.5` (at minimum).

**Review**: `import.review.blocking_banner` (with `{n}`), `import.review.ready_banner`, `import.review.summary`, `import.review.assumptions` (with `{n}`), `import.review.issues`, `import.review.warnings` (with `{n}`), `import.review.preview`, `import.review.preview.assets`, `import.review.preview.trades`, `import.review.preview.snapshots`.

**Gaps**: `import.gaps.affected_rows` (with `{rows}`), `import.gaps.input_placeholder`, `import.gaps.save`.

**Action bar**: `import.review.confirm`, `import.review.cancel`, `import.review.status.needs_review`, `import.review.status.ready`.

**Outcomes**: `import.success` (with `{assets}`, `{trades}`, `{snapshots}`, `{warnings}`), `import.cancelled`, `import.session_expired`, `import.failure_generic`.

**Restore**: `import.restore.submit`, `import.restore.success_title`, `import.restore.col.imported`, `import.restore.col.skipped`, `import.restore.row.assets`, `import.restore.row.trades`, `import.restore.row.snapshots`, `import.restore.row.snapshot_entries`, `import.restore.try_again`, `import.restore.error_generic`.

**Export**: `import.export.submit`.

**Shared**: `common.cancel`.

---

## Error handling

| Situation | Surface |
|---|---|
| `file_upload` local size violation (analyze ≥ 5 MB, restore ≥ 10 MB) | Inline in dropzone via `error_message_size`. No round-trip. |
| `file_upload` local format mismatch | Inline in dropzone via `error_message_format`. No round-trip. |
| BE `IMPORT_FILE_TOO_LARGE` | `replace ai-import-card` with idle + error banner; file preserved via `prefill_filename`. |
| BE AI parse failure | `replace ai-import-card` with idle + error banner using BE message; file + hint preserved. |
| BE 422 on `resolve_gaps` | `replace import-modal-slot` with review + error banner; user values preserved. |
| BE 422 on `confirm` | `replace import-modal-slot` with review + error banner; root untouched. |
| BE 404 on `confirm` / `resolve_gaps` | `replace import-root` + warning snackbar `import.session_expired`. |
| BE 404 on `cancel` | Treated as success — root replace + info snackbar. |
| BE 5xx / network on any action | Snackbar error `import.failure_generic`; card/modal keeps current state. |
| BE `RESTORE_FILE_TOO_LARGE` | `replace restore-card` with idle (file preserved) + error banner + snackbar. |
| BE restore parse failure | Same as above. |
| Export proxy upstream failure | Browser receives upstream error body. v1 acceptable. |

---

## Acceptance criteria

- `GET /screens/import` without a valid JWT returns `401` with the documented redirect.
- With a valid JWT the screen renders four regions under `import-root`: header, `ai-import-card`, `export-restore-group` (containing `export-card` + `restore-card`), and an empty `import-modal-slot`.
- No HideValues toggle in the screen header; no `sensitive: true` on any component.
- `ai-import-card` idle state renders a `file_upload` (5 MB cap, multi-format), a `textarea` for hint (max 500 chars), and an `Analyze file` button disabled until a file is selected.
- `file_upload` performs local validation: a file ≥ 5 MB or with a wrong extension triggers an inline error and does not submit.
- Successful analyze response replaces `import-modal-slot` with the review modal.
- Review modal contains: banner, AI Summary card with collapsible assumptions, issues section (only if blocking>0), warnings section (only if warnings exist), preview section, and sticky action bar.
- Confirm is disabled when `session.status != "ready"`. Status badge is amber for `needs_review`, green for `ready`.
- Save resolutions replaces the modal-slot; Confirm enables when status flips to `ready`.
- Confirm success replaces `import-root` with fresh tree + success snackbar interpolating four counts.
- Cancel success replaces `import-root` + info snackbar. 404 from BE treated as success.
- Session-expired 404 from confirm/resolve_gaps → root replace + `import.session_expired` warning snackbar.
- BE 5xx → `import.failure_generic` snackbar; card/modal unchanged.
- `export-card` renders an `Export all data` button with `open_url` to `/actions/import/export`. Handler streams the download through.
- `restore-card` idle renders `file_upload` (10 MB, `.csv`) + disabled Restore button.
- Restore success renders a 4-row result table + `Restore another file` button (embedded-tree replace, no extra round-trip).
- Restore failure replaces `restore-card` with idle + error banner + snackbar.
- On mobile, `export-restore-group` stacks to single column.
- All user-facing strings resolve via the i18n keys in §i18n keys for `en` and `es`.
