# Import & Export Screen — Design

Design spec for the Import & Export screen in the vk-investment middleend. Bundles two independent flows on a single screen: an AI-driven import of broker exports (CSV / TSV / XLS / XLSX / TXT) and a backup-grade export/restore of all user data. Introduces two SDUI additions: a custom `file_upload` component and an extended `loading` indicator that accepts cycling messages.

---

## 1. Overview

Single screen at `/screens/import` (sidebar entry: `Import & Export`) that lets the user:

1. **AI Import** — upload a broker / exchange export, let the backend's AI parse it, review the AI summary + assumptions, resolve any blocking gaps, inspect a preview of assets / trades / snapshots, and confirm the import.
2. **Export** — download a single CSV containing all of the user's data (backup + migration).
3. **Restore** — upload a previously-exported CSV; the backend creates missing records and skips existing ones (idempotent).

The two flows live on the same screen but operate independently. AI Import uses a `file_upload` form in its sub-section plus a review modal in the screen's modal slot. Export is a single button that triggers a browser download. Restore is a self-contained `file_upload` + result table in its own sub-card.

Shipping this screen requires:

- One new custom SDUI component: `file_upload` (drag-and-drop + click-to-browse + local validation + multipart submit).
- One extension to the SDUI `loading` indicator: it now accepts `{scope, messages[]}` so handlers can stream a list of localized progress phrases that the frontend cycles through during long waits. Backwards compatible with the current string-token form.

The HideValues toggle and `sensitive: true` masking are **not** added to this screen — they remain portfolio-only.

---

## 2. Endpoints

All endpoints are **protected** (JWT). Missing / invalid / expired JWT → `401 {"error":"unauthorized","redirect":"/login"}`. Backend 5xx / network / malformed → `502 BACKEND_ERROR`. Invalid query / body → `400 BAD_REQUEST`. Sessions that no longer exist on the backend (404 from confirm / resolve / cancel) are handled gracefully (see §6).

| Method | Path | Purpose |
|---|---|---|
| `GET`  | `/screens/import` | Full screen render. No query params (no filters / pagination). |
| `POST` | `/actions/import/analyze` | Multipart `{file, hint?}`. Calls `POST /v1/import/sessions` and **blocks** until the BE response. Success → `replace import-modal-slot` with the review subtree. Failure → `replace ai-import-card` with form re-emitted + error banner. Loading: full + cycling messages. |
| `POST` | `/actions/import/sessions/:id/resolve_gaps` | Body: `{ resolutions: [{gap_id, value}] }`. Calls `PATCH /v1/import/sessions/:id/gaps`. Success → `replace import-modal-slot` with refreshed review (button states recompute). 422 → same replace + inline error banner. Loading: section over `import-modal-slot`. |
| `POST` | `/actions/import/sessions/:id/confirm` | Calls `POST /v1/import/sessions/:id/confirm`. Success → `replace import-root` with fresh tree (idle state) + success snackbar interpolating counts. 422 → `replace import-modal-slot` with the review re-emitted + error banner. Loading: full. |
| `POST` | `/actions/import/sessions/:id/cancel` | Calls `DELETE /v1/import/sessions/:id`. Success → `replace import-root` + info snackbar. 404 from BE is treated as success (idempotent). Loading: full. |
| `GET`  | `/actions/import/export` | Streaming proxy of `GET /v1/export`. Copies `Content-Disposition` header verbatim. Action type is `open_url` — the browser handles the download; no SDUI replace happens. |
| `POST` | `/actions/import/restore` | Multipart `{file}`. Calls `POST /v1/restore`. Success → `replace restore-card` with the success state (result table). Validation error → `replace restore-card` with idle re-emitted + error banner + error snackbar. Loading: section over `restore-card`. |

Every non-screen endpoint returns an `ActionResponse`. Successful mutations either:

- Replace the screen root (`import-root`) entirely — used by Confirm and Cancel of an in-progress AI import — collapsing the modal slot back to empty and resetting both sub-cards.
- Replace a smaller subtree (`ai-import-card`, `restore-card`, `import-modal-slot`) — used by sub-actions that don't end the flow.

The middleend stores **no session state**: every review handler embeds the full session in the response, so subsequent sub-actions only need the session id to round-trip with the backend.

### Backend dependencies

- `POST /v1/import/sessions` — multipart `{file, hint?}`. Synchronous: blocks until the AI finishes parsing. Returns the full session object (`id`, `status`, `ai_summary`, `assumptions`, `preview: {assets, trades, snapshots}`, `gaps`, `gap_counts`). Errors: `IMPORT_FILE_TOO_LARGE` (400), `INVALID_REQUEST` (400), `INTERNAL_ERROR` (500), plus AI parse-failure codes. The middleend forwards the `Authorization` header and applies a generous timeout (≥ 90s — the AI parse can be long).
- `GET /v1/import/sessions/:id` — not used by the middleend in v1. The session travels in handler responses; refresh on demand happens by re-calling resolve_gaps / confirm / cancel.
- `PATCH /v1/import/sessions/:id/gaps` — body: `{resolutions:[{gap_id, value}]}`. Returns the updated session. Validation errors (422) surface inline.
- `POST /v1/import/sessions/:id/confirm` — no body. Returns `{assets_created, trades_imported, snapshots_imported, warnings}`. 404 if session expired; 422 if blocking gaps remain.
- `DELETE /v1/import/sessions/:id` — no body. Returns 204. Idempotent (404 ≡ success).
- `GET /v1/export` — returns `text/csv; charset=utf-8` with `Content-Disposition: attachment; filename="vk_tracker_export_<YYYY-MM-DD>.csv"`. The middleend streams the body through to the client unchanged.
- `POST /v1/restore` — multipart `{file}` (CSV up to 10 MB). Returns `{assets_imported, assets_skipped, trades_imported, trades_skipped, snapshots_imported, snapshots_skipped, snapshot_entries_imported, snapshot_entries_skipped}`. Errors: `RESTORE_FILE_TOO_LARGE` (400), `INVALID_REQUEST` (400), `INTERNAL_ERROR` (500), plus parse-failure codes.

No assets catalog dependency on this screen — preview rows reference tickers that the backend already resolved into `parsedAssetResponse.ticker / name / asset_type / currency`.

---

## 3. Layout

Four logical regions, three of them visible at once. The shape mirrors the project's standard `<screen>-root` / `<screen>-section` / `<screen>-modal-slot` triplet, with the section split into two stacked sub-areas:

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

- The header is title-only. No HideValues toggle, no global filters.
- `ai-import-card` and `restore-card` are independent **switcheable cards**: each has internal states that cycle via `replace target_id="<card-id>"`. They never replace each other.
- `export-restore-group` lays its two cards in a 2-column grid on desktop and a single-column stack on mobile. Tooling: same responsive primitives the trades / snapshots screens already use.
- `import-modal-slot` follows the standard project modal-slot pattern (sibling of the section, frontend renders its child as an overlay dialog on desktop and a drawer/sheet on mobile). Only the AI Import review uses it. Restore and Export never inject into it.

---

## 4. Data and business rules

### 4.1 AI Import card states

`ai-import-card` is replaced as a whole via `replace target_id="ai-import-card"`. Its content cycles through these states:

**Idle** (initial render, post-success of confirm, post-cancel, post-failure-with-retry):

- Header: localized title `import.ai.title` ("Import historical data") and description `import.ai.description`.
- `file_upload` (see §5.1): `name="file"`, `accept=".csv,.tsv,.xls,.xlsx,.txt"`, `max_size_bytes=5_242_880` (5 MB), localized labels.
- `textarea`: `name="hint"`, optional, max length 500, localized label `import.hint.label` and placeholder `import.hint.placeholder` ("e.g. trade history from Broker X, amounts in USD").
- Primary button `import.analyze` ("Analyze file"), disabled until `file_upload` has a file. Action: `submit` to `POST /actions/import/analyze` with `loading: { scope: "full", messages: [<5–7 localized phrases>] }`.

**Failure** (the analyze action returned an error response — file too large server-side, AI parse failed, BE 5xx):

- Same render as idle, with two differences:
  - An `error` banner above the file_upload showing the localized BE message.
  - The file and hint are preserved (the middleend re-emits them as the file_upload's pre-filled state — see §5.1's `prefill_filename` prop).
- The button stays enabled — clicking re-submits as a retry.

The `file_upload` component owns local validation for size and format. Files exceeding `max_size_bytes` or not matching `accept` raise an inline error inside the dropzone and never reach the middleend.

### 4.2 Review modal (in `import-modal-slot`)

When `POST /actions/import/analyze` succeeds, the middleend returns:

```json
{
  "action": "replace",
  "target_id": "import-modal-slot",
  "tree": { "type": "modal", "id": "import-review-modal", "props": {...}, "children": [...] }
}
```

The modal is `dismissible: false` (closing requires the explicit Cancel button — there is no X in the corner that bypasses the cancel network call). It scrolls vertically on overflow. Sections, in order:

1. **Banner (always present).**
   - `gap_counts.blocking > 0` → variant `warning`, text `import.review.blocking_banner` interpolating the count: *"This file has {n} issues that need your input before importing."*
   - `gap_counts.blocking == 0` → variant `info`, text `import.review.ready_banner` ("Ready to import — review the preview and confirm.").
   - Not dismissible.

2. **AI Summary card** (`title import.review.summary`).
   - `text` rendering `session.ai_summary` (plain).
   - If `session.assumptions.length > 0`: a `toggle` row "Assumptions ({n})" that expands to a bullet list of `session.assumptions`.

3. **Issues section** (only if `gap_counts.blocking > 0`; title `import.review.issues`).
   - One `card` per blocking gap with `border-destructive/40` styling:
     - Badge with `gap.type` (e.g. "missing_currency", "ambiguous_date").
     - `text` rendering `gap.description`.
     - `text` (smaller, muted) `import.gaps.affected_rows` interpolating `gap.affected_rows.join(", ")`.
     - `text` (italic, muted) rendering `gap.suggestion`.
     - `input`: `name="resolutions[<gap_id>]"`, type `text`, placeholder localized, pre-filled with `gap.resolution` if the BE has stored a previous resolution.
   - Below the cards: button `import.gaps.save` ("Save resolutions") that `submit`s the resolve_gaps form (see §4.3).

4. **Warnings section** (only if any gap has `severity == "warning"`).
   - A collapsed-by-default `toggle` row "{n} warnings" (singular: "1 warning"). Expanding reveals each warning as a row with a `secondary` badge (`gap.type`) + a `text` of `gap.description`. Read-only, no inputs.

5. **Preview section** (always present; title `import.review.preview`).
   - Three independent collapsible blocks (open by default), one per entity, each with a header showing `<entity_name> ({count})`:
     - **Assets** (`import.review.preview.assets`): table with columns Ticker (mono) · Name · Type · Currency · Action (badge).
     - **Trades** (`import.review.preview.trades`): table with columns Row · Ticker (mono) · Type · Date · Qty · Price · Fees · Status (badge: `secondary` for "ok", `destructive` for "blocked"). Rows where `t.status == "blocked"` get a faint destructive background.
     - **Snapshots** (`import.review.preview.snapshots`): table with columns Date · Entries (count) · Status (badge). Same blocked-row tinting.
   - All values are pre-formatted strings from the backend's preview response.

6. **Action bar (sticky bottom, inside the modal).**
   - `badge` with `session.status` text — `secondary` styling, color cue:
     - `needs_review` → amber (`import.review.status.needs_review`).
     - `ready` → green (`import.review.status.ready`).
   - Button ghost `import.review.cancel`. Action: `submit` to `POST /actions/import/sessions/:id/cancel`, `loading: full`.
   - Button primary `import.review.confirm`. Action: `submit` to `POST /actions/import/sessions/:id/confirm`, `loading: full`. Disabled when `session.status != "ready"`.

When the modal contains **no resolvable gaps** (i.e. blocking == 0), the Issues section is omitted entirely; the rest of the modal is identical.

### 4.3 Resolve gaps sub-flow

Triggered by `Save resolutions` in the Issues section.

1. Frontend submits the form (only the issues sub-form's inputs — names match `resolutions[<gap_id>]`). The middleend handler parses these into `[{gap_id, value}]`, dropping entries with empty values.
2. Middleend calls `PATCH /v1/import/sessions/:id/gaps` with the resolutions.
3. BE returns the updated session (status may now be `ready`, gaps may have new resolutions, or some still unresolved).
4. Middleend re-emits the **same review modal** as in §4.2 with the updated session, and returns `replace target_id="import-modal-slot"`. The Confirm button enables / disables according to the new `status`.
5. On 422 (invalid resolution format / value): same replace, but with an `error` banner inside the modal and the user's submitted values preserved in the inputs.
6. On 404 (session expired): the middleend instead replaces `import-root` and emits a warning snackbar `import.session_expired`. The user goes back to the upload form.

### 4.4 Confirm

Triggered by the primary button in the action bar.

1. Frontend submits with no body (the session id is in the action URL).
2. Middleend calls `POST /v1/import/sessions/:id/confirm`.
3. BE returns `{assets_created, trades_imported, snapshots_imported, warnings}` on success.
4. Middleend builds a fresh `import-root` tree (both sub-cards in idle, modal-slot empty) and returns:
   ```json
   {
     "action": "replace",
     "target_id": "import-root",
     "tree": <fresh root>,
     "feedback": {"type": "snackbar", "variant": "success", "message": "<localized import.success>"}
   }
   ```
   The success message interpolates the four counts.
5. On 422 (e.g. blocking gaps somehow remained): replace `import-modal-slot` with the review re-emitted + error banner; do not touch the screen root.
6. On 404 (session expired between resolve_gaps and confirm): replace root + warning snackbar `import.session_expired`.

### 4.5 Cancel

Triggered by the ghost button in the action bar.

1. Frontend submits with no body.
2. Middleend calls `DELETE /v1/import/sessions/:id`.
3. On 204 or 404 (idempotent): replace `import-root` with a fresh tree + info snackbar `import.cancelled` ("Import cancelled.").
4. On 5xx: snackbar error, modal stays open at the same review state.

### 4.6 Export card

Single static state. The card contains:

- Header: `import.export.title` ("Export data") + `import.export.description`.
- Button `import.export.submit` ("Export all data") with action:
  ```json
  { "trigger": "click", "type": "open_url", "url": "/actions/import/export", "target": "self" }
  ```
- The middleend's handler at `GET /actions/import/export` proxies `GET /v1/export` from the backend: forwards the Authorization header, copies the `Content-Disposition` and `Content-Type` headers from the BE response, and streams the body through to the client. The browser handles the download.
- No SDUI replace, no loading indicator (the action is `open_url`, not `submit`).

Because this endpoint is hit directly by the browser (not as an SDUI action), the standard `WithAuth` wrapping that emits `401 {error: "unauthorized", redirect: "/login"}` JSON does not work — the browser cannot interpret it. Instead, this handler responds to a 401 from the backend (or a missing/invalid JWT) with an HTTP **302 redirect** to `/login` so the browser follows it natively. Backend 5xx surfaces as a 502 plain-text response — the browser displays it as a download with the upstream error body. v1 acceptable given how rare this is; an enhancement could intercept and emit a snackbar via a pre-flight HEAD, but that doubles the round-trips.

### 4.7 Restore card

`restore-card` is replaced as a whole via `replace target_id="restore-card"`. States:

**Idle:**

- Header: `import.restore.title` ("Restore from backup") + `import.restore.description`.
- `file_upload`: `name="file"`, `accept=".csv"`, `max_size_bytes=10_485_760` (10 MB), localized labels.
- Button `import.restore.submit` ("Restore"), disabled until file present. Action: `submit` to `POST /actions/import/restore` with `loading: section` over `restore-card` (no messages — restore is fast, no fake-progress justification).

**Failure** (validation error, parse failure, 5xx):

- Same render as idle, with file preserved (via `file_upload`'s `prefill_filename`) and an `error` banner above the file_upload showing the localized BE message. A snackbar error also fires.

**Success:**

- Header in green: `import.restore.success_title` ("Restored successfully").
- Compact `table` of 4 fixed rows (assets, trades, snapshots, snapshot entries), 3 columns: label · `Imported` count · `Skipped` count. Counts use tabular-nums; columns Imported and Skipped are right-aligned. Localized labels via `import.restore.col.imported`, `import.restore.col.skipped`, `import.restore.row.assets`, `import.restore.row.trades`, `import.restore.row.snapshots`, `import.restore.row.snapshot_entries`.
- Button outline `import.restore.try_again` ("Restore another file"). Action: `reload` against `GET /actions/import/restore_idle` targeting `restore-card` (returns the idle subtree). Alternatively the middleend can embed the idle subtree literally in a `replace` action with `tree: <idle>` to avoid the extra round-trip — handler chooses; both are valid and equivalent. **This spec adopts the embedded-tree variant** to keep the round-trip cost down.

The middleend's handler at `POST /actions/import/restore`:

1. Reads the multipart upload (cap at 10 MB to mirror the BE).
2. Calls `POST /v1/restore` with the bytes.
3. Maps the BE response to the success-state subtree and returns `replace target_id="restore-card"`.
4. On error: replace `restore-card` with idle (file preserved) + error banner; also emit error snackbar via `feedback`.

---

## 5. Custom components and SDUI extensions

### 5.1 `file_upload` (new custom component)

Drag-and-drop + click-to-browse file picker with local validation, used wherever a flow needs a file submitted as part of a multipart form. Lives in `spec/sdui-custom-components.md` as a new top-level entry. Generic by design — reusable beyond Import (e.g. future profile-picture uploads, snapshot CSV imports, etc.).

**Props**

| Prop | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Multipart field name on submit (e.g. `"file"`). |
| `label` | string | yes | Visible label rendered above the dropzone. Localized by the middleend. |
| `placeholder` | string | yes | Dropzone copy when no file is selected (e.g. *"Drop a file here or click to browse"*). Localized. |
| `hint` | string | no | Auxiliary copy beneath the dropzone (formats / size limit). Localized. |
| `accept` | string | no | Comma-separated list of extensions / MIME types (e.g. `".csv,.tsv,.xlsx"`). Drives the native input's `accept` attribute and the local format check. Absent → any file. |
| `max_size_bytes` | int | no | Local size limit in bytes. When the user picks a larger file, render `error_message_size` inline and clear the selection. Absent → no local limit. |
| `error_message_size` | string | no | Localized message when `max_size_bytes` is exceeded. May contain `{limit}` (rendered as a human-readable size, e.g. "5 MB"). |
| `error_message_format` | string | no | Localized message when the file's extension / type doesn't match `accept`. |
| `prefill_filename` | string | no | When set, render the dropzone in the "file selected" state with this filename **but no actual File object behind it** — purely informational. Used by the middleend when re-emitting a form after a server-side error so the user sees what they had attempted. To re-submit the user must re-pick the file (browsers do not let JS programmatically reattach a previously-picked File across re-renders). The dropzone signals this state with a small caption (`import.upload.reattach_hint` — "Re-select the file to retry"). |

**Frontend behavior**

- Render: a dashed-bordered dropzone (~10rem tall) with an upload icon in the center and the placeholder text below. When a file is selected, the placeholder is replaced by the filename (mono-friendly truncation if long).
- Hover, drag-over, and focus states use the same color scheme as other interactive controls in the design system.
- Native `<input type="file">` is hidden; the dropzone forwards click to it. Drop events on the dropzone are captured (`preventDefault` on dragover, intercept the file from `dataTransfer.files[0]` on drop).
- On a new file selection (drop or pick): run the format check against `accept` (if set), then the size check against `max_size_bytes` (if set). On failure: show the corresponding error inline beneath the dropzone, do not retain the file.
- On the submit of the enclosing form: the component contributes its file to the `multipart/form-data` body under `name`. If no file is present, the form-level submit button is responsible for being disabled (the file_upload does not own form-level disabling).
- Reset: when the component receives a fresh `replace` from the server (matching `id`), it clears any local file and any local error. `prefill_filename` lets the server hint at the previously-uploaded filename.

**Example (AI Import idle):**

```json
{
  "type": "file_upload",
  "id": "import-file",
  "props": {
    "name": "file",
    "label": "File",
    "placeholder": "Drop a file here or click to browse",
    "hint": "CSV, TSV, XLS, XLSX, TXT — max 5 MB",
    "accept": ".csv,.tsv,.xls,.xlsx,.txt",
    "max_size_bytes": 5242880,
    "error_message_size": "File exceeds the {limit} limit.",
    "error_message_format": "Unsupported file format."
  }
}
```

### 5.2 Loading indicator extension

The `loading` field on actions (defined in `spec/sdui-actions.md` §2b) is extended from a string token to **string OR object**. Both forms are valid; existing handlers do not need to change.

**Form A (existing, unchanged):** `"loading": "section"` or `"loading": "full"` or absent.

**Form B (new):**

```json
"loading": {
  "scope": "full",
  "messages": [
    "Detecting columns…",
    "Mapping tickers…",
    "Resolving currencies…",
    "Building preview…",
    "Validating consistency…"
  ]
}
```

**Object fields**

| Field | Type | Required | Description |
|---|---|---|---|
| `scope` | enum | yes | `"section"` (semi-transparent overlay on the subtree at `target_id`) or `"full"` (fullscreen overlay). Same semantics as the string-token form. |
| `messages` | string[] | no | Localized phrases the frontend rotates through while the action is in flight. Empty / absent → behave like the bare token. |

**Frontend behavior**

- Render: spinner (unchanged) plus, when `messages` is non-empty, a single line of text below the spinner that displays the active phrase.
- Cycling: the frontend rotates through `messages` in order at a fixed interval of **2 seconds**, with a brief cross-fade between phrases. Once the last entry is reached, the cycle restarts from the first entry.
- The displayed text is purely cosmetic — it has no semantic relationship to actual progress on the server. The action's loading still ends when the response arrives, regardless of where in the cycle the displayed phrase is.
- For Form A (string token) and for Form B with `messages: []`, render only the spinner — backwards compatible.

**Backend usage on this screen**

Only `POST /actions/import/analyze` uses Form B in v1. Its handler injects 5–7 localized phrases under keys `import.loading.analyze.1` … `import.loading.analyze.7`. The other long-ish action (`POST /actions/import/sessions/:id/confirm`) uses Form A (`loading: full`) — confirms are usually fast enough that the fake-progress carousel adds noise.

Future flows that may benefit (out of this spec's scope): the `analysis` IA stream, large export proxies, slow snapshot auto-fetches.

---

## 6. Error handling

The standard project envelope (401 + redirect, 502 BACKEND_ERROR, 400 BAD_REQUEST) applies. Screen-specific behaviors:

| Situation | Surface |
|---|---|
| `file_upload` local size violation (analyze ≥ 5 MB, restore ≥ 10 MB) | Inline in the dropzone via `error_message_size`. No middleend round-trip. |
| `file_upload` local format mismatch | Inline in the dropzone via `error_message_format`. No round-trip. |
| BE `IMPORT_FILE_TOO_LARGE` (server-side check) | `replace ai-import-card` with idle + error banner. File is preserved via `prefill_filename` so the user sees it; they need to pick a smaller file to proceed. |
| BE AI parse failure (codes vary; e.g. unparseable rows, AI service unavailable) | `replace ai-import-card` with idle + error banner using the BE's localized message; preserve file + hint. |
| BE 422 on `resolve_gaps` | `replace import-modal-slot` with the review + error banner; user-submitted resolution values preserved. |
| BE 422 on `confirm` (e.g. unresolved gaps, expired data) | `replace import-modal-slot` with the review + error banner; do not touch root. |
| BE 404 on `confirm` / `resolve_gaps` (session expired) | `replace import-root` + warning snackbar `import.session_expired`. The user starts over from the upload form. |
| BE 404 on `cancel` | Treated as success — same root replace + info snackbar. |
| BE 5xx / network on any action | Snackbar error using `import.failure_generic`. The card / modal stays in its current state; the user can retry. |
| BE `RESTORE_FILE_TOO_LARGE` (server-side) | `replace restore-card` with idle (file preserved) + error banner + snackbar. |
| BE restore parse failure | Same as above. |
| Export proxy upstream failure | The browser receives the upstream error body (typically a small JSON or text). v1 acceptable; an enhancement could intercept and emit a snackbar via a pre-flight HEAD, but that doubles the round-trips. |

Backend codes that surface as inline banners on this screen: `IMPORT_FILE_TOO_LARGE`, `IMPORT_PARSE_FAILED`, any other code in the import error namespace, plus `RESTORE_FILE_TOO_LARGE` and any restore parse code. The middleend uses the localized `message` from the BE body verbatim; it does not re-translate codes.

---

## 7. i18n keys

Namespace `import.*` plus `common.*`. Locales `en` and `es`. Missing-key fallback: `en`, then the key itself.

**Screen / sections**

`import.title`, `import.ai.title`, `import.ai.description`, `import.export.title`, `import.export.description`, `import.restore.title`, `import.restore.description`.

**File upload (shared between AI Import and Restore — keys are reused with screen-specific values)**

`import.upload.label`, `import.upload.placeholder`, `import.upload.hint_ai` (mentions the multi-format list + 5 MB), `import.upload.hint_restore` (CSV + 10 MB), `import.upload.error_size` (with `{limit}`), `import.upload.error_format`, `import.upload.reattach_hint`.

**AI Import form**

`import.hint.label`, `import.hint.placeholder`, `import.analyze`.

**Loading messages (analyze)**

`import.loading.analyze.1` ("Detecting columns…"), `…2` ("Mapping tickers…"), `…3` ("Resolving currencies…"), `…4` ("Building preview…"), `…5` ("Validating consistency…"). (At minimum 5; the handler may add more.)

**Review banners and sections**

`import.review.blocking_banner` (with `{n}`), `import.review.ready_banner`, `import.review.summary`, `import.review.assumptions` (with `{n}`), `import.review.issues`, `import.review.warnings` (with `{n}`), `import.review.preview`, `import.review.preview.assets`, `import.review.preview.trades`, `import.review.preview.snapshots`.

**Issues**

`import.gaps.affected_rows` (with `{rows}`), `import.gaps.input_placeholder`, `import.gaps.save`.

**Action bar**

`import.review.confirm`, `import.review.cancel`, `import.review.status.needs_review`, `import.review.status.ready`.

**Outcomes**

`import.success` (with `{assets} {trades} {snapshots} {warnings}`), `import.cancelled`, `import.session_expired`, `import.failure_generic`.

**Restore**

`import.restore.submit`, `import.restore.success_title`, `import.restore.col.imported`, `import.restore.col.skipped`, `import.restore.row.assets`, `import.restore.row.trades`, `import.restore.row.snapshots`, `import.restore.row.snapshot_entries`, `import.restore.try_again`, `import.restore.error_generic`.

**Export**

`import.export.submit`.

**Shared**

`common.cancel`.

Concrete strings live in `locales/en.json` and `locales/es.json`.

---

## 8. Acceptance criteria

- [ ] `GET /screens/import` without a valid JWT returns `401` with the documented redirect.
- [ ] With a valid JWT the screen renders four regions under `import-root`: header, `ai-import-card`, `export-restore-group` (containing `export-card` + `restore-card`), and an empty `import-modal-slot`.
- [ ] No HideValues toggle in the screen header; no `sensitive: true` on any component in this screen.
- [ ] `ai-import-card` idle state renders a `file_upload` (5 MB cap, multi-format), a `textarea` for hint (max 500 chars), and an `Analyze file` button disabled until a file is selected.
- [ ] `file_upload` performs local validation: a file ≥ 5 MB or with an extension not in `accept` triggers an inline error inside the dropzone and does not submit. The same applies to the Restore upload at 10 MB / `.csv`.
- [ ] Click `Analyze file` triggers a `submit` action with `loading: { scope: "full", messages: [...] }`. The frontend renders the fullscreen overlay with a spinner plus a single-line message that rotates through `messages` every 2 seconds in order, looping at the end. The overlay clears when the response arrives.
- [ ] Successful analyze response replaces `import-modal-slot` with the review modal: banner (warning if blocking>0, info otherwise), summary card with collapsible assumptions, issues section (only if blocking>0) with one card per blocking gap, warnings section (only if any warning gap, collapsed by default), preview section with three collapsible tables (Assets / Trades / Snapshots), and a sticky action bar with status badge + Cancel + Confirm.
- [ ] Confirm is disabled when `session.status != "ready"`. The status badge is amber for `needs_review`, green for `ready`.
- [ ] Save resolutions submits `resolutions[<gap_id>]` inputs to `POST /actions/import/sessions/:id/resolve_gaps` with `loading: section` over the modal-slot. On success, the modal-slot replace re-renders with the new session and the Confirm button enables when status flips to `ready`. On 422, the same modal-slot replace also adds an inline error banner and preserves user-submitted values.
- [ ] Confirm submits to `POST /actions/import/sessions/:id/confirm` with `loading: full`. On success the screen root is replaced with a fresh tree (cards in idle, modal-slot empty) plus a success snackbar interpolating the four counts.
- [ ] Cancel submits to `POST /actions/import/sessions/:id/cancel` with `loading: full`. The root is replaced + info snackbar `import.cancelled`. A 404 from the BE is treated as success.
- [ ] A 404 from confirm / resolve_gaps (session expired) triggers a root replace + warning snackbar `import.session_expired`.
- [ ] BE 5xx surfaces as `import.failure_generic` snackbar; the card / modal keeps its previous state.
- [ ] AI Import failure (size, parse, etc.) replaces `ai-import-card` with idle + error banner, preserving file (via `prefill_filename`) and hint.
- [ ] `export-card` renders title, description, and an `Export all data` button whose action is `open_url` to `/actions/import/export`. The middleend handler streams `GET /v1/export` from the BE through to the client with the same `Content-Disposition` header. No SDUI replace happens.
- [ ] `restore-card` idle state renders title, description, `file_upload` (10 MB, `.csv`), and a disabled-until-file `Restore` button. Action: `submit` to `POST /actions/import/restore` with `loading: section` over `restore-card`.
- [ ] Restore success replaces `restore-card` with a 4-row table (Assets / Trades / Snapshots / Snapshot entries) showing imported and skipped counts (right-aligned, tabular-nums), a green title, and a `Restore another file` button that replaces `restore-card` with the idle subtree (embedded `tree`, no extra round-trip).
- [ ] Restore failure replaces `restore-card` with idle + error banner (file preserved) + error snackbar.
- [ ] On mobile, `export-restore-group` collapses from a 2-column grid to a 1-column stack while preserving the same children and behavior.
- [ ] Custom component `file_upload` is documented in `spec/sdui-custom-components.md` with all props from §5.1, the local-validation behavior, the `prefill_filename` semantics, and the multipart submit contract.
- [ ] SDUI `loading` field in `spec/sdui-actions.md` §2b accepts both the string token form and the new `{scope, messages?}` object form. Existing string-form usages elsewhere in the project remain unchanged and behave identically.
- [ ] All user-facing strings resolve via the i18n keys listed in §7 for `en` and `es`. Backend validation messages surface localized per `Accept-Language`; otherwise the BE `code` is shown.

---

## 9. Out of scope (v1)

- Asynchronous / pollable AI import (would require backend changes — non-goal of this project).
- File preview before submitting (e.g. sniffing the first N rows in the browser to show columns). The hint textarea covers the user's intent.
- Multiple concurrent import sessions in flight. The user can have at most one active session; opening a new analyze while a review modal is open implies cancelling the prior session first (the modal's Cancel button).
- Granular per-row deselection in the preview tables. v1 is all-or-nothing on confirm.
- HideValues toggle on this screen (kept portfolio-only).
- `tabs` SDUI primitive (the design avoids tabs entirely).
- Async retry / background queueing for the export proxy.
