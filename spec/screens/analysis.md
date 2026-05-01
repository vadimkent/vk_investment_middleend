# Analysis Screen

Single screen at `/screens/analysis` that surfaces a streaming AI conversation about the user's portfolio. Two visual states — a **start state** (focus textarea + submit button) and a **chat state** (reset row + `analysis_chat` streaming component) — both anchored at the same URL. Submitting the start form replaces the content area with the chat state; clicking "New analysis" replaces it back. The screen opens no modals. The middleend is stateless: `session_id` lives in the `analysis_chat` component's local state and is forwarded via URL on follow-ups.

---

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET`  | `/screens/analysis` | Full screen render (start state). |
| `POST` | `/actions/analysis/start` | Body: `{focus?}`. Validates `focus` <= 500 chars. No backend call. Returns `replace target_id="analysis-content"` with the chat-state subtree. Loading: full. |
| `GET`  | `/actions/analysis/reset` | Returns `replace target_id="analysis-content"` with the start-state subtree. |
| `GET`  | `/actions/analysis/stream` | Query: `focus` (optional). SSE proxy: opens `POST /v1/analysis/sessions` to the backend, bypasses SSE stream byte-for-byte. |
| `POST` | `/actions/analysis/sessions/:id/messages` | Body: `{content}`. SSE proxy: opens `POST /v1/analysis/sessions/:id/messages`, bypasses SSE stream byte-for-byte. |

All endpoints require a valid JWT. Missing/invalid JWT returns `401 {"error":"unauthorized","redirect":"/login"}`. `start` and `reset` return standard `ActionResponse` JSON. `stream` and `sessions/:id/messages` return `Content-Type: text/event-stream`.

### Backend dependencies

- `POST /v1/analysis/sessions` — body `{focus?: string}`. Streams SSE: `event: session` (carries `session_id`), then `event: delta` (`{text}`), then `event: done` or `event: error`. Rate-limited; `429` returned as HTTP status. Errors: `ANALYSIS_FOCUS_TOO_LONG`, `AI_PROVIDER_UNAVAILABLE`, `AI_RATE_LIMITED`, `AI_TIMEOUT`, `AI_CONTEXT_TOO_LARGE`.
- `POST /v1/analysis/sessions/:id/messages` — body `{content}`. Same SSE stream shape. Additional errors: `ANALYSIS_SESSION_NOT_FOUND`, `ANALYSIS_SESSION_EXPIRED`, `ANALYSIS_TOO_MANY_MESSAGES`.
- `GET /v1/analysis/sessions/:id` — JSON history. Not used in v1.
- `DELETE /v1/analysis/sessions/:id` — cleanup. Not called in v1 (TTL on the backend handles expiration).

---

## Layout

```
screen "analysis-screen"           (title: "Analysis")
└── column "analysis-root"         (gap: lg)
    ├── header "analysis-header"   (title only — no actions)
    └── column "analysis-content"  (the switcheable region)
        └── (start_state | chat_state)
```

`analysis-content` is replaced as a whole via `replace target_id="analysis-content"`.

### Start state (initial render, post-reset)

```
column "analysis-content"  (align_items: center, justify_items: center, gap: lg)
└── card "analysis-start-card"  (max-width: lg)
    └── form "analysis-start-form"
        └── column "analysis-start-body"  (gap: md)
            ├── icon "brain-circuit" (muted, centered)
            ├── text "analysis-start-description"  (sm, muted, centered)
            ├── textarea "analysis-focus"
            │     name: "focus"
            │     label: analysis.start.focus_label
            │     placeholder: analysis.start.focus_placeholder
            │     max_length: 500
            └── row "analysis-start-actions"  ([1fr, auto], gap: sm)
                  ├── spacer
                  └── button "analysis-start-submit"
                        action: submit POST /actions/analysis/start
                        target_id: "analysis-start-form"
                        loading: full
                        label: analysis.start.submit
```

### Chat state

```
column "analysis-content"  (gap: md)
├── row "analysis-chat-header"  ([1fr, auto], gap: sm)
│   ├── spacer
│   └── button "analysis-new-btn"  (size sm, secondary ghost)
│         action: reload GET /actions/analysis/reset
│         target_id: "analysis-content"
│         loading: section
│         label: analysis.new_analysis
└── analysis_chat "analysis-chat"  (props per custom component spec)
```

The reset button lives inside `analysis-content` so the `reset` action replaces both the button and the chat in one swap.

---

## Custom component

The `analysis_chat` component spec is in [`../sdui-custom-components.md`](../sdui-custom-components.md) section 5. It is a self-contained streaming chat surface: SSE attachment on mount, incremental `delta` append, blinking cursor, auto-scroll, local `session_id` state, and error mode bifurcation (recoverable/terminal).

---

## Data and business rules

### SSE event protocol (proxied unchanged)

| Event | Payload | Component behavior |
|---|---|---|
| `session` | `{session_id: string}` | Stash `session_id`. Append placeholder assistant message. Show streaming cursor. |
| `delta` | `{text: string}` | Append `text` to the last assistant message. Auto-scroll to bottom. |
| `done` | `{}` | Hide cursor. Re-enable input. |
| `error` | `{code: string, message: string}` | Render inline error banner. If `code` is in `terminal_error_codes`: disable input and show CTA. Otherwise: keep input enabled. Remove empty placeholder assistant message if no delta received. |

The middleend does not transform event names or payloads.

### Validation

- `focus` in `POST /actions/analysis/start`: > 500 chars returns the start form with an inline error banner. No backend call.
- `focus` is forwarded from the start form body to the `initial_endpoint` query parameter as `?focus=<urlencoded>`. Omitted when empty.

---

## SSE proxy

Both `GET /actions/analysis/stream` and `POST /actions/analysis/sessions/:id/messages` use the same proxy pattern:

| Upstream response | Middleend action |
|---|---|
| `200 OK` | Set SSE headers; copy body in chunks to client; flush after each write. |
| `401 Unauthorized` | Return `401 {"error":"unauthorized","redirect":"/login"}` (JSON, no SSE headers). |
| `429 Too Many Requests` | Set SSE headers; emit `event: error` with `code: RATE_LIMITED`; close. |
| `5xx` | Set SSE headers; emit `event: error` with `code: AI_PROVIDER_UNAVAILABLE`; close. |
| Other `4xx` | Emit `event: error` with `code: INTERNAL_ERROR`; close. |
| Pre-stream BE error with body `{error:{code,message}}` | Pass BE's `code` and `message` through verbatim. |
| Mid-stream upstream drop | Emit `event: error` with `code: INTERNAL_ERROR, message: "connection lost"`; close. |

Client cancellation: request context is cancelled when the client disconnects; upstream connection closes automatically. No `Client.Timeout` (breaks long streams); `ResponseHeaderTimeout` only.

SSE response headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`, `X-Accel-Buffering: no`.

---

## i18n keys

Namespace `analysis.*`.

**Screen / start state:** `analysis.title`, `analysis.start.description`, `analysis.start.focus_label`, `analysis.start.focus_placeholder`, `analysis.start.submit`.

**Chat state:** `analysis.new_analysis`, `analysis.chat.placeholder`, `analysis.chat.submit_label`, `analysis.chat.streaming_label`, `analysis.chat.terminal_cta`.

**Error messages:** `analysis.error.session_not_found`, `analysis.error.session_expired`, `analysis.error.too_many_messages`, `analysis.error.focus_too_long`, `analysis.error.provider_unavailable`, `analysis.error.rate_limited`, `analysis.error.timeout`, `analysis.error.context_too_large`, `analysis.error.internal`, `analysis.error.default`.

**Feedback:** `analysis.feedback.start_failed` (snackbar fallback if `start` action errors).

---

## Error handling

| Situation | Surface |
|---|---|
| `focus` > 500 chars in `start` | Replace `analysis-start-form` with form + inline error banner. No backend call. |
| `start` or `reset` connection error | Snackbar via `ActionResponse{action:"none", feedback: snackbar}`. |
| SSE `401` pre-stream | `401 {"error":"unauthorized","redirect":"/login"}` (JSON, no SSE). |
| SSE `429` pre-stream | SSE `event: error` with `code: RATE_LIMITED`, then close. |
| SSE `5xx` pre-stream | SSE `event: error` with `code: AI_PROVIDER_UNAVAILABLE`, then close. |
| SSE other `4xx` pre-stream | SSE `event: error` with `code: INTERNAL_ERROR`, then close. |
| SSE mid-stream upstream drop | SSE `event: error` with `code: INTERNAL_ERROR`, then close. |
| SSE recoverable error | Inline banner; input stays enabled. |
| SSE terminal error | Inline banner; input disabled; CTA visible; CTA executes `reset_action`. |

---

## Acceptance criteria

- `GET /screens/analysis` without a valid JWT returns `401` with the documented redirect.
- With a valid JWT, the screen renders `analysis-screen` containing `analysis-root` with a header (title only) and `analysis-content` in start state.
- Submit of the start form triggers `POST /actions/analysis/start`. Handler returns `replace target_id="analysis-content"` with the chat-state subtree. No backend call.
- `focus` > 500 chars returns the start form with inline error banner; no backend call.
- The `analysis_chat` `initial_endpoint` is `/actions/analysis/stream?focus=<urlencoded>` when focus non-empty, or `/actions/analysis/stream` otherwise.
- `GET /actions/analysis/stream` proxies `POST /v1/analysis/sessions`, forwards `Authorization`, bypasses SSE chunk-for-chunk with SSE headers set.
- `POST /actions/analysis/sessions/:id/messages` proxies `POST /v1/analysis/sessions/:id/messages`. SSE stream bypassed.
- Pre-stream BE errors surfaced per the SSE proxy table above.
- Mid-stream upstream errors emit `event: error` with `code: INTERNAL_ERROR`.
- Client disconnection cancels the upstream request via context.
- `GET /actions/analysis/reset` returns `replace target_id="analysis-content"` with the start-state subtree.
- All user-facing strings resolve via `analysis.*` keys in `en` and `es`.
