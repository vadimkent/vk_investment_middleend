# Analysis Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the Analysis screen in the vk-investment middleend: a streaming AI chat surface for portfolio analysis, backed by the backend's SSE-based `/v1/analysis/sessions` endpoints. Introduces one new SDUI custom component (`analysis_chat`) and the project's first SSE proxy.

**Architecture:** New `internal/analysis/` package, mirroring the shape of `internal/imports/`: a backend `Client` with two streaming methods (`StreamSession`, `AddMessage`) returning `*http.Response`, and one handler per HTTP route. Two SSE handlers (`stream`, `messages/:id`) bypass upstream chunks byte-for-byte to the client without buffering, with synthesized error events for pre-stream and mid-stream failures. The `analysis_chat` component lives in `internal/components/`. The middleend stores no analysis state — `session_id` lives on the FE in the component's local state, and follow-up requests carry it back via the URL path.

**Tech Stack:** Go 1.x · Gin · `net/http` with a tuned `Transport` (`ResponseHeaderTimeout` only — no `Client.Timeout` so streams can run indefinitely; cancellation governed by request context) · `httptest` for handler tests · `internal/components` SDUI helpers · `internal/i18n` JSON-flat translations · `locales/en.json` & `locales/es.json`.

**Reference design spec:** `docs/superpowers/specs/2026-04-30-analysis-screen-design.md`. Read it first — every task here implements a section of it.

**Pre-flight:** From a clean main (or worktree off main). Verify with `git status` (clean) and `make test` (green) before Task A1.

---

## Phase A — SDUI primitives

### Task A1: Doc — Add `analysis_chat` to `sdui-custom-components.md`

**Files:**
- Modify: `spec/sdui-custom-components.md` (insert before `## 5. Custom Attributes`)

- [ ] **Step 1: Locate insertion point**

Run: `grep -n "^## 5. Custom Attributes" spec/sdui-custom-components.md`
Expected: a single match line. The new section will be inserted immediately before it as `## 5. analysis_chat`, and current §5/§6 shift to §6/§7.

- [ ] **Step 2: Insert the new section**

Open `spec/sdui-custom-components.md`. Just before `## 5. Custom Attributes`, insert this content:

````markdown
## 5. `analysis_chat`

Self-contained streaming chat surface for the Analysis screen. Opens an SSE channel to a configured endpoint on mount, captures `session_id` from the first SSE event, appends `delta` events to the last assistant message, and accepts follow-up messages that open new SSE channels. Renders markdown in assistant messages and plain text in user messages.

### Why custom

The component combines several behaviors no base primitive offers:

- SSE attachment via `fetch`+`ReadableStream`, kept alive across local re-renders triggered by message-append updates.
- Incremental append: `delta` events extend the last assistant message in-place without an SDUI server round-trip per chunk.
- Streaming cursor (blinking) while a response is in flight.
- Auto-scroll on every new chunk.
- Local `session_id` state captured from the first SSE `session` event, used to fill the `{session_id}` placeholder in `followup_endpoint`.
- Error mode bifurcation (recoverable / terminal) with input gating, all client-side.

### Props

| Prop | Type | Required | Description |
|---|---|---|---|
| `initial_endpoint` | string | yes | URL the component opens an SSE channel to **on mount**. The first event must be `session` carrying `session_id`; subsequent events are `delta`, then `done` or `error`. |
| `followup_endpoint` | string | yes | URL template for follow-up messages. Must contain `{session_id}`, which the component substitutes at send time using the captured id. The follow-up request is a `POST` with body `{content}` and is handled as another SSE stream. |
| `placeholder` | string | yes | Text displayed in the input when empty. Localized. |
| `submit_label` | string | yes | Aria-label for the icon-only send button. Localized. |
| `streaming_label` | string | no | Small muted text rendered alongside the blinking cursor while a response is streaming. If absent, only the cursor renders. Localized. |
| `max_input_length` | int | no | Maximum characters allowed in the input. Default `2000`. |
| `error_messages` | `map<string, string>` | yes | Map of error code to localized message. Must include `default` as fallback. Codes the component cares about: `ANALYSIS_SESSION_NOT_FOUND`, `ANALYSIS_SESSION_EXPIRED`, `ANALYSIS_TOO_MANY_MESSAGES`, `ANALYSIS_FOCUS_TOO_LONG`, `AI_PROVIDER_UNAVAILABLE`, `AI_RATE_LIMITED`, `AI_TIMEOUT`, `AI_CONTEXT_TOO_LARGE`, `RATE_LIMITED`, `INTERNAL_ERROR`, `default`. |
| `terminal_error_codes` | `string[]` | yes | Codes that transition the component into terminal mode (input disabled + CTA visible). |
| `terminal_cta_label` | string | yes | Label for the CTA button in terminal mode. Localized. |
| `reset_action` | `Action` | yes | Action executed by the terminal CTA. Typically `Reload(/actions/analysis/reset, target_id="analysis-content")`. |

### SSE event protocol (passed through unchanged from the backend)

| Event | Payload | Component behavior |
|---|---|---|
| `session` | `{session_id: string}` | Stash `session_id`. Append a placeholder assistant message. Show streaming cursor. |
| `delta` | `{text: string}` | Append `text` to the last assistant message's `content`. Auto-scroll to bottom. |
| `done` | `{}` | Hide cursor on the last message. Re-enable input. |
| `error` | `{code: string, message: string}` | Render inline error banner using `error_messages[code] ?? error_messages["default"]`. If code ∈ `terminal_error_codes`: disable input, show CTA. Otherwise: keep input enabled. Remove the empty placeholder assistant message if it never received any `delta`. |

### Frontend behavior

1. **Mount**: open SSE to `initial_endpoint`. Initialize `messages: []`, `session_id: null`, `is_streaming: true`, `error: null`, `is_terminal: false`. The first `session` event captures `session_id`; the component pushes a placeholder assistant message.
2. **Streaming render**: messages list scrolls automatically as content grows. Each `delta` appends to the last assistant message and triggers scroll-to-bottom.
3. **`done`**: clear cursor; `is_streaming = false`.
4. **Send follow-up** (Enter without Shift, or Send button):
   - Validate: trimmed length > 0 and ≤ `max_input_length`.
   - Push `{role: "user", content}`; push `{role: "assistant", content: ""}`.
   - Open SSE to `followup_endpoint` with `{session_id}` resolved, body `{content}`.
   - Same delta/done/error loop.
5. **Error inline**: banner above the input, persists until next send (recoverable) or terminal CTA click.
6. **Terminal mode**: input disabled, send button disabled, banner persists, CTA button executes `reset_action`.
7. **Markdown**: assistant messages via remark-gfm (tables, lists, code, headings). User messages: plain text with `whitespace: pre-wrap`.
8. **Character counter**: bottom-right of input, only when value length crosses ~75% of `max_input_length`. Format `<current> / <max>`. Destructive color when over.
9. **Disconnection**: `fetch` aborts (network drop, navigation) → surface as `INTERNAL_ERROR` recoverable.
10. **Enter-to-send**: Enter (no Shift, no IME composition) invokes Send; Shift-Enter inserts newline.
11. **Unmount cleanup**: on unmount the component aborts any in-flight `fetch`+SSE before being torn down.

### Layout

- Outer: column flex, fills available height of the parent slot.
- Messages area: `flex: 1`, `overflow-y: auto`, centered max-width container; user bubbles right-aligned, assistant bubbles left-aligned with prose styling for markdown.
- Input area: pinned bottom, border-top separator, padding; centered max-width row containing `[textarea, send-button]`. Textarea auto-resizes between 1 and ~4 rows.
- Error banner between messages and input when `error` is set; terminal CTA below the banner when in terminal mode.

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

````

- [ ] **Step 3: Verify section ordering**

Run: `grep -n "^## " spec/sdui-custom-components.md`
Expected: sections numbered 1, 2, 3, 4, 5, 6, 7 in order: `line_chart`, `pie_chart`, `wizard`, `file_upload`, `analysis_chat`, `Custom Attributes`, `Custom Actions`. Renumber section 5 → 6 and section 6 → 7 if needed.

- [ ] **Step 4: Commit**

```bash
git add spec/sdui-custom-components.md
git commit -m "docs(spec): add analysis_chat custom component"
```

---

### Task A2: Implement `AnalysisChat` helper in `internal/components/`

**Files:**
- Create: `internal/components/analysis_chat.go`
- Create: `internal/components/analysis_chat_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/components/analysis_chat_test.go`:

```go
package components

import (
	"encoding/json"
	"testing"
)

func TestAnalysisChat_RequiredProps(t *testing.T) {
	c := AnalysisChat("analysis-chat", AnalysisChatProps{
		InitialEndpoint:    "/actions/analysis/stream?focus=x",
		FollowupEndpoint:   "/actions/analysis/sessions/{session_id}/messages",
		Placeholder:        "Ask a follow-up question…",
		SubmitLabel:        "Send",
		ErrorMessages:      map[string]string{"default": "Something went wrong."},
		TerminalErrorCodes: []string{"ANALYSIS_SESSION_EXPIRED"},
		TerminalCTALabel:   "Start a new analysis",
		ResetAction: Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: "/actions/analysis/reset",
			TargetID: "analysis-content",
			Loading:  "section",
		},
	})
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	for _, want := range []string{
		`"type":"analysis_chat"`,
		`"id":"analysis-chat"`,
		`"initial_endpoint":"/actions/analysis/stream?focus=x"`,
		`"followup_endpoint":"/actions/analysis/sessions/{session_id}/messages"`,
		`"placeholder":"Ask a follow-up question…"`,
		`"submit_label":"Send"`,
		`"terminal_cta_label":"Start a new analysis"`,
		`"terminal_error_codes":["ANALYSIS_SESSION_EXPIRED"]`,
		`"reset_action":{`,
	} {
		if !contains(got, want) {
			t.Errorf("expected %q in JSON, got: %s", want, got)
		}
	}
	for _, omitted := range []string{`"streaming_label"`, `"max_input_length"`} {
		if contains(got, omitted) {
			t.Errorf("expected %q to be omitted when zero, got: %s", omitted, got)
		}
	}
}

func TestAnalysisChat_OptionalProps(t *testing.T) {
	c := AnalysisChat("analysis-chat", AnalysisChatProps{
		InitialEndpoint:    "/x",
		FollowupEndpoint:   "/y",
		Placeholder:        "p",
		SubmitLabel:        "s",
		StreamingLabel:     "AI is thinking…",
		MaxInputLength:     2000,
		ErrorMessages:      map[string]string{"default": "x"},
		TerminalErrorCodes: []string{},
		TerminalCTALabel:   "t",
		ResetAction:        Action{Trigger: "click", Type: "reload", Endpoint: "/r", TargetID: "t"},
	})
	b, _ := json.Marshal(c)
	got := string(b)
	for _, want := range []string{
		`"streaming_label":"AI is thinking…"`,
		`"max_input_length":2000`,
	} {
		if !contains(got, want) {
			t.Errorf("expected %q, got: %s", want, got)
		}
	}
}
```

The `contains` helper already exists in `internal/components/actions_test.go`; tests in the same package share it.

- [ ] **Step 2: Run tests — expect failure**

Run: `go test ./internal/components/ -run TestAnalysisChat -v`
Expected: `AnalysisChat` and `AnalysisChatProps` undefined; build error.

- [ ] **Step 3: Implement helper**

Create `internal/components/analysis_chat.go`:

```go
package components

// AnalysisChatProps captures all configuration for an analysis_chat component.
// Required: InitialEndpoint, FollowupEndpoint, Placeholder, SubmitLabel,
// ErrorMessages, TerminalErrorCodes, TerminalCTALabel, ResetAction. Other
// fields are optional and omitted from the rendered Props map when zero.
type AnalysisChatProps struct {
	InitialEndpoint    string
	FollowupEndpoint   string
	Placeholder        string
	SubmitLabel        string
	StreamingLabel     string
	MaxInputLength     int
	ErrorMessages      map[string]string
	TerminalErrorCodes []string
	TerminalCTALabel   string
	ResetAction        Action
}

// AnalysisChat creates an analysis_chat custom component. See
// spec/sdui-custom-components.md §5 for the contract.
func AnalysisChat(id string, p AnalysisChatProps) Component {
	props := map[string]any{
		"initial_endpoint":     p.InitialEndpoint,
		"followup_endpoint":    p.FollowupEndpoint,
		"placeholder":          p.Placeholder,
		"submit_label":         p.SubmitLabel,
		"error_messages":       p.ErrorMessages,
		"terminal_error_codes": p.TerminalErrorCodes,
		"terminal_cta_label":   p.TerminalCTALabel,
		"reset_action":         p.ResetAction,
	}
	if p.StreamingLabel != "" {
		props["streaming_label"] = p.StreamingLabel
	}
	if p.MaxInputLength > 0 {
		props["max_input_length"] = p.MaxInputLength
	}
	return Component{
		Type:  "analysis_chat",
		ID:    id,
		Props: props,
	}
}
```

- [ ] **Step 4: Run tests — expect pass**

Run: `go test ./internal/components/ -v`
Expected: every test passes.

- [ ] **Step 5: Commit**

```bash
git add internal/components/analysis_chat.go internal/components/analysis_chat_test.go
git commit -m "feat(components): add analysis_chat custom component helper"
```

---

## Phase B — Canonical screen spec

### Task B1: Write `spec/screens/analysis.md` and update `spec/spec.md`

**Files:**
- Create: `spec/screens/analysis.md`
- Modify: `spec/spec.md` (the Screens table — change `Analysis | screens/analysis.md — TBD` to a real link)

- [ ] **Step 1: Create `spec/screens/analysis.md`**

The canonical spec mirrors the design doc but is terser (no rationale, no out-of-scope). Use `docs/superpowers/specs/2026-04-30-analysis-screen-design.md` as the authoring source. Write top-level sections in this order:

1. `# Analysis Screen` — one paragraph (design doc §1 condensed).
2. `## Endpoints` — exact table from §2 (the middleend endpoints) plus the "Backend dependencies" sub-section.
3. `## Layout` — the tree diagrams from §3 plus bullet notes.
4. `## Custom component` — short paragraph linking `../sdui-custom-components.md#5-analysis_chat`.
5. `## Data and business rules` — §4 SSE protocol table + frontend behavior summarized as the canonical contract.
6. `## SSE proxy` — §5 condensed (status code mapping table only — no full implementation prose).
7. `## i18n keys` — §7 list.
8. `## Error handling` — §6 table.
9. `## Acceptance criteria` — §8 list.

Tone: terse. Match the prose density of `spec/screens/snapshots.md` and `spec/screens/import.md`. Do not link out to `docs/superpowers/`.

- [ ] **Step 2: Update `spec/spec.md`**

Run: `grep -n "Analysis" spec/spec.md`
Expected: a row reading something like `| Analysis | \`screens/analysis.md\` — TBD |`. Replace it with:

```
| Analysis | [`screens/analysis.md`](screens/analysis.md) |
```

- [ ] **Step 3: Verify links**

Run: `grep -n "screens/analysis.md" spec/spec.md`
Expected: one match, no "TBD".

Run: `ls spec/screens/analysis.md`
Expected: file exists.

- [ ] **Step 4: Commit**

```bash
git add spec/spec.md spec/screens/analysis.md
git commit -m "docs(spec): add canonical Analysis screen spec"
```

---

## Phase C — Backend client (SSE)

The `analysis` package follows the shape of `imports`: a `Client` that talks to the backend over HTTP. Unlike `imports`, the `Client` returns live `*http.Response` for streaming endpoints — the caller closes the body and copies chunks downstream.

### Task C1: Skeleton — types, errors, Client with tuned Transport

**Files:**
- Create: `internal/analysis/client.go`
- Create: `internal/analysis/types.go`

- [ ] **Step 1: Create `internal/analysis/types.go`**

```go
package analysis

// BackendError carries a code + message that the BE returns on validation
// failures (4xx/5xx with a JSON body). The middleend forwards code and
// message to the FE so the analysis_chat component can map the code to a
// localized message client-side.
type BackendError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *BackendError) Error() string {
	return e.Code + ": " + e.Message
}
```

- [ ] **Step 2: Create `internal/analysis/client.go`**

```go
package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrUnauthorized    = errors.New("backend unauthorized")
	ErrBackend         = errors.New("backend error")
	ErrSessionNotFound = errors.New("analysis session not found")
)

// Client streams SSE from the backend's analysis endpoints. Tuned for SSE:
// ResponseHeaderTimeout caps the wait for the upstream to start streaming;
// Client.Timeout is left zero so the body can stream for as long as the
// backend wants. Cancellation is governed by the request context.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds a Client. The headerTimeout argument is a safety net for
// cases where the backend never responds; once headers arrive the body can
// stream indefinitely.
func NewClient(baseURL string, headerTimeout time.Duration) *Client {
	if headerTimeout <= 0 {
		headerTimeout = 30 * time.Second
	}
	transport := &http.Transport{
		ResponseHeaderTimeout: headerTimeout,
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Transport: transport},
	}
}

// do issues req and inspects the response status. On 200 it returns the live
// response (caller must close). On error statuses it consumes the body, maps
// to one of the package errors, and returns nil response.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return resp, nil
	case http.StatusUnauthorized:
		_ = resp.Body.Close()
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		_ = resp.Body.Close()
		return nil, ErrSessionNotFound
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if be := parseBackendError(resp.StatusCode, body); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// parseBackendError reads {error:{code,message}} from a response body.
// Returns nil when the body is empty or not in that shape.
func parseBackendError(httpStatus int, body []byte) *BackendError {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil
	}
	var wrapper struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil
	}
	if wrapper.Error.Code == "" && wrapper.Error.Message == "" {
		return nil
	}
	return &BackendError{HTTPStatus: httpStatus, Code: wrapper.Error.Code, Message: wrapper.Error.Message}
}

// Compile-time guard: keep helper imports referenced even if a method
// arrives in a later task.
var _ = strings.NewReader
var _ = url.PathEscape
var _ = json.Marshal
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/analysis/...`
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/analysis/types.go internal/analysis/client.go
git commit -m "feat(analysis): scaffold backend client and types"
```

---

### Task C2: `Client.StreamSession` — POST /v1/analysis/sessions

**Files:**
- Modify: `internal/analysis/client.go` (append method)
- Create: `internal/analysis/client_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/analysis/client_test.go`:

```go
package analysis

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(h)
	c := NewClient(srv.URL, 5*time.Second)
	return c, srv.Close
}

func TestStreamSession_PostsAndReturnsLiveResponse(t *testing.T) {
	var receivedAuth, receivedCT string
	var receivedBody []byte
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/analysis/sessions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		receivedAuth = r.Header.Get("Authorization")
		receivedCT = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: session\ndata: {\"session_id\":\"sess-1\"}\n\n"))
	})
	defer cleanup()

	resp, err := c.StreamSession(context.Background(), "Bearer t", "risk")
	if err != nil {
		t.Fatalf("StreamSession: %v", err)
	}
	defer resp.Body.Close()

	if receivedAuth != "Bearer t" {
		t.Fatalf("authorization not forwarded: %q", receivedAuth)
	}
	if receivedCT != "application/json" {
		t.Fatalf("content-type: %q", receivedCT)
	}
	if !strings.Contains(string(receivedBody), `"focus":"risk"`) {
		t.Fatalf("focus body: %q", receivedBody)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "session_id") {
		t.Fatalf("body bypass: %q", body)
	}
}

func TestStreamSession_EmptyFocusSendsEmptyJSON(t *testing.T) {
	var receivedBody []byte
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	defer cleanup()

	resp, err := c.StreamSession(context.Background(), "", "")
	if err != nil {
		t.Fatalf("StreamSession: %v", err)
	}
	resp.Body.Close()
	if string(receivedBody) != `{}` && string(receivedBody) != `{"focus":""}` {
		t.Fatalf("expected empty-focus body to be {} or {\"focus\":\"\"}, got: %q", receivedBody)
	}
}

func TestStreamSession_Unauthorized(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer cleanup()
	_, err := c.StreamSession(context.Background(), "", "")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestStreamSession_BackendError(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"ANALYSIS_FOCUS_TOO_LONG","message":"too long"}}`))
	})
	defer cleanup()
	_, err := c.StreamSession(context.Background(), "", "")
	var be *BackendError
	if !errors.As(err, &be) {
		t.Fatalf("expected *BackendError, got %v", err)
	}
	if be.Code != "ANALYSIS_FOCUS_TOO_LONG" || be.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("unexpected: %+v", be)
	}
}

func TestStreamSession_RateLimited(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"RATE_LIMITED","message":"slow down"}}`))
	})
	defer cleanup()
	_, err := c.StreamSession(context.Background(), "", "")
	var be *BackendError
	if !errors.As(err, &be) || be.HTTPStatus != http.StatusTooManyRequests {
		t.Fatalf("expected 429 BackendError, got %v", err)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/analysis/ -v`
Expected: undefined `StreamSession`.

- [ ] **Step 3: Implement**

Append to `internal/analysis/client.go`:

```go
// StreamSession opens POST /v1/analysis/sessions with body {focus} and returns
// the live SSE response. Caller must close resp.Body.
func (c *Client) StreamSession(ctx context.Context, authorization, focus string) (*http.Response, error) {
	body, _ := json.Marshal(map[string]string{"focus": focus})
	if focus == "" {
		body = []byte(`{}`)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/analysis/sessions", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	return c.do(req)
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/analysis/ -v`
Expected: all 5 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/analysis/client.go internal/analysis/client_test.go
git commit -m "feat(analysis): add Client.StreamSession (SSE POST to BE)"
```

---

### Task C3: `Client.AddMessage` — POST /v1/analysis/sessions/:id/messages

**Files:**
- Modify: `internal/analysis/client.go`
- Modify: `internal/analysis/client_test.go`

- [ ] **Step 1: Add failing tests**

Append to `client_test.go`:

```go
func TestAddMessage_PostsAndReturnsLiveResponse(t *testing.T) {
	var receivedBody []byte
	var receivedPath string
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: delta\ndata: {\"text\":\"hi\"}\n\n"))
	})
	defer cleanup()

	resp, err := c.AddMessage(context.Background(), "", "sess-1", "hello")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	defer resp.Body.Close()
	if receivedPath != "/v1/analysis/sessions/sess-1/messages" {
		t.Fatalf("path: %q", receivedPath)
	}
	if !strings.Contains(string(receivedBody), `"content":"hello"`) {
		t.Fatalf("body: %q", receivedBody)
	}
}

func TestAddMessage_SessionNotFound(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()
	_, err := c.AddMessage(context.Background(), "", "sess-x", "x")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/analysis/ -run TestAddMessage -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Append to `client.go`:

```go
// AddMessage opens POST /v1/analysis/sessions/:id/messages with body {content}
// and returns the live SSE response. Caller must close resp.Body.
func (c *Client) AddMessage(ctx context.Context, authorization, sessionID, content string) (*http.Response, error) {
	body, _ := json.Marshal(map[string]string{"content": content})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/analysis/sessions/"+url.PathEscape(sessionID)+"/messages",
		strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	return c.do(req)
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/analysis/ -v`
Expected: all client tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/analysis/client.go internal/analysis/client_test.go
git commit -m "feat(analysis): add Client.AddMessage"
```

---

## Phase D — Builders

### Task D1: Builder skeleton + start state

**Files:**
- Create: `internal/analysis/builder.go`
- Create: `internal/analysis/builder_test.go`

**Goal:** `BuildRoot(lang)` (full Screen for the initial render), `BuildRootColumn(lang)` (just the screen-root column for replace responses — same split pattern as imports), and `BuildStartState(lang, focusValue, errorMessage)` (the start form subtree).

- [ ] **Step 1: Write failing tests**

Create `internal/analysis/builder_test.go`:

```go
package analysis

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/project/vk-investment-middleend/internal/i18n"
)

func loadAnalysisLocales(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	en := `{
		"analysis": {
			"title": "Analysis",
			"start": {
				"description": "Get an AI-powered analysis of your portfolio — positions, allocation, risk, and opportunities.",
				"focus_label": "Focus area (optional)",
				"focus_placeholder": "e.g. risk exposure, crypto allocation, dividend potential",
				"submit": "Analyze my portfolio"
			},
			"new_analysis": "New analysis",
			"chat": {
				"placeholder": "Ask a follow-up question…",
				"submit_label": "Send",
				"streaming_label": "AI is thinking…",
				"terminal_cta": "Start a new analysis"
			},
			"error": {
				"session_not_found": "Session not found.",
				"session_expired": "Session expired. Start a new analysis.",
				"too_many_messages": "Conversation length limit reached. Start a new analysis.",
				"focus_too_long": "Focus area is too long.",
				"provider_unavailable": "AI provider unavailable. Please retry.",
				"rate_limited": "AI rate limit reached. Please retry shortly.",
				"timeout": "AI request timed out. Please retry.",
				"context_too_large": "Portfolio context is too large for the AI.",
				"internal": "Connection lost. Please try again.",
				"default": "Something went wrong. Please retry."
			},
			"feedback": { "start_failed": "Could not start analysis. Please retry." }
		}
	}`
	if err := os.WriteFile(dir+"/en.json", []byte(en), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := i18n.Load(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBuildRoot_Shape(t *testing.T) {
	loadAnalysisLocales(t)
	root := BuildRoot("en")
	b, _ := json.MarshalIndent(root, "", "  ")
	js := string(b)
	for _, want := range []string{
		`"id": "analysis-screen"`,
		`"id": "analysis-root"`,
		`"id": "analysis-header"`,
		`"id": "analysis-content"`,
		`"id": "analysis-start-card"`,
		`"id": "analysis-start-form"`,
		`"id": "analysis-focus"`,
		`"id": "analysis-start-submit"`,
		`"title": "Analysis"`,
		`Analyze my portfolio`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in root tree", want)
		}
	}
}

func TestBuildStartState_PrefillAndError(t *testing.T) {
	loadAnalysisLocales(t)
	c := BuildStartState("en", "risk exposure", "Focus area is too long.")
	b, _ := json.Marshal(c)
	js := string(b)
	if !strings.Contains(js, `risk exposure`) {
		t.Error("missing prefilled focus value")
	}
	if !strings.Contains(js, `Focus area is too long.`) {
		t.Error("missing error message")
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/analysis/ -run TestBuild -v`
Expected: undefined `BuildRoot`, `BuildStartState`.

- [ ] **Step 3: Implement**

Create `internal/analysis/builder.go`:

```go
package analysis

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildRoot is the top-level screen tree at GET /screens/analysis.
func BuildRoot(lang string) components.Component {
	return components.Screen("analysis-screen", i18n.T(lang, "analysis.title"), BuildRootColumn(lang))
}

// BuildRootColumn returns just the analysis-root column. Used when a replace
// targets analysis-root or analysis-content with a fresh start state.
func BuildRootColumn(lang string) components.Component {
	header := components.Component{
		Type: "row",
		ID:   "analysis-header",
		Props: map[string]any{
			"align_items": "center",
		},
		Children: []components.Component{
			components.Text("analysis-title", i18n.T(lang, "analysis.title"), "xl", "bold"),
		},
	}
	content := BuildContentStart(lang, "", "")
	return components.Component{
		Type:  "column",
		ID:    "analysis-root",
		Props: map[string]any{"gap": "lg"},
		Children: []components.Component{
			header,
			content,
		},
	}
}

// BuildContentStart returns the analysis-content column populated with the
// start state (the start form). focusValue and errorMessage are optional —
// when set, focusValue pre-fills the focus textarea and errorMessage shows
// as an inline banner above the form.
func BuildContentStart(lang, focusValue, errorMessage string) components.Component {
	return components.Component{
		Type: "column",
		ID:   "analysis-content",
		Props: map[string]any{
			"gap":           "lg",
			"align_items":   "center",
			"justify_items": "center",
		},
		Children: []components.Component{
			BuildStartState(lang, focusValue, errorMessage),
		},
	}
}

// BuildStartState wraps the start form in a centered card.
func BuildStartState(lang, focusValue, errorMessage string) components.Component {
	bodyChildren := make([]components.Component, 0, 5)

	bodyChildren = append(bodyChildren,
		components.TextStyled("analysis-start-description",
			i18n.T(lang, "analysis.start.description"),
			"sm", "normal", "block", "muted", "", ""),
	)

	if errorMessage != "" {
		bodyChildren = append(bodyChildren, components.Component{
			Type: "banner",
			ID:   "analysis-start-error",
			Props: map[string]any{
				"variant": "error",
				"message": errorMessage,
			},
		})
	}

	focus := components.TextareaFull(
		"analysis-focus", "focus",
		i18n.T(lang, "analysis.start.focus_label"),
		i18n.T(lang, "analysis.start.focus_placeholder"),
		focusValue, 2, 500,
	)
	bodyChildren = append(bodyChildren, focus)

	submitBtn := components.ButtonFull("analysis-start-submit",
		i18n.T(lang, "analysis.start.submit"),
		"", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/analysis/start",
			Method:   "POST",
			TargetID: "analysis-start-form",
			Loading:  "full",
		},
	)
	submitBtn.Props["size"] = "sm"

	actions := components.RowWithGap("analysis-start-actions",
		[]string{"1fr", "auto"}, "sm",
		components.Spacer("analysis-start-spacer", "none"),
		submitBtn,
	)
	bodyChildren = append(bodyChildren, actions)

	formBody := components.ColumnWithGap("analysis-start-body", "md", bodyChildren...)
	form := components.Form("analysis-start-form", formBody)
	card := components.Card("analysis-start-card", form)
	card.Props["max_width"] = "lg"
	return card
}
```

> **Note:** the existing `TextareaFull(id, name, label, placeholder, defaultValue string, rows, maxLength int)` signature is what's in the codebase today. If your local `internal/components/base.go` differs, adjust the call. Search with `grep -n "func TextareaFull" internal/components/base.go`.

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/analysis/ -v`
Expected: TestBuildRoot_Shape and TestBuildStartState_PrefillAndError pass.

- [ ] **Step 5: Commit**

```bash
git add internal/analysis/builder.go internal/analysis/builder_test.go
git commit -m "feat(analysis): add root/start-state builders"
```

---

### Task D2: Chat state subtree

**Files:**
- Modify: `internal/analysis/builder.go`
- Modify: `internal/analysis/builder_test.go`

**Goal:** `BuildContentChat(lang, focus)` — the analysis-content column populated with the chat state (reset row + analysis_chat).

- [ ] **Step 1: Add failing tests**

Append to `builder_test.go`:

```go
func TestBuildContentChat_Shape(t *testing.T) {
	loadAnalysisLocales(t)
	c := BuildContentChat("en", "risk exposure")
	b, _ := json.Marshal(c)
	js := string(b)

	for _, want := range []string{
		`"id":"analysis-content"`,
		`"id":"analysis-new-btn"`,
		`"id":"analysis-chat"`,
		`"type":"analysis_chat"`,
		`"initial_endpoint":"/actions/analysis/stream?focus=risk+exposure"`,
		`"followup_endpoint":"/actions/analysis/sessions/{session_id}/messages"`,
		`"endpoint":"/actions/analysis/reset"`,
		`AI is thinking…`,
		`Start a new analysis`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in chat state", want)
		}
	}
}

func TestBuildContentChat_EmptyFocusOmitsQueryParam(t *testing.T) {
	loadAnalysisLocales(t)
	c := BuildContentChat("en", "")
	b, _ := json.Marshal(c)
	js := string(b)
	if strings.Contains(js, `?focus=`) {
		t.Errorf("expected no ?focus= when empty, got: %s", js)
	}
	if !strings.Contains(js, `"initial_endpoint":"/actions/analysis/stream"`) {
		t.Errorf("expected initial_endpoint without query, got: %s", js)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/analysis/ -run TestBuildContentChat -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Append to `builder.go`:

```go
import (
	"net/url"
)
```

(Add `"net/url"` to the existing imports block at the top of the file.)

Append the following functions:

```go
// BuildContentChat returns the analysis-content column populated with the
// chat state — a small reset row (the "New analysis" button right-aligned)
// stacked above the analysis_chat component. focus is the original focus
// string from the start form; it's URL-encoded into initial_endpoint.
func BuildContentChat(lang, focus string) components.Component {
	resetBtn := components.ButtonFull("analysis-new-btn",
		i18n.T(lang, "analysis.new_analysis"),
		"", "secondary", "ghost",
		components.Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: "/actions/analysis/reset",
			TargetID: "analysis-content",
			Loading:  "section",
		},
	)
	resetBtn.Props["size"] = "sm"

	resetRow := components.RowWithGap("analysis-chat-header",
		[]string{"1fr", "auto"}, "sm",
		components.Spacer("analysis-chat-header-spacer", "none"),
		resetBtn,
	)

	chat := buildAnalysisChat(lang, focus)

	return components.Component{
		Type: "column",
		ID:   "analysis-content",
		Props: map[string]any{
			"gap": "md",
		},
		Children: []components.Component{
			resetRow,
			chat,
		},
	}
}

func buildAnalysisChat(lang, focus string) components.Component {
	initial := "/actions/analysis/stream"
	if focus != "" {
		initial += "?focus=" + url.QueryEscape(focus)
	}
	errorMessages := map[string]string{
		"ANALYSIS_SESSION_NOT_FOUND":  i18n.T(lang, "analysis.error.session_not_found"),
		"ANALYSIS_SESSION_EXPIRED":    i18n.T(lang, "analysis.error.session_expired"),
		"ANALYSIS_TOO_MANY_MESSAGES":  i18n.T(lang, "analysis.error.too_many_messages"),
		"ANALYSIS_FOCUS_TOO_LONG":     i18n.T(lang, "analysis.error.focus_too_long"),
		"AI_PROVIDER_UNAVAILABLE":     i18n.T(lang, "analysis.error.provider_unavailable"),
		"AI_RATE_LIMITED":             i18n.T(lang, "analysis.error.rate_limited"),
		"AI_TIMEOUT":                  i18n.T(lang, "analysis.error.timeout"),
		"AI_CONTEXT_TOO_LARGE":        i18n.T(lang, "analysis.error.context_too_large"),
		"RATE_LIMITED":                i18n.T(lang, "analysis.error.rate_limited"),
		"INTERNAL_ERROR":              i18n.T(lang, "analysis.error.internal"),
		"default":                     i18n.T(lang, "analysis.error.default"),
	}
	terminalCodes := []string{
		"ANALYSIS_SESSION_EXPIRED",
		"ANALYSIS_SESSION_NOT_FOUND",
		"ANALYSIS_TOO_MANY_MESSAGES",
	}
	resetAction := components.Action{
		Trigger:  "click",
		Type:     "reload",
		Endpoint: "/actions/analysis/reset",
		TargetID: "analysis-content",
		Loading:  "section",
	}
	return components.AnalysisChat("analysis-chat", components.AnalysisChatProps{
		InitialEndpoint:    initial,
		FollowupEndpoint:   "/actions/analysis/sessions/{session_id}/messages",
		Placeholder:        i18n.T(lang, "analysis.chat.placeholder"),
		SubmitLabel:        i18n.T(lang, "analysis.chat.submit_label"),
		StreamingLabel:     i18n.T(lang, "analysis.chat.streaming_label"),
		MaxInputLength:     2000,
		ErrorMessages:      errorMessages,
		TerminalErrorCodes: terminalCodes,
		TerminalCTALabel:   i18n.T(lang, "analysis.chat.terminal_cta"),
		ResetAction:        resetAction,
	})
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/analysis/ -v`
Expected: all tests pass, including TestBuildContentChat_*.

- [ ] **Step 5: Commit**

```bash
git add internal/analysis/builder.go internal/analysis/builder_test.go
git commit -m "feat(analysis): add chat-state builder"
```

---

## Phase E — Handlers

### Task E1: SSE proxy helper

**Files:**
- Create: `internal/analysis/sse.go`
- Create: `internal/analysis/sse_test.go`

**Goal:** Two helpers used by the streaming handlers in E5/E6:

- `proxySSE(c *gin.Context, upstream *http.Response)` — sets SSE headers, copies body chunks with flush, emits a synthetic `INTERNAL_ERROR` event on mid-stream upstream failure.
- `handleStreamError(c *gin.Context, err error)` — converts a pre-stream client error into either a 401 JSON envelope (auth) or a single SSE error event (everything else).

- [ ] **Step 1: Write failing tests**

Create `internal/analysis/sse_test.go`:

```go
package analysis

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newGinRecorder() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func TestHandleStreamError_Unauthorized(t *testing.T) {
	c, w := newGinRecorder()
	handleStreamError(c, ErrUnauthorized)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unauthorized") {
		t.Fatalf("expected unauthorized envelope, got: %s", w.Body.String())
	}
}

func TestHandleStreamError_BackendError429EmitsRateLimited(t *testing.T) {
	c, w := newGinRecorder()
	handleStreamError(c, &BackendError{HTTPStatus: http.StatusTooManyRequests, Code: "RATE_LIMITED", Message: "slow"})
	body := w.Body.String()
	if !strings.Contains(body, `event: error`) {
		t.Fatalf("expected SSE error event, got: %s", body)
	}
	if !strings.Contains(body, `"code":"RATE_LIMITED"`) {
		t.Fatalf("expected RATE_LIMITED code, got: %s", body)
	}
}

func TestHandleStreamError_BackendError5xxEmitsProviderUnavailable(t *testing.T) {
	c, w := newGinRecorder()
	handleStreamError(c, &BackendError{HTTPStatus: http.StatusBadGateway, Code: "", Message: "bad"})
	body := w.Body.String()
	if !strings.Contains(body, `"code":"AI_PROVIDER_UNAVAILABLE"`) {
		t.Fatalf("expected AI_PROVIDER_UNAVAILABLE, got: %s", body)
	}
}

func TestHandleStreamError_BackendError4xxPassesThroughCode(t *testing.T) {
	c, w := newGinRecorder()
	handleStreamError(c, &BackendError{HTTPStatus: http.StatusBadRequest, Code: "ANALYSIS_FOCUS_TOO_LONG", Message: "too long"})
	body := w.Body.String()
	if !strings.Contains(body, `"code":"ANALYSIS_FOCUS_TOO_LONG"`) {
		t.Fatalf("expected pass-through code, got: %s", body)
	}
}

func TestHandleStreamError_OtherErrorEmitsInternal(t *testing.T) {
	c, w := newGinRecorder()
	handleStreamError(c, errors.New("network ded"))
	body := w.Body.String()
	if !strings.Contains(body, `"code":"INTERNAL_ERROR"`) {
		t.Fatalf("expected INTERNAL_ERROR, got: %s", body)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/analysis/ -run "TestHandleStream|TestProxySSE" -v`
Expected: undefined `handleStreamError`, `proxySSE`.

- [ ] **Step 3: Implement**

Create `internal/analysis/sse.go`:

```go
package analysis

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// setSSEHeaders writes the standard SSE response headers on the gin writer.
// Must be called before any body bytes are written.
func setSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
}

// writeSSEErrorEvent serializes {code, message} as a single SSE error event.
// Caller is expected to have already called setSSEHeaders + Status(200).
func writeSSEErrorEvent(c *gin.Context, code, message string) {
	payload, _ := json.Marshal(map[string]string{"code": code, "message": message})
	fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", payload)
}

// handleStreamError converts a pre-stream client error into either a 401 JSON
// envelope (auth — same shape as the rest of the project) or a single SSE
// error event (everything else). Mapping:
//   - ErrUnauthorized → 401 {"error":"unauthorized","redirect":"/login"}
//   - BackendError 429 → SSE error with code "RATE_LIMITED"
//   - BackendError 5xx → SSE error with code "AI_PROVIDER_UNAVAILABLE"
//     (or pass-through if the BE returned its own code)
//   - BackendError other → SSE error with the BE's code (or "INTERNAL_ERROR")
//   - any other error  → SSE error with code "INTERNAL_ERROR"
func handleStreamError(c *gin.Context, err error) {
	if errors.Is(err, ErrUnauthorized) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
		return
	}

	setSSEHeaders(c)
	c.Status(http.StatusOK)

	var be *BackendError
	if errors.As(err, &be) {
		switch {
		case be.HTTPStatus == http.StatusTooManyRequests:
			writeSSEErrorEvent(c, ifEmptyCode(be.Code, "RATE_LIMITED"), be.Message)
		case be.HTTPStatus >= 500:
			writeSSEErrorEvent(c, ifEmptyCode(be.Code, "AI_PROVIDER_UNAVAILABLE"), be.Message)
		default:
			writeSSEErrorEvent(c, ifEmptyCode(be.Code, "INTERNAL_ERROR"), be.Message)
		}
	} else {
		writeSSEErrorEvent(c, "INTERNAL_ERROR", err.Error())
	}
	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}
}

// ifEmptyCode returns code unless empty, in which case fallback is returned.
func ifEmptyCode(code, fallback string) string {
	if code == "" {
		return fallback
	}
	return code
}

// proxySSE bypasses upstream's SSE response body to the gin context. Sets the
// SSE response headers, then streams body chunks with periodic flush. On
// mid-stream upstream error (network drop), emits a synthetic INTERNAL_ERROR
// event before closing.
func proxySSE(c *gin.Context, upstream *http.Response) {
	setSSEHeaders(c)
	c.Status(http.StatusOK)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		// Defensive: fall back to a plain copy without flush. Should not
		// happen with gin's writer.
		_, _ = io.Copy(c.Writer, upstream.Body)
		return
	}
	buf := make([]byte, 4096)
	for {
		n, err := upstream.Body.Read(buf)
		if n > 0 {
			if _, werr := c.Writer.Write(buf[:n]); werr != nil {
				return // client gone
			}
			flusher.Flush()
		}
		if err == io.EOF {
			return
		}
		if err != nil {
			writeSSEErrorEvent(c, "INTERNAL_ERROR", "connection lost")
			flusher.Flush()
			return
		}
	}
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/analysis/ -v`
Expected: every test passes.

- [ ] **Step 5: Commit**

```bash
git add internal/analysis/sse.go internal/analysis/sse_test.go
git commit -m "feat(analysis): add SSE proxy and pre-stream error helpers"
```

---

### Task E2: `GET /screens/analysis` handler

**Files:**
- Create: `internal/analysis/handler.go`
- Create: `internal/analysis/handler_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/analysis/handler_test.go`:

```go
package analysis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newRouter(setup func(*gin.Engine)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "u1")
		c.Set("authorization", "Bearer t")
		c.Next()
	})
	setup(r)
	return r
}

func TestScreenHandler_RendersStartState(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.GET("/screens/analysis", NewHandler().Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/screens/analysis", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["id"] != "analysis-screen" {
		t.Fatalf("expected id=analysis-screen, got %v", got["id"])
	}
	if !strings.Contains(rec.Body.String(), "analysis-start-form") {
		t.Fatal("missing start form in render")
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/analysis/ -run TestScreenHandler -v`
Expected: undefined `NewHandler`.

- [ ] **Step 3: Implement**

Create `internal/analysis/handler.go`:

```go
package analysis

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler renders the screen tree for GET /screens/analysis.
type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Get(c *gin.Context) {
	lang := resolveLang(c)
	c.JSON(http.StatusOK, BuildRoot(lang))
}

func resolveLang(c *gin.Context) string {
	if l := c.Query("lang"); l != "" {
		return l
	}
	if l := c.GetHeader("Accept-Language"); l != "" {
		if len(l) >= 2 {
			return l[:2]
		}
	}
	return "en"
}

func resolveAuth(c *gin.Context) string {
	if v, ok := c.Get("authorization"); ok {
		if s, ok2 := v.(string); ok2 && s != "" {
			return s
		}
	}
	return c.GetHeader("Authorization")
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/analysis/ -run TestScreenHandler -v`
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/analysis/handler.go internal/analysis/handler_test.go
git commit -m "feat(analysis): add GET /screens/analysis handler"
```

---

### Task E3: `POST /actions/analysis/start`

**Files:**
- Create: `internal/analysis/start_handler.go`
- Create: `internal/analysis/start_handler_test.go`

**Goal:** Validate `focus` ≤ 500 chars (server-side defense). Does not touch the backend. On success, return `replace target_id="analysis-content"` with the chat-state subtree. On `focus` too long, return `replace target_id="analysis-start-form"` with the form re-emitted + an inline error banner.

- [ ] **Step 1: Write failing tests**

Create `internal/analysis/start_handler_test.go`:

```go
package analysis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStartHandler_SuccessReplacesContentWithChat(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/analysis/start", NewStartHandler().Post)
	})

	form := url.Values{}
	form.Set("focus", "risk exposure")
	req := httptest.NewRequest(http.MethodPost, "/actions/analysis/start",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["action"] != "replace" || got["target_id"] != "analysis-content" {
		t.Fatalf("unexpected: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "analysis-chat") {
		t.Fatal("missing analysis_chat in tree")
	}
	if !strings.Contains(rec.Body.String(), `risk+exposure`) {
		t.Fatal("expected URL-encoded focus in initial_endpoint")
	}
}

func TestStartHandler_EmptyFocusOK(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/analysis/start", NewStartHandler().Post)
	})

	req := httptest.NewRequest(http.MethodPost, "/actions/analysis/start",
		strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"initial_endpoint":"/actions/analysis/stream"`) {
		t.Fatalf("expected stream endpoint without ?focus= when empty, got: %s", body)
	}
}

func TestStartHandler_FocusTooLongReplacesFormWithError(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/analysis/start", NewStartHandler().Post)
	})

	long := strings.Repeat("a", 501)
	form := url.Values{}
	form.Set("focus", long)
	req := httptest.NewRequest(http.MethodPost, "/actions/analysis/start",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"target_id":"analysis-start-form"`) {
		t.Fatalf("expected replace of analysis-start-form on validation error, got: %s", body)
	}
	if !strings.Contains(body, "Focus area is too long.") {
		t.Fatal("expected error message in tree")
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/analysis/ -run TestStartHandler -v`
Expected: undefined `NewStartHandler`.

- [ ] **Step 3: Implement**

Create `internal/analysis/start_handler.go`:

```go
package analysis

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

const maxFocusLen = 500

type StartHandler struct{}

func NewStartHandler() *StartHandler { return &StartHandler{} }

func (h *StartHandler) Post(c *gin.Context) {
	lang := resolveLang(c)
	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	focus := strings.TrimSpace(c.Request.PostForm.Get("focus"))

	if len([]rune(focus)) > maxFocusLen {
		// Re-emit the form with the (truncated for display) focus + inline error.
		formTree := BuildStartState(lang, focus, i18n.T(lang, "analysis.error.focus_too_long"))
		// The replace target is the form id, so we replace just the form, not
		// the whole content area. The tree we send is the card+form returned
		// by BuildStartState — but the FE expects to find the form as the
		// root of what it replaces. Strategy: send only the form subtree.
		// BuildStartState returns the card; we want only the form inside.
		formChild := extractStartForm(formTree)
		c.JSON(http.StatusOK, components.ReplaceResponse("analysis-start-form", formChild, nil))
		return
	}

	tree := BuildContentChat(lang, focus)
	c.JSON(http.StatusOK, components.ReplaceResponse("analysis-content", tree, nil))
}

// extractStartForm digs into the BuildStartState card to return the inner
// Form component (id="analysis-start-form"). Used to re-emit only the form
// when the replace target is the form id.
func extractStartForm(card components.Component) components.Component {
	if card.Type == "form" && card.ID == "analysis-start-form" {
		return card
	}
	for _, ch := range card.Children {
		if found := extractStartForm(ch); found.Type == "form" {
			return found
		}
	}
	return card // fallback: send whatever we have
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/analysis/ -run TestStartHandler -v`
Expected: all 3 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/analysis/start_handler.go internal/analysis/start_handler_test.go
git commit -m "feat(analysis): add POST /actions/analysis/start handler"
```

---

### Task E4: `GET /actions/analysis/reset`

**Files:**
- Create: `internal/analysis/reset_handler.go`
- Create: `internal/analysis/reset_handler_test.go`

**Goal:** "New analysis" button. Returns `replace target_id="analysis-content"` with a fresh start state.

- [ ] **Step 1: Write failing tests**

Create `internal/analysis/reset_handler_test.go`:

```go
package analysis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResetHandler_ReplacesContentWithStartState(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/analysis/reset", NewResetHandler().Get)
	})

	req := httptest.NewRequest(http.MethodGet, "/actions/analysis/reset", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["action"] != "replace" || got["target_id"] != "analysis-content" {
		t.Fatalf("unexpected: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "analysis-start-form") {
		t.Fatal("missing start form in reset response")
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/analysis/ -run TestResetHandler -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/analysis/reset_handler.go`:

```go
package analysis

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
)

type ResetHandler struct{}

func NewResetHandler() *ResetHandler { return &ResetHandler{} }

func (h *ResetHandler) Get(c *gin.Context) {
	lang := resolveLang(c)
	tree := BuildContentStart(lang, "", "")
	c.JSON(http.StatusOK, components.ReplaceResponse("analysis-content", tree, nil))
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/analysis/ -run TestResetHandler -v`
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/analysis/reset_handler.go internal/analysis/reset_handler_test.go
git commit -m "feat(analysis): add GET /actions/analysis/reset handler"
```

---

### Task E5: `GET /actions/analysis/stream` — SSE proxy for initial session

**Files:**
- Create: `internal/analysis/stream_handler.go`
- Create: `internal/analysis/stream_handler_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/analysis/stream_handler_test.go`:

```go
package analysis

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newAnalysisClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	return NewClient(baseURL, 5*time.Second)
}

func TestStreamHandler_BypassesUpstreamSSE(t *testing.T) {
	loadAnalysisLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl := w.(http.Flusher)
		_, _ = w.Write([]byte("event: session\ndata: {\"session_id\":\"sess-1\"}\n\n"))
		fl.Flush()
		_, _ = w.Write([]byte("event: delta\ndata: {\"text\":\"hi\"}\n\n"))
		fl.Flush()
		_, _ = w.Write([]byte("event: done\ndata: {}\n\n"))
		fl.Flush()
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/analysis/stream", NewStreamHandler(newAnalysisClient(t, be.URL)).Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/analysis/stream?focus=risk", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content-type: %q", got)
	}
	body := rec.Body.String()
	for _, want := range []string{"event: session", "event: delta", "event: done", "session_id"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in bypass body", want)
		}
	}
}

func TestStreamHandler_PreStreamRateLimitedEmitsErrorEvent(t *testing.T) {
	loadAnalysisLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"RATE_LIMITED","message":"slow down"}}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/analysis/stream", NewStreamHandler(newAnalysisClient(t, be.URL)).Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/analysis/stream", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `event: error`) || !strings.Contains(body, `"code":"RATE_LIMITED"`) {
		t.Fatalf("expected RATE_LIMITED SSE error, got: %s", body)
	}
}

func TestStreamHandler_Unauthorized401Envelope(t *testing.T) {
	loadAnalysisLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/analysis/stream", NewStreamHandler(newAnalysisClient(t, be.URL)).Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/analysis/stream", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unauthorized") {
		t.Fatal("expected unauthorized envelope")
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/analysis/ -run TestStreamHandler -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/analysis/stream_handler.go`:

```go
package analysis

import (
	"github.com/gin-gonic/gin"
)

type StreamHandler struct {
	client *Client
}

func NewStreamHandler(c *Client) *StreamHandler { return &StreamHandler{client: c} }

func (h *StreamHandler) Get(c *gin.Context) {
	focus := c.Query("focus")
	resp, err := h.client.StreamSession(c.Request.Context(), resolveAuth(c), focus)
	if err != nil {
		handleStreamError(c, err)
		return
	}
	defer resp.Body.Close()
	proxySSE(c, resp)
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/analysis/ -run TestStreamHandler -v`
Expected: all 3 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/analysis/stream_handler.go internal/analysis/stream_handler_test.go
git commit -m "feat(analysis): add GET /actions/analysis/stream SSE proxy"
```

---

### Task E6: `POST /actions/analysis/sessions/:id/messages` — SSE proxy for follow-ups

**Files:**
- Create: `internal/analysis/messages_handler.go`
- Create: `internal/analysis/messages_handler_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/analysis/messages_handler_test.go`:

```go
package analysis

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMessagesHandler_BypassesUpstreamSSE(t *testing.T) {
	loadAnalysisLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/analysis/sessions/sess-1/messages" {
			t.Fatalf("unexpected upstream path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl := w.(http.Flusher)
		_, _ = w.Write([]byte("event: delta\ndata: {\"text\":\"hello\"}\n\n"))
		fl.Flush()
		_, _ = w.Write([]byte("event: done\ndata: {}\n\n"))
		fl.Flush()
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/analysis/sessions/:id/messages", NewMessagesHandler(newAnalysisClient(t, be.URL)).Post)
	})

	req := httptest.NewRequest(http.MethodPost,
		"/actions/analysis/sessions/sess-1/messages",
		strings.NewReader(`{"content":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: delta") || !strings.Contains(body, "hello") {
		t.Fatalf("missing bypass content: %s", body)
	}
}

func TestMessagesHandler_BadRequestOnMissingContent(t *testing.T) {
	loadAnalysisLocales(t)
	r := newRouter(func(r *gin.Engine) {
		// Use a stub client: no upstream call expected since we error out early.
		r.POST("/actions/analysis/sessions/:id/messages", NewMessagesHandler(newAnalysisClient(t, "http://example.invalid")).Post)
	})
	req := httptest.NewRequest(http.MethodPost,
		"/actions/analysis/sessions/sess-1/messages",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
}

func TestMessagesHandler_SessionNotFoundEmitsErrorEvent(t *testing.T) {
	loadAnalysisLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/analysis/sessions/:id/messages", NewMessagesHandler(newAnalysisClient(t, be.URL)).Post)
	})

	req := httptest.NewRequest(http.MethodPost,
		"/actions/analysis/sessions/sess-x/messages",
		strings.NewReader(`{"content":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body := rec.Body.String()
	// 404 from BE → ErrSessionNotFound → handleStreamError synthesizes an
	// SSE error event; component will treat it as terminal via its
	// terminal_error_codes config.
	if !strings.Contains(body, `event: error`) {
		t.Fatalf("expected SSE error event for 404, got: %s", body)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/analysis/ -run TestMessagesHandler -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/analysis/messages_handler.go`:

```go
package analysis

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MessagesHandler struct {
	client *Client
}

func NewMessagesHandler(c *Client) *MessagesHandler { return &MessagesHandler{client: c} }

type messageRequest struct {
	Content string `json:"content"`
}

func (h *MessagesHandler) Post(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "missing session id"}})
		return
	}
	var req messageRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "missing required field: content"}})
		return
	}

	resp, err := h.client.AddMessage(c.Request.Context(), resolveAuth(c), id, req.Content)
	if err != nil {
		// Map ErrSessionNotFound through handleStreamError as a normal
		// pre-stream error: it'll synthesize an SSE error event with code
		// ANALYSIS_SESSION_NOT_FOUND-like content. To get the right code we
		// translate ErrSessionNotFound into a BackendError carrying the
		// standard code so downstream handlers don't need to know.
		if errors.Is(err, ErrSessionNotFound) {
			err = &BackendError{HTTPStatus: http.StatusNotFound, Code: "ANALYSIS_SESSION_NOT_FOUND", Message: "session not found"}
		}
		handleStreamError(c, err)
		return
	}
	defer resp.Body.Close()
	proxySSE(c, resp)
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/analysis/ -run TestMessagesHandler -v`
Expected: all 3 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/analysis/messages_handler.go internal/analysis/messages_handler_test.go
git commit -m "feat(analysis): add POST /actions/analysis/sessions/:id/messages SSE proxy"
```

---

## Phase F — Wiring + i18n + smoke

### Task F1: Add `analysis.*` i18n keys to `en.json` and `es.json`

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/es.json`

- [ ] **Step 1: Add the namespace to `locales/en.json`**

Find a suitable insertion point (e.g. just before `"common"`) and insert this block (keeping JSON valid — comma after the previous namespace):

```json
"analysis": {
  "title": "Analysis",
  "start": {
    "description": "Get an AI-powered analysis of your portfolio — positions, allocation, risk, and opportunities.",
    "focus_label": "Focus area (optional)",
    "focus_placeholder": "e.g. risk exposure, crypto allocation, dividend potential",
    "submit": "Analyze my portfolio"
  },
  "new_analysis": "New analysis",
  "chat": {
    "placeholder": "Ask a follow-up question…",
    "submit_label": "Send",
    "streaming_label": "AI is thinking…",
    "terminal_cta": "Start a new analysis"
  },
  "error": {
    "session_not_found": "Session not found.",
    "session_expired": "Session expired. Start a new analysis.",
    "too_many_messages": "Conversation length limit reached. Start a new analysis.",
    "focus_too_long": "Focus area is too long.",
    "provider_unavailable": "AI provider unavailable. Please retry.",
    "rate_limited": "Too many requests. Please wait a moment before trying again.",
    "timeout": "AI request timed out. Please retry.",
    "context_too_large": "Portfolio context is too large for the AI.",
    "internal": "Connection lost. Please try again.",
    "default": "Something went wrong. Please retry."
  },
  "feedback": {
    "start_failed": "Could not start analysis. Please retry."
  }
}
```

- [ ] **Step 2: Mirror in `locales/es.json`**

Same structure, translated:

```json
"analysis": {
  "title": "Análisis",
  "start": {
    "description": "Obtené un análisis IA de tu portafolio — posiciones, asignación, riesgo y oportunidades.",
    "focus_label": "Área de enfoque (opcional)",
    "focus_placeholder": "ej. exposición a riesgo, asignación cripto, potencial de dividendos",
    "submit": "Analizar mi portafolio"
  },
  "new_analysis": "Nuevo análisis",
  "chat": {
    "placeholder": "Hacé una pregunta de seguimiento…",
    "submit_label": "Enviar",
    "streaming_label": "La IA está pensando…",
    "terminal_cta": "Iniciar un nuevo análisis"
  },
  "error": {
    "session_not_found": "Sesión no encontrada.",
    "session_expired": "La sesión expiró. Iniciá un nuevo análisis.",
    "too_many_messages": "Llegaste al límite de mensajes. Iniciá un nuevo análisis.",
    "focus_too_long": "El área de enfoque es demasiado larga.",
    "provider_unavailable": "Proveedor de IA no disponible. Probá de nuevo.",
    "rate_limited": "Demasiadas solicitudes. Esperá un momento antes de reintentar.",
    "timeout": "Tiempo de espera agotado. Probá de nuevo.",
    "context_too_large": "El contexto del portafolio es demasiado grande para la IA.",
    "internal": "Conexión perdida. Probá de nuevo.",
    "default": "Algo salió mal. Probá de nuevo."
  },
  "feedback": {
    "start_failed": "No se pudo iniciar el análisis. Probá de nuevo."
  }
}
```

- [ ] **Step 3: Verify JSON validity**

Run: `python3 -c 'import json; json.load(open("locales/en.json")); json.load(open("locales/es.json")); print("ok")'`
Expected: `ok`.

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "feat(i18n): add analysis screen translations (en/es)"
```

---

### Task F2: Wire routes in `internal/server/server.go`

**Files:**
- Modify: `internal/server/server.go`

- [ ] **Step 1: Add import**

In the import block at the top of `server.go`, add (alphabetically placed):

```go
	"github.com/project/vk-investment-middleend/internal/analysis"
```

- [ ] **Step 2: Register routes**

In `setupRoutes`, after the imports block (the last `imports.New*Handler` route), append:

```go
	// --- analysis ---
	analysisClient := analysis.NewClient(s.cfg.BackendURL, 30*time.Second)
	protected.GET("/screens/analysis", analysis.NewHandler().Get)
	protected.POST("/actions/analysis/start", analysis.NewStartHandler().Post)
	protected.GET("/actions/analysis/reset", analysis.NewResetHandler().Get)
	protected.GET("/actions/analysis/stream", analysis.NewStreamHandler(analysisClient).Get)
	protected.POST("/actions/analysis/sessions/:id/messages", analysis.NewMessagesHandler(analysisClient).Post)
```

(Ensure `time` is imported in the file. It already is in the current `server.go`.)

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 4: Run full suite**

Run: `make test`
Expected: green.

- [ ] **Step 5: Commit**

```bash
git add internal/server/server.go
git commit -m "feat(server): wire analysis routes"
```

---

### Task F3: Smoke test — restart and verify

**Files:** none (manual verification + revert).

- [ ] **Step 1: Restart middleend**

```bash
lsof -ti :8082 | xargs -r kill 2>/dev/null
sleep 1
nohup ./cli run > /tmp/middleend.log 2>&1 &
sleep 3
grep "/screens/analysis\|/actions/analysis" /tmp/middleend.log | head -10
```

Expected: 5 routes registered:

- `GET /screens/analysis`
- `POST /actions/analysis/start`
- `GET /actions/analysis/reset`
- `GET /actions/analysis/stream`
- `POST /actions/analysis/sessions/:id/messages`

- [ ] **Step 2: No commit**

This task only validates wiring; no source files change. Do NOT curl the running app — per project convention, the user verifies through the frontend. The route registration in the log is sufficient confirmation that wiring is correct.

---

## Self-review

Run through this checklist with the spec open:

1. **Spec coverage** — every section of `docs/superpowers/specs/2026-04-30-analysis-screen-design.md` should map to at least one task:
   - §1 Overview → B1 (canonical spec).
   - §2 Endpoints → C1–C3 (clients), E2–E6 (handlers), F2 (wiring).
   - §3 Layout → D1, D2.
   - §4 `analysis_chat` component → A1 (doc), A2 (helper).
   - §5 SSE proxy → E1 (helpers), E5/E6 (handlers).
   - §6 Error handling → covered across E1 (helpers), E3 (focus too long), E5/E6 (pre-stream and mid-stream).
   - §7 i18n → F1.
   - §8 Acceptance → covered by per-handler tests + F3 smoke.

2. **Placeholder scan** — none of the forbidden patterns ("TBD", "TODO", "Similar to Task N", "add appropriate error handling", etc.) appear.

3. **Type consistency**:
   - `Client` returns `*http.Response` from `StreamSession` and `AddMessage` consistently (C2, C3, E5, E6).
   - `BackendError{HTTPStatus, Code, Message}` defined in C1 and consumed in E1, E5, E6.
   - `BuildRoot` / `BuildRootColumn` / `BuildContentStart` / `BuildStartState` / `BuildContentChat` defined in D1/D2 and consumed in E2 (handler), E3 (start), E4 (reset).
   - `AnalysisChatProps` / `AnalysisChat` defined in A2 and consumed by D2.
   - `proxySSE` / `handleStreamError` / `setSSEHeaders` / `writeSSEErrorEvent` / `ifEmptyCode` defined in E1 and consumed in E5/E6.
   - i18n keys used in builders (D1, D2) match exactly what F1 declares in en.json/es.json.

No issues found — plan is internally consistent.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-30-analysis-screen.md`. Two execution options:

**1. Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks, fast iteration. Good fit for ~16 small TDD tasks.

**2. Inline Execution** — execute tasks in this session using `superpowers:executing-plans`, batch with checkpoints.

Which approach?
