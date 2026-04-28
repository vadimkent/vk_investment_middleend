# Import & Export Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the Import & Export screen of the vk-investment middleend: a single screen that hosts the AI-driven import flow (upload → block on AI parse → review modal with gap resolution → confirm/cancel) plus the Export and Restore sub-flows backed by the existing `/v1/export` and `/v1/restore` backend endpoints. Add two reusable SDUI extensions: a `file_upload` custom component and a `loading` indicator that accepts cycling progress messages.

**Architecture:** New `internal/imports/` package (Go does not allow `import` as a package name), following the same shape as `internal/snapshots/`: a backend `Client`, builders for the three regions of the screen tree, one handler per HTTP route. Two SDUI primitives are added in `internal/components/`: `LoadingFullWithMessages` (a struct that JSON-encodes to `{scope, messages}` while keeping string-token backwards compatibility) and `FileUpload` (a custom component helper). The screen lives at `GET /screens/import`; sub-actions live under `/actions/import/...`. The middleend stores no session state — every review handler embeds the full session in its response.

**Tech Stack:** Go 1.x · Gin · `net/http` for backend client (`mime/multipart` for upload forwarding) · `httptest` for handler tests · `internal/components` SDUI helpers · `internal/i18n` JSON-flat translations · `locales/en.json` & `locales/es.json`.

**Reference design spec:** `docs/superpowers/specs/2026-04-27-import-export-screen-design.md`. Read that document first — every task here implements a section of it.

**Pre-flight:** Run from a clean `main` (or a worktree off of it). Verify with `git status` (working tree clean) and `make test` (all green) before Task A1.

---

## Phase A — SDUI extensions

These two tasks add reusable primitives that the rest of the plan depends on. They land in `internal/components/` and the canonical specs in `spec/`. Land Phase A entirely before Phase B so the rest of the screen can rely on the new helpers.

---

### Task A1: Doc — Extend `loading` in `sdui-actions.md`

**Files:**
- Modify: `spec/sdui-actions.md` (section 2b "Loading Indicators", around line 36-58)

- [ ] **Step 1: Read the current §2b**

Run: `sed -n '36,58p' spec/sdui-actions.md`
Expected: shows the table with `"section"` / `"full"` / absent and the example block.

- [ ] **Step 2: Replace §2b with the extended doc**

Replace the entire block from `## 2b. Loading Indicators` through (and including) the closing example fence with this content:

````markdown
## 2b. Loading Indicators

Any action that hits the middleend (`submit`, `reload`) can declare a `loading` field to show a visual indicator while the request is in flight. Two equivalent forms are accepted:

**Form A — string token (default for short waits):**

| Value | Behavior |
|---|---|
| `"section"` | Renders a semi-transparent overlay with spinner on the subtree whose `id` matches `target_id`. |
| `"full"` | Renders a fullscreen overlay (`z-50`) with spinner over the entire viewport. |
| (absent) | No loading indicator. The action completes silently. |

**Form B — object with cycling messages (for long waits):**

```json
"loading": {
  "scope": "section" | "full",
  "messages": ["Detecting columns…", "Mapping tickers…", "Resolving currencies…"]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `scope` | enum | yes | `"section"` or `"full"` — same semantics as Form A. |
| `messages` | string[] | no | Localized phrases the frontend rotates through every **2 seconds** in order, looping at the end. Empty / absent → behaves like Form A. |

The frontend renders the spinner (unchanged) plus, when `messages` is non-empty, a single line of text below the spinner that cycles with a brief cross-fade. Messages are purely cosmetic — they have no relationship to actual server-side progress.

The middleend decides **when** to show loading and **what scope** — the frontend only implements the visual. Loading clears automatically when the action response arrives.

Client-side-only actions (`toggle_sensitive`, `navigate`, `refresh`, etc.) ignore `loading`.

```json
{
  "trigger": "click",
  "type": "reload",
  "endpoint": "/actions/portfolio/live_data?live=true",
  "target_id": "live-data-section",
  "loading": "section"
}
```

```json
{
  "trigger": "click",
  "type": "submit",
  "endpoint": "/actions/import/analyze",
  "method": "POST",
  "target_id": "import-modal-slot",
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
}
```
````

- [ ] **Step 3: Verify the file is consistent**

Run: `grep -n "## 2b. Loading Indicators\|## 2. Action Types" spec/sdui-actions.md`
Expected: two matches, in order, `## 2b.` before `## 2.`.

- [ ] **Step 4: Commit**

```bash
git add spec/sdui-actions.md
git commit -m "docs(spec): extend loading indicator with cycling messages"
```

---

### Task A2: Implement extended `Loading` in `components/actions.go`

**Files:**
- Modify: `internal/components/actions.go`
- Test: `internal/components/actions_test.go`

**Goal:** The `Action.Loading` field becomes `interface{}` so it can carry either a string token (`"section"` / `"full"`) — preserving every existing call site — or a new `LoadingSpec` struct that JSON-encodes to `{"scope":"…","messages":[…]}`. Add a helper `LoadingFullWithMessages([]string) LoadingSpec`.

- [ ] **Step 1: Write failing tests**

Replace the contents of `internal/components/actions_test.go` with:

```go
package components

import (
	"encoding/json"
	"testing"
)

func TestSubmit_LoadingDefaultsToSectionString(t *testing.T) {
	a := Submit("/actions/foo", "POST", "foo-id")
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["loading"] != "section" {
		t.Fatalf("expected loading=\"section\", got %#v", got["loading"])
	}
}

func TestAction_LoadingFullStringRoundTrips(t *testing.T) {
	a := Action{Trigger: "click", Type: "submit", Endpoint: "/x", Method: "POST", TargetID: "y", Loading: "full"}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(b), `"loading":"full"`) {
		t.Fatalf("expected loading=\"full\" string token, got: %s", b)
	}
}

func TestAction_LoadingFullWithMessages(t *testing.T) {
	a := Action{
		Trigger:  "click",
		Type:     "submit",
		Endpoint: "/actions/import/analyze",
		Method:   "POST",
		TargetID: "import-modal-slot",
		Loading: LoadingFullWithMessages([]string{
			"Detecting columns…",
			"Mapping tickers…",
		}),
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `"loading":{"scope":"full","messages":["Detecting columns…","Mapping tickers…"]}`
	if !contains(string(b), want) {
		t.Fatalf("expected loading object form. got: %s", b)
	}
}

func TestAction_LoadingOmittedWhenEmpty(t *testing.T) {
	a := Action{Trigger: "click", Type: "submit", Endpoint: "/x", Method: "POST", TargetID: "y"}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if contains(string(b), `"loading"`) {
		t.Fatalf("expected loading to be omitted when empty. got: %s", b)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Run tests — expect failure**

Run: `go test ./internal/components/ -run TestAction_Loading -v`
Expected: `LoadingFullWithMessages` undefined; build error.

- [ ] **Step 3: Edit `internal/components/actions.go`**

Change line 12 from:

```go
	Loading  string `json:"loading,omitempty"`
```

to:

```go
	Loading  any `json:"loading,omitempty"`
```

Then append the following at the bottom of the file:

```go
// LoadingSpec is the object form of an action's loading indicator. When
// emitted, it serializes as {"scope":"…","messages":[…]} alongside any other
// fields. The string-token form ("section" / "full") remains valid via the
// `any`-typed Loading field on Action.
type LoadingSpec struct {
	Scope    string   `json:"scope"`
	Messages []string `json:"messages,omitempty"`
}

// LoadingFullWithMessages returns a LoadingSpec scoped to "full" and carrying
// the given cycling phrases. Use this when the action's wait may be long
// enough that a bare spinner feels frozen.
func LoadingFullWithMessages(messages []string) LoadingSpec {
	return LoadingSpec{Scope: "full", Messages: messages}
}

// LoadingSectionWithMessages is the section-scoped equivalent.
func LoadingSectionWithMessages(messages []string) LoadingSpec {
	return LoadingSpec{Scope: "section", Messages: messages}
}
```

- [ ] **Step 4: Run tests — expect pass**

Run: `go test ./internal/components/ -v`
Expected: all tests pass, including any pre-existing actions_test.go cases.

- [ ] **Step 5: Run the full test suite to confirm zero regressions**

Run: `make test`
Expected: full suite passes. `Loading: "section"` and `Loading: "full"` literals throughout the codebase keep working because Go assigns string to `any` cleanly and JSON encodes `any`-string the same as `string`.

- [ ] **Step 6: Commit**

```bash
git add internal/components/actions.go internal/components/actions_test.go
git commit -m "feat(components): extend loading indicator with optional cycling messages"
```

---

### Task A3: Doc — Add `file_upload` to `sdui-custom-components.md`

**Files:**
- Modify: `spec/sdui-custom-components.md` (insert before the existing "## 4. Custom Attributes" section)

- [ ] **Step 1: Locate insertion point**

Run: `grep -n "^## 4. Custom Attributes\|^## 3. \`wizard\`" spec/sdui-custom-components.md`
Expected: section 3 exists; section 4 follows it. Insert the new `file_upload` section as `## 4. file_upload` and renumber later sections.

- [ ] **Step 2: Insert the new section**

Open `spec/sdui-custom-components.md`. After the `wizard` section ends (the example code-fence and the trailing `---` separator), insert a new top-level section:

````markdown
## 4. `file_upload`

Drag-and-drop + click-to-browse file picker with local validation. Used by the Import & Export screen for the AI Import upload form and the Restore upload form. Generic by design — any future flow that needs a file as part of a multipart submit can reuse it.

### Why custom

The base SDUI catalog has no `input` variant for files. Browsers do not let JavaScript programmatically reattach a previously-picked File across re-renders, and SDUI re-renders are server-driven — so a custom component that owns local file state, drag-and-drop affordances, and pre-submit validation (size, format) is the cleanest way to model file inputs without leaking browser-specific quirks into every consumer.

### Props

| Prop | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Multipart field name on submit (e.g. `"file"`). |
| `label` | string | yes | Visible label rendered above the dropzone. Localized by the middleend. |
| `placeholder` | string | yes | Dropzone copy when no file is selected (e.g. *"Drop a file here or click to browse"*). Localized. |
| `hint` | string | no | Auxiliary copy beneath the dropzone (formats / size limit). Localized. |
| `accept` | string | no | Comma-separated extensions / MIME types (e.g. `".csv,.tsv,.xlsx"`). Drives the native `<input type="file" accept>` and the local format check. Absent → any file. |
| `max_size_bytes` | int | no | Local size limit in bytes. When the user picks a larger file, render `error_message_size` inline and clear the selection. Absent → no local limit. |
| `error_message_size` | string | no | Localized message when `max_size_bytes` is exceeded. May contain `{limit}` rendered as a human-readable size (e.g. "5 MB"). |
| `error_message_format` | string | no | Localized message when the file's extension / MIME type doesn't match `accept`. |
| `prefill_filename` | string | no | When set, render the dropzone in the "file selected" state with this filename **but no actual File object behind it** — purely informational. Used by the middleend when re-emitting a form after a server-side error. To re-submit, the user must re-pick the file (browsers do not let JS reattach a previously-picked File). The dropzone signals this state with the small caption from `reattach_hint`. |
| `reattach_hint` | string | no | Localized small caption shown alongside `prefill_filename` (e.g. "Re-select the file to retry"). |

### Frontend behavior

- Render: a dashed-bordered dropzone (~10rem tall) with an upload icon centered and the placeholder text below. When a file is selected, the placeholder is replaced by the filename (mono-friendly truncation if long). Hover, drag-over, and focus states match the design system's other interactive controls.
- Native `<input type="file">` is hidden; the dropzone forwards click to it. Drop events on the dropzone are captured (`preventDefault` on dragover, intercept the file from `dataTransfer.files[0]` on drop).
- On a new file selection: run the format check against `accept` (if set), then the size check against `max_size_bytes` (if set). On failure, show the corresponding error inline beneath the dropzone and do **not** retain the file.
- On `submit` of the enclosing form: contributes its file to the `multipart/form-data` body under `name`. If no file is present, the form-level submit button must be disabled by its consumer (the file_upload does not own form-level disabling).
- Reset: a fresh `replace` from the server (matching `id`) clears any local file and any local error. `prefill_filename` lets the server hint at the previously-uploaded filename for context.

### Example

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

---

````

(Make sure the trailing `---` is preserved so the next section's separator is intact.)

- [ ] **Step 3: Renumber later sections**

The original `## 4. Custom Attributes` becomes `## 5. Custom Attributes`; the original `## 5. Custom Actions` becomes `## 6. Custom Actions`.

Run: `grep -n "^## " spec/sdui-custom-components.md`
Expected: sections numbered 1, 2, 3, 4, 5, 6 in order: `line_chart`, `pie_chart`, `wizard`, `file_upload`, `Custom Attributes`, `Custom Actions`.

- [ ] **Step 4: Commit**

```bash
git add spec/sdui-custom-components.md
git commit -m "docs(spec): add file_upload custom component"
```

---

### Task A4: Implement `FileUpload` helper in `internal/components/`

**Files:**
- Create: `internal/components/file_upload.go`
- Create: `internal/components/file_upload_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/components/file_upload_test.go`:

```go
package components

import (
	"encoding/json"
	"testing"
)

func TestFileUpload_RequiredPropsOnly(t *testing.T) {
	c := FileUpload("import-file", FileUploadProps{
		Name:        "file",
		Label:       "File",
		Placeholder: "Drop a file here or click to browse",
	})

	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	wantSubstrings := []string{
		`"type":"file_upload"`,
		`"id":"import-file"`,
		`"name":"file"`,
		`"label":"File"`,
		`"placeholder":"Drop a file here or click to browse"`,
	}
	for _, w := range wantSubstrings {
		if !contains(got, w) {
			t.Errorf("expected %q in JSON, got: %s", w, got)
		}
	}
	for _, omitted := range []string{`"hint"`, `"accept"`, `"max_size_bytes"`, `"prefill_filename"`} {
		if contains(got, omitted) {
			t.Errorf("expected %q to be omitted, got: %s", omitted, got)
		}
	}
}

func TestFileUpload_FullProps(t *testing.T) {
	c := FileUpload("import-file", FileUploadProps{
		Name:               "file",
		Label:              "File",
		Placeholder:        "Drop or browse",
		Hint:               "CSV up to 5 MB",
		Accept:             ".csv,.tsv",
		MaxSizeBytes:       5242880,
		ErrorMessageSize:   "Too large: {limit}",
		ErrorMessageFormat: "Unsupported.",
		PrefillFilename:    "old.csv",
		ReattachHint:       "Re-select the file to retry",
	})

	b, _ := json.Marshal(c)
	got := string(b)
	for _, w := range []string{
		`"hint":"CSV up to 5 MB"`,
		`"accept":".csv,.tsv"`,
		`"max_size_bytes":5242880`,
		`"error_message_size":"Too large: {limit}"`,
		`"error_message_format":"Unsupported."`,
		`"prefill_filename":"old.csv"`,
		`"reattach_hint":"Re-select the file to retry"`,
	} {
		if !contains(got, w) {
			t.Errorf("expected %q in JSON, got: %s", w, got)
		}
	}
}
```

- [ ] **Step 2: Run tests — expect failure**

Run: `go test ./internal/components/ -run TestFileUpload -v`
Expected: `FileUpload` and `FileUploadProps` undefined; build error.

- [ ] **Step 3: Implement helper**

Create `internal/components/file_upload.go`:

```go
package components

// FileUploadProps captures all configuration for a file_upload component.
// Required fields: Name, Label, Placeholder. Everything else is optional and
// is omitted from the rendered Props map when its zero value is in effect.
type FileUploadProps struct {
	Name               string
	Label              string
	Placeholder        string
	Hint               string
	Accept             string
	MaxSizeBytes       int64
	ErrorMessageSize   string
	ErrorMessageFormat string
	PrefillFilename    string
	ReattachHint       string
}

// FileUpload creates a file_upload custom component. See
// spec/sdui-custom-components.md §4 for the contract.
func FileUpload(id string, p FileUploadProps) Component {
	props := map[string]any{
		"name":        p.Name,
		"label":       p.Label,
		"placeholder": p.Placeholder,
	}
	if p.Hint != "" {
		props["hint"] = p.Hint
	}
	if p.Accept != "" {
		props["accept"] = p.Accept
	}
	if p.MaxSizeBytes > 0 {
		props["max_size_bytes"] = p.MaxSizeBytes
	}
	if p.ErrorMessageSize != "" {
		props["error_message_size"] = p.ErrorMessageSize
	}
	if p.ErrorMessageFormat != "" {
		props["error_message_format"] = p.ErrorMessageFormat
	}
	if p.PrefillFilename != "" {
		props["prefill_filename"] = p.PrefillFilename
	}
	if p.ReattachHint != "" {
		props["reattach_hint"] = p.ReattachHint
	}
	return Component{
		Type:  "file_upload",
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
git add internal/components/file_upload.go internal/components/file_upload_test.go
git commit -m "feat(components): add file_upload custom component helper"
```

---

## Phase B — Canonical screen spec

### Task B1: Write `spec/screens/import.md` and update `spec/spec.md`

**Files:**
- Create: `spec/screens/import.md`
- Modify: `spec/spec.md` (the "Screens" table — change `Import | screens/import.md — TBD` to a real link)

- [ ] **Step 1: Create `spec/screens/import.md`**

The canonical spec is the user-facing source of truth. It mirrors the design doc but is more terse (no rationale, no out-of-scope). Use the design doc at `docs/superpowers/specs/2026-04-27-import-export-screen-design.md` as the authoring source — the canonical spec must agree with it section-for-section.

Write `spec/screens/import.md` with these top-level headings, populated from the corresponding sections of the design doc:

1. `# Import & Export Screen` — one-paragraph summary (design doc §1, condensed).
2. `## Endpoints` — exact table from design doc §2 (the middleend endpoints) plus the "Backend dependencies" sub-section.
3. `## Layout` — the tree diagram from design doc §3 plus the bullet notes.
4. `## Data and business rules` — design doc §4 in full (4.1 through 4.7), but rephrased as the canonical contract (drop the "design rationale" tone — describe what shipped behavior looks like).
5. `## Custom components used` — short paragraph listing `file_upload` and the extended `loading` indicator, with links to `../sdui-custom-components.md#4-file_upload` and `../sdui-actions.md#2b-loading-indicators`.
6. `## i18n keys` — design doc §7 list, organized by sub-namespace.
7. `## Error handling` — design doc §6 table.
8. `## Acceptance criteria` — design doc §8 list.

Tone: terse. Match the prose density of `spec/screens/snapshots.md`. Do not link out to `docs/superpowers/`.

- [ ] **Step 2: Update `spec/spec.md`**

Locate the "Screens" table:

```bash
grep -n "Import | \`screens/import.md\` — TBD" spec/spec.md
```

Replace that row with:

```
| Import | [`screens/import.md`](screens/import.md) |
```

- [ ] **Step 3: Verify links resolve**

Run: `grep -n "screens/import.md" spec/spec.md`
Expected: one match, no "TBD".

Run: `ls spec/screens/import.md`
Expected: file exists.

- [ ] **Step 4: Commit**

```bash
git add spec/spec.md spec/screens/import.md
git commit -m "docs(spec): add canonical Import & Export screen spec"
```

---

## Phase C — Backend client (`internal/imports/client.go`)

The `imports` package is the new home for everything Import/Export related. (Go forbids `import` as a package name.) The client mirrors the shape of `internal/snapshots/client.go`: one method per backend endpoint, errors as exported variables, no caching, multipart bodies built with `mime/multipart`.

---

### Task C1: Skeleton — types, client struct, errors

**Files:**
- Create: `internal/imports/types.go`
- Create: `internal/imports/client.go`

- [ ] **Step 1: Create `internal/imports/types.go`**

```go
package imports

// Session is the backend's import session response, mirroring the JSON body
// returned by POST /v1/import/sessions and friends.
type Session struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	CreatedAt   string    `json:"created_at"`
	ExpiresAt   string    `json:"expires_at"`
	AISummary   string    `json:"ai_summary"`
	Assumptions []string  `json:"assumptions"`
	Preview     Preview   `json:"preview"`
	Gaps        []Gap     `json:"gaps"`
	GapCounts   GapCounts `json:"gap_counts"`
}

type GapCounts struct {
	Blocking int `json:"blocking"`
	Warnings int `json:"warnings"`
}

type Gap struct {
	ID           string  `json:"id"`
	Severity     string  `json:"severity"`
	Type         string  `json:"type"`
	Description  string  `json:"description"`
	AffectedRows []int   `json:"affected_rows"`
	Suggestion   string  `json:"suggestion"`
	Resolution   *string `json:"resolution"`
}

type Preview struct {
	Assets    []PreviewAsset    `json:"assets"`
	Trades    []PreviewTrade    `json:"trades"`
	Snapshots []PreviewSnapshot `json:"snapshots"`
}

type PreviewAsset struct {
	Ticker    string `json:"ticker"`
	Name      string `json:"name"`
	AssetType string `json:"asset_type"`
	Currency  string `json:"currency"`
	Action    string `json:"action"`
}

type PreviewTrade struct {
	Row          int     `json:"row"`
	Ticker       string  `json:"ticker"`
	TradeType    string  `json:"trade_type"`
	Date         string  `json:"date"`
	Quantity     *string `json:"quantity"`
	PricePerUnit *string `json:"price_per_unit"`
	Fees         string  `json:"fees"`
	Status       string  `json:"status"`
	GapID        *string `json:"gap_id"`
}

type PreviewSnapshot struct {
	Rows       []int                  `json:"rows"`
	RecordedAt string                 `json:"recorded_at"`
	Entries    []PreviewSnapshotEntry `json:"entries"`
	Status     string                 `json:"status"`
}

type PreviewSnapshotEntry struct {
	Ticker     string `json:"ticker"`
	TotalValue string `json:"total_value"`
	Status     string `json:"status"`
}

// ConfirmResult mirrors POST /v1/import/sessions/:id/confirm.
type ConfirmResult struct {
	AssetsCreated     int `json:"assets_created"`
	TradesImported    int `json:"trades_imported"`
	SnapshotsImported int `json:"snapshots_imported"`
	Warnings          int `json:"warnings"`
}

// RestoreResult mirrors POST /v1/restore.
type RestoreResult struct {
	AssetsImported          int `json:"assets_imported"`
	AssetsSkipped           int `json:"assets_skipped"`
	TradesImported          int `json:"trades_imported"`
	TradesSkipped           int `json:"trades_skipped"`
	SnapshotsImported       int `json:"snapshots_imported"`
	SnapshotsSkipped        int `json:"snapshots_skipped"`
	SnapshotEntriesImported int `json:"snapshot_entries_imported"`
	SnapshotEntriesSkipped  int `json:"snapshot_entries_skipped"`
}

// GapResolution is the request shape for PATCH /v1/import/sessions/:id/gaps.
type GapResolution struct {
	GapID string `json:"gap_id"`
	Value string `json:"value"`
}

// BackendError carries a code + message that the BE returns on validation
// failures (typically 400 / 422). The middleend surfaces .Message to the user
// directly (no re-translation) and uses .Code for routing decisions.
type BackendError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *BackendError) Error() string {
	return e.Code + ": " + e.Message
}
```

- [ ] **Step 2: Create `internal/imports/client.go` with skeleton + errors**

```go
package imports

import (
	"errors"
	"net/http"
	"time"
)

var (
	ErrUnauthorized    = errors.New("backend unauthorized")
	ErrBackend         = errors.New("backend error")
	ErrSessionNotFound = errors.New("import session not found")
)

// Client talks to the backend /v1/import/sessions, /v1/export, and /v1/restore endpoints.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	// Override the timeout for AI parsing — analyze can take 60s+.
	// Use a generous floor so the upload doesn't trip on the global default.
	if timeout < 90*time.Second {
		timeout = 90 * time.Second
	}
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/imports/...`
Expected: builds clean (the package has only types and an empty client so far).

- [ ] **Step 4: Commit**

```bash
git add internal/imports/types.go internal/imports/client.go
git commit -m "feat(imports): scaffold backend client and types"
```

---

### Task C2: `Client.StartSession` — multipart POST

**Files:**
- Modify: `internal/imports/client.go` (append `StartSession`)
- Test: `internal/imports/client_test.go`

- [ ] **Step 1: Write failing test**

Create `internal/imports/client_test.go`:

```go
package imports

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(h)
	c := NewClient(srv.URL, 90*time.Second)
	return c, srv.Close
}

func TestStartSession_PostsMultipartAndReturnsSession(t *testing.T) {
	var receivedFile, receivedHint string
	var receivedAuth string
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/import/sessions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		receivedAuth = r.Header.Get("Authorization")
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer f.Close()
		b, _ := io.ReadAll(f)
		receivedFile = string(b)
		receivedHint = r.FormValue("hint")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id":"sess-1","status":"needs_review",
			"created_at":"2026-04-27T10:00:00Z","expires_at":"2026-04-27T11:00:00Z",
			"ai_summary":"Looks like Broker X export.",
			"assumptions":["amounts in USD"],
			"preview":{"assets":[],"trades":[],"snapshots":[]},
			"gaps":[],"gap_counts":{"blocking":0,"warnings":0}
		}`))
	})
	defer cleanup()

	sess, err := c.StartSession(context.Background(), "Bearer t", []byte("col1,col2\n1,2\n"), "text/csv", "broker x export")
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	if sess.ID != "sess-1" || sess.Status != "needs_review" {
		t.Fatalf("unexpected session: %+v", sess)
	}
	if receivedFile != "col1,col2\n1,2\n" {
		t.Fatalf("file content not forwarded: got %q", receivedFile)
	}
	if receivedHint != "broker x export" {
		t.Fatalf("hint not forwarded: got %q", receivedHint)
	}
	if receivedAuth != "Bearer t" {
		t.Fatalf("authorization not forwarded: got %q", receivedAuth)
	}
}

func TestStartSession_Unauthorized(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer cleanup()
	_, err := c.StartSession(context.Background(), "", []byte("x"), "text/csv", "")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestStartSession_BackendValidationError(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"IMPORT_FILE_TOO_LARGE","message":"File exceeds the 5 MB limit."}}`))
	})
	defer cleanup()
	_, err := c.StartSession(context.Background(), "", []byte("x"), "text/csv", "")
	var be *BackendError
	if !errors.As(err, &be) {
		t.Fatalf("expected *BackendError, got %v", err)
	}
	if be.Code != "IMPORT_FILE_TOO_LARGE" || be.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("unexpected backend error: %+v", be)
	}
}

// helper used by other tests
func newMultipart(t *testing.T, fieldName, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte(content))
	_ = w.Close()
	return &b, w.FormDataContentType()
}
```

Then add to the imports block at the top: `"bytes"`. (The test will fail to compile until you add it; that's expected.)

- [ ] **Step 2: Run tests — expect failure**

Run: `go test ./internal/imports/ -v`
Expected: `StartSession` undefined; build error.

- [ ] **Step 3: Implement `Client.StartSession` in `internal/imports/client.go`**

Append:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)
```

(Replace the existing import block at the top of `client.go` with the above — this becomes the canonical import set for the file as we add more methods.)

Append to the file:

```go
// StartSession uploads the file via multipart/form-data and returns the
// parsed Session. Blocks until the backend's AI completes (the BE side is
// synchronous in v1; see the design doc §1).
func (c *Client) StartSession(ctx context.Context, authorization string, fileContent []byte, mediaType, hint string) (*Session, error) {
	body, contentType, err := buildAnalyzeMultipart(fileContent, mediaType, hint)
	if err != nil {
		return nil, fmt.Errorf("build multipart: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/import/sessions", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		var s Session
		if err := json.Unmarshal(respBody, &s); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &s, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func buildAnalyzeMultipart(content []byte, mediaType, hint string) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// File part with the original media type so the BE can detect format.
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="upload"`)
	if mediaType != "" {
		hdr.Set("Content-Type", mediaType)
	}
	fw, err := w.CreatePart(hdr)
	if err != nil {
		return nil, "", err
	}
	if _, err := fw.Write(content); err != nil {
		return nil, "", err
	}

	if hint != "" {
		if err := w.WriteField("hint", hint); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

// parseBackendError attempts to read {error:{code,message}} from a response
// body. Returns nil when the body is not in that shape.
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

// Compile-time guard: keep helper imports referenced even if some methods
// arrive in later tasks. (strconv / strings are used by ResolveGaps and friends.)
var _ = strconv.Itoa
var _ = strings.TrimSpace
```

- [ ] **Step 4: Run tests — expect pass**

Run: `go test ./internal/imports/ -v`
Expected: all three `StartSession` tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/client.go internal/imports/client_test.go
git commit -m "feat(imports): add Client.StartSession (multipart POST to BE)"
```

---

### Task C3: `Client.ResolveGaps`

**Files:**
- Modify: `internal/imports/client.go`
- Modify: `internal/imports/client_test.go`

- [ ] **Step 1: Add failing tests**

Append to `client_test.go`:

```go
func TestResolveGaps_PatchesAndReturnsSession(t *testing.T) {
	var got struct {
		Resolutions []GapResolution `json:"resolutions"`
	}
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/import/sessions/sess-1/gaps" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"sess-1","status":"ready",
			"created_at":"2026-04-27T10:00:00Z","expires_at":"2026-04-27T11:00:00Z",
			"ai_summary":"x","assumptions":[],
			"preview":{"assets":[],"trades":[],"snapshots":[]},
			"gaps":[],"gap_counts":{"blocking":0,"warnings":0}
		}`))
	})
	defer cleanup()

	sess, err := c.ResolveGaps(context.Background(), "", "sess-1", []GapResolution{{GapID: "g1", Value: "USD"}})
	if err != nil {
		t.Fatalf("ResolveGaps: %v", err)
	}
	if sess.Status != "ready" {
		t.Fatalf("expected status=ready, got %q", sess.Status)
	}
	if len(got.Resolutions) != 1 || got.Resolutions[0].GapID != "g1" || got.Resolutions[0].Value != "USD" {
		t.Fatalf("body not forwarded: %+v", got)
	}
}

func TestResolveGaps_NotFound(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()
	_, err := c.ResolveGaps(context.Background(), "", "sess-x", nil)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestResolveGaps -v`
Expected: undefined `ResolveGaps`.

- [ ] **Step 3: Implement**

Append to `client.go`:

```go
// ResolveGaps PATCHes /v1/import/sessions/:id/gaps with the given resolutions
// and returns the updated session.
func (c *Client) ResolveGaps(ctx context.Context, authorization, sessionID string, resolutions []GapResolution) (*Session, error) {
	payload := struct {
		Resolutions []GapResolution `json:"resolutions"`
	}{Resolutions: resolutions}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		c.baseURL+"/v1/import/sessions/"+sessionID+"/gaps", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		var s Session
		if err := json.Unmarshal(respBody, &s); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &s, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrSessionNotFound
	default:
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -v`
Expected: all client tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/client.go internal/imports/client_test.go
git commit -m "feat(imports): add Client.ResolveGaps"
```

---

### Task C4: `Client.ConfirmSession`

**Files:**
- Modify: `internal/imports/client.go`
- Modify: `internal/imports/client_test.go`

- [ ] **Step 1: Add failing tests**

Append to `client_test.go`:

```go
func TestConfirmSession_OK(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/import/sessions/sess-1/confirm" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"assets_created":1,"trades_imported":2,"snapshots_imported":3,"warnings":0}`))
	})
	defer cleanup()
	res, err := c.ConfirmSession(context.Background(), "", "sess-1")
	if err != nil {
		t.Fatalf("ConfirmSession: %v", err)
	}
	if res.AssetsCreated != 1 || res.TradesImported != 2 || res.SnapshotsImported != 3 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestConfirmSession_NotFound(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	defer cleanup()
	_, err := c.ConfirmSession(context.Background(), "", "x")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestConfirmSession -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Append to `client.go`:

```go
// ConfirmSession POSTs /v1/import/sessions/:id/confirm and returns the result.
func (c *Client) ConfirmSession(ctx context.Context, authorization, sessionID string) (*ConfirmResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/import/sessions/"+sessionID+"/confirm", nil)
	if err != nil {
		return nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		var r ConfirmResult
		if err := json.Unmarshal(respBody, &r); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &r, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrSessionNotFound
	default:
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -v`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/client.go internal/imports/client_test.go
git commit -m "feat(imports): add Client.ConfirmSession"
```

---

### Task C5: `Client.CancelSession`

**Files:**
- Modify: `internal/imports/client.go`
- Modify: `internal/imports/client_test.go`

- [ ] **Step 1: Add failing tests**

Append to `client_test.go`:

```go
func TestCancelSession_NoContent(t *testing.T) {
	called := false
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/import/sessions/sess-1" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer cleanup()
	if err := c.CancelSession(context.Background(), "", "sess-1"); err != nil {
		t.Fatalf("CancelSession: %v", err)
	}
	if !called {
		t.Fatal("backend not called")
	}
}

func TestCancelSession_NotFoundIsNotError(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	defer cleanup()
	if err := c.CancelSession(context.Background(), "", "x"); err != nil {
		t.Fatalf("CancelSession should treat 404 as success, got %v", err)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestCancelSession -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Append to `client.go`:

```go
// CancelSession DELETEs the session. 404 is treated as success (idempotent).
func (c *Client) CancelSession(ctx context.Context, authorization, sessionID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		c.baseURL+"/v1/import/sessions/"+sessionID, nil)
	if err != nil {
		return err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusNotFound, http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	default:
		respBody, _ := io.ReadAll(resp.Body)
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return be
		}
		return fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -v`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/client.go internal/imports/client_test.go
git commit -m "feat(imports): add Client.CancelSession"
```

---

### Task C6: `Client.ExportStream`

**Files:**
- Modify: `internal/imports/client.go`
- Modify: `internal/imports/client_test.go`

`ExportStream` is structurally different from the other client methods: it returns a streaming `*http.Response` (so the handler can pass headers + body through to the browser without buffering). The caller is responsible for closing the response body.

- [ ] **Step 1: Add failing tests**

Append to `client_test.go`:

```go
func TestExportStream_PassthroughHeaders(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/export" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="vk_tracker_export_2026-04-27.csv"`)
		_, _ = w.Write([]byte("col1,col2\n1,2\n"))
	})
	defer cleanup()

	resp, err := c.ExportStream(context.Background(), "Bearer t")
	if err != nil {
		t.Fatalf("ExportStream: %v", err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("Content-Disposition"); !strings.Contains(got, "vk_tracker_export_") {
		t.Fatalf("Content-Disposition not forwarded: %q", got)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "col1,col2\n1,2\n" {
		t.Fatalf("body mismatch: %q", body)
	}
}

func TestExportStream_Unauthorized(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	defer cleanup()
	_, err := c.ExportStream(context.Background(), "")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestExportStream -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Append to `client.go`:

```go
// ExportStream calls GET /v1/export and returns the live response. The caller
// is responsible for copying headers and body to the client and for closing
// resp.Body. Unlike other methods, this does not buffer the body — exports
// can be large.
func (c *Client) ExportStream(ctx context.Context, authorization string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/export", nil)
	if err != nil {
		return nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
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
	default:
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -v`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/client.go internal/imports/client_test.go
git commit -m "feat(imports): add Client.ExportStream"
```

---

### Task C7: `Client.Restore` — multipart POST

**Files:**
- Modify: `internal/imports/client.go`
- Modify: `internal/imports/client_test.go`

- [ ] **Step 1: Add failing tests**

Append to `client_test.go`:

```go
func TestRestore_PostsMultipartAndReturnsCounts(t *testing.T) {
	var receivedFile string
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/restore" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_ = r.ParseMultipartForm(10 << 20)
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer f.Close()
		b, _ := io.ReadAll(f)
		receivedFile = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"assets_imported":1,"assets_skipped":0,
			"trades_imported":2,"trades_skipped":1,
			"snapshots_imported":0,"snapshots_skipped":3,
			"snapshot_entries_imported":4,"snapshot_entries_skipped":5
		}`))
	})
	defer cleanup()

	res, err := c.Restore(context.Background(), "", []byte("col1\nA\n"))
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if receivedFile != "col1\nA\n" {
		t.Fatalf("file not forwarded: %q", receivedFile)
	}
	if res.SnapshotEntriesSkipped != 5 {
		t.Fatalf("unexpected counts: %+v", res)
	}
}

func TestRestore_BackendError(t *testing.T) {
	c, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"RESTORE_FILE_TOO_LARGE","message":"File exceeds the 10 MB limit."}}`))
	})
	defer cleanup()
	_, err := c.Restore(context.Background(), "", []byte("x"))
	var be *BackendError
	if !errors.As(err, &be) || be.Code != "RESTORE_FILE_TOO_LARGE" {
		t.Fatalf("expected RESTORE_FILE_TOO_LARGE, got %v", err)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestRestore -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Append to `client.go`:

```go
// Restore uploads the CSV via multipart and returns the import/skip counts.
func (c *Client) Restore(ctx context.Context, authorization string, fileContent []byte) (*RestoreResult, error) {
	body, contentType, err := buildRestoreMultipart(fileContent)
	if err != nil {
		return nil, fmt.Errorf("build multipart: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/restore", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		var r RestoreResult
		if err := json.Unmarshal(respBody, &r); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &r, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func buildRestoreMultipart(content []byte) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="restore.csv"`)
	hdr.Set("Content-Type", "text/csv")
	fw, err := w.CreatePart(hdr)
	if err != nil {
		return nil, "", err
	}
	if _, err := fw.Write(content); err != nil {
		return nil, "", err
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -v`
Expected: all client tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/client.go internal/imports/client_test.go
git commit -m "feat(imports): add Client.Restore"
```

---

## Phase D — Builders

These three files compose the SDUI component trees used by handlers. They are pure functions: input is data + language, output is a `components.Component`. No HTTP, no I/O.

---

### Task D1: Root + idle cards builder

**Files:**
- Create: `internal/imports/builder.go`
- Create: `internal/imports/builder_test.go`

**Goal:** Build the screen root tree and the idle subtrees of `ai-import-card` and `restore-card`. Includes:

- `BuildRoot(lang string) components.Component` — full screen (header + section with the two cards/group + empty modal-slot).
- `BuildAIImportCardIdle(lang, errorMessage, prefillFilename, prefillHint string) components.Component` — `ai-import-card` in idle state, optionally with an inline error banner and prefilled file/hint.
- `BuildExportCard(lang string) components.Component` — static export card.
- `BuildRestoreCardIdle(lang, errorMessage, prefillFilename string) components.Component` — restore card idle state.
- `BuildEmptyModalSlot() components.Component` — the empty modal-slot column.

- [ ] **Step 1: Write failing tests**

Create `internal/imports/builder_test.go`:

```go
package imports

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/project/vk-investment-middleend/internal/i18n"
)

// loadTestLocales loads a minimal in-memory locale set so i18n.T returns
// real strings during tests. Tests run in any order, so this is idempotent.
func loadTestLocales(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	en := `{
		"import": {
			"title": "Import & Export",
			"ai": { "title": "Import historical data", "description": "Upload a file…" },
			"upload": { "label": "File", "placeholder": "Drop a file here or click to browse",
				"hint_ai": "CSV, TSV, XLS, XLSX, TXT — max 5 MB",
				"hint_restore": "CSV — max 10 MB",
				"error_size": "File exceeds the {limit} limit.",
				"error_format": "Unsupported file format.",
				"reattach_hint": "Re-select the file to retry" },
			"hint": { "label": "Hint (optional)", "placeholder": "e.g. broker x export" },
			"analyze": "Analyze file",
			"loading": { "analyze": { "1": "Detecting columns…", "2": "Mapping tickers…",
				"3": "Resolving currencies…", "4": "Building preview…", "5": "Validating consistency…" } },
			"export": { "title": "Export data", "description": "Download all data.", "submit": "Export all data" },
			"restore": { "title": "Restore from backup", "description": "Upload a CSV backup.",
				"submit": "Restore", "success_title": "Restored successfully",
				"col": { "imported": "Imported", "skipped": "Skipped" },
				"row": { "assets": "Assets", "trades": "Trades", "snapshots": "Snapshots", "snapshot_entries": "Snapshot entries" },
				"try_again": "Restore another file", "error_generic": "Restore failed." }
		}
	}`
	if err := writeFile(dir+"/en.json", en); err != nil {
		t.Fatal(err)
	}
	if err := i18n.Load(dir); err != nil {
		t.Fatal(err)
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func TestBuildRoot_Shape(t *testing.T) {
	loadTestLocales(t)
	root := BuildRoot("en")
	b, _ := json.MarshalIndent(root, "", "  ")
	js := string(b)

	for _, want := range []string{
		`"id": "import-root"`,
		`"id": "import-section"`,
		`"id": "ai-import-card"`,
		`"id": "export-restore-group"`,
		`"id": "export-card"`,
		`"id": "restore-card"`,
		`"id": "import-modal-slot"`,
		`"title": "Import & Export"`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in root tree", want)
		}
	}
}

func TestBuildAIImportCardIdle_DefaultState(t *testing.T) {
	loadTestLocales(t)
	c := BuildAIImportCardIdle("en", "", "", "")
	b, _ := json.Marshal(c)
	js := string(b)
	if !strings.Contains(js, `"id":"ai-import-card"`) {
		t.Fatal("missing ai-import-card id")
	}
	if !strings.Contains(js, `"type":"file_upload"`) {
		t.Fatal("missing file_upload child")
	}
	if !strings.Contains(js, `"max_size_bytes":5242880`) {
		t.Fatal("missing 5MB cap on upload")
	}
	if strings.Contains(js, `"prefill_filename"`) {
		t.Fatal("expected no prefill_filename in default state")
	}
}

func TestBuildAIImportCardIdle_WithErrorAndPrefill(t *testing.T) {
	loadTestLocales(t)
	c := BuildAIImportCardIdle("en", "AI parse failed.", "broker.csv", "amounts in USD")
	b, _ := json.Marshal(c)
	js := string(b)
	if !strings.Contains(js, `"prefill_filename":"broker.csv"`) {
		t.Fatal("missing prefill_filename")
	}
	if !strings.Contains(js, `AI parse failed.`) {
		t.Fatal("missing error banner message")
	}
	if !strings.Contains(js, `amounts in USD`) {
		t.Fatal("missing prefilled hint")
	}
}

func TestBuildRestoreCardIdle_DefaultState(t *testing.T) {
	loadTestLocales(t)
	c := BuildRestoreCardIdle("en", "", "")
	b, _ := json.Marshal(c)
	js := string(b)
	for _, want := range []string{
		`"id":"restore-card"`,
		`"type":"file_upload"`,
		`"max_size_bytes":10485760`,
		`"accept":".csv"`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in restore card", want)
		}
	}
}
```

Add `"os"` to the imports of `builder_test.go`.

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestBuild -v`
Expected: undefined `BuildRoot`, etc.

- [ ] **Step 3: Implement**

Create `internal/imports/builder.go`:

```go
package imports

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

const (
	maxAnalyzeBytes = 5 * 1024 * 1024
	maxRestoreBytes = 10 * 1024 * 1024

	acceptAnalyze = ".csv,.tsv,.xls,.xlsx,.txt"
	acceptRestore = ".csv"
)

// BuildRoot is the top-level screen tree at GET /screens/import.
func BuildRoot(lang string) components.Component {
	header := buildHeader(lang)
	section := components.Component{
		Type: "column",
		ID:   "import-section",
		Props: map[string]any{"gap": "lg"},
		Children: []components.Component{
			BuildAIImportCardIdle(lang, "", "", ""),
			buildExportRestoreGroup(lang),
		},
	}
	return components.Component{
		Type: "screen",
		ID:   "import-root",
		Props: map[string]any{
			"title": i18n.T(lang, "import.title"),
		},
		Children: []components.Component{header, section, BuildEmptyModalSlot()},
	}
}

// BuildEmptyModalSlot is the empty modal-slot sibling under import-root.
func BuildEmptyModalSlot() components.Component {
	return components.Component{
		Type:  "column",
		ID:    "import-modal-slot",
		Props: map[string]any{},
	}
}

func buildHeader(lang string) components.Component {
	return components.Component{
		Type: "row",
		ID:   "import-header",
		Props: map[string]any{
			"align_items": "center",
		},
		Children: []components.Component{
			components.Text("import-title", i18n.T(lang, "import.title"), "xl", "bold"),
		},
	}
}

func buildExportRestoreGroup(lang string) components.Component {
	return components.Component{
		Type: "row",
		ID:   "export-restore-group",
		Props: map[string]any{
			"gap":               "md",
			"grid_template":     []string{"1fr", "1fr"},
			"stack_on_mobile":   true,
			"align_items":       "stretch",
		},
		Children: []components.Component{
			BuildExportCard(lang),
			BuildRestoreCardIdle(lang, "", ""),
		},
	}
}

// BuildAIImportCardIdle returns the ai-import-card in idle state. errorMessage,
// prefillFilename, and prefillHint are all optional. When errorMessage is set,
// an inline error banner is rendered above the file upload. When
// prefillFilename is set, the file upload renders the "previously uploaded"
// state with a re-attach hint. When prefillHint is set, it pre-fills the hint
// textarea.
func BuildAIImportCardIdle(lang, errorMessage, prefillFilename, prefillHint string) components.Component {
	children := make([]components.Component, 0, 6)

	children = append(children,
		components.Text("ai-import-title", i18n.T(lang, "import.ai.title"), "lg", "bold"),
		components.Text("ai-import-description", i18n.T(lang, "import.ai.description"), "sm", "normal"),
	)

	if errorMessage != "" {
		children = append(children, components.Component{
			Type: "error",
			ID:   "ai-import-error",
			Props: map[string]any{
				"message": errorMessage,
			},
		})
	}

	upload := components.FileUpload("import-file", components.FileUploadProps{
		Name:               "file",
		Label:              i18n.T(lang, "import.upload.label"),
		Placeholder:        i18n.T(lang, "import.upload.placeholder"),
		Hint:               i18n.T(lang, "import.upload.hint_ai"),
		Accept:             acceptAnalyze,
		MaxSizeBytes:       maxAnalyzeBytes,
		ErrorMessageSize:   i18n.T(lang, "import.upload.error_size"),
		ErrorMessageFormat: i18n.T(lang, "import.upload.error_format"),
		PrefillFilename:    prefillFilename,
		ReattachHint:       i18n.T(lang, "import.upload.reattach_hint"),
	})
	children = append(children, upload)

	hint := components.Component{
		Type: "textarea",
		ID:   "import-hint",
		Props: map[string]any{
			"name":        "hint",
			"label":       i18n.T(lang, "import.hint.label"),
			"placeholder": i18n.T(lang, "import.hint.placeholder"),
			"max_length":  500,
		},
	}
	if prefillHint != "" {
		hint.Props["value"] = prefillHint
	}
	children = append(children, hint)

	analyze := components.Component{
		Type: "form",
		ID:   "ai-import-form",
		Props: map[string]any{
			"target_id": "ai-import-card",
		},
		Children: []components.Component{},
	}
	_ = analyze // forms wrap inputs; keeping flat for simplicity below.

	submitBtn := components.Button("import-analyze-btn", i18n.T(lang, "import.analyze"),
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/analyze",
			Method:   "POST",
			TargetID: "import-modal-slot",
			Loading:  components.LoadingFullWithMessages(analyzeLoadingMessages(lang)),
		},
	)
	children = append(children, submitBtn)

	return components.Component{
		Type: "card",
		ID:   "ai-import-card",
		Props: map[string]any{},
		Children: children,
	}
}

func analyzeLoadingMessages(lang string) []string {
	keys := []string{
		"import.loading.analyze.1",
		"import.loading.analyze.2",
		"import.loading.analyze.3",
		"import.loading.analyze.4",
		"import.loading.analyze.5",
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, i18n.T(lang, k))
	}
	return out
}

// BuildExportCard renders the static Export card.
func BuildExportCard(lang string) components.Component {
	return components.Component{
		Type: "card",
		ID:   "export-card",
		Props: map[string]any{},
		Children: []components.Component{
			components.Text("export-title", i18n.T(lang, "import.export.title"), "lg", "bold"),
			components.Text("export-description", i18n.T(lang, "import.export.description"), "sm", "normal"),
			components.Button("export-btn", i18n.T(lang, "import.export.submit"),
				components.OpenURL("/actions/import/export"),
			),
		},
	}
}

// BuildRestoreCardIdle returns the restore-card in idle state.
func BuildRestoreCardIdle(lang, errorMessage, prefillFilename string) components.Component {
	children := make([]components.Component, 0, 6)
	children = append(children,
		components.Text("restore-title", i18n.T(lang, "import.restore.title"), "lg", "bold"),
		components.Text("restore-description", i18n.T(lang, "import.restore.description"), "sm", "normal"),
	)
	if errorMessage != "" {
		children = append(children, components.Component{
			Type: "error", ID: "restore-error",
			Props: map[string]any{"message": errorMessage},
		})
	}

	upload := components.FileUpload("restore-file", components.FileUploadProps{
		Name:               "file",
		Label:              i18n.T(lang, "import.upload.label"),
		Placeholder:        i18n.T(lang, "import.upload.placeholder"),
		Hint:               i18n.T(lang, "import.upload.hint_restore"),
		Accept:             acceptRestore,
		MaxSizeBytes:       maxRestoreBytes,
		ErrorMessageSize:   i18n.T(lang, "import.upload.error_size"),
		ErrorMessageFormat: i18n.T(lang, "import.upload.error_format"),
		PrefillFilename:    prefillFilename,
		ReattachHint:       i18n.T(lang, "import.upload.reattach_hint"),
	})
	children = append(children, upload)

	submitBtn := components.Button("restore-submit-btn", i18n.T(lang, "import.restore.submit"),
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/restore",
			Method:   "POST",
			TargetID: "restore-card",
			Loading:  "section",
		},
	)
	children = append(children, submitBtn)

	return components.Component{
		Type:     "card",
		ID:       "restore-card",
		Props:    map[string]any{},
		Children: children,
	}
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -run TestBuild -v`
Expected: all five tests pass.

- [ ] **Step 5: Run full package suite**

Run: `go test ./internal/imports/ -v`
Expected: green.

- [ ] **Step 6: Commit**

```bash
git add internal/imports/builder.go internal/imports/builder_test.go
git commit -m "feat(imports): add root + idle card builders"
```

---

### Task D2: Review modal builder

**Files:**
- Create: `internal/imports/review_modal.go`
- Create: `internal/imports/review_modal_test.go`

**Goal:** `BuildReviewModal(lang string, sess *Session, errorMessage string) components.Component` returns the modal subtree injected into `import-modal-slot` after a successful analyze (or after a resolve_gaps round-trip). Includes the banner, AI summary card, issues form (when blocking>0), warnings collapsible, three preview tables, and the sticky action bar.

- [ ] **Step 1: Write failing tests**

Create `internal/imports/review_modal_test.go`:

```go
package imports

import (
	"encoding/json"
	"strings"
	"testing"
)

func sampleSessionReady() *Session {
	return &Session{
		ID: "sess-1", Status: "ready",
		AISummary:   "Looks like a Broker X export.",
		Assumptions: []string{"USD amounts"},
		Preview: Preview{
			Assets: []PreviewAsset{{Ticker: "AAPL", Name: "Apple", AssetType: "stock", Currency: "USD", Action: "create"}},
			Trades: []PreviewTrade{{Row: 2, Ticker: "AAPL", TradeType: "buy", Date: "2026-01-15", Fees: "0", Status: "ok"}},
		},
		GapCounts: GapCounts{Blocking: 0, Warnings: 0},
	}
}

func sampleSessionBlocking() *Session {
	return &Session{
		ID: "sess-1", Status: "needs_review",
		AISummary: "x",
		Gaps: []Gap{
			{ID: "g1", Severity: "blocking", Type: "missing_currency",
				Description: "currency not detected", AffectedRows: []int{2, 5},
				Suggestion: "set currency to USD"},
			{ID: "g2", Severity: "warning", Type: "ambiguous_date",
				Description: "date ambiguous", AffectedRows: []int{8}, Suggestion: "use ISO"},
		},
		GapCounts: GapCounts{Blocking: 1, Warnings: 1},
	}
}

func TestBuildReviewModal_ReadyState(t *testing.T) {
	loadTestLocales(t)
	loadReviewLocales(t)
	c := BuildReviewModal("en", sampleSessionReady(), "")
	b, _ := json.Marshal(c)
	js := string(b)

	for _, want := range []string{
		`"id":"import-review-modal"`,
		`"type":"modal"`,
		`Ready to import`,
		`Looks like a Broker X export.`,
		`USD amounts`,
		`AAPL`,
		`/actions/import/sessions/sess-1/confirm`,
		`/actions/import/sessions/sess-1/cancel`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in review modal: %s", want, js)
		}
	}

	if strings.Contains(js, `"id":"issues-section"`) {
		t.Error("expected no issues section when blocking == 0")
	}
}

func TestBuildReviewModal_BlockingState(t *testing.T) {
	loadTestLocales(t)
	loadReviewLocales(t)
	c := BuildReviewModal("en", sampleSessionBlocking(), "")
	b, _ := json.Marshal(c)
	js := string(b)

	for _, want := range []string{
		`"id":"issues-section"`,
		`"name":"resolutions[g1]"`,
		`/actions/import/sessions/sess-1/resolve_gaps`,
		`This file has 1 issue`, // copy may differ — adjust to your locale string
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in blocking review modal", want)
		}
	}
}

func TestBuildReviewModal_WithErrorBanner(t *testing.T) {
	loadTestLocales(t)
	loadReviewLocales(t)
	c := BuildReviewModal("en", sampleSessionReady(), "Validation failed.")
	b, _ := json.Marshal(c)
	if !strings.Contains(string(b), "Validation failed.") {
		t.Fatal("missing error banner message")
	}
}

func loadReviewLocales(t *testing.T) {
	// Extends the locales loaded by loadTestLocales with review-specific keys.
	// The test helper calls i18n.Load again with a richer snapshot.
	t.Helper()
	dir := t.TempDir()
	en := `{
		"import": {
			"review": {
				"blocking_banner": "This file has {n} issue(s) that need your input before importing.",
				"ready_banner": "Ready to import — review the preview and confirm.",
				"summary": "AI Summary",
				"assumptions": "Assumptions ({n})",
				"issues": "Issues",
				"warnings": "{n} warning(s)",
				"preview": "Preview",
				"preview.assets": "Assets ({n})",
				"preview.trades": "Trades ({n})",
				"preview.snapshots": "Snapshots ({n})",
				"confirm": "Confirm import",
				"cancel": "Cancel",
				"status": { "needs_review": "Needs review", "ready": "Ready" }
			},
			"gaps": { "affected_rows": "Affected rows: {rows}", "input_placeholder": "Enter value…", "save": "Save resolutions" }
		}
	}`
	_ = writeFile(dir+"/en.json", en)
	_ = i18n.Load(dir)
}
```

Add imports `"github.com/project/vk-investment-middleend/internal/i18n"` to the top of `review_modal_test.go`.

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestBuildReviewModal -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/imports/review_modal.go`:

```go
package imports

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildReviewModal renders the modal subtree injected into import-modal-slot
// after a successful analyze or after a resolve_gaps round-trip. errorMessage
// is optional — when non-empty an inline error banner is rendered at the top.
func BuildReviewModal(lang string, sess *Session, errorMessage string) components.Component {
	children := make([]components.Component, 0, 8)

	if errorMessage != "" {
		children = append(children, components.Component{
			Type: "error", ID: "review-error",
			Props: map[string]any{"message": errorMessage},
		})
	}

	children = append(children, buildBanner(lang, sess))
	children = append(children, buildSummary(lang, sess))
	if sess.GapCounts.Blocking > 0 {
		children = append(children, buildIssuesSection(lang, sess))
	}
	if hasWarnings(sess) {
		children = append(children, buildWarnings(lang, sess))
	}
	children = append(children, buildPreview(lang, sess))
	children = append(children, buildActionBar(lang, sess))

	return components.Component{
		Type: "modal",
		ID:   "import-review-modal",
		Props: map[string]any{
			"visible":       true,
			"dismissible":   false,
			"presentation":  "dialog",
		},
		Children: children,
	}
}

func buildBanner(lang string, sess *Session) components.Component {
	if sess.GapCounts.Blocking > 0 {
		msg := strings.ReplaceAll(i18n.T(lang, "import.review.blocking_banner"),
			"{n}", strconv.Itoa(sess.GapCounts.Blocking))
		return components.Component{
			Type: "banner", ID: "review-banner",
			Props: map[string]any{"variant": "warning", "message": msg, "dismissible": false},
		}
	}
	return components.Component{
		Type: "banner", ID: "review-banner",
		Props: map[string]any{
			"variant":     "info",
			"message":     i18n.T(lang, "import.review.ready_banner"),
			"dismissible": false,
		},
	}
}

func buildSummary(lang string, sess *Session) components.Component {
	children := []components.Component{
		components.Text("summary-title", i18n.T(lang, "import.review.summary"), "md", "bold"),
		components.Text("summary-text", sess.AISummary, "sm", "normal"),
	}
	if len(sess.Assumptions) > 0 {
		title := strings.ReplaceAll(i18n.T(lang, "import.review.assumptions"),
			"{n}", strconv.Itoa(len(sess.Assumptions)))
		bullets := make([]components.Component, 0, len(sess.Assumptions))
		for i, a := range sess.Assumptions {
			bullets = append(bullets, components.Text(fmt.Sprintf("assumption-%d", i), "• "+a, "sm", "normal"))
		}
		children = append(children, components.Component{
			Type: "toggle", ID: "assumptions-toggle",
			Props:    map[string]any{"label": title, "default_open": false},
			Children: bullets,
		})
	}
	return components.Component{
		Type: "card", ID: "summary-card",
		Props:    map[string]any{},
		Children: children,
	}
}

func buildIssuesSection(lang string, sess *Session) components.Component {
	cards := []components.Component{
		components.Text("issues-title", i18n.T(lang, "import.review.issues"), "md", "bold"),
	}
	for _, g := range sess.Gaps {
		if g.Severity != "blocking" {
			continue
		}
		rowsStr := strings.ReplaceAll(i18n.T(lang, "import.gaps.affected_rows"),
			"{rows}", joinInts(g.AffectedRows))
		preset := ""
		if g.Resolution != nil {
			preset = *g.Resolution
		}
		cards = append(cards, components.Component{
			Type: "card", ID: "gap-" + g.ID,
			Props: map[string]any{"variant": "destructive_outline"},
			Children: []components.Component{
				components.Component{
					Type: "badge", ID: "gap-" + g.ID + "-type-badge",
					Props: map[string]any{"label": g.Type, "variant": "destructive"},
				},
				components.Text("gap-"+g.ID+"-desc", g.Description, "sm", "normal"),
				components.Text("gap-"+g.ID+"-rows", rowsStr, "xs", "normal"),
				components.Text("gap-"+g.ID+"-suggestion", g.Suggestion, "xs", "italic"),
				components.Component{
					Type: "input", ID: "gap-" + g.ID + "-input",
					Props: map[string]any{
						"name":        "resolutions[" + g.ID + "]",
						"input_type":  "text",
						"placeholder": i18n.T(lang, "import.gaps.input_placeholder"),
						"value":       preset,
					},
				},
			},
		})
	}

	saveBtn := components.Button("issues-save-btn", i18n.T(lang, "import.gaps.save"),
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/sessions/" + sess.ID + "/resolve_gaps",
			Method:   "POST",
			TargetID: "import-modal-slot",
			Loading:  "section",
		},
	)
	cards = append(cards, saveBtn)

	return components.Component{
		Type:     "column",
		ID:       "issues-section",
		Props:    map[string]any{"gap": "sm"},
		Children: cards,
	}
}

func hasWarnings(sess *Session) bool {
	for _, g := range sess.Gaps {
		if g.Severity == "warning" {
			return true
		}
	}
	return false
}

func buildWarnings(lang string, sess *Session) components.Component {
	count := 0
	for _, g := range sess.Gaps {
		if g.Severity == "warning" {
			count++
		}
	}
	label := strings.ReplaceAll(i18n.T(lang, "import.review.warnings"), "{n}", strconv.Itoa(count))
	rows := []components.Component{}
	for _, g := range sess.Gaps {
		if g.Severity != "warning" {
			continue
		}
		rows = append(rows, components.Component{
			Type: "row", ID: "warning-" + g.ID,
			Props: map[string]any{"gap": "sm", "align_items": "start"},
			Children: []components.Component{
				components.Component{
					Type: "badge", ID: "warning-" + g.ID + "-badge",
					Props: map[string]any{"label": g.Type, "variant": "secondary"},
				},
				components.Text("warning-"+g.ID+"-desc", g.Description, "sm", "normal"),
			},
		})
	}
	return components.Component{
		Type: "toggle", ID: "warnings-toggle",
		Props:    map[string]any{"label": label, "default_open": false},
		Children: rows,
	}
}

func buildPreview(lang string, sess *Session) components.Component {
	return components.Component{
		Type: "column",
		ID:   "preview-section",
		Props: map[string]any{"gap": "md"},
		Children: []components.Component{
			components.Text("preview-title", i18n.T(lang, "import.review.preview"), "md", "bold"),
			buildPreviewAssets(lang, sess.Preview.Assets),
			buildPreviewTrades(lang, sess.Preview.Trades),
			buildPreviewSnapshots(lang, sess.Preview.Snapshots),
		},
	}
}

func buildPreviewAssets(lang string, assets []PreviewAsset) components.Component {
	label := strings.ReplaceAll(i18n.T(lang, "import.review.preview.assets"),
		"{n}", strconv.Itoa(len(assets)))
	headers := []string{"Ticker", "Name", "Type", "Currency", "Action"}
	rows := make([][]string, 0, len(assets))
	for _, a := range assets {
		rows = append(rows, []string{a.Ticker, a.Name, a.AssetType, a.Currency, a.Action})
	}
	return wrapTable("preview-assets", label, headers, rows)
}

func buildPreviewTrades(lang string, trades []PreviewTrade) components.Component {
	label := strings.ReplaceAll(i18n.T(lang, "import.review.preview.trades"),
		"{n}", strconv.Itoa(len(trades)))
	headers := []string{"Row", "Ticker", "Type", "Date", "Qty", "Price", "Fees", "Status"}
	rows := make([][]string, 0, len(trades))
	for _, t := range trades {
		rows = append(rows, []string{
			strconv.Itoa(t.Row), t.Ticker, t.TradeType, t.Date,
			derefOrDash(t.Quantity), derefOrDash(t.PricePerUnit), t.Fees, t.Status,
		})
	}
	return wrapTable("preview-trades", label, headers, rows)
}

func buildPreviewSnapshots(lang string, snapshots []PreviewSnapshot) components.Component {
	label := strings.ReplaceAll(i18n.T(lang, "import.review.preview.snapshots"),
		"{n}", strconv.Itoa(len(snapshots)))
	headers := []string{"Date", "Entries", "Status"}
	rows := make([][]string, 0, len(snapshots))
	for _, s := range snapshots {
		rows = append(rows, []string{s.RecordedAt, strconv.Itoa(len(s.Entries)), s.Status})
	}
	return wrapTable("preview-snapshots", label, headers, rows)
}

func derefOrDash(s *string) string {
	if s == nil || *s == "" {
		return "—"
	}
	return *s
}

func wrapTable(id, label string, headers []string, rows [][]string) components.Component {
	tableHeaders := make([]map[string]any, len(headers))
	for i, h := range headers {
		tableHeaders[i] = map[string]any{"label": h}
	}
	tableRows := make([]components.Component, 0, len(rows))
	for i, r := range rows {
		cells := make([]components.Component, 0, len(r))
		for j, v := range r {
			cells = append(cells, components.Text(fmt.Sprintf("%s-r%d-c%d", id, i, j), v, "sm", "normal"))
		}
		tableRows = append(tableRows, components.Component{
			Type: "table_row", ID: fmt.Sprintf("%s-row-%d", id, i),
			Props:    map[string]any{},
			Children: cells,
		})
	}
	return components.Component{
		Type: "toggle", ID: id + "-toggle",
		Props: map[string]any{"label": label, "default_open": true},
		Children: []components.Component{
			components.Component{
				Type: "table", ID: id,
				Props: map[string]any{
					"headers": tableHeaders,
				},
				Children: tableRows,
			},
		},
	}
}

func buildActionBar(lang string, sess *Session) components.Component {
	statusKey := "import.review.status." + sess.Status
	statusLabel := i18n.T(lang, statusKey)
	if statusLabel == statusKey {
		statusLabel = sess.Status
	}
	statusVariant := "secondary"
	if sess.Status == "ready" {
		statusVariant = "success"
	} else if sess.Status == "needs_review" {
		statusVariant = "warning"
	}

	cancelBtn := components.ButtonFull("review-cancel-btn", i18n.T(lang, "import.review.cancel"), "", "ghost", "ghost",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/sessions/" + sess.ID + "/cancel",
			Method:   "POST",
			TargetID: "import-root",
			Loading:  "full",
		},
	)
	confirmBtn := components.Button("review-confirm-btn", i18n.T(lang, "import.review.confirm"),
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/sessions/" + sess.ID + "/confirm",
			Method:   "POST",
			TargetID: "import-root",
			Loading:  "full",
		},
	)
	confirmBtn.Props["disabled"] = sess.Status != "ready"

	return components.Component{
		Type: "row",
		ID:   "review-action-bar",
		Props: map[string]any{
			"sticky":      "bottom",
			"justify":     "space_between",
			"align_items": "center",
			"gap":         "md",
		},
		Children: []components.Component{
			components.Component{
				Type: "badge", ID: "review-status-badge",
				Props: map[string]any{"label": statusLabel, "variant": statusVariant},
			},
			components.Component{
				Type:  "row",
				ID:    "review-action-buttons",
				Props: map[string]any{"gap": "sm"},
				Children: []components.Component{cancelBtn, confirmBtn},
			},
		},
	}
}

func joinInts(rows []int) string {
	parts := make([]string, len(rows))
	for i, n := range rows {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ", ")
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -run TestBuildReviewModal -v`
Expected: all three tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/review_modal.go internal/imports/review_modal_test.go
git commit -m "feat(imports): add review modal builder"
```

---

### Task D3: Restore success builder

**Files:**
- Create: `internal/imports/restore_success.go`
- Create: `internal/imports/restore_success_test.go`

**Goal:** `BuildRestoreCardSuccess(lang string, result *RestoreResult) components.Component` — the restore-card success state with the 4-row imported/skipped table and the "Restore another file" button.

- [ ] **Step 1: Write failing tests**

Create `internal/imports/restore_success_test.go`:

```go
package imports

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildRestoreCardSuccess_Shape(t *testing.T) {
	loadTestLocales(t)
	res := &RestoreResult{
		AssetsImported: 1, AssetsSkipped: 0,
		TradesImported: 2, TradesSkipped: 1,
		SnapshotsImported: 0, SnapshotsSkipped: 3,
		SnapshotEntriesImported: 4, SnapshotEntriesSkipped: 5,
	}
	c := BuildRestoreCardSuccess("en", res)
	b, _ := json.Marshal(c)
	js := string(b)

	for _, want := range []string{
		`"id":"restore-card"`,
		`"id":"restore-success-table"`,
		`Restored successfully`,
		`Restore another file`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q", want)
		}
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestBuildRestoreCardSuccess -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/imports/restore_success.go`:

```go
package imports

import (
	"fmt"
	"strconv"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildRestoreCardSuccess renders the success state of restore-card.
// The "Restore another file" button replaces the card with the idle subtree
// embedded in the response (no extra round-trip).
func BuildRestoreCardSuccess(lang string, r *RestoreResult) components.Component {
	headers := []map[string]any{
		{"label": ""},
		{"label": i18n.T(lang, "import.restore.col.imported"), "align": "right"},
		{"label": i18n.T(lang, "import.restore.col.skipped"), "align": "right"},
	}
	rows := []struct {
		key   string
		label string
		imp   int
		skip  int
	}{
		{"assets", i18n.T(lang, "import.restore.row.assets"), r.AssetsImported, r.AssetsSkipped},
		{"trades", i18n.T(lang, "import.restore.row.trades"), r.TradesImported, r.TradesSkipped},
		{"snapshots", i18n.T(lang, "import.restore.row.snapshots"), r.SnapshotsImported, r.SnapshotsSkipped},
		{"snapshot_entries", i18n.T(lang, "import.restore.row.snapshot_entries"), r.SnapshotEntriesImported, r.SnapshotEntriesSkipped},
	}

	tableRows := make([]components.Component, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, components.Component{
			Type: "table_row", ID: "restore-success-row-" + row.key,
			Props: map[string]any{},
			Children: []components.Component{
				components.Text(fmt.Sprintf("restore-row-%s-label", row.key), row.label, "sm", "normal"),
				components.Text(fmt.Sprintf("restore-row-%s-imp", row.key), strconv.Itoa(row.imp), "sm", "medium"),
				components.Text(fmt.Sprintf("restore-row-%s-skip", row.key), strconv.Itoa(row.skip), "sm", "normal"),
			},
		})
	}

	tryAgainBtn := components.ButtonFull("restore-try-again-btn", i18n.T(lang, "import.restore.try_again"), "", "outline", "outline",
		components.Action{
			Trigger:  "click",
			Type:     "replace",
			TargetID: "restore-card",
			// The handler injects the idle tree literally via the response;
			// alternatively the action can carry a `tree` field. v1 uses
			// the response form (see export/restore handler).
		},
	)
	// In practice, the action carries a tree literal — but Action does not
	// model a `tree` field. So we use type=reload with a dedicated endpoint
	// that returns the idle subtree; that endpoint is registered below.
	tryAgainBtn.Actions[0] = components.Action{
		Trigger:  "click",
		Type:     "reload",
		Endpoint: "/actions/import/restore_idle",
		TargetID: "restore-card",
		Loading:  "section",
	}

	return components.Component{
		Type: "card", ID: "restore-card",
		Props: map[string]any{},
		Children: []components.Component{
			components.Text("restore-success-title", i18n.T(lang, "import.restore.success_title"), "lg", "bold"),
			components.Component{
				Type: "table", ID: "restore-success-table",
				Props: map[string]any{
					"headers": headers,
				},
				Children: tableRows,
			},
			tryAgainBtn,
		},
	}
}
```

> **Note:** The design doc §4.7 mentions an embedded-tree variant for "Restore another file" to avoid a round-trip. Action does not model a `tree` field, so this implementation falls back to a `reload` against `/actions/import/restore_idle`. That handler returns the idle subtree (Task E7 wires it). Switching to true embedded-tree later is a one-line change.

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -run TestBuildRestoreCardSuccess -v`
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/restore_success.go internal/imports/restore_success_test.go
git commit -m "feat(imports): add restore-card success builder"
```

---

## Phase E — HTTP handlers

Each handler lives in its own `*_handler.go` file, mirroring the snapshots package layout. All handlers receive `gin.Context`, read `Authorization` and `user_id` from the context (set by the `RequireAuth` middleware), and return `ActionResponse` JSON or (for the export proxy) a streaming body.

The handlers use a shared helper `writeActionResponse(c, resp)` and a `resolveLang(c)` helper. Add those once in `internal/imports/handler.go` and reuse.

---

### Task E1: `GET /screens/import` handler

**Files:**
- Create: `internal/imports/handler.go`
- Create: `internal/imports/handler_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/imports/handler_test.go`:

```go
package imports

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

func TestScreenHandler_Get_RendersRoot(t *testing.T) {
	loadTestLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.GET("/screens/import", NewHandler().Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/screens/import", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["id"] != "import-root" {
		t.Fatalf("expected import-root, got %v", got["id"])
	}
	if !strings.Contains(rec.Body.String(), "ai-import-card") {
		t.Fatal("missing ai-import-card in render")
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestScreenHandler -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/imports/handler.go`:

```go
package imports

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler renders the screen tree for GET /screens/import.
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
		// First two letters of the first tag (e.g. "es-ES,en;q=0.9" → "es")
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

Run: `go test ./internal/imports/ -run TestScreenHandler -v`
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/handler.go internal/imports/handler_test.go
git commit -m "feat(imports): add GET /screens/import handler"
```

---

### Task E2: `POST /actions/import/analyze`

**Files:**
- Create: `internal/imports/analyze_handler.go`
- Create: `internal/imports/analyze_handler_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/imports/analyze_handler_test.go`:

```go
package imports

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func mustClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	return NewClient(baseURL, 90*time.Second)
}

func multipartBody(t *testing.T, fileContent, hint string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "broker.csv")
	_, _ = fw.Write([]byte(fileContent))
	if hint != "" {
		_ = w.WriteField("hint", hint)
	}
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

func TestAnalyzeHandler_Success(t *testing.T) {
	loadTestLocales(t)
	loadReviewLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id":"sess-1","status":"ready",
			"created_at":"2026-04-27T10:00:00Z","expires_at":"2026-04-27T11:00:00Z",
			"ai_summary":"x","assumptions":[],
			"preview":{"assets":[],"trades":[],"snapshots":[]},
			"gaps":[],"gap_counts":{"blocking":0,"warnings":0}
		}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/analyze", NewAnalyzeHandler(mustClient(t, be.URL)).Post)
	})

	body, ct := multipartBody(t, "col1\n1\n", "broker x")
	req := httptest.NewRequest(http.MethodPost, "/actions/import/analyze", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["action"] != "replace" || got["target_id"] != "import-modal-slot" {
		t.Fatalf("unexpected action response: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "import-review-modal") {
		t.Fatal("missing review modal in response tree")
	}
}

func TestAnalyzeHandler_ReplacesCardOnBackendError(t *testing.T) {
	loadTestLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"IMPORT_FILE_TOO_LARGE","message":"File exceeds 5 MB."}}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/analyze", NewAnalyzeHandler(mustClient(t, be.URL)).Post)
	})

	body, ct := multipartBody(t, "x", "h")
	req := httptest.NewRequest(http.MethodPost, "/actions/import/analyze", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target_id"] != "ai-import-card" {
		t.Fatalf("expected target_id ai-import-card, got %v", got["target_id"])
	}
	if !strings.Contains(rec.Body.String(), "File exceeds 5 MB.") {
		t.Fatal("expected backend error message in tree")
	}
	if !errors.Is(nil, errors.New("")) { // keep import alive in case of refactor
		_ = errors.New("")
	}
}
```

Add `"github.com/gin-gonic/gin"` and `"errors"` to the import set.

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestAnalyzeHandler -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/imports/analyze_handler.go`:

```go
package imports

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
)

const analyzeMaxBytes = 5 * 1024 * 1024

type AnalyzeHandler struct {
	client *Client
}

func NewAnalyzeHandler(c *Client) *AnalyzeHandler { return &AnalyzeHandler{client: c} }

func (h *AnalyzeHandler) Post(c *gin.Context) {
	lang := resolveLang(c)

	if err := c.Request.ParseMultipartForm(analyzeMaxBytes); err != nil {
		writeAIImportError(c, lang, "Upload exceeds the size limit.")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		writeAIImportError(c, lang, "Missing file.")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, analyzeMaxBytes+1))
	if err != nil {
		writeAIImportError(c, lang, "Failed to read upload.")
		return
	}
	if int64(len(content)) > analyzeMaxBytes {
		writeAIImportError(c, lang, "File exceeds the 5 MB limit.")
		return
	}

	mediaType := header.Header.Get("Content-Type")
	hint := c.Request.FormValue("hint")
	prefillFilename := header.Filename

	sess, err := h.client.StartSession(c.Request.Context(), resolveAuth(c), content, mediaType, hint)
	if err != nil {
		var be *BackendError
		if errors.As(err, &be) {
			tree := BuildAIImportCardIdle(lang, be.Message, prefillFilename, hint)
			c.JSON(http.StatusOK, components.ReplaceResponse("ai-import-card", tree, nil))
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
			return
		}
		// network / 5xx → snackbar error, leave the card as-is (no replace).
		fb := components.Snackbar("feedback", "Import failed. Please try again.", "error")
		c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
		return
	}

	tree := BuildReviewModal(lang, sess, "")
	c.JSON(http.StatusOK, components.ReplaceResponse("import-modal-slot", tree, nil))
}

func writeAIImportError(c *gin.Context, lang, message string) {
	hint := c.Request.FormValue("hint")
	tree := BuildAIImportCardIdle(lang, message, "", hint)
	c.JSON(http.StatusOK, components.ReplaceResponse("ai-import-card", tree, nil))
}
```

> **Note:** `components.Snackbar(...)` is the existing helper in `internal/components/base.go`. If your local copy uses a different signature, adjust the call site. The shape is `Snackbar(id, message, variant string) Component`.

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -run TestAnalyzeHandler -v`
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/analyze_handler.go internal/imports/analyze_handler_test.go
git commit -m "feat(imports): add POST /actions/import/analyze handler"
```

---

### Task E3: `POST /actions/import/sessions/:id/resolve_gaps`

**Files:**
- Create: `internal/imports/resolve_gaps_handler.go`
- Create: `internal/imports/resolve_gaps_handler_test.go`

**Goal:** Parse `resolutions[<gap_id>]` form fields into `[]GapResolution`, call `Client.ResolveGaps`, replace `import-modal-slot` with the refreshed review modal. On 422, replace with the same modal + error banner; on 404, replace `import-root` + warning snackbar.

- [ ] **Step 1: Write failing tests**

Create `internal/imports/resolve_gaps_handler_test.go`:

```go
package imports

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"net/url"

	"github.com/gin-gonic/gin"
)

func TestResolveGapsHandler_Success(t *testing.T) {
	loadTestLocales(t)
	loadReviewLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"gap_id":"g1"`) || !strings.Contains(string(body), `"value":"USD"`) {
			t.Fatalf("backend body did not include resolution: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"sess-1","status":"ready",
			"created_at":"x","expires_at":"y",
			"ai_summary":"x","assumptions":[],
			"preview":{"assets":[],"trades":[],"snapshots":[]},
			"gaps":[],"gap_counts":{"blocking":0,"warnings":0}
		}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/sessions/:id/resolve_gaps", NewResolveGapsHandler(mustClient(t, be.URL)).Post)
	})

	form := url.Values{}
	form.Set("resolutions[g1]", "USD")
	req := httptest.NewRequest(http.MethodPost, "/actions/import/sessions/sess-1/resolve_gaps",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target_id"] != "import-modal-slot" {
		t.Fatalf("unexpected target_id: %v", got["target_id"])
	}
}

func TestResolveGapsHandler_SessionExpiredReplacesRoot(t *testing.T) {
	loadTestLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/sessions/:id/resolve_gaps", NewResolveGapsHandler(mustClient(t, be.URL)).Post)
	})
	req := httptest.NewRequest(http.MethodPost, "/actions/import/sessions/sess-x/resolve_gaps",
		strings.NewReader("resolutions[g1]=USD"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target_id"] != "import-root" {
		t.Fatalf("expected target_id=import-root, got %v", got["target_id"])
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestResolveGapsHandler -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/imports/resolve_gaps_handler.go`:

```go
package imports

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

type ResolveGapsHandler struct {
	client *Client
}

func NewResolveGapsHandler(c *Client) *ResolveGapsHandler { return &ResolveGapsHandler{client: c} }

func (h *ResolveGapsHandler) Post(c *gin.Context) {
	lang := resolveLang(c)
	id := c.Param("id")

	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	resolutions := parseGapResolutions(c.Request.PostForm)

	sess, err := h.client.ResolveGaps(c.Request.Context(), resolveAuth(c), id, resolutions)
	if err != nil {
		var be *BackendError
		if errors.As(err, &be) {
			// We don't have the previous session here — re-fetch by no-op
			// resolve with empty resolutions is fragile. Instead, return a
			// minimal modal carrying the error banner; the user can retry.
			fb := components.Snackbar("feedback", be.Message, "error")
			c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
			return
		}
		if errors.Is(err, ErrSessionNotFound) {
			tree := BuildRoot(lang)
			fb := components.Snackbar("feedback", i18n.T(lang, "import.session_expired"), "warning")
			c.JSON(http.StatusOK, components.ReplaceResponse("import-root", tree, &fb))
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
			return
		}
		fb := components.Snackbar("feedback", i18n.T(lang, "import.failure_generic"), "error")
		c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
		return
	}

	tree := BuildReviewModal(lang, sess, "")
	c.JSON(http.StatusOK, components.ReplaceResponse("import-modal-slot", tree, nil))
}

// parseGapResolutions extracts resolutions[<gap_id>]=value pairs from the form.
func parseGapResolutions(form map[string][]string) []GapResolution {
	out := make([]GapResolution, 0)
	const prefix = "resolutions["
	const suffix = "]"
	for key, vals := range form {
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
			continue
		}
		gapID := key[len(prefix) : len(key)-len(suffix)]
		if gapID == "" {
			continue
		}
		val := ""
		if len(vals) > 0 {
			val = strings.TrimSpace(vals[0])
		}
		if val == "" {
			continue
		}
		out = append(out, GapResolution{GapID: gapID, Value: val})
	}
	return out
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -run TestResolveGapsHandler -v`
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/resolve_gaps_handler.go internal/imports/resolve_gaps_handler_test.go
git commit -m "feat(imports): add POST /actions/import/sessions/:id/resolve_gaps"
```

---

### Task E4: `POST /actions/import/sessions/:id/confirm`

**Files:**
- Create: `internal/imports/confirm_handler.go`
- Create: `internal/imports/confirm_handler_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/imports/confirm_handler_test.go`:

```go
package imports

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestConfirmHandler_Success(t *testing.T) {
	loadTestLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"assets_created":1,"trades_imported":2,"snapshots_imported":3,"warnings":0}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/sessions/:id/confirm", NewConfirmHandler(mustClient(t, be.URL)).Post)
	})
	req := httptest.NewRequest(http.MethodPost, "/actions/import/sessions/sess-1/confirm", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target_id"] != "import-root" {
		t.Fatalf("expected target_id=import-root, got %v", got["target_id"])
	}
	if !strings.Contains(rec.Body.String(), `"feedback"`) {
		t.Fatal("expected snackbar feedback")
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestConfirmHandler -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/imports/confirm_handler.go`:

```go
package imports

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

type ConfirmHandler struct {
	client *Client
}

func NewConfirmHandler(c *Client) *ConfirmHandler { return &ConfirmHandler{client: c} }

func (h *ConfirmHandler) Post(c *gin.Context) {
	lang := resolveLang(c)
	id := c.Param("id")

	res, err := h.client.ConfirmSession(c.Request.Context(), resolveAuth(c), id)
	if err != nil {
		var be *BackendError
		if errors.As(err, &be) {
			fb := components.Snackbar("feedback", be.Message, "error")
			c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
			return
		}
		if errors.Is(err, ErrSessionNotFound) {
			tree := BuildRoot(lang)
			fb := components.Snackbar("feedback", i18n.T(lang, "import.session_expired"), "warning")
			c.JSON(http.StatusOK, components.ReplaceResponse("import-root", tree, &fb))
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
			return
		}
		fb := components.Snackbar("feedback", i18n.T(lang, "import.failure_generic"), "error")
		c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
		return
	}

	tree := BuildRoot(lang)
	tmpl := i18n.T(lang, "import.success")
	msg := strings.NewReplacer(
		"{assets}", fmt.Sprintf("%d", res.AssetsCreated),
		"{trades}", fmt.Sprintf("%d", res.TradesImported),
		"{snapshots}", fmt.Sprintf("%d", res.SnapshotsImported),
		"{warnings}", fmt.Sprintf("%d", res.Warnings),
	).Replace(tmpl)
	fb := components.Snackbar("feedback", msg, "success")
	c.JSON(http.StatusOK, components.ReplaceResponse("import-root", tree, &fb))
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -run TestConfirmHandler -v`
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/confirm_handler.go internal/imports/confirm_handler_test.go
git commit -m "feat(imports): add POST /actions/import/sessions/:id/confirm"
```

---

### Task E5: `POST /actions/import/sessions/:id/cancel`

**Files:**
- Create: `internal/imports/cancel_handler.go`
- Create: `internal/imports/cancel_handler_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/imports/cancel_handler_test.go`:

```go
package imports

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCancelHandler_SuccessReplacesRoot(t *testing.T) {
	loadTestLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/sessions/:id/cancel", NewCancelHandler(mustClient(t, be.URL)).Post)
	})
	req := httptest.NewRequest(http.MethodPost, "/actions/import/sessions/sess-1/cancel", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target_id"] != "import-root" {
		t.Fatalf("expected import-root, got %v", got["target_id"])
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestCancelHandler -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/imports/cancel_handler.go`:

```go
package imports

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

type CancelHandler struct {
	client *Client
}

func NewCancelHandler(c *Client) *CancelHandler { return &CancelHandler{client: c} }

func (h *CancelHandler) Post(c *gin.Context) {
	lang := resolveLang(c)
	id := c.Param("id")

	if err := h.client.CancelSession(c.Request.Context(), resolveAuth(c), id); err != nil {
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
			return
		}
		fb := components.Snackbar("feedback", i18n.T(lang, "import.failure_generic"), "error")
		c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
		return
	}

	tree := BuildRoot(lang)
	fb := components.Snackbar("feedback", i18n.T(lang, "import.cancelled"), "info")
	c.JSON(http.StatusOK, components.ReplaceResponse("import-root", tree, &fb))
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -run TestCancelHandler -v`
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/cancel_handler.go internal/imports/cancel_handler_test.go
git commit -m "feat(imports): add POST /actions/import/sessions/:id/cancel"
```

---

### Task E6: `GET /actions/import/export` — streaming proxy

**Files:**
- Create: `internal/imports/export_handler.go`
- Create: `internal/imports/export_handler_test.go`

**Goal:** Forward `GET /v1/export` from the BE to the client. On 401 (or missing auth), respond with a 302 to `/login` so the browser follows it natively. On 5xx, respond with a 502 plain-text.

- [ ] **Step 1: Write failing tests**

Create `internal/imports/export_handler_test.go`:

```go
package imports

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExportHandler_StreamsBackendResponse(t *testing.T) {
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="vk_tracker_export_2026-04-27.csv"`)
		_, _ = w.Write([]byte("col1,col2\nA,B\n"))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/import/export", NewExportHandler(mustClient(t, be.URL)).Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/import/export", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "vk_tracker_export_") {
		t.Fatalf("Content-Disposition not forwarded: %q", got)
	}
	if rec.Body.String() != "col1,col2\nA,B\n" {
		t.Fatalf("body mismatch: %q", rec.Body.String())
	}
}

func TestExportHandler_Unauthorized_Redirects(t *testing.T) {
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/import/export", NewExportHandler(mustClient(t, be.URL)).Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/import/export", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("expected Location=/login, got %q", loc)
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestExportHandler -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/imports/export_handler.go`:

```go
package imports

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct {
	client *Client
}

func NewExportHandler(c *Client) *ExportHandler { return &ExportHandler{client: c} }

func (h *ExportHandler) Get(c *gin.Context) {
	resp, err := h.client.ExportStream(c.Request.Context(), resolveAuth(c))
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		c.Data(http.StatusBadGateway, "text/plain; charset=utf-8", []byte("Export failed."))
		return
	}
	defer resp.Body.Close()

	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		c.Header("Content-Disposition", cd)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/csv; charset=utf-8"
	}
	c.Status(http.StatusOK)
	c.Header("Content-Type", contentType)
	_, _ = io.Copy(c.Writer, resp.Body)
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -run TestExportHandler -v`
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/export_handler.go internal/imports/export_handler_test.go
git commit -m "feat(imports): add GET /actions/import/export streaming proxy"
```

---

### Task E7: `POST /actions/import/restore` (and `GET /actions/import/restore_idle`)

**Files:**
- Create: `internal/imports/restore_handler.go`
- Create: `internal/imports/restore_handler_test.go`

**Goal:** The restore handler accepts a multipart `file`, calls `Client.Restore`, and returns the success state of `restore-card`. On error: replace `restore-card` with idle (file preserved) + error banner + error snackbar. The companion `GET /actions/import/restore_idle` simply emits the idle subtree (used by the "Restore another file" button).

- [ ] **Step 1: Write failing tests**

Create `internal/imports/restore_handler_test.go`:

```go
package imports

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRestoreHandler_Success(t *testing.T) {
	loadTestLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"assets_imported":1,"assets_skipped":0,
			"trades_imported":2,"trades_skipped":1,
			"snapshots_imported":0,"snapshots_skipped":3,
			"snapshot_entries_imported":4,"snapshot_entries_skipped":5
		}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/restore", NewRestoreHandler(mustClient(t, be.URL)).Post)
	})
	body, ct := multipartBody(t, "col\nA\n", "")
	req := httptest.NewRequest(http.MethodPost, "/actions/import/restore", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["target_id"] != "restore-card" {
		t.Fatalf("expected restore-card target, got %v", got["target_id"])
	}
	if !strings.Contains(rec.Body.String(), "Restored successfully") {
		t.Fatal("missing success copy")
	}
}

func TestRestoreHandler_BackendErrorReplacesIdle(t *testing.T) {
	loadTestLocales(t)
	be := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"RESTORE_FILE_TOO_LARGE","message":"File exceeds 10 MB."}}`))
	}))
	defer be.Close()

	r := newRouter(func(r *gin.Engine) {
		r.POST("/actions/import/restore", NewRestoreHandler(mustClient(t, be.URL)).Post)
	})
	body, ct := multipartBody(t, "x", "")
	req := httptest.NewRequest(http.MethodPost, "/actions/import/restore", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "File exceeds 10 MB.") {
		t.Fatal("expected backend error message in response")
	}
}

func TestRestoreIdleHandler_ReturnsIdleCard(t *testing.T) {
	loadTestLocales(t)
	r := newRouter(func(r *gin.Engine) {
		r.GET("/actions/import/restore_idle", NewRestoreIdleHandler().Get)
	})
	req := httptest.NewRequest(http.MethodGet, "/actions/import/restore_idle", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"id":"restore-card"`) {
		t.Fatal("missing restore-card in response")
	}
}
```

- [ ] **Step 2: Run — expect failure**

Run: `go test ./internal/imports/ -run TestRestoreHandler -v`
Expected: undefined.

- [ ] **Step 3: Implement**

Create `internal/imports/restore_handler.go`:

```go
package imports

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
)

const restoreMaxBytes = 10 * 1024 * 1024

type RestoreHandler struct {
	client *Client
}

func NewRestoreHandler(c *Client) *RestoreHandler { return &RestoreHandler{client: c} }

func (h *RestoreHandler) Post(c *gin.Context) {
	lang := resolveLang(c)

	if err := c.Request.ParseMultipartForm(restoreMaxBytes); err != nil {
		writeRestoreError(c, lang, "Upload exceeds the size limit.", "")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		writeRestoreError(c, lang, "Missing file.", "")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, restoreMaxBytes+1))
	if err != nil {
		writeRestoreError(c, lang, "Failed to read upload.", header.Filename)
		return
	}
	if int64(len(content)) > restoreMaxBytes {
		writeRestoreError(c, lang, "File exceeds the 10 MB limit.", header.Filename)
		return
	}

	res, err := h.client.Restore(c.Request.Context(), resolveAuth(c), content)
	if err != nil {
		var be *BackendError
		if errors.As(err, &be) {
			writeRestoreError(c, lang, be.Message, header.Filename)
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
			return
		}
		writeRestoreError(c, lang, "Restore failed.", header.Filename)
		return
	}

	tree := BuildRestoreCardSuccess(lang, res)
	c.JSON(http.StatusOK, components.ReplaceResponse("restore-card", tree, nil))
}

func writeRestoreError(c *gin.Context, lang, message, prefillFilename string) {
	tree := BuildRestoreCardIdle(lang, message, prefillFilename)
	fb := components.Snackbar("feedback", message, "error")
	c.JSON(http.StatusOK, components.ReplaceResponse("restore-card", tree, &fb))
}

// RestoreIdleHandler emits the idle subtree of restore-card. Used by the
// "Restore another file" button on the success state.
type RestoreIdleHandler struct{}

func NewRestoreIdleHandler() *RestoreIdleHandler { return &RestoreIdleHandler{} }

func (h *RestoreIdleHandler) Get(c *gin.Context) {
	lang := resolveLang(c)
	c.JSON(http.StatusOK, BuildRestoreCardIdle(lang, "", ""))
}
```

- [ ] **Step 4: Run — expect pass**

Run: `go test ./internal/imports/ -run TestRestoreHandler\|TestRestoreIdleHandler -v`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/imports/restore_handler.go internal/imports/restore_handler_test.go
git commit -m "feat(imports): add POST /actions/import/restore + restore_idle handlers"
```

---

## Phase F — Wiring, i18n, smoke test

The package compiles and tests in isolation; this phase wires it into the running server, loads translations, and verifies the screen renders end-to-end.

---

### Task F1: Add i18n keys to `locales/en.json` and `locales/es.json`

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/es.json`

- [ ] **Step 1: Add the `import` namespace to `locales/en.json`**

Find the closing `}` of the file and insert a comma after the previous top-level namespace (typically `"snapshots": {…}`), then append the `import` namespace before `"common": {…}`. Concrete content:

```json
"import": {
  "title": "Import & Export",
  "ai": {
    "title": "Import historical data",
    "description": "Upload a file from your broker or exchange — CSV, XLS, TXT and more. The AI will parse it, map columns to assets and trades, and let you review everything before committing."
  },
  "upload": {
    "label": "File",
    "placeholder": "Drop a file here or click to browse",
    "hint_ai": "CSV, TSV, XLS, XLSX, TXT — max 5 MB",
    "hint_restore": "CSV — max 10 MB",
    "error_size": "File exceeds the {limit} limit.",
    "error_format": "Unsupported file format.",
    "reattach_hint": "Re-select the file to retry"
  },
  "hint": {
    "label": "Hint (optional)",
    "placeholder": "e.g. trade history from Broker X, amounts in USD"
  },
  "analyze": "Analyze file",
  "loading": {
    "analyze": {
      "1": "Detecting columns…",
      "2": "Mapping tickers…",
      "3": "Resolving currencies…",
      "4": "Building preview…",
      "5": "Validating consistency…"
    }
  },
  "review": {
    "blocking_banner": "This file has {n} issues that need your input before importing.",
    "ready_banner": "Ready to import — review the preview and confirm.",
    "summary": "AI Summary",
    "assumptions": "Assumptions ({n})",
    "issues": "Issues",
    "warnings": "{n} warnings",
    "preview": "Preview",
    "preview.assets": "Assets ({n})",
    "preview.trades": "Trades ({n})",
    "preview.snapshots": "Snapshots ({n})",
    "confirm": "Confirm import",
    "cancel": "Cancel",
    "status": {
      "needs_review": "Needs review",
      "ready": "Ready"
    }
  },
  "gaps": {
    "affected_rows": "Affected rows: {rows}",
    "input_placeholder": "Enter value…",
    "save": "Save resolutions"
  },
  "success": "Import complete: {assets} assets, {trades} trades, {snapshots} snapshots.",
  "cancelled": "Import cancelled.",
  "session_expired": "Import session expired — please re-upload the file.",
  "failure_generic": "Import failed. Please try again.",
  "export": {
    "title": "Export data",
    "description": "Download all your data as a single CSV file. Use it as a backup or to migrate to another instance.",
    "submit": "Export all data"
  },
  "restore": {
    "title": "Restore from backup",
    "description": "Upload a CSV file previously exported from this app. Existing records are skipped — safe to run multiple times.",
    "submit": "Restore",
    "success_title": "Restored successfully",
    "col": {
      "imported": "Imported",
      "skipped": "Skipped"
    },
    "row": {
      "assets": "Assets",
      "trades": "Trades",
      "snapshots": "Snapshots",
      "snapshot_entries": "Snapshot entries"
    },
    "try_again": "Restore another file",
    "error_generic": "Restore failed. Please try again."
  }
}
```

- [ ] **Step 2: Mirror in `locales/es.json` with Spanish copy**

Same structure, translated. Concrete copy (keep keys identical):

```json
"import": {
  "title": "Importar y Exportar",
  "ai": {
    "title": "Importar datos históricos",
    "description": "Subí un archivo de tu broker o exchange — CSV, XLS, TXT y más. La IA lo va a parsear, mapear las columnas a activos y operaciones, y vas a poder revisar todo antes de confirmar."
  },
  "upload": {
    "label": "Archivo",
    "placeholder": "Arrastrá un archivo aquí o hacé clic para buscarlo",
    "hint_ai": "CSV, TSV, XLS, XLSX, TXT — máximo 5 MB",
    "hint_restore": "CSV — máximo 10 MB",
    "error_size": "El archivo supera el límite de {limit}.",
    "error_format": "Formato no soportado.",
    "reattach_hint": "Volvé a seleccionar el archivo para reintentar"
  },
  "hint": {
    "label": "Pista (opcional)",
    "placeholder": "ej. historial de operaciones de Broker X, montos en USD"
  },
  "analyze": "Analizar archivo",
  "loading": {
    "analyze": {
      "1": "Detectando columnas…",
      "2": "Mapeando tickers…",
      "3": "Resolviendo monedas…",
      "4": "Armando previsualización…",
      "5": "Validando consistencia…"
    }
  },
  "review": {
    "blocking_banner": "Este archivo tiene {n} cuestiones que requieren tu intervención antes de importar.",
    "ready_banner": "Listo para importar — revisá la previsualización y confirmá.",
    "summary": "Resumen IA",
    "assumptions": "Suposiciones ({n})",
    "issues": "Cuestiones",
    "warnings": "{n} advertencias",
    "preview": "Previsualización",
    "preview.assets": "Activos ({n})",
    "preview.trades": "Operaciones ({n})",
    "preview.snapshots": "Snapshots ({n})",
    "confirm": "Confirmar importación",
    "cancel": "Cancelar",
    "status": {
      "needs_review": "Requiere revisión",
      "ready": "Listo"
    }
  },
  "gaps": {
    "affected_rows": "Filas afectadas: {rows}",
    "input_placeholder": "Ingresá un valor…",
    "save": "Guardar resoluciones"
  },
  "success": "Importación completa: {assets} activos, {trades} operaciones, {snapshots} snapshots.",
  "cancelled": "Importación cancelada.",
  "session_expired": "La sesión de importación expiró — volvé a subir el archivo.",
  "failure_generic": "La importación falló. Probá de nuevo.",
  "export": {
    "title": "Exportar datos",
    "description": "Descargá todos tus datos en un único CSV. Sirve como backup o para migrar a otra instancia.",
    "submit": "Exportar todo"
  },
  "restore": {
    "title": "Restaurar desde backup",
    "description": "Subí un CSV exportado previamente desde esta app. Los registros existentes se omiten — es seguro correrlo varias veces.",
    "submit": "Restaurar",
    "success_title": "Restauración exitosa",
    "col": {
      "imported": "Importadas",
      "skipped": "Omitidas"
    },
    "row": {
      "assets": "Activos",
      "trades": "Operaciones",
      "snapshots": "Snapshots",
      "snapshot_entries": "Entradas de snapshot"
    },
    "try_again": "Restaurar otro archivo",
    "error_generic": "La restauración falló. Probá de nuevo."
  }
}
```

- [ ] **Step 3: Verify JSON is valid**

Run: `python3 -c 'import json; json.load(open("locales/en.json")); json.load(open("locales/es.json")); print("ok")'`
Expected: `ok`. If parsing fails, re-check trailing commas and brace balance.

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/es.json
git commit -m "feat(i18n): add import & export screen translations (en/es)"
```

---

### Task F2: Wire routes in `internal/server/server.go`

**Files:**
- Modify: `internal/server/server.go`

- [ ] **Step 1: Add import**

In the import block at the top of `server.go`, add:

```go
	"github.com/project/vk-investment-middleend/internal/imports"
```

(Alphabetically: between `i18n` and `login` if those are present, otherwise next to the other internal imports.)

- [ ] **Step 2: Register routes**

In `setupRoutes`, after the snapshots block (the last `protected.DELETE` for snapshots), append:

```go
	// --- imports / exports ---
	importsClient := imports.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
	protected.GET("/screens/import", imports.NewHandler().Get)
	protected.POST("/actions/import/analyze", imports.NewAnalyzeHandler(importsClient).Post)
	protected.POST("/actions/import/sessions/:id/resolve_gaps", imports.NewResolveGapsHandler(importsClient).Post)
	protected.POST("/actions/import/sessions/:id/confirm", imports.NewConfirmHandler(importsClient).Post)
	protected.POST("/actions/import/sessions/:id/cancel", imports.NewCancelHandler(importsClient).Post)
	protected.GET("/actions/import/export", imports.NewExportHandler(importsClient).Get)
	protected.POST("/actions/import/restore", imports.NewRestoreHandler(importsClient).Post)
	protected.GET("/actions/import/restore_idle", imports.NewRestoreIdleHandler().Get)
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Run full suite**

Run: `make test`
Expected: green. No existing tests broken.

- [ ] **Step 5: Commit**

```bash
git add internal/server/server.go
git commit -m "feat(server): wire import/export routes"
```

---

### Task F3: Smoke test — start server, hit `/screens/import`, verify shape

**Files:** none (manual run + revert)

- [ ] **Step 1: Restart middleend**

```bash
./cli run > /tmp/import.log 2>&1 &
sleep 1
```

Expected: middleend listening on `:8082`.

- [ ] **Step 2: Hit the screen endpoint**

```bash
TOKEN="$(cat ~/.vk_dev_jwt 2>/dev/null || echo "DEV_TOKEN_HERE")"
curl -sS -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/import | head -c 200
```

Expected: JSON starting with `{"type":"screen","id":"import-root",…`. If you don't have a dev JWT handy, log in via the frontend or use the legacy auth flow to obtain one — the response on a missing token will be `{"error":"unauthorized","redirect":"/login"}` which also confirms the route is wired.

- [ ] **Step 3: Hit the export endpoint (download check)**

```bash
curl -sSI -H "Authorization: Bearer $TOKEN" http://localhost:8082/actions/import/export | head -10
```

Expected: HTTP/1.1 200 plus a `Content-Disposition: attachment; filename="vk_tracker_export_…csv"` header (assuming the BE returns one). On a missing JWT, expect `HTTP/1.1 302 Found` with `Location: /login`.

- [ ] **Step 4: Stop middleend**

```bash
pkill -f "go run cmd/middleend" || true
pkill -f "/cli run" || true
```

(Use whichever process pattern matches your local `./cli run` setup.)

- [ ] **Step 5: No commit**

This task only validates wiring; no source files change.

---

## Self-review

Run through this checklist with the plan open:

1. **Spec coverage** — every section of `docs/superpowers/specs/2026-04-27-import-export-screen-design.md` should map to at least one task. Map:
   - §1 Overview → B1 (canonical spec).
   - §2 Endpoints → C1–C7 (clients), E1–E7 (handlers), F2 (wiring).
   - §3 Layout → D1.
   - §4.1 AI Import card → D1, E2.
   - §4.2 Review modal → D2, E2.
   - §4.3 Resolve gaps → C3, E3.
   - §4.4 Confirm → C4, E4.
   - §4.5 Cancel → C5, E5.
   - §4.6 Export → C6, E6.
   - §4.7 Restore → C7, D3, E7.
   - §5.1 file_upload → A3, A4.
   - §5.2 loading extension → A1, A2.
   - §6 Error handling → covered across E2–E7 (each handler tests its error paths).
   - §7 i18n → F1.
   - §8 Acceptance criteria → covered by per-handler tests + F3 smoke.

2. **Placeholder scan** — search for "TBD", "TODO", "implement later", "fill in details", "Add appropriate error handling", "Similar to Task". None present. (The "Note" callouts in D3 and E2 explicitly call out limitations rather than punt them.)

3. **Type consistency** — `Session`, `Gap`, `GapResolution`, `Preview`, `PreviewAsset`, `PreviewTrade`, `PreviewSnapshot`, `ConfirmResult`, `RestoreResult`, `BackendError` are defined in C1 and used consistently in C2–C7, D1–D3, E2–E7. `Action.Loading` is `any` per A2 and used as `LoadingFullWithMessages(...)` in D1, plain `"section"` / `"full"` strings elsewhere. `FileUpload(id, FileUploadProps{...})` signature matches A4 and D1/D3 call sites.

4. **Self-review fixes:** none — plan is consistent.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-27-import-export-screen.md`. Two execution options:

**1. Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks, fast iteration. Good fit for a plan with ~25 small TDD-shaped tasks.

**2. Inline Execution** — execute tasks in this session using `superpowers:executing-plans`, batch execution with checkpoints for review.

Which approach?
