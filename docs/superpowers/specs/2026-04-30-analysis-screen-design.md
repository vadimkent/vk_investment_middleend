# Analysis Screen — Design

Design spec for the Analysis screen in the vk-investment middleend. Surfaces a streaming AI conversation about the user's portfolio, backed by the backend's SSE-based `/v1/analysis/sessions` endpoints. Introduces one new SDUI primitive: a custom `analysis_chat` component that opens its own SSE channel, manages incremental message append with markdown rendering, and handles error states locally without round-tripping the SDUI tree per chunk.

---

## 1. Overview

Single screen at `/screens/analysis` (sidebar entry: `Analysis`) with two visual states:

1. **Start state (no session)** — a centered card with an optional "Focus area" textarea and an "Analyze my portfolio" button.
2. **Chat state (active session)** — a "New analysis" reset button atop a chat surface streaming the assistant's response and accepting follow-up questions.

Same URL throughout. Submitting the start form replaces the screen's content area with the chat state; clicking "New analysis" replaces it back to the start state. The screen does not open modals.

The interaction with the backend is fundamentally streaming. The middleend proxies SSE through to the browser without transforming events; the `analysis_chat` component owns the SSE attachment, parses events as they arrive, and updates its own state. The middleend is stateless: it does not buffer responses, does not cache sessions, does not retain `session_id` across requests.

---

## 2. Endpoints

All endpoints are **protected** (JWT). Missing / invalid / expired JWT → `401 {"error":"unauthorized","redirect":"/login"}`. Backend 5xx / network / malformed → `502 BACKEND_ERROR` for non-SSE actions; for SSE actions, see §6.

| Method | Path | Purpose |
|---|---|---|
| `GET`  | `/screens/analysis` | Full screen render (start state). |
| `POST` | `/actions/analysis/start` | Submits `{focus}` from the start form. **Does not touch the backend.** Returns `replace target_id="analysis-content"` with the chat-state subtree (reset row + `analysis_chat`). Server-side validates `focus` ≤ 500 chars (defense in depth — the textarea client-side caps at 500). |
| `GET`  | `/actions/analysis/reset` | "New analysis" button. Returns `replace target_id="analysis-content"` with the start-state subtree. |
| `GET`  | `/actions/analysis/stream` | Query: `focus` (optional). SSE proxy: opens `POST /v1/analysis/sessions` to the backend with body `{focus}` and bypasses the SSE stream byte-for-byte to the client. |
| `POST` | `/actions/analysis/sessions/:id/messages` | Body: `{content}`. SSE proxy: opens `POST /v1/analysis/sessions/:id/messages` to the backend with the same body and bypasses the SSE stream. |

`start` and `reset` return standard `ActionResponse` JSON. `stream` and `sessions/:id/messages` are **streaming endpoints** with `Content-Type: text/event-stream`; their bodies are SSE byte streams unmodified from upstream.

The middleend stores **no analysis state**: `session_id` lives in the `analysis_chat` component's local state on the FE; the middleend re-receives it via the URL on each follow-up.

### Backend dependencies

- `POST /v1/analysis/sessions` — body `{focus?: string}`. Streams via SSE: `event: session` (carries `session_id`), then `event: delta` (`{text}`) repeatedly, then `event: done` or `event: error`. Rate-limited by user_id with cooldown; `429` returned as HTTP status when limit exceeded. Errors: `ANALYSIS_FOCUS_TOO_LONG`, `AI_PROVIDER_UNAVAILABLE`, `AI_RATE_LIMITED`, `AI_TIMEOUT`, `AI_CONTEXT_TOO_LARGE`.
- `POST /v1/analysis/sessions/:id/messages` — body `{content}`. Streams the same way. Errors include the above plus `ANALYSIS_SESSION_NOT_FOUND`, `ANALYSIS_SESSION_EXPIRED`, `ANALYSIS_TOO_MANY_MESSAGES`.
- `GET /v1/analysis/sessions/:id` — JSON, full message history. **Not used in v1** (the FE keeps state locally; no rehydration).
- `DELETE /v1/analysis/sessions/:id` — explicit cleanup. **Not called in v1** (TTL on the backend handles expiration).

---

## 3. Layout

Two regions stacked vertically under `analysis-root`:

```
screen "analysis-screen"           (title: "Analysis")
└── column "analysis-root"         (gap: lg)
    ├── header "analysis-header"   (title only — no actions)
    └── column "analysis-content"  (the switcheable region)
        └── (start_state | chat_state)
```

**`analysis-content` is replaced as a whole** via `replace target_id="analysis-content"`. It carries the entire content for whichever state the screen is in.

### Start state (initial render, post-reset)

```
column "analysis-content"  (align_items: center, justify_items: center, gap: lg)
└── card "analysis-start-card"  (max-width: lg)
    └── form "analysis-start-form"
        └── column "analysis-start-body"  (gap: md)
            ├── icon "brain-circuit" (or similar; muted, centered)
            ├── text "analysis-start-description"  (sm, muted, centered)
            ├── textarea "analysis-focus"
            │     name: "focus"
            │     label: "Focus area (optional)"
            │     placeholder: "e.g. risk exposure, crypto allocation, dividend potential"
            │     max_length: 500
            └── row "analysis-start-actions"  ([1fr, auto], gap: sm)
                  ├── spacer
                  └── button "analysis-start-submit"  (size sm, primary solid)
                        action: submit POST /actions/analysis/start, target_id="analysis-start-form", loading: full
                        label: "Analyze my portfolio"
```

### Chat state

```
column "analysis-content"  (gap: md)
├── row "analysis-chat-header"  ([1fr, auto], gap: sm)
│   ├── spacer
│   └── button "analysis-new-btn"  (size sm, secondary ghost)
│         action: reload GET /actions/analysis/reset, target_id="analysis-content", loading: section
│         label: "New analysis"
└── analysis_chat "analysis-chat"  (props per §4)
```

The "New analysis" button lives **inside** `analysis-content`, not in the header — so a re-render via `reset` cleanly replaces both the button and the chat in one swap. `loading: section` covers `analysis-content` while waiting for the start subtree to come back.

---

## 4. Custom component `analysis_chat`

A self-contained streaming chat surface. Lives in `spec/sdui-custom-components.md` as a new top-level entry alongside `line_chart`, `pie_chart`, `wizard`, `file_upload`.

### Why custom

The component combines several behaviors that no base primitive offers:

- **SSE attachment**: opens a `fetch`+`ReadableStream` channel on mount and keeps it alive across re-renders triggered locally by message-append updates.
- **Incremental append**: `delta` events extend the last assistant message in-place without a server round-trip.
- **Markdown render** in assistant messages (remark-gfm) + plain `whitespace: pre-wrap` in user messages.
- **Streaming cursor** (blinking) while a response is in flight.
- **Auto-scroll** on every new chunk.
- **Local session_id state** captured from the first SSE `session` event, used to fill the `{session_id}` placeholder in `followup_endpoint` for subsequent posts.
- **Error mode bifurcation** (recoverable vs terminal) without round-trips.

Composing this from base primitives (`text`, `input`, `button`, `replace`) is not feasible: every chunk would require an SDUI replace, defeating the streaming feel.

### Props

| Prop | Type | Required | Description |
|---|---|---|---|
| `initial_endpoint` | string | yes | URL the component opens an SSE channel to **on mount**. The first event must be `session` carrying `session_id`; subsequent events are `delta`, then `done` or `error`. |
| `followup_endpoint` | string | yes | URL template used for follow-up messages. Must contain `{session_id}`, which the component substitutes at send time using the id captured from `initial_endpoint`. The follow-up request is a `POST` with body `{content}` and is handled as another SSE stream. |
| `placeholder` | string | yes | Text displayed in the input when empty (e.g. *"Ask a follow-up question…"*). Localized by the middleend. |
| `submit_label` | string | yes | Aria-label for the icon-only send button (e.g. *"Send"*). Localized. |
| `streaming_label` | string | no | Small muted text rendered alongside the blinking cursor while a response is streaming (e.g. *"AI is thinking…"*). If absent, only the cursor renders. Localized. |
| `max_input_length` | int | no | Maximum characters allowed in the input. Default `2000`. A character counter appears in the input's bottom-right corner when the value approaches the limit (see Frontend behavior below). |
| `error_messages` | `map<string, string>` | yes | Map of backend error code (per §6) to localized message. Must include a `"default"` key as fallback. Keys: `ANALYSIS_SESSION_NOT_FOUND`, `ANALYSIS_SESSION_EXPIRED`, `ANALYSIS_TOO_MANY_MESSAGES`, `ANALYSIS_FOCUS_TOO_LONG`, `AI_PROVIDER_UNAVAILABLE`, `AI_RATE_LIMITED`, `AI_TIMEOUT`, `AI_CONTEXT_TOO_LARGE`, `RATE_LIMITED`, `INTERNAL_ERROR`, `default`. |
| `terminal_error_codes` | `string[]` | yes | Codes that, when received, transition the component into terminal mode (input disabled + CTA visible). Recommended set: `["ANALYSIS_SESSION_EXPIRED", "ANALYSIS_SESSION_NOT_FOUND", "ANALYSIS_TOO_MANY_MESSAGES"]`. Other codes are treated as recoverable (banner stays until the next user send, input remains enabled). |
| `terminal_cta_label` | string | yes | Label for the CTA button shown in terminal mode (e.g. *"Start a new analysis"*). Localized. |
| `reset_action` | `Action` | yes | Action executed by the terminal CTA. Typically `Reload(/actions/analysis/reset, target_id="analysis-content")`. |

### SSE event protocol

The component consumes the backend's SSE event names **unchanged** (the proxy does not transform them):

| Event | Payload | Component behavior |
|---|---|---|
| `session` | `{session_id: string}` | Stash `session_id` into local state. Append a placeholder assistant message to the conversation array. Show streaming cursor. |
| `delta` | `{text: string}` | Append `text` to the `content` of the last assistant message. Auto-scroll to bottom. |
| `done` | `{}` | Hide the streaming cursor on the last message. Re-enable the input. Keep the SSE channel closed (the server has closed it). |
| `error` | `{code: string, message: string}` | Render an inline error banner above the input with `error_messages[code] ?? error_messages["default"]`. If `code ∈ terminal_error_codes`: disable the input, show the CTA. Otherwise: keep the input enabled. Remove the empty placeholder assistant message if it never received any `delta`. |

Connection-level errors (network drop, fetch abort) are surfaced internally as `error` with `code: "INTERNAL_ERROR"`. The middleend also emits this code when the upstream connection drops mid-stream (see §6).

### Frontend behavior

1. **Mount**: open SSE to `initial_endpoint`. Initialize `messages: []`, `session_id: null`, `is_streaming: true`, `error: null`, `is_terminal: false`. As soon as the first `session` event arrives, stash `session_id` and push a placeholder assistant message (`{role: "assistant", content: ""}`).
2. **Streaming render**: the messages list scrolls automatically as content grows. Each `delta` appends to the last assistant message and triggers a scroll-to-bottom. The streaming cursor is a small blinking block at the end of the assistant's content.
3. **`done`**: clear cursor on the last message. Set `is_streaming: false`.
4. **Send follow-up** (user types in input → presses Enter without Shift, or clicks Send):
   - Validate: trimmed length > 0 and ≤ `max_input_length`.
   - Push `{role: "user", content}` to messages.
   - Push `{role: "assistant", content: ""}` placeholder.
   - Open SSE to `followup_endpoint` with `{session_id}` resolved, body `{content}`.
   - Same delta/done/error loop as initial.
5. **Error inline**: banner above the input area, variant warning (recoverable) or error (terminal). The banner persists until either the user starts another send (recoverable) or the user clicks the terminal CTA (terminal).
6. **Terminal mode**: input disabled, send button disabled, banner persists, `terminal_cta_label` button rendered below the banner. Clicking it executes `reset_action`.
7. **Markdown**: assistant messages render via remark-gfm with table support, code blocks, lists, headings, etc. User messages render as plain text with `whitespace: pre-wrap`.
8. **Character counter**: shown in the bottom-right of the input area only when the current value's length crosses ~75% of `max_input_length`. Format: `<current> / <max>`. Turns destructive color when over `max_input_length`.
9. **Disconnection**: if `fetch` aborts (user navigated away, network drop), the component stops the loop. If the abort was not user-initiated (no manual cancel), surface as `INTERNAL_ERROR`.
10. **Enter-to-send**: pressing Enter (no Shift, no IME composition) in the input invokes Send. Shift-Enter inserts a newline.
11. **Unmount cleanup**: on unmount (the parent slot is replaced — e.g. user clicks "New analysis" mid-stream), the component aborts any in-flight `fetch`+SSE before being torn down. This prevents leaked connections to the middleend (and through it, to the backend).

### Layout (visual structure)

- Outer: column flex, full available height of the parent slot, `gap: 0`.
- **Messages area**: `flex: 1`, `overflow-y: auto`, padding horizontal. Inside: a centered max-width container (e.g. 3xl) with vertical-stacked message bubbles.
  - User: right-aligned, max-width 85%, primary background, rounded.
  - Assistant: left-aligned (or full-width depending on design), muted/transparent background, rounded, prose styling for markdown.
- **Input area**: pinned to the bottom, `border-top` separator, padding. Inside: a centered max-width container with a row containing `[textarea, send-button]`. Textarea auto-resizes between 1 and ~4 rows. Send button is icon-only (`send` icon).
- **Error banner**: rendered between the messages area and the input area when `error` is present.
- **Terminal CTA**: rendered below the error banner when in terminal mode.

### Example

```json
{
  "type": "analysis_chat",
  "id": "analysis-chat",
  "props": {
    "initial_endpoint": "/actions/analysis/stream?focus=risk%20exposure",
    "followup_endpoint": "/actions/analysis/sessions/{session_id}/messages",
    "placeholder": "Ask a follow-up question…",
    "submit_label": "Send",
    "streaming_label": "AI is thinking…",
    "max_input_length": 2000,
    "error_messages": {
      "ANALYSIS_SESSION_NOT_FOUND": "Session not found.",
      "ANALYSIS_SESSION_EXPIRED": "Session expired. Start a new analysis.",
      "ANALYSIS_TOO_MANY_MESSAGES": "Conversation length limit reached. Start a new analysis.",
      "ANALYSIS_FOCUS_TOO_LONG": "Focus area is too long.",
      "AI_PROVIDER_UNAVAILABLE": "AI provider unavailable. Please retry.",
      "AI_RATE_LIMITED": "AI rate limit reached. Please retry shortly.",
      "AI_TIMEOUT": "AI request timed out. Please retry.",
      "AI_CONTEXT_TOO_LARGE": "Portfolio context is too large for the AI.",
      "RATE_LIMITED": "Too many requests. Please wait a moment before trying again.",
      "INTERNAL_ERROR": "Connection lost. Please try again.",
      "default": "Something went wrong. Please retry."
    },
    "terminal_error_codes": [
      "ANALYSIS_SESSION_EXPIRED",
      "ANALYSIS_SESSION_NOT_FOUND",
      "ANALYSIS_TOO_MANY_MESSAGES"
    ],
    "terminal_cta_label": "Start a new analysis",
    "reset_action": {
      "trigger": "click",
      "type": "reload",
      "endpoint": "/actions/analysis/reset",
      "target_id": "analysis-content",
      "loading": "section"
    }
  }
}
```

---

## 5. SSE proxy implementation

Both `GET /actions/analysis/stream` and `POST /actions/analysis/sessions/:id/messages` are streaming proxies. The implementation pattern is the same:

1. **HTTP client setup**. Use a dedicated client whose `Transport` has a tuned `ResponseHeaderTimeout` (e.g. 30 s — generous enough for the backend to start its stream) **but not** `Client.Timeout` (that timeout aborts mid-body, breaking long streams). Body reads are governed by the request `context.Context` only.
2. **Open the upstream request**. Forward `Authorization` from the incoming request. For `stream`: `POST <backend>/v1/analysis/sessions` with body `{focus}` (omit when empty). For `messages/:id`: `POST <backend>/v1/analysis/sessions/:id/messages` with body `{content}` from the incoming request.
3. **Inspect the upstream response**:
   - **`200 OK`** — proceed to bypass.
   - **`401 Unauthorized`** — return `401 {"error":"unauthorized","redirect":"/login"}` to the client (no SSE headers were sent yet).
   - **`429 Too Many Requests`** (rate limited) — set SSE headers (so the client's SSE parser can read it), emit one event: `event: error\ndata: {"code":"RATE_LIMITED","message":"<from BE>"}\n\n`, flush, close. The client treats this as a recoverable error.
   - **`5xx`** — emit `event: error\ndata: {"code":"AI_PROVIDER_UNAVAILABLE","message":"..."}`, close. Recoverable.
   - **Other 4xx** (404, 422, etc.) — emit `event: error\ndata: {"code":"INTERNAL_ERROR","message":"..."}`, close. Recoverable.
   - **Pre-stream errors that carry a code in the body** (`{error:{code, message}}` shape from BE) — pass the BE's `code` and `message` through verbatim instead of synthesizing one. This covers cases like `ANALYSIS_FOCUS_TOO_LONG` returned as `400` from `/v1/analysis/sessions`.
4. **Bypass the body**:
   - Set `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`, `X-Accel-Buffering: no` on the response.
   - Read upstream body in chunks (e.g. 4 KB buffer) and write to `c.Writer`, calling `Flush()` after each successful write. Do **not** buffer the full body.
   - Stop on EOF (upstream completed) or on context cancellation (client disconnected).
5. **Mid-stream upstream errors** (network drop while reading body): emit `event: error\ndata: {"code":"INTERNAL_ERROR","message":"connection lost"}\n\n`, flush, close.
6. **Client cancellation**: the request context is canceled when the client closes the connection. The HTTP request to the backend uses the same context, so the upstream connection is closed automatically. No explicit cleanup needed.

The backend's SSE event names (`session`, `delta`, `done`, `error`) and JSON payloads are passed through unchanged. The middleend is a transparent SSE relay.

---

## 6. Error handling

Beyond the standard envelope (401 redirect, 502 BACKEND_ERROR for non-SSE, 400 BAD_REQUEST):

| Situation | Surface |
|---|---|
| `start` body has `focus` > 500 chars (server-side) | `replace target_id="analysis-start-form"` with the form re-emitted + an inline error banner ("Focus area is too long."). No round-trip to backend. |
| `start` connection error (extremely unlikely since it doesn't hit the backend) | Snackbar error via `ActionResponse{action:"none", feedback: snackbar}`. |
| `reset` connection error | Snackbar error. |
| SSE pre-stream HTTP error (`401`, `429`, `5xx`) | See §5 step 3. `401` → JSON unauthorized. Others → SSE single-event-error then close. |
| SSE recoverable error mid-stream (`AI_TIMEOUT`, `AI_PROVIDER_UNAVAILABLE`, `AI_RATE_LIMITED`, `AI_CONTEXT_TOO_LARGE`, `RATE_LIMITED`, `INTERNAL_ERROR`) | Component shows inline banner, input stays enabled. User can retry by sending again. |
| SSE terminal error (`ANALYSIS_SESSION_EXPIRED`, `ANALYSIS_SESSION_NOT_FOUND`, `ANALYSIS_TOO_MANY_MESSAGES`) | Component shows inline banner, input disabled, CTA visible. CTA triggers `reset_action`. |
| `ANALYSIS_FOCUS_TOO_LONG` from the BE (defense in depth in case the FE max_length is bypassed) | Component treats as recoverable error. The `focus` is no longer editable from the chat state — the user clicks the terminal CTA (or "New analysis" reset row) to go back to the start form. So this is in practice a "soft terminal" — listed in `terminal_error_codes` if desired. **Default: keep recoverable, expect the FE max_length to prevent it.** |
| Client→middleend SSE drop | Component surfaces as `INTERNAL_ERROR` recoverable. |
| Middleend→backend SSE drop mid-stream | Middleend emits `INTERNAL_ERROR` event before closing (per §5 step 5). |

The middleend never re-translates BE error messages. It passes the BE's localized `message` through. The component prefers `error_messages[code]` (its own localized copy) over the message text — using `error_messages` ensures consistent UX even when the BE message is technical. The BE message is a fallback only when the code is unknown.

---

## 7. i18n keys

Namespace `analysis.*`. Locales `en` and `es`.

**Screen / start state**
`analysis.title`, `analysis.start.description`, `analysis.start.focus_label`, `analysis.start.focus_placeholder`, `analysis.start.submit`.

**Chat state**
`analysis.new_analysis`, `analysis.chat.placeholder`, `analysis.chat.submit_label`, `analysis.chat.streaming_label`, `analysis.chat.terminal_cta`.

**Error messages**
`analysis.error.session_not_found`, `analysis.error.session_expired`, `analysis.error.too_many_messages`, `analysis.error.focus_too_long`, `analysis.error.provider_unavailable`, `analysis.error.rate_limited`, `analysis.error.timeout`, `analysis.error.context_too_large`, `analysis.error.internal`, `analysis.error.default`.

**Feedback**
`analysis.feedback.start_failed` (snackbar fallback if `start` action errors).

Concrete strings live in `locales/en.json` and `locales/es.json`. Missing-key fallback: `en` → key.

---

## 8. Acceptance criteria

- [ ] `GET /screens/analysis` without a valid JWT returns `401` with the documented redirect.
- [ ] With a valid JWT, the screen renders `analysis-screen` containing `analysis-root` with a header (title only) and `analysis-content` in start state (icon + description + focus textarea + Analyze button right-aligned).
- [ ] Submit of the start form triggers `POST /actions/analysis/start` with `loading: full`. The handler does not call the backend and returns `replace target_id="analysis-content"` with the chat-state subtree (a row with the "New analysis" button + the `analysis_chat` component).
- [ ] The `analysis_chat` `initial_endpoint` is `/actions/analysis/stream?focus=<urlencoded>` when focus is non-empty, or `/actions/analysis/stream` otherwise.
- [ ] Server-side validation of `focus` in `start`: > 500 chars → replace the form with an inline error banner (no backend call).
- [ ] `GET /actions/analysis/stream` opens `POST /v1/analysis/sessions` to the backend, forwards `Authorization`, sends body `{focus}` (omit when empty), and bypasses the SSE response chunk-for-chunk to the client. Headers `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`, `X-Accel-Buffering: no` are set on the proxied response.
- [ ] `POST /actions/analysis/sessions/:id/messages` opens `POST /v1/analysis/sessions/:id/messages` with body `{content}` and bypasses the SSE response.
- [ ] Pre-stream BE errors (401, 429, 5xx) are surfaced as a single SSE `event: error` (or 401 JSON for unauth), then close.
- [ ] Mid-stream upstream connection errors emit `event: error` with `code: INTERNAL_ERROR` before closing.
- [ ] Client disconnection cancels the upstream request via context.
- [ ] The middleend does not transform SSE event names or payloads.
- [ ] `GET /actions/analysis/reset` returns `replace target_id="analysis-content"` with the start-state subtree.
- [ ] The `analysis_chat` component is registered in the FE under type `analysis_chat`. Required props per §4 are emitted.
- [ ] Component opens SSE on mount, captures `session_id` from the first event, appends `delta`s to the last assistant message, hides cursor on `done`, surfaces `error` per recoverable/terminal split.
- [ ] Send follow-up: user message pushed, assistant placeholder pushed, SSE opened to `followup_endpoint` with `{session_id}` substituted. Body `{content}`.
- [ ] Terminal mode: input disabled, CTA visible, CTA executes `reset_action`.
- [ ] Markdown render in assistant messages (remark-gfm, tables, lists, code blocks).
- [ ] Auto-scroll on new content; character counter near the input limit.
- [ ] All user-facing strings resolve via `analysis.*` keys in `en` and `es`.

---

## 9. Out of scope (v1)

- Persisting conversation history across navigations / page reloads.
- "Use live prices" toggle (the backend does not consume it).
- Explicit cleanup of the backend session when the user clicks "New analysis" (TTL on the BE handles it).
- Re-anchoring to an existing session via deep link (`?session_id=…`).
- Rehydrating from `GET /v1/analysis/sessions/:id`.
- Multiple concurrent sessions.
- Exporting the conversation.
- HideValues toggle (portfolio-only).
- Stop/cancel button while streaming.
- Inline file attachments in the chat.
