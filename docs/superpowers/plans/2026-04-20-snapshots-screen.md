# Snapshots Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the full Snapshots SDUI screen (browse with expandable rows, manual-wizard create, auto-snapshot, edit via wizard, delete) end-to-end, plus the two SDUI extensions it requires: a new `wizard` custom component and an `expandable` / `details` slot on the base `table_row`.

**Architecture:** Mirror `internal/trades/` and `internal/assets/` for the screen package structure. Three distinct concerns:
1. **SDUI base/custom extensions** — land **first** because the screen depends on them. `table_row.expandable/details` in `internal/components/table.go`; `wizard` helper in `internal/components/wizard.go`. Canonical specs updated in `spec/sdui-base-components.md` and `spec/sdui-custom-components.md`.
2. **Canonical screen spec** — `spec/screens/snapshots.md`, written as the contract (mirrors trades.md/assets.md shape).
3. **`internal/snapshots/`** — types, BE client, builder, wizard builder, use case, handlers.

Follow the trades/assets TDD pattern: test first, minimal code, commit; restart middleend after any server/handler change.

**Tech Stack:** Go · Gin · testify · stdlib `net/http/httptest`. Existing SDUI library in `internal/components/`. Existing `internal/shared/format` and `internal/shared/assetscatalog`. Brainstorm design at `docs/superpowers/specs/2026-04-20-snapshots-screen-design.md`.

---

## Conventions used throughout this plan

- **All paths are repo-relative to `/Users/vadimkent/repos/vk_investment_middleend_v2/`.**
- **TDD:** each behavior gets a failing test first, then the minimum code to pass, then a commit.
- **Reference files to emulate** (read first when starting each phase; they show the established pattern):
  - `internal/trades/types.go` / `client.go` / `mutate_client.go` / `get_usecase.go`
  - `internal/trades/builder.go` / `modal_builder.go` / `request.go`
  - `internal/trades/handler.go` / `list_handler.go` / `create_handler.go` / `update_handler.go` / `delete_handler.go`
  - `internal/trades/create_modal_handler.go` / `edit_modal_handler.go` / `delete_modal_handler.go`
  - `internal/trades/*_test.go` for test style
  - `internal/components/table.go`, `base.go`, `actions.go`, `charts.go` for SDUI conventions
  - `internal/server/server.go` for wiring
  - `spec/screens/trades.md` for canonical spec shape
  - `spec/sdui-base-components.md` / `spec/sdui-custom-components.md` for SDUI contract shape
  - `locales/en.json`, `locales/es.json` — `trades.*` namespace for key style
- **Commit message style:** Conventional Commits (`feat(snapshots): …`, `feat(components): …`, `docs(spec): …`, `test(snapshots): …`). **No Claude co-author trailer** unless the user explicitly asks.
- **Middleend restart:** after any phase that touches server/handlers/routes, kill port `:8082` and run `./cli run` in background.
- **Brainstorm design is reference, not contract.** Canonical specs in `spec/` are the contract. If implementation reveals a better path, update the canonical spec (and note the deviation in the commit message).

---

## Phase 0 — Preparation

### Task 0.1: Read the existing code, spec, and design

**Files to read (no changes):**
- `docs/superpowers/specs/2026-04-20-snapshots-screen-design.md` — the full design.
- `spec/screens/trades.md` and `spec/screens/assets.md` — canonical spec shape to mirror.
- `spec/sdui-base-components.md` (full) — especially `§table`, `§table_row`, `§modal`.
- `spec/sdui-custom-components.md` (full) — especially `§line_chart`, `§pie_chart` as custom-component precedent.
- `spec/sdui-actions.md` — to understand `submit`, `replace`, `dismiss`, `ActionResponse`.
- `spec/shared/assets-catalog.md` — the catalog helper snapshots will reuse.
- `be_specs/api/snapshots.md` — the backend contract in full.
- `internal/trades/` (all files) — the mirror for the screen package.
- `internal/components/table.go`, `internal/components/base.go`, `internal/components/charts.go`.
- `internal/server/server.go`.

- [ ] **Step 1: Read and internalize. No code changes.**

---

## Phase 1 — SDUI base/custom extensions

The screen cannot be built until `table_row.expandable/details` and `wizard` exist in the middleend's component library.

### Task 1.1: Extend `table_row` with `expandable` + `details`

**Files:**
- Modify: `internal/components/table.go`
- Modify: `internal/components/table_test.go`

Backward-compatible addition: existing `TableRow(id, children...)` unchanged. New helper adds an `expandable: true` prop and a `details` slot carried in `Props["details"]` as a `[]Component`. This serializes correctly via `json.Marshal` on `Component.Props map[string]any` — the nested `Component` values have their own JSON tags, so recursion works.

- [ ] **Step 1: Write the failing tests in `internal/components/table_test.go`.**

```go
package components

import (
	"encoding/json"
	"testing"
)

func TestTableRowExpandable_SetsProps(t *testing.T) {
	details := []Component{Text("entry-1", "detail cell", "sm", "normal")}
	row := TableRowExpandable("row-1",
		[]Component{Text("c1", "Date", "sm", "normal")},
		details...,
	)

	if row.Type != "table_row" {
		t.Fatalf("type = %q, want table_row", row.Type)
	}
	if row.ID != "row-1" {
		t.Fatalf("id = %q, want row-1", row.ID)
	}
	if got := row.Props["expandable"]; got != true {
		t.Fatalf("props.expandable = %v, want true", got)
	}
	got, ok := row.Props["details"].([]Component)
	if !ok {
		t.Fatalf("props.details not []Component, got %T", row.Props["details"])
	}
	if len(got) != 1 || got[0].ID != "entry-1" {
		t.Fatalf("details mismatch: %+v", got)
	}
	if len(row.Children) != 1 || row.Children[0].ID != "c1" {
		t.Fatalf("cells mismatch: %+v", row.Children)
	}
}

func TestTableRowExpandable_JSONShape(t *testing.T) {
	row := TableRowExpandable("row-1",
		[]Component{Text("c1", "hello", "sm", "normal")},
		Text("d1", "detail", "sm", "normal"),
	)
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		Props struct {
			Expandable bool        `json:"expandable"`
			Details    []Component `json:"details"`
		} `json:"props"`
		Children []Component `json:"children"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Props.Expandable {
		t.Fatalf("json props.expandable not true: %s", string(b))
	}
	if len(out.Props.Details) != 1 || out.Props.Details[0].ID != "d1" {
		t.Fatalf("json props.details bad: %s", string(b))
	}
	if len(out.Children) != 1 || out.Children[0].ID != "c1" {
		t.Fatalf("json children bad: %s", string(b))
	}
}

func TestTableRow_UnchangedByExpandableAddition(t *testing.T) {
	// Regression: the original TableRow helper must not set expandable/details.
	row := TableRow("row-1", Text("c1", "hello", "sm", "normal"))
	if _, ok := row.Props["expandable"]; ok {
		t.Fatalf("TableRow should not set expandable prop: %+v", row.Props)
	}
	if _, ok := row.Props["details"]; ok {
		t.Fatalf("TableRow should not set details prop: %+v", row.Props)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail.**

Run: `go test ./internal/components/ -run TestTableRow -v`
Expected: FAIL with `undefined: TableRowExpandable`.

- [ ] **Step 3: Implement `TableRowExpandable` in `internal/components/table.go`.**

Append below the existing `TableRow` function:

```go
// TableRowExpandable is a table_row that carries a details subtree rendered as
// a full-width panel when the row is expanded. Cells are the main-row cells
// (one per column). details is the subtree rendered beneath the row on expand.
// When any row in a table is expandable, the frontend auto-adds a chevron
// column to the left of the header to preserve column alignment.
func TableRowExpandable(id string, cells []Component, details ...Component) Component {
	return Component{
		Type: "table_row",
		ID:   id,
		Props: map[string]any{
			"expandable": true,
			"details":    details,
		},
		Children: cells,
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass.**

Run: `go test ./internal/components/ -run TestTableRow -v`
Expected: PASS.

- [ ] **Step 5: Run the full package tests to catch regressions.**

Run: `go test ./internal/components/ -v`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/components/table.go internal/components/table_test.go
git commit -m "feat(components): add TableRowExpandable with details slot"
```

---

### Task 1.2: Add `wizard` custom component helper

**Files:**
- Create: `internal/components/wizard.go`
- Create: `internal/components/wizard_test.go`

- [ ] **Step 1: Write the failing tests in `internal/components/wizard_test.go`.**

```go
package components

import (
	"encoding/json"
	"testing"
)

func TestWizard_BasicShape(t *testing.T) {
	submit := Submit("/actions/snapshots/create")
	dismiss := Action{Trigger: "click", Type: "replace", TargetID: "modal-slot"}

	steps := []WizardStep{
		{
			ID:             "info",
			Label:          "Info",
			Kind:           "info",
			Skippable:      false,
			IncludeDefault: true,
			Children:       []Component{Text("info-text", "Enter info", "sm", "normal")},
		},
		{
			ID:             "asset-1",
			Label:          "AAPL",
			Kind:           "entry",
			Skippable:      true,
			IncludeDefault: false,
			Children:       []Component{Text("asset-1-text", "AAPL details", "sm", "normal")},
		},
		{
			ID:             "summary",
			Label:          "Summary",
			Kind:           "summary",
			Skippable:      false,
			IncludeDefault: true,
			Children:       []Component{Text("summary-text", "Review", "sm", "normal")},
		},
	}
	w := Wizard("snapshot-wizard", "create", "New Snapshot", steps, submit, dismiss, nil, "")

	if w.Type != "wizard" {
		t.Fatalf("type = %q, want wizard", w.Type)
	}
	if w.ID != "snapshot-wizard" {
		t.Fatalf("id = %q", w.ID)
	}
	if w.Props["mode"] != "create" {
		t.Fatalf("mode = %v", w.Props["mode"])
	}
	if w.Props["title"] != "New Snapshot" {
		t.Fatalf("title = %v", w.Props["title"])
	}
	if _, ok := w.Props["steps"].([]WizardStep); !ok {
		t.Fatalf("steps not []WizardStep: %T", w.Props["steps"])
	}
	if _, ok := w.Props["submit_action"].(Action); !ok {
		t.Fatalf("submit_action not Action: %T", w.Props["submit_action"])
	}
	if _, ok := w.Props["dismiss_action"].(Action); !ok {
		t.Fatalf("dismiss_action not Action: %T", w.Props["dismiss_action"])
	}
	if _, hasBanner := w.Props["banner"]; hasBanner {
		t.Fatalf("banner should be omitted when nil")
	}
	if _, hasInitial := w.Props["initial_step_id"]; hasInitial {
		t.Fatalf("initial_step_id should be omitted when empty")
	}
}

func TestWizard_WithBannerAndInitialStep(t *testing.T) {
	submit := Submit("/x")
	dismiss := Action{Trigger: "click", Type: "replace", TargetID: "slot"}
	banner := &WizardBanner{Variant: "info", Message: "Created automatically", Dismissible: true}

	w := Wizard("w", "edit", "Edit", []WizardStep{
		{ID: "info", Label: "Info", Kind: "info", Skippable: false, IncludeDefault: true},
	}, submit, dismiss, banner, "info")

	gotBanner, ok := w.Props["banner"].(*WizardBanner)
	if !ok {
		t.Fatalf("banner not *WizardBanner: %T", w.Props["banner"])
	}
	if gotBanner.Variant != "info" || gotBanner.Message != "Created automatically" || !gotBanner.Dismissible {
		t.Fatalf("banner fields wrong: %+v", gotBanner)
	}
	if w.Props["initial_step_id"] != "info" {
		t.Fatalf("initial_step_id = %v", w.Props["initial_step_id"])
	}
}

func TestWizard_JSONShape(t *testing.T) {
	submit := Submit("/x")
	dismiss := Action{Trigger: "click", Type: "replace", TargetID: "slot"}

	w := Wizard("w", "create", "T", []WizardStep{
		{ID: "info", Label: "Info", Kind: "info", Skippable: false, IncludeDefault: true,
			Children: []Component{Text("t1", "hi", "sm", "normal")}},
	}, submit, dismiss, nil, "")

	b, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out struct {
		Type  string `json:"type"`
		ID    string `json:"id"`
		Props struct {
			Mode          string       `json:"mode"`
			Title         string       `json:"title"`
			Steps         []WizardStep `json:"steps"`
			SubmitAction  Action       `json:"submit_action"`
			DismissAction Action       `json:"dismiss_action"`
		} `json:"props"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Type != "wizard" || out.ID != "w" {
		t.Fatalf("type/id wrong: %s", string(b))
	}
	if out.Props.Mode != "create" || out.Props.Title != "T" {
		t.Fatalf("mode/title wrong: %s", string(b))
	}
	if len(out.Props.Steps) != 1 || out.Props.Steps[0].Kind != "info" {
		t.Fatalf("steps wrong: %s", string(b))
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail.**

Run: `go test ./internal/components/ -run TestWizard -v`
Expected: FAIL with `undefined: Wizard`, `undefined: WizardStep`, `undefined: WizardBanner`.

- [ ] **Step 3: Implement `internal/components/wizard.go`.**

```go
package components

// WizardStep is one step of a wizard. `Kind` drives which buttons render:
//   - "info":    Next.
//   - "entry":   Back, Skip (if Skippable), Include (create / new-entry in edit).
//                When mode=edit and the step is pre-existing (Skippable=false,
//                IncludeDefault=true), the frontend replaces Include with Update
//                and hides Skip.
//   - "summary": Back, Submit.
type WizardStep struct {
	ID             string      `json:"id"`
	Label          string      `json:"label"`
	Kind           string      `json:"kind"`
	Skippable      bool        `json:"skippable"`
	IncludeDefault bool        `json:"include_default"`
	Children       []Component `json:"children,omitempty"`
}

// WizardBanner is an optional banner rendered above the step content.
// Variants: "info", "success", "warning", "error". Title is an optional bold
// prefix; Dismissible controls whether the user can close the banner.
type WizardBanner struct {
	Variant     string `json:"variant"`
	Message     string `json:"message"`
	Title       string `json:"title,omitempty"`
	Dismissible bool   `json:"dismissible,omitempty"`
}

// Wizard builds a wizard custom component. See spec/sdui-custom-components.md.
//
// mode:           "create" / "edit".
// title:          localized title.
// steps:          ordered steps, at least one.
// submitAction:   action fired from the summary step (typically Submit(endpoint)).
// dismissAction:  action fired when the user closes the wizard.
// banner:         nil to omit.
// initialStepID:  "" to start on the first step; otherwise the id to focus.
func Wizard(id, mode, title string, steps []WizardStep, submitAction, dismissAction Action, banner *WizardBanner, initialStepID string) Component {
	props := map[string]any{
		"mode":           mode,
		"title":          title,
		"steps":          steps,
		"submit_action":  submitAction,
		"dismiss_action": dismissAction,
	}
	if banner != nil {
		props["banner"] = banner
	}
	if initialStepID != "" {
		props["initial_step_id"] = initialStepID
	}
	return Component{Type: "wizard", ID: id, Props: props}
}
```

- [ ] **Step 4: Run the tests to verify they pass.**

Run: `go test ./internal/components/ -run TestWizard -v`
Expected: PASS.

- [ ] **Step 5: Run the full package tests.**

Run: `go test ./internal/components/ -v`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add internal/components/wizard.go internal/components/wizard_test.go
git commit -m "feat(components): add wizard custom component helper"
```

---

### Task 1.3: Update canonical SDUI specs

**Files:**
- Modify: `spec/sdui-base-components.md` — extend the `table_row` section.
- Modify: `spec/sdui-custom-components.md` — add a new `§wizard` section after `pie_chart`.

- [ ] **Step 1: Update `spec/sdui-base-components.md` — `table_row` section.**

Locate the `### table_row` section (around line 239 per the current file). Replace its body with an updated version that adds the `expandable` prop, the `details` slot, the auto-chevron column behavior, state semantics, and the new Go helper. Preserve all existing content (cell mapping, subgrid, etc.) and only *add* the new contract.

Add this text to the `### table_row` section:

```markdown
| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `expandable` | bool | no | Default `false`. When `true`, the row is toggleable — clicking the main row expands / collapses a full-width `details` panel rendered beneath it. The frontend renders a chevron indicator. |

In addition to the cell `children`, expandable rows carry a `details` slot:

| Slot | Type | Description |
|------|------|-------------|
| `details` | `Component[]` | Subtree rendered as a full-width panel directly beneath the row when expanded. Pre-emitted in the tree (not fetched on expand). Breaks the subgrid only for its own row. |

When **any** row in a `table` is `expandable: true`, the frontend automatically prepends a 24px chevron column to the header (and to all rows — expandable or not — to preserve column alignment). This column is not part of `columns`; it is purely presentational.

Expand state is client-side, keyed per `row.id`. Multiple rows may be expanded simultaneously. State resets on any `replace` that rebuilds the table subtree.

```go
TableRowExpandable(id string, cells []Component, details ...Component) Component
```

The existing `TableRow(id, children...)` signature is unchanged — non-expandable rows remain the default.
```

- [ ] **Step 2: Update `spec/sdui-custom-components.md` — add a `§ wizard` section after `§ 2. pie_chart`.**

Insert a new numbered section `## 3. wizard` (and renumber the existing `## 3. Custom Attributes` and `## 4. Custom Actions` to `## 4.` and `## 5.`). Use the contract as defined in the design doc §9, at the same level of detail as `line_chart` / `pie_chart`:

- Intro paragraph: "Multi-step form container with local step state, Back/Next navigation without round-trips, per-step include/skip logic. Used by the snapshots screen's create/edit flow; reusable for other multi-step flows (import, analysis)."
- "Why custom" subsection.
- Props table (`mode`, `title`, `steps`, `submit_action`, `dismiss_action`, `banner`, `initial_step_id`).
- Sub-types `Step` and `Banner` tables.
- Frontend behavior (step indicator, buttons per kind, include map, navigation, summary step, submit, dismiss).
- Hidden input naming conventions.
- Validation / BE error handling.
- Minimal JSON example.

Copy the content from `docs/superpowers/specs/2026-04-20-snapshots-screen-design.md` §9 as the starting text — it is already self-contained.

- [ ] **Step 3: Verify the spec files still render as valid Markdown.**

Run: `head -5 spec/sdui-base-components.md && head -5 spec/sdui-custom-components.md`
Expected: no rendering errors; headings are correct.

- [ ] **Step 4: Commit.**

```bash
git add spec/sdui-base-components.md spec/sdui-custom-components.md
git commit -m "docs(spec): add wizard custom component and expandable table_row"
```

---

## Phase 2 — Canonical screen spec

### Task 2.1: Write `spec/screens/snapshots.md`

**Files:**
- Create: `spec/screens/snapshots.md`

Mirror the shape of `spec/screens/trades.md`. Sections: Purpose, Endpoints, Backend dependencies, Layout, Data and business rules (List / Expanded row / Empty states / Create / Auto / Edit / Delete / Post-mutation refresh / Filter and page preservation), i18n keys, Error handling, Acceptance criteria.

- [ ] **Step 1: Write the canonical spec.**

Source: `docs/superpowers/specs/2026-04-20-snapshots-screen-design.md`. Translate sections §1–§13 into the canonical spec shape (reuse the prose — same level of detail as trades.md). Drop the design-doc-only sections (§14 SDUI additions summary, §15 Canonical spec updates, §16 Out of scope). Trim forward-looking or "TBD" phrasing; this file describes shipped behavior.

- [ ] **Step 2: Sanity check — `wc -l spec/screens/snapshots.md`.**

Expected: ~200–230 lines (trades.md is 224 lines; this screen has similar complexity plus the wizard + auto flow).

- [ ] **Step 3: Commit.**

```bash
git add spec/screens/snapshots.md
git commit -m "docs(spec): add snapshots screen canonical spec"
```

### Task 2.2: Update `spec/spec.md` index

**Files:**
- Modify: `spec/spec.md`

- [ ] **Step 1: Replace the `TBD` entry for Snapshots with a real link.**

Find:
```
| Snapshots | `screens/snapshots.md` — TBD |
```
Replace with:
```
| Snapshots | [`screens/snapshots.md`](screens/snapshots.md) |
```

- [ ] **Step 2: Commit.**

```bash
git add spec/spec.md
git commit -m "docs(spec): link snapshots screen in index"
```

---

## Phase 3 — Domain types

### Task 3.1: `internal/snapshots/types.go`

**Files:**
- Create: `internal/snapshots/types.go`
- Create: `internal/snapshots/types_test.go`

Mirror `internal/trades/types.go`. Money-ish fields (`CurrentPrice`, `CurrentValueOverride`, `Quantity`) stay as strings to preserve precision; formatting in the builder via `internal/shared/format`. Complex assets can have `Quantity: ""` (BE returns `null`).

- [ ] **Step 1: Write `internal/snapshots/types_test.go`.**

```go
package snapshots

import "testing"

func TestParseListResponse_Basic(t *testing.T) {
	body := []byte(`{
		"snapshots": [
			{"id":"s1","recorded_at":"2024-01-10T10:00:00Z","is_full_snapshot":true,"notes":"hi",
			 "entries":[{"asset_id":"a1","quantity":"10.5","current_price":"150.00","current_value_override":null,"source":"MANUAL"}],
			 "created_at":"2024-01-10T10:00:00Z"}
		],
		"total": 1, "size": 10, "offset": 0
	}`)
	res, err := ParseListResponse(body)
	if err != nil {
		t.Fatalf("ParseListResponse err: %v", err)
	}
	if res.Total != 1 || res.Size != 10 || res.Offset != 0 {
		t.Fatalf("pagination wrong: %+v", res)
	}
	if len(res.Snapshots) != 1 {
		t.Fatalf("want 1 snapshot, got %d", len(res.Snapshots))
	}
	s := res.Snapshots[0]
	if s.ID != "s1" || !s.IsFullSnapshot || s.Notes != "hi" {
		t.Fatalf("header wrong: %+v", s)
	}
	if len(s.Entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(s.Entries))
	}
	e := s.Entries[0]
	if e.AssetID != "a1" || e.Quantity != "10.5" || e.CurrentPrice != "150.00" || e.CurrentValueOverride != "" || e.Source != "MANUAL" {
		t.Fatalf("entry wrong: %+v", e)
	}
}

func TestParseListResponse_NullQuantity(t *testing.T) {
	body := []byte(`{
		"snapshots":[
			{"id":"s1","recorded_at":"2024-01-10T10:00:00Z","is_full_snapshot":false,"notes":"",
			 "entries":[{"asset_id":"a1","quantity":null,"current_price":null,"current_value_override":"1000.00","source":"MANUAL"}],
			 "created_at":"2024-01-10T10:00:00Z"}
		],"total":1,"size":10,"offset":0
	}`)
	res, err := ParseListResponse(body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	e := res.Snapshots[0].Entries[0]
	if e.Quantity != "" {
		t.Fatalf("null quantity should parse to empty string, got %q", e.Quantity)
	}
	if e.CurrentPrice != "" {
		t.Fatalf("null current_price should parse to empty string, got %q", e.CurrentPrice)
	}
	if e.CurrentValueOverride != "1000.00" {
		t.Fatalf("current_value_override wrong: %q", e.CurrentValueOverride)
	}
}

func TestParseSnapshot_Single(t *testing.T) {
	body := []byte(`{"id":"s1","recorded_at":"2024-01-10T10:00:00Z","is_full_snapshot":true,"notes":"x",
		"entries":[{"asset_id":"a1","quantity":"1","current_price":"100","current_value_override":null,"source":"COINGECKO"}],
		"created_at":"2024-01-10T10:00:00Z"}`)
	s, err := ParseSnapshot(body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if s.ID != "s1" || len(s.Entries) != 1 || s.Entries[0].Source != "COINGECKO" {
		t.Fatalf("parsed wrong: %+v", s)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail.**

Run: `go test ./internal/snapshots/ -v`
Expected: FAIL (`no Go files`, or undefined types).

- [ ] **Step 3: Implement `internal/snapshots/types.go`.**

```go
// Package snapshots implements the Snapshots SDUI screen (browse with
// expandable rows, wizard create/edit, auto-snapshot, delete).
package snapshots

import "encoding/json"

// Entry is a single asset entry within a snapshot.
// Quantity, CurrentPrice, CurrentValueOverride are kept as strings to preserve
// decimal precision; empty string encodes BE null.
type Entry struct {
	AssetID              string
	Quantity             string
	CurrentPrice         string
	CurrentValueOverride string
	Source               string
}

// Snapshot is a timestamped portfolio capture.
type Snapshot struct {
	ID             string
	RecordedAt     string
	IsFullSnapshot bool
	Notes          string
	Entries        []Entry
	CreatedAt      string
}

// ListParams captures the list endpoint query parameters.
type ListParams struct {
	IsFullSnapshot *bool // nil = no filter; pointer distinguishes "false" from "unset"
	Offset         int
}

// ListResult wraps the parsed backend list response.
type ListResult struct {
	Snapshots []Snapshot
	Total     int
	Size      int
	Offset    int
}

type rawEntry struct {
	AssetID              string  `json:"asset_id"`
	Quantity             *string `json:"quantity"`
	CurrentPrice         *string `json:"current_price"`
	CurrentValueOverride *string `json:"current_value_override"`
	Source               string  `json:"source"`
}

func (r rawEntry) toDomain() Entry {
	return Entry{
		AssetID:              r.AssetID,
		Quantity:             deref(r.Quantity),
		CurrentPrice:         deref(r.CurrentPrice),
		CurrentValueOverride: deref(r.CurrentValueOverride),
		Source:               r.Source,
	}
}

type rawSnapshot struct {
	ID             string     `json:"id"`
	RecordedAt     string     `json:"recorded_at"`
	IsFullSnapshot bool       `json:"is_full_snapshot"`
	Notes          string     `json:"notes"`
	Entries        []rawEntry `json:"entries"`
	CreatedAt      string     `json:"created_at"`
}

func (r rawSnapshot) toDomain() Snapshot {
	entries := make([]Entry, 0, len(r.Entries))
	for _, e := range r.Entries {
		entries = append(entries, e.toDomain())
	}
	return Snapshot{
		ID:             r.ID,
		RecordedAt:     r.RecordedAt,
		IsFullSnapshot: r.IsFullSnapshot,
		Notes:          r.Notes,
		Entries:        entries,
		CreatedAt:      r.CreatedAt,
	}
}

type rawListResponse struct {
	Snapshots []rawSnapshot `json:"snapshots"`
	Total     int           `json:"total"`
	Size      int           `json:"size"`
	Offset    int           `json:"offset"`
}

// ParseListResponse parses the backend GET /v1/snapshots body.
func ParseListResponse(body []byte) (*ListResult, error) {
	var r rawListResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := &ListResult{Total: r.Total, Size: r.Size, Offset: r.Offset}
	out.Snapshots = make([]Snapshot, 0, len(r.Snapshots))
	for _, rs := range r.Snapshots {
		out.Snapshots = append(out.Snapshots, rs.toDomain())
	}
	return out, nil
}

// ParseSnapshot parses a single-snapshot backend response body.
func ParseSnapshot(body []byte) (*Snapshot, error) {
	var r rawSnapshot
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	s := r.toDomain()
	return &s, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
```

- [ ] **Step 4: Run tests.**

Run: `go test ./internal/snapshots/ -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/snapshots/types.go internal/snapshots/types_test.go
git commit -m "feat(snapshots): add domain types and response parsers"
```

---

## Phase 4 — Backend client

### Task 4.1: `client.go` — List and GetSnapshot

**Files:**
- Create: `internal/snapshots/client.go`
- Create: `internal/snapshots/client_test.go`

Mirror `internal/trades/client.go`. Always sends `size=10`, `sort=recorded_at`, `order=desc`, `offset`. Optional `is_full_snapshot` (`"true"`/`"false"`). Forwards `Authorization`. `ErrUnauthorized` on 401, `ErrBackend` otherwise. Also provide `GetSnapshot(id)` → `ErrSnapshotNotFound` on 404 (defined here, moved to `mutate_client.go` if preferred later for symmetry with trades — but for the List file it's fine to define both client methods here).

- [ ] **Step 1: Write `internal/snapshots/client_test.go` with table-driven HTTP-round-trip tests.**

Test shape mirrors `internal/trades/client_test.go`. Use `httptest.NewServer` and a small handler that inspects `r.URL.Query()` and returns canned JSON. Cover:
  - `List` with no filter → query has `size=10&sort=recorded_at&order=desc&offset=0` and no `is_full_snapshot`.
  - `List` with `IsFullSnapshot=true` → query carries `is_full_snapshot=true`.
  - `List` with `IsFullSnapshot=false` → query carries `is_full_snapshot=false` (distinguishing from nil).
  - `List` with `offset=20` → query carries `offset=20`.
  - `List` 401 → `ErrUnauthorized`.
  - `List` 500 → `ErrBackend`.
  - `List` malformed JSON → `ErrBackend`.
  - `GetSnapshot` 200 → correct Snapshot.
  - `GetSnapshot` 404 → `ErrSnapshotNotFound`.
  - `GetSnapshot` 401 → `ErrUnauthorized`.

(Use the trades client_test.go as the template — copy structure, adapt types.)

- [ ] **Step 2: Run tests, verify failures (undefined symbols).**

Run: `go test ./internal/snapshots/ -run TestClient -v`
Expected: FAIL.

- [ ] **Step 3: Implement `internal/snapshots/client.go`.**

```go
package snapshots

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

var (
	ErrUnauthorized      = errors.New("backend unauthorized")
	ErrBackend           = errors.New("backend error")
	ErrSnapshotNotFound  = errors.New("snapshot not found")
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

// List calls GET /v1/snapshots, always sending size=10, sort=recorded_at,
// order=desc, offset. Forwards Authorization. Emits is_full_snapshot only when
// the filter is set.
func (c *Client) List(ctx context.Context, authorization string, p ListParams) (*ListResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/snapshots", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("size", "10")
	q.Set("sort", "recorded_at")
	q.Set("order", "desc")
	q.Set("offset", strconv.Itoa(p.Offset))
	if p.IsFullSnapshot != nil {
		q.Set("is_full_snapshot", strconv.FormatBool(*p.IsFullSnapshot))
	}
	req.URL.RawQuery = q.Encode()
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		res, err := ParseListResponse(body)
		if err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return res, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// GetSnapshot fetches a single snapshot by id.
func (c *Client) GetSnapshot(ctx context.Context, authorization, id string) (*Snapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/snapshots/"+id, nil)
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return ParseSnapshot(body)
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrSnapshotNotFound
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
```

- [ ] **Step 4: Run tests, verify PASS.**

Run: `go test ./internal/snapshots/ -run TestClient -v`

- [ ] **Step 5: Commit.**

```bash
git add internal/snapshots/client.go internal/snapshots/client_test.go
git commit -m "feat(snapshots): add backend List and GetSnapshot client"
```

---

### Task 4.2: `mutate_client.go` — Create, Auto, Update, Delete + validation errors

**Files:**
- Create: `internal/snapshots/mutate_client.go`
- Create: `internal/snapshots/mutate_client_test.go`

Mirror `internal/trades/mutate_client.go`. Adds:
- `BackendValidationError` struct (`Code`, `Message`) — same shape as trades.
- `CreateSnapshot(ctx, auth, body)` → `POST /v1/snapshots`, expect 201.
- `AutoSnapshot(ctx, auth, notes)` → `POST /v1/snapshots/auto` with body `{"notes": notes}` (or `{}` if empty). Expect 201. Returns the parsed `Snapshot` and the `warnings` list. (Warnings are structured: `[{asset_id, ticker, error}]`.)
- `UpdateSnapshot(ctx, auth, id, body)` → `PATCH /v1/snapshots/:id`, expect 200.
- `DeleteSnapshot(ctx, auth, id)` → `DELETE /v1/snapshots/:id`, expect 204 (or 200).

Validation errors (422/400/409) → `BackendValidationError` with `Code` + `Message` from BE body.

Auto-snapshot response parsing needs a dedicated type:

```go
// AutoWarning identifies an asset whose provider price fetch failed during an auto-snapshot.
type AutoWarning struct {
	AssetID string
	Ticker  string
	Error   string
}

// AutoResult is the parsed response of POST /v1/snapshots/auto.
type AutoResult struct {
	Snapshot Snapshot
	Warnings []AutoWarning // nil when the BE omits the field (no warnings)
}
```

- [ ] **Step 1: Write `internal/snapshots/mutate_client_test.go`.**

Test cases (mirror trades/mutate_client_test.go structure):
  - `CreateSnapshot` 201 with a snapshot body → parses correctly.
  - `CreateSnapshot` 422 with `FUTURE_DATED_SNAPSHOT` code → `BackendValidationError` with that code and message.
  - `CreateSnapshot` 422 with `CONFLICTING_SNAPSHOT_VALUE` → same pattern.
  - `CreateSnapshot` 401 → `ErrUnauthorized`.
  - `AutoSnapshot` 201 with `{snapshot:{...}, warnings:[{...}]}` → returns both; sends `{"notes":""}` or `{}` as body.
  - `AutoSnapshot` 201 without `warnings` → returns `nil` warnings (not empty slice).
  - `AutoSnapshot` 422 `NO_PRICE_PROVIDERS_CONFIGURED` → `BackendValidationError`.
  - `AutoSnapshot` 502 `ALL_PROVIDERS_FAILED` — since this is technically `5xx`, BE still returns a body with a code. The middleend should surface it as a `BackendValidationError` (the handler will decide how to render it — snackbar, not inline). **Design note:** the spec says `ALL_PROVIDERS_FAILED` is 502 from the BE's perspective; the middleend client must treat it as a structured validation-like error, not a generic `ErrBackend`. Add a 502-path in the parse logic that extracts the code. If no code present → plain `ErrBackend`.
  - `UpdateSnapshot` 200 happy path.
  - `UpdateSnapshot` 404 → `ErrSnapshotNotFound`.
  - `UpdateSnapshot` 422 → `BackendValidationError`.
  - `DeleteSnapshot` 204 → nil.
  - `DeleteSnapshot` 404 → `ErrSnapshotNotFound`.

- [ ] **Step 2: Run tests, verify failures.**

Run: `go test ./internal/snapshots/ -run TestMutate -v`

- [ ] **Step 3: Implement `internal/snapshots/mutate_client.go`.**

Structure:

```go
package snapshots

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// BackendValidationError carries a 4xx/5xx-with-code validation error from the
// backend. Code is e.g. "FUTURE_DATED_SNAPSHOT"; Message is the BE's localized
// human-readable message.
type BackendValidationError struct {
	Code    string
	Message string
}

func (e *BackendValidationError) Error() string {
	return fmt.Sprintf("backend validation: %s: %s", e.Code, e.Message)
}

type backendErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// AutoWarning identifies an asset whose provider price fetch failed during an
// auto-snapshot.
type AutoWarning struct {
	AssetID string `json:"asset_id"`
	Ticker  string `json:"ticker"`
	Error   string `json:"error"`
}

// AutoResult is the parsed POST /v1/snapshots/auto response.
type AutoResult struct {
	Snapshot Snapshot
	Warnings []AutoWarning
}

type rawAutoResult struct {
	Snapshot rawSnapshot   `json:"snapshot"`
	Warnings []AutoWarning `json:"warnings"`
}

// CreateSnapshot posts the given body to /v1/snapshots.
func (c *Client) CreateSnapshot(ctx context.Context, authorization string, body map[string]any) (*Snapshot, error) {
	return c.doSnapshotWithBody(ctx, authorization, http.MethodPost, "/v1/snapshots", body, http.StatusCreated)
}

// UpdateSnapshot patches an existing snapshot.
func (c *Client) UpdateSnapshot(ctx context.Context, authorization, id string, body map[string]any) (*Snapshot, error) {
	return c.doSnapshotWithBody(ctx, authorization, http.MethodPatch, "/v1/snapshots/"+id, body, http.StatusOK)
}

// DeleteSnapshot deletes a snapshot by id.
func (c *Client) DeleteSnapshot(ctx context.Context, authorization, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/v1/snapshots/"+id, nil)
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
	raw, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrSnapshotNotFound
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict:
		return parseValidationError(raw)
	default:
		return fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// AutoSnapshot triggers POST /v1/snapshots/auto. notes is sent verbatim (empty
// string is fine — the BE defaults it). Returns the created Snapshot plus any
// per-asset provider warnings. Terminal failures (NO_PRICE_PROVIDERS_CONFIGURED,
// ALL_PROVIDERS_FAILED) come back as *BackendValidationError when the BE body
// includes a code.
func (c *Client) AutoSnapshot(ctx context.Context, authorization, notes string) (*AutoResult, error) {
	body := map[string]any{}
	if notes != "" {
		body["notes"] = notes
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/snapshots/auto", bytes.NewReader(buf))
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
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case http.StatusCreated:
		var r rawAutoResult
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &AutoResult{Snapshot: r.Snapshot.toDomain(), Warnings: r.Warnings}, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict,
		http.StatusBadGateway, http.StatusInternalServerError:
		// BE may emit a structured error for terminal provider failures.
		if ve := parseValidationError(raw); !errors.Is(ve, ErrBackend) {
			return nil, ve
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func (c *Client) doSnapshotWithBody(ctx context.Context, authorization, method, path string, body map[string]any, successStatus int) (*Snapshot, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(buf))
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
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case successStatus:
		return ParseSnapshot(raw)
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrSnapshotNotFound
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict:
		return nil, parseValidationError(raw)
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func parseValidationError(body []byte) error {
	var b backendErrorBody
	if err := json.Unmarshal(body, &b); err != nil || b.Error.Code == "" {
		return fmt.Errorf("%w: status 4xx", ErrBackend)
	}
	return &BackendValidationError{Code: b.Error.Code, Message: b.Error.Message}
}
```

- [ ] **Step 4: Run tests, verify PASS.**

Run: `go test ./internal/snapshots/ -run TestMutate -v`

- [ ] **Step 5: Commit.**

```bash
git add internal/snapshots/mutate_client.go internal/snapshots/mutate_client_test.go
git commit -m "feat(snapshots): add create/auto/update/delete client with validation errors"
```

---

## Phase 5 — Use case (list)

### Task 5.1: `get_usecase.go`

**Files:**
- Create: `internal/snapshots/get_usecase.go`
- Create: `internal/snapshots/get_usecase_test.go`

Mirror `internal/trades/get_usecase.go`. Interfaces:

```go
type snapshotFetcher interface {
	List(ctx context.Context, authorization string, p ListParams) (*ListResult, error)
}

type catalogFetcher interface {
	List(ctx context.Context, authorization string) ([]assetscatalog.Asset, error)
}
```

Two methods: `Execute` (full screen) and `ExecuteSection` (list region only).

- [ ] **Step 1: Write `internal/snapshots/get_usecase_test.go`** using fakes for both interfaces. Cover: success path returns a `components.Component`, snapshot-list error propagates verbatim, catalog error propagates verbatim, snapshot-list error short-circuits catalog call.

- [ ] **Step 2: Run tests, verify failures.**

- [ ] **Step 3: Implement `internal/snapshots/get_usecase.go`.**

```go
package snapshots

import (
	"context"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

type snapshotFetcher interface {
	List(ctx context.Context, authorization string, p ListParams) (*ListResult, error)
}

type catalogFetcher interface {
	List(ctx context.Context, authorization string) ([]assetscatalog.Asset, error)
}

type GetUseCase struct {
	client  snapshotFetcher
	catalog catalogFetcher
}

func NewGetUseCase(client snapshotFetcher, catalog catalogFetcher) *GetUseCase {
	return &GetUseCase{client: client, catalog: catalog}
}

func (uc *GetUseCase) Execute(ctx context.Context, authorization string, p ListParams, lang string) (components.Component, error) {
	res, err := uc.client.List(ctx, authorization, p)
	if err != nil {
		return components.Component{}, err
	}
	cat, err := uc.catalog.List(ctx, authorization)
	if err != nil {
		return components.Component{}, err
	}
	return BuildScreen(res, cat, p, lang), nil
}

func (uc *GetUseCase) ExecuteSection(ctx context.Context, authorization string, p ListParams, lang string) (components.Component, error) {
	res, err := uc.client.List(ctx, authorization, p)
	if err != nil {
		return components.Component{}, err
	}
	cat, err := uc.catalog.List(ctx, authorization)
	if err != nil {
		return components.Component{}, err
	}
	return BuildSnapshotsSection(res, cat, p, lang), nil
}
```

- [ ] **Step 4: Run tests.** (They will fail on `BuildScreen`/`BuildSnapshotsSection` being undefined — that's expected; the tests use fakes but the symbols still must exist. Defer running these tests to Phase 6 after `builder.go` lands, or stub the builders now. Pick one approach:

  **Approach A (preferred):** Add stubs now in a new `builder.go`:

  ```go
  package snapshots

  import (
      "github.com/project/vk-investment-middleend/internal/components"
      "github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
  )

  // Stub — filled in in Phase 6.
  func BuildScreen(_ *ListResult, _ []assetscatalog.Asset, _ ListParams, _ string) components.Component {
      return components.Component{Type: "screen", ID: "snapshots-screen"}
  }

  // Stub — filled in in Phase 6.
  func BuildSnapshotsSection(_ *ListResult, _ []assetscatalog.Asset, _ ListParams, _ string) components.Component {
      return components.Component{Type: "column", ID: "snapshots-section"}
  }
  ```

  then `go test ./internal/snapshots/ -v` passes.

- [ ] **Step 5: Commit.**

```bash
git add internal/snapshots/get_usecase.go internal/snapshots/get_usecase_test.go internal/snapshots/builder.go
git commit -m "feat(snapshots): add GetUseCase (list + section builders stubbed)"
```

---

## Phase 6 — List builder

### Task 6.1: `builder.go` — full screen + section + header + filter + table + pagination + expanded entries panel

**Files:**
- Modify: `internal/snapshots/builder.go`
- Create: `internal/snapshots/builder_test.go`

Mirror `internal/trades/builder.go`. New-to-this-screen pieces:
- Header with **two** buttons (`New Snapshot` + `Auto Snapshot`), not one.
- Filter: a single `is_full_snapshot` select.
- Table main row: 6 cells (Date, Type, Entries, Sources, Notes, Actions) + `expandable: true` + `details` = a nested entries `table`.
- Entries panel: 5-column table (Asset, Quantity, Price, Value Override, Source).

Expose ID constants at the top (mirror trades):

```go
const (
	ScreenID    = "snapshots-screen"
	SectionID   = "snapshots-section"
	ModalSlotID = "snapshots-modal-slot"
)
```

- [ ] **Step 1: Write `internal/snapshots/builder_test.go`.**

Table-driven tests covering:
  - `BuildScreen` returns a `screen` with id `ScreenID` and a child tree that includes the header + section + modal slot.
  - Header has buttons for New Snapshot and Auto Snapshot with correct endpoints.
  - Empty state (no filter): uses `snapshots.empty_title` / `snapshots.empty_subtitle`.
  - Empty state (filter active): uses `snapshots.empty_filtered_title` / `snapshots.empty_filtered_subtitle`.
  - List with 2 snapshots: table has 2 rows, each `expandable: true` with a `details` subtree containing a nested entries table.
  - Entries panel rendering: quantity `—` when empty, price/override via `format.FormatMoney`, source badge for each entry.
  - Full/Partial badge.
  - Sources compact display (up to 3, then `+N`).
  - Notes truncation at 40 chars.
  - Pagination: omitted when `total <= size`; present when `total > size`; Prev disabled at offset=0; Next disabled when `offset+size >= total`.
  - Each pagination button URL carries the filter + target offset.

- [ ] **Step 2: Run tests, verify failures (stubs don't have the right shape).**

- [ ] **Step 3: Implement `builder.go`** (replace the stub). Template, copied and adapted from `internal/trades/builder.go`:

Key implementation notes (not code — use the reference file to fill in):
  - Header via `buildHeader(lang)` returning a `row` with title + spacer + New Snapshot button + Auto Snapshot button. Buttons carry `navigate`/`refresh` actions pointing at `/actions/snapshots/create_wizard` and `/actions/snapshots/auto` respectively. (For the auto button, use a `submit` action since it's a POST with empty body; the wizard pathway is a GET so it's a `refresh` targeting `snapshots-modal-slot` with `endpoint=/actions/snapshots/create_wizard?...`.)
  - Filter via `buildFilter(p, lang)` — single Select, options `Any` / `Full` / `Partial`; value is the stringified `IsFullSnapshot` (empty / `"true"` / `"false"`). On-change action: `refresh` targeting `SectionID` with endpoint `/actions/snapshots/list?is_full_snapshot=<v>&offset=0`.
  - Table via `buildTable(snapshots, catalog, p, lang)`:
    - Columns: `date`, `type`, `entries`, `sources`, `notes`, `actions`.
    - Each row: `TableRowExpandable(rowID, cells, detailsTable)`.
    - `detailsTable` = `components.Table(rowID+"-entries", entryCols, entryRows...)`.
    - Entry row cells: use `format.FormatQuantity`, `format.FormatMoney` with `asset.Currency` (look up via `byID := indexCatalog(catalog)`).
  - Pagination via `buildPagination(res, p, lang)` — mirror trades, but URL params are `is_full_snapshot` + `offset`.
  - Helper `buildIsFullSnapshotQuery(p ListParams) url.Values`.

- [ ] **Step 4: Run tests.** Iterate until PASS.

Run: `go test ./internal/snapshots/ -v`

- [ ] **Step 5: Commit.**

```bash
git add internal/snapshots/builder.go internal/snapshots/builder_test.go
git commit -m "feat(snapshots): build screen, list section, filter, table with expandable rows"
```

---

## Phase 7 — Wizard builder + delete modal builder

### Task 7.1: `wizard_builder.go` — BuildCreateWizard

**Files:**
- Create: `internal/snapshots/wizard_builder.go`
- Create: `internal/snapshots/wizard_builder_test.go`

Builds a `wizard` component with:
- Step `info`: recorded_at datetime-local input (required, `max=now`), notes textarea (optional, max 500).
- Steps `entry` — one per asset in the catalog:
  - Header row: ticker (bold) + name + type badge.
  - If complex: only `current_value_override` input.
  - Else: segmented toggle (`mode` = `price` / `override`) + the input bound to the selected mode (via `visible_when` on two inputs, one per mode).
  - `name` attributes on inputs follow the bracket convention: `entries[<asset_id>].mode`, `entries[<asset_id>].current_price`, `entries[<asset_id>].current_value_override`.
  - `skippable: true`, `include_default: false`.
- Step `summary`: a short descriptive text. (The frontend derives the entry list from the include map client-side.)

Constants:

```go
const (
	WizardID = "snapshots-wizard"
)
```

Signature:

```go
func BuildCreateWizard(catalog []assetscatalog.Asset, p ListParams, lang, inlineError, initialStepID string) components.Component
```

`inlineError != ""` adds a banner `variant: "error"` (uses `*components.WizardBanner`); otherwise no banner. `initialStepID` is passed straight to the wizard constructor — empty string means "first step" (the wizard default). Both are set by the `create_handler` when re-emitting the wizard on a BE validation error; handler and tests must account for them.

- [ ] **Step 1: Write `internal/snapshots/wizard_builder_test.go`** covering:
  - Wizard type + id + mode=create.
  - First step is `kind=info` with required `recorded_at` and optional `notes` inputs.
  - For a catalog with one complex asset + one non-complex asset: two entry steps, correctly typed (complex step has no toggle, non-complex has the toggle + both inputs with `visible_when`).
  - Last step is `kind=summary`.
  - `submit_action.endpoint` = `/actions/snapshots/create?is_full_snapshot=...&offset=...`.
  - `dismiss_action` targets `ModalSlotID`.
  - `inlineError="..."` → banner variant=error with that message.
  - `inlineError=""` → no banner prop present.

- [ ] **Step 2: Run, verify failures.**

- [ ] **Step 3: Implement `wizard_builder.go`.**

Key detail (segmented toggle): SDUI currently has no `segmented_toggle` primitive — use a `radio_group` (or a `select` with two options) with `name="entries[<id>].mode"` and options `{price, override}`. Then the two inputs have `visible_when: {control: "entries[<id>].mode", value: "price"|"override"}`. This reuses existing form primitives (see `spec/sdui-base-components.md` §602).

- [ ] **Step 4: Run tests, verify PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/snapshots/wizard_builder.go internal/snapshots/wizard_builder_test.go
git commit -m "feat(snapshots): build create wizard with per-asset entry steps"
```

### Task 7.2: `wizard_builder.go` — BuildEditWizard

**Files:**
- Modify: `internal/snapshots/wizard_builder.go`
- Modify: `internal/snapshots/wizard_builder_test.go`

Signature:

```go
func BuildEditWizard(s *Snapshot, catalog []assetscatalog.Asset, p ListParams, lang, inlineError, initialStepID string, banner *components.WizardBanner) components.Component
```

`initialStepID` matches the create wizard's semantics — `""` means "first step"; the update handler sets it on validation-error re-emit.

Differences from create:
- `mode: edit`.
- Step `info`: `recorded_at` rendered as **static `text`** (not an input) with a small "read-only" label; `notes` textarea pre-filled with `s.Notes`.
- Entry steps: index `s.Entries` by `asset_id`. For each asset in catalog:
  - If entry exists: `skippable: false`, `include_default: true`; pre-fill input with `CurrentPrice` or `CurrentValueOverride` and init the mode toggle. Attach a small `text` element with `snapshots.wizard.already_included` copy.
  - Else: same as create (`skippable: true`, `include_default: false`).
- `submit_action.endpoint` = `/actions/snapshots/:id?is_full_snapshot=...&offset=...` — PATCH method handled at the handler level via the wizard's `Submit` action type.
- `banner` parameter used by the auto-snapshot flow to attach the "created automatically" info banner + optional warnings.

- [ ] **Step 1: Extend the test file with edit-mode cases.**

Cover:
  - Edit mode title and mode=edit.
  - `recorded_at` rendered as `text`, not `input`.
  - `notes` pre-populated.
  - Asset-in-snapshot step has `skippable: false`, `include_default: true`, pre-filled value, and an `already_included` text child.
  - Asset-not-in-snapshot step behaves like create.
  - Banner passed through correctly when non-nil.
  - `submit_action.endpoint` uses the snapshot id.

- [ ] **Step 2: Run tests, verify failures.**

- [ ] **Step 3: Implement `BuildEditWizard`.** Factor shared code with `BuildCreateWizard` into helpers (`buildInfoStep`, `buildEntryStep`) so the wizard step construction stays DRY.

- [ ] **Step 4: Run, verify PASS.**

- [ ] **Step 5: Commit.**

```bash
git add internal/snapshots/wizard_builder.go internal/snapshots/wizard_builder_test.go
git commit -m "feat(snapshots): build edit wizard with pre-filled entries and banner support"
```

### Task 7.3: Delete modal

**Files:**
- Create: `internal/snapshots/modal_builder.go`
- Create: `internal/snapshots/modal_builder_test.go`

Simple confirmation modal. Signature:

```go
func BuildDeleteModal(s *Snapshot, p ListParams, lang string) components.Component
```

Title = `snapshots.delete.title`; body text = interpolated `snapshots.delete.confirm` with the `recorded_at` formatted as `YYYY-MM-DD HH:mm`. Cancel (`dismiss`, replaces `ModalSlotID` with empty column) + destructive Delete (`submit` → `/actions/snapshots/:id?is_full_snapshot=<f>&offset=<n>`, method DELETE).

Constant:

```go
const DeleteModalID = "snapshots-delete-modal"
```

- [ ] **Step 1: Write tests — structure (modal title), interpolated body, buttons' endpoints.**
- [ ] **Step 2: Run, verify failures.**
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run, verify PASS.**
- [ ] **Step 5: Commit.**

```bash
git add internal/snapshots/modal_builder.go internal/snapshots/modal_builder_test.go
git commit -m "feat(snapshots): build delete confirmation modal"
```

---

## Phase 8 — Screen + list handlers

### Task 8.1: `handler.go` (main `GET /screens/snapshots`)

**Files:**
- Create: `internal/snapshots/handler.go`
- Create: `internal/snapshots/handler_test.go`

Mirror `internal/trades/handler.go`. Exports: `Handler`, `NewHandler(uc *GetUseCase)`, `(*Handler).Get(c)`.

Query parsing helper `parseListParams`:
- `is_full_snapshot` must be absent, `"true"`, or `"false"`.
- `offset` must be a non-negative integer.

Auth handling: propagate `ErrUnauthorized` and `assetscatalog.ErrUnauthorized` → 401 with `/login` redirect via `shared.RespondUnauthorized`.

Bad query → 400 `BAD_REQUEST`. BE error → 502 `BACKEND_ERROR`.

- [ ] **Step 1: Write `handler_test.go`.**

Use `httptest` + `gin.New()` + install the handler. Cover: happy path (full tree), missing auth → 401, bad `is_full_snapshot` → 400, bad `offset` → 400, BE failure → 502.

- [ ] **Step 2..5: Red / Green / Commit.**

```bash
git add internal/snapshots/handler.go internal/snapshots/handler_test.go
git commit -m "feat(snapshots): add GET /screens/snapshots handler"
```

### Task 8.2: `list_handler.go`

**Files:**
- Create: `internal/snapshots/list_handler.go`
- Create: `internal/snapshots/list_handler_test.go`

Mirror `internal/trades/list_handler.go`. Returns `ActionResponse{action:"replace", target_id: SectionID, tree: <section>}`.

- [ ] **Step 1..5: Red / Green / Commit.**

```bash
git add internal/snapshots/list_handler.go internal/snapshots/list_handler_test.go
git commit -m "feat(snapshots): add GET /actions/snapshots/list handler"
```

---

## Phase 9 — Wizard GET handlers + delete modal GET handler

### Task 9.1: `create_wizard_handler.go`

**Files:**
- Create: `internal/snapshots/create_wizard_handler.go`
- Create: `internal/snapshots/create_wizard_handler_test.go`

Mirror `internal/trades/create_modal_handler.go`. Depends on `catalogFetcher`. Returns `ActionResponse{replace, target_id: ModalSlotID, tree: <BuildCreateWizard(...)>}`.

- [ ] **Step 1..5: Red / Green / Commit.**

```bash
git add internal/snapshots/create_wizard_handler.go internal/snapshots/create_wizard_handler_test.go
git commit -m "feat(snapshots): add GET /actions/snapshots/create_wizard handler"
```

### Task 9.2: `edit_wizard_handler.go`

**Files:**
- Create: `internal/snapshots/edit_wizard_handler.go`
- Create: `internal/snapshots/edit_wizard_handler_test.go`

Mirror `internal/trades/edit_modal_handler.go`. Depends on a fetcher interface for `GetSnapshot` and on `catalogFetcher`. Returns `ActionResponse{replace, target_id: ModalSlotID, tree: <BuildEditWizard(...)>}`. On `ErrSnapshotNotFound` → 404. On other errors same as list handler.

This handler does **not** take a banner param (banner is only added by the auto-snapshot flow). Pass `banner=nil`.

- [ ] **Step 1..5: Red / Green / Commit.**

```bash
git add internal/snapshots/edit_wizard_handler.go internal/snapshots/edit_wizard_handler_test.go
git commit -m "feat(snapshots): add GET /actions/snapshots/edit_wizard handler"
```

### Task 9.3: `delete_modal_handler.go`

**Files:**
- Create: `internal/snapshots/delete_modal_handler.go`
- Create: `internal/snapshots/delete_modal_handler_test.go`

Mirror `internal/trades/delete_modal_handler.go`. Fetches the snapshot (for the date), returns `BuildDeleteModal`.

- [ ] **Step 1..5: Red / Green / Commit.**

```bash
git add internal/snapshots/delete_modal_handler.go internal/snapshots/delete_modal_handler_test.go
git commit -m "feat(snapshots): add GET /actions/snapshots/delete_modal handler"
```

---

## Phase 10 — Mutation handlers

### Task 10.1: `request.go` — parse wizard-style form body

**Files:**
- Create: `internal/snapshots/request.go`
- Create: `internal/snapshots/request_test.go`

The wizard submits a flat JSON body with bracket-notation keys. Example:

```json
{
  "recorded_at": "2024-01-10T10:00:00Z",
  "notes": "hi",
  "entries[a1].mode": "price",
  "entries[a1].current_price": "150.00",
  "entries[a2].mode": "override",
  "entries[a2].current_value_override": "2000.00"
}
```

We need two utilities:

```go
// parseJSONBody reuses the trades pattern.
func parseJSONBody(c *gin.Context) (map[string]any, error)

// parseWizardEntries extracts the flat "entries[<id>].<field>" keys into a map
// keyed by asset_id, preserving the per-asset mode/value-or-override choice.
// Entries whose asset_id appears but has neither current_price nor
// current_value_override are excluded.
func parseWizardEntries(body map[string]any) []wizardEntry

type wizardEntry struct {
	AssetID              string
	Mode                 string // "price" / "override" / ""
	CurrentPrice         string
	CurrentValueOverride string
}
```

`parseWizardEntries` is the part that needs tests. Corner cases:
- No entries at all → empty slice.
- One complex asset (no `mode`, just `current_value_override`) → entry with `Mode=""`, only the override field set.
- One non-complex asset with `mode=price` + `current_price` → entry with `Mode="price"`, only the price field set.
- Mix of both.
- Malformed key (e.g. `entries[].mode`) → silently dropped (no panic).
- Same asset_id appears with multiple fields (mode + current_price + current_value_override) — produces one entry with all three.

- [ ] **Step 1: Write `request_test.go`.**
- [ ] **Step 2: Run, verify failures.**
- [ ] **Step 3: Implement `request.go`.**

Regex for key parsing: `^entries\[([^\]]+)\]\.(\w+)$`. Copy `asString` / `parseJSONBody` from the trades `request.go` (they're identical).

- [ ] **Step 4: Run, verify PASS.**
- [ ] **Step 5: Commit.**

```bash
git add internal/snapshots/request.go internal/snapshots/request_test.go
git commit -m "feat(snapshots): parse wizard entries[uuid].field body shape"
```

### Task 10.2: `create_handler.go`

**Files:**
- Create: `internal/snapshots/create_handler.go`
- Create: `internal/snapshots/create_handler_test.go`

Mirror `internal/trades/create_handler.go`. Steps:
1. Parse `is_full_snapshot` + `offset` from query (for filter/offset preservation).
2. Parse body → `wizardEntry[]`.
3. Build BE body `{recorded_at, notes?, entries: [...]}`:
   - For each wizard entry: include `asset_id`; include `current_price` when `Mode="price"` and the value is non-empty; include `current_value_override` when `Mode="override"` or when `Mode=""` and override is non-empty (complex).
   - Entries where both fields are empty are **dropped** (they represent skipped steps).
4. `CreateSnapshot` call.
5. On `BackendValidationError`: re-fetch catalog, call `BuildCreateWizard(catalog, params, lang, be.Message, initialStepID)` where `initialStepID = "info"` for `FUTURE_DATED_SNAPSHOT` and `"summary"` for all other codes. Return `ActionResponse{replace, ModalSlotID, <wizard>}`.
6. On success: call `uc.Execute(...)` for the refreshed tree, attach `Feedback` snackbar with `snapshots.create.success`, return `ActionResponse{replace, ScreenID, tree, feedback}`.

- [ ] **Step 1..5: Red / Green / Commit.**

```bash
git add internal/snapshots/create_handler.go internal/snapshots/create_handler_test.go
git commit -m "feat(snapshots): add POST /actions/snapshots/create handler"
```

### Task 10.3: `update_handler.go`

**Files:**
- Create: `internal/snapshots/update_handler.go`
- Create: `internal/snapshots/update_handler_test.go`

Mirror `internal/trades/update_handler.go`. Core logic:
1. Parse params + body (same as create).
2. **Re-fetch the snapshot** (`GetSnapshot(id)`) to compare against submitted values. 404 → 404 response.
3. Diff:
   - `notes`: include only if submitted `notes != original.Notes`.
   - `entries`: include only entries where (a) `asset_id` not in original, or (b) values changed.
   - If both `notes` and `entries` end up empty → treat as no-op success (still call `uc.Execute` to refresh the screen).
4. `UpdateSnapshot(id, body)`.
5. Same error/success handling as create (use `BuildEditWizard(originalSnapshot, ...)` for inline-error replay).

- [ ] **Step 1..5: Red / Green / Commit.**

```bash
git add internal/snapshots/update_handler.go internal/snapshots/update_handler_test.go
git commit -m "feat(snapshots): add PATCH /actions/snapshots/:id handler with diff"
```

### Task 10.4: `delete_handler.go`

**Files:**
- Create: `internal/snapshots/delete_handler.go`
- Create: `internal/snapshots/delete_handler_test.go`

Mirror `internal/trades/delete_handler.go`. No force flag.

- [ ] **Step 1..5: Red / Green / Commit.**

```bash
git add internal/snapshots/delete_handler.go internal/snapshots/delete_handler_test.go
git commit -m "feat(snapshots): add DELETE /actions/snapshots/:id handler"
```

### Task 10.5: `auto_handler.go`

**Files:**
- Create: `internal/snapshots/auto_handler.go`
- Create: `internal/snapshots/auto_handler_test.go`

New-to-this-screen flow. Signature:

```go
type AutoHandler struct {
	client  interface {
		AutoSnapshot(ctx context.Context, authorization, notes string) (*AutoResult, error)
	}
	uc      *GetUseCase
	catalog catalogFetcher
}
```

Behavior:
1. Parse `is_full_snapshot` + `offset` query.
2. Call `AutoSnapshot(ctx, auth, "")`.
3. On success (non-nil `AutoResult`):
   - Build the refreshed list tree (`uc.Execute(...)`).
   - Build the edit wizard on the new snapshot, with a banner:
     - If warnings: two banners — `info` (`snapshots.auto.banner`) and `warning` (`snapshots.auto.warnings_title` + a bulleted list of warning tickers). Since the wizard has a single `banner` prop, concatenate them into one banner with `variant: info` and a `message` that includes the warnings list, OR extend the wizard's `banner` prop to `banners: []Banner`. **Decision:** keep `banner` as a single banner; when warnings exist, the `message` is extended client-side — but that's awkward. Simpler: render the warnings as an inline note at the top of the wizard tree by adding a `text` element as the first child of the `info` step's children. This way the single `banner` stays as the "created automatically" info banner, and the warnings are inline content.
   - Return an `ActionResponse` that does two replaces. Since `ActionResponse` today does one replace + optional feedback (see `internal/components/actions.go`), add a **multi-replace** variant or adapt: simplest path is to replace the screen root (which includes both the list and the empty modal slot) with a tree that **already contains the edit wizard in the modal slot**. Build a custom screen tree: `BuildScreen(...)` but with `modalSlot = BuildEditWizard(...)` instead of empty `column`. Add a helper `BuildScreenWithModal` that accepts the modal subtree.
4. On `BackendValidationError` with code `NO_PRICE_PROVIDERS_CONFIGURED`: return `ActionResponse{feedback: Snackbar(snapshots.auto.no_providers, "warning")}`, no replace.
5. On `BackendValidationError` with code `ALL_PROVIDERS_FAILED`: same pattern, `snapshots.auto.all_failed`.
6. On `ErrUnauthorized`: 401 + redirect.
7. On other BE errors: 502 `BACKEND_ERROR`.

Add `BuildScreenWithModal` to `builder.go` (extract the modal-slot injection from `BuildScreen`, which becomes `BuildScreen(res, cat, p, lang) = BuildScreenWithModal(res, cat, p, lang, components.Column(ModalSlotID))`).

- [ ] **Step 1: Write `auto_handler_test.go`.**

Cover:
  - Happy path: handler calls `AutoSnapshot`, returns `ActionResponse{replace, ScreenID, tree}` whose tree contains the edit wizard in the modal slot.
  - With warnings: the edit wizard's info step contains an inline warnings notice.
  - `NO_PRICE_PROVIDERS_CONFIGURED` → snackbar, no replace.
  - `ALL_PROVIDERS_FAILED` → snackbar, no replace.
  - `ErrUnauthorized` → 401 redirect.
  - Generic BE error → 502.

- [ ] **Step 2..5: Red / Green / Commit.**

```bash
git add internal/snapshots/auto_handler.go internal/snapshots/auto_handler_test.go internal/snapshots/builder.go
git commit -m "feat(snapshots): add POST /actions/snapshots/auto handler with pre-filled edit wizard"
```

---

## Phase 11 — i18n

### Task 11.1: Add `snapshots.*` keys

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/es.json`

Use the key list in `spec/screens/snapshots.md` §i18n keys (and in the design doc §11). Add a single top-level `"snapshots"` object mirroring the structure used for `trades`.

Sample EN values (use your judgment, keep consistent with trades tone):

```
"snapshots": {
  "title": "Snapshots",
  "new": "New Snapshot",
  "auto": "Auto Snapshot",
  "empty_title": "No snapshots yet",
  "empty_subtitle": "Record your first portfolio snapshot from the header.",
  "empty_filtered_title": "No snapshots match this filter",
  "empty_filtered_subtitle": "Clear the filter to see all snapshots.",
  "filter": {
    "type": "Type",
    "type_any": "Any",
    "type_full": "Full",
    "type_partial": "Partial"
  },
  "col": {
    "date": "Date",
    "type": "Type",
    "entries": "Entries",
    "sources": "Sources",
    "notes": "Notes"
  },
  "entries": {
    "col": {
      "asset": "Asset",
      "quantity": "Quantity",
      "price": "Price",
      "value_override": "Value Override",
      "source": "Source"
    }
  },
  "type": {
    "full": "Full",
    "partial": "Partial"
  },
  "source": {
    "manual": "MANUAL",
    "coingecko": "COINGECKO",
    "twelve_data": "TWELVE_DATA",
    "alpha_vantage": "ALPHA_VANTAGE"
  },
  "pagination": {
    "prev": "Previous",
    "next": "Next",
    "page_of": "Page {current} of {total}"
  },
  "wizard": {
    "info": "Info",
    "summary": "Summary",
    "step_of": "Step {current} of {total}",
    "back": "Back",
    "next": "Next",
    "skip": "Skip",
    "include": "Include",
    "update": "Update",
    "already_included": "Already in snapshot, cannot be removed"
  },
  "create": {
    "title": "New Snapshot",
    "submit": "Record snapshot",
    "success": "Snapshot recorded"
  },
  "edit": {
    "title": "Edit snapshot from {date}",
    "submit": "Save changes",
    "success": "Snapshot updated"
  },
  "delete": {
    "title": "Delete snapshot",
    "confirm": "Delete snapshot from {date}? This will affect portfolio calculations.",
    "submit": "Delete",
    "success": "Snapshot deleted"
  },
  "auto": {
    "success": "Auto snapshot created",
    "banner": "Snapshot created automatically. Adjust entries or close to keep as-is.",
    "warnings_title": "Some assets could not be updated",
    "no_providers": "No assets have a price provider configured. Set one up on the Assets screen first.",
    "all_failed": "All provider calls failed. Try again in a moment."
  },
  "form": {
    "recorded_at": "Recorded at",
    "recorded_at_readonly": "Recorded at (read-only)",
    "notes": "Notes",
    "notes_placeholder": "Optional notes",
    "toggle_price": "Price",
    "toggle_override": "Value Override",
    "current_price": "Price",
    "current_value_override": "Value override"
  }
}
```

For `es.json`, translate. Use the trades/assets Spanish copy as tone reference.

- [ ] **Step 1: Edit `locales/en.json`** and add the `"snapshots"` block. Preserve JSON validity.

- [ ] **Step 2: Edit `locales/es.json`** and add the same shape in Spanish.

- [ ] **Step 3: Sanity check.**

Run: `python3 -m json.tool locales/en.json > /dev/null && python3 -m json.tool locales/es.json > /dev/null`
Expected: no output, no errors.

- [ ] **Step 4: Commit.**

```bash
git add locales/en.json locales/es.json
git commit -m "feat(snapshots): add en/es i18n keys"
```

### Task 11.2: Rendered-key smoke test

**Files:**
- Create: `internal/snapshots/i18n_keys_test.go`

Mirror `internal/trades/i18n_keys_test.go`. A single test that builds a representative tree (BuildScreen with 1 snapshot; BuildCreateWizard; BuildEditWizard on a snapshot; BuildDeleteModal) and walks the result, collecting all string props, then asserts no string equals a known i18n key pattern (e.g. any string that starts with `snapshots.` is an unrendered key — a bug).

- [ ] **Step 1: Write the test.**
- [ ] **Step 2: Run, expect pass if all keys were looked up via `i18n.T(lang, key)`; if any fail, fix the builder.**
- [ ] **Step 3: Commit.**

```bash
git add internal/snapshots/i18n_keys_test.go
git commit -m "test(snapshots): assert all i18n keys are rendered in the tree"
```

---

## Phase 12 — Server wiring + smoke test

### Task 12.1: Wire routes in `server.go`

**Files:**
- Modify: `internal/server/server.go`

After the `// --- trades ---` block (line 76 area), add:

```go
// --- snapshots ---
snapshotsClient := snapshots.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
snapshotsUC := snapshots.NewGetUseCase(snapshotsClient, catalog)
protected.GET("/screens/snapshots", snapshots.NewHandler(snapshotsUC).Get)
protected.GET("/actions/snapshots/list", snapshots.NewListHandler(snapshotsUC).Get)
protected.GET("/actions/snapshots/create_wizard", snapshots.NewCreateWizardHandler(catalog).Get)
protected.GET("/actions/snapshots/edit_wizard", snapshots.NewEditWizardHandler(snapshotsClient, catalog).Get)
protected.GET("/actions/snapshots/delete_modal", snapshots.NewDeleteModalHandler(snapshotsClient).Get)
protected.POST("/actions/snapshots/create", snapshots.NewCreateHandler(snapshotsClient, snapshotsUC, catalog).Post)
protected.POST("/actions/snapshots/auto", snapshots.NewAutoHandler(snapshotsClient, snapshotsUC, catalog).Post)
protected.PATCH("/actions/snapshots/:id", snapshots.NewUpdateHandler(snapshotsClient, snapshotsUC, catalog).Patch)
protected.DELETE("/actions/snapshots/:id", snapshots.NewDeleteHandler(snapshotsClient, snapshotsUC).Delete)
```

Add the import at the top: `"github.com/project/vk-investment-middleend/internal/snapshots"`.

- [ ] **Step 1: Edit `server.go`.**

- [ ] **Step 2: Build to verify compilation.**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Run the full test suite.**

Run: `go test ./...`
Expected: all pass.

- [ ] **Step 4: Commit.**

```bash
git add internal/server/server.go
git commit -m "feat(server): wire snapshots routes"
```

### Task 12.2: Restart and smoke-test

- [ ] **Step 1: Kill the existing middleend listener on `:8082` and restart.**

Run (in project root):
```bash
lsof -ti:8082 | xargs -r kill -9 ; ./cli run
```

- [ ] **Step 2: Hit the health endpoint.**

```bash
curl -s http://localhost:8082/health
```
Expected: `{"status":"healthy","service":"vk-investment-middleend"}`.

- [ ] **Step 3: Hit `/screens/snapshots` with a valid JWT.**

Use an existing token from the environment / an earlier `login` call. Expected: JSON tree with `"type":"screen","id":"snapshots-screen"`, a header containing New Snapshot and Auto Snapshot buttons, an empty or populated table, and an empty modal slot.

- [ ] **Step 4: Hit `/actions/snapshots/create_wizard`.** Expected: `ActionResponse` with `target_id:"snapshots-modal-slot"` and a `wizard` tree.

- [ ] **Step 5: (optional) Integration smoke on auto-snapshot.** Only if a test user has a price provider configured. Else skip.

- [ ] **Step 6: No commit (runtime verification only).**

---

## Phase 13 — Canonical spec sync

### Task 13.1: Reconcile `spec/screens/snapshots.md` with shipped code

**Files:**
- Modify (if needed): `spec/screens/snapshots.md`

During implementation the design may shift. Walk the canonical spec and verify each statement matches what the code does. Likely deltas to watch:
- Toggle implementation — the spec says "segmented toggle"; the code uses a `radio_group` or `select`. Update the spec to match.
- Button action shape on Auto Snapshot — whatever the implementation used (`submit` with empty body), describe it.
- Inline warnings rendering — single banner vs first-of-info-step text. Document the actual approach.

- [ ] **Step 1: Walk the canonical spec with the shipped code open. Note any deviations.**
- [ ] **Step 2: Edit the spec to reflect shipped behavior.**
- [ ] **Step 3: Commit.**

```bash
git add spec/screens/snapshots.md
git commit -m "docs(spec): reconcile snapshots spec with shipped code"
```

---

## Self-Review

After all phases land, walk the plan one more time:

**1. Spec coverage:** Every acceptance criterion in `spec/screens/snapshots.md` §Acceptance maps to a task above. Gaps mean an acceptance test was missed — add a task.

**2. SDUI extensions coverage:** The two net-new component contracts (§§ 9 and 10 of the design) both have a creation task (1.1, 1.2), a spec-update task (1.3), and are consumed by subsequent tasks. Verify the consumption: builder.go uses `TableRowExpandable`; wizard_builder.go uses `Wizard`.

**3. i18n coverage:** Every `snapshots.*` key named in any builder test has a row in `locales/en.json` and `locales/es.json`. The smoke test (11.2) catches unrendered keys.

**4. No-placeholder scan:** Search the plan for `TBD`, `TODO`, `implement later`, `fill in details`, `similar to Task N`. Fix any found.

**5. Route wiring:** Every handler created in Phases 8–10 appears in `server.go` (Task 12.1).

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-20-snapshots-screen.md`. Two execution options:

**1. Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — execute tasks in this session using superpowers:executing-plans, batch execution with checkpoints.

Which approach?
