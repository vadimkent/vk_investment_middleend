package snapshots

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// mixedCatalog returns one complex + one non-complex asset.
func mixedCatalog() []assetscatalog.Asset {
	return []assetscatalog.Asset{
		{ID: "complex-1", Ticker: "BTC-FUND", Name: "Bitcoin Fund", AssetType: "CRYPTO", Currency: "USD", IsComplex: true},
		{ID: "simple-2", Ticker: "AAPL", Name: "Apple Inc", AssetType: "STOCK", Currency: "USD", IsComplex: false},
	}
}

func defaultParams() ListParams {
	return ListParams{Offset: 0}
}

// wizardSteps extracts steps from a wizard component.
func wizardSteps(t *testing.T, w components.Component) []components.WizardStep {
	t.Helper()
	steps, ok := w.Props["steps"].([]components.WizardStep)
	require.True(t, ok, "expected props[steps] to be []components.WizardStep")
	return steps
}

// findChildByProp searches a step's children for one whose props[key] == value.
func findChildByProp(children []components.Component, key, value string) *components.Component {
	for i := range children {
		c := &children[i]
		if v, ok := c.Props[key].(string); ok && v == value {
			return c
		}
		// Recurse one level for nested containers.
		if found := findChildByProp(c.Children, key, value); found != nil {
			return found
		}
	}
	return nil
}

// flattenChildren returns all descendants (DFS) of the given children list.
func flattenChildren(children []components.Component) []components.Component {
	var out []components.Component
	for _, c := range children {
		out = append(out, c)
		out = append(out, flattenChildren(c.Children)...)
	}
	return out
}

// Test 1: Wizard type and id.
func TestBuildCreateWizard_TypeAndID(t *testing.T) {
	w := BuildCreateWizard(nil, defaultParams(), "en", "", "")
	assert.Equal(t, "wizard", w.Type)
	assert.Equal(t, WizardID, w.ID)
}

// Test 2: mode = "create" and title is localized.
func TestBuildCreateWizard_ModeAndTitle(t *testing.T) {
	w := BuildCreateWizard(nil, defaultParams(), "en", "", "")
	assert.Equal(t, "create", w.Props["mode"])
	// Title must be non-empty (actual i18n string "snapshots.create.title" key may render as key if not loaded).
	title, ok := w.Props["title"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, title)
}

// Test 3: submit_action endpoint contains /actions/snapshots/create? and carries list context.
func TestBuildCreateWizard_SubmitAction(t *testing.T) {
	isFull := true
	p := ListParams{IsFullSnapshot: &isFull, Offset: 20}
	w := BuildCreateWizard(nil, p, "en", "", "")

	submit, ok := w.Props["submit_action"].(components.Action)
	require.True(t, ok, "expected submit_action to be components.Action")

	assert.Contains(t, submit.Endpoint, "/actions/snapshots/create?")
	assert.Contains(t, submit.Endpoint, "is_full_snapshot=true")
	assert.Contains(t, submit.Endpoint, "offset=20")
	assert.Equal(t, "POST", submit.Method)
	assert.Equal(t, ScreenID, submit.TargetID)
}

// Test 4: dismiss_action is the client-side Dismiss action (matches assets pattern).
func TestBuildCreateWizard_DismissAction(t *testing.T) {
	w := BuildCreateWizard(nil, defaultParams(), "en", "", "")

	dismiss, ok := w.Props["dismiss_action"].(components.Action)
	require.True(t, ok, "expected dismiss_action to be components.Action")

	assert.Equal(t, "click", dismiss.Trigger)
	assert.Equal(t, "dismiss", dismiss.Type)
}

// Test 5: First step is "info" with recorded_at input and notes textarea.
func TestBuildCreateWizard_InfoStep(t *testing.T) {
	w := BuildCreateWizard(nil, defaultParams(), "en", "", "")
	steps := wizardSteps(t, w)

	require.NotEmpty(t, steps, "expected at least one step")
	info := steps[0]
	assert.Equal(t, "info", info.ID)
	assert.Equal(t, "info", info.Kind)
	assert.False(t, info.Skippable)
	assert.True(t, info.IncludeDefault)
	require.Len(t, info.Children, 2, "info step must have exactly 2 children")

	// Child 0: recorded_at input.
	ra := info.Children[0]
	assert.Equal(t, "input", ra.Type)
	assert.Equal(t, "snapshots-wizard-recorded-at", ra.ID)
	assert.Equal(t, "recorded_at", ra.Props["name"])
	assert.Equal(t, "datetime-local", ra.Props["input_type"])
	assert.Equal(t, true, ra.Props["required"])

	// Child 1: notes textarea.
	notes := info.Children[1]
	assert.Equal(t, "textarea", notes.Type)
	assert.Equal(t, "snapshots-wizard-notes", notes.ID)
	assert.Equal(t, "notes", notes.Props["name"])
	assert.Equal(t, 500, notes.Props["max_length"])
	_, reqSet := notes.Props["required"]
	assert.False(t, reqSet, "notes must not be required")
}

// Test 6: step count = 1 + len(catalog) + 1.
func TestBuildCreateWizard_StepCount(t *testing.T) {
	catalog := mixedCatalog()
	w := BuildCreateWizard(catalog, defaultParams(), "en", "", "")
	steps := wizardSteps(t, w)
	assert.Len(t, steps, 4, "expected 1 info + 2 entry + 1 summary = 4 steps")
}

// Test 7: Complex asset step has exactly one input (current_value_override), no radio_group, no visible_when.
func TestBuildCreateWizard_ComplexEntryStep(t *testing.T) {
	catalog := mixedCatalog() // complex-1 is first
	w := BuildCreateWizard(catalog, defaultParams(), "en", "", "")
	steps := wizardSteps(t, w)

	complexStep := steps[1] // first entry step
	assert.Equal(t, "entry-complex-1", complexStep.ID)
	assert.Equal(t, "BTC-FUND", complexStep.Label)
	assert.True(t, complexStep.Skippable)
	assert.False(t, complexStep.IncludeDefault)

	all := flattenChildren(complexStep.Children)

	// Count inputs.
	var inputs []components.Component
	for _, c := range all {
		if c.Type == "input" {
			inputs = append(inputs, c)
		}
	}
	require.Len(t, inputs, 1, "complex step must have exactly one input")
	assert.Equal(t, "entries[complex-1].current_value_override", inputs[0].Props["name"])

	// No radio_group.
	for _, c := range all {
		assert.NotEqual(t, "radio_group", c.Type, "complex step must not have a radio_group")
	}

	// No visible_when on any child.
	for _, c := range all {
		_, hasVW := c.Props["visible_when"]
		assert.False(t, hasVW, "complex step children must not have visible_when")
	}
}

// Test 8: Non-complex step has radio_group + two inputs with visible_when.
func TestBuildCreateWizard_NonComplexEntryStep(t *testing.T) {
	catalog := mixedCatalog() // simple-2 is second
	w := BuildCreateWizard(catalog, defaultParams(), "en", "", "")
	steps := wizardSteps(t, w)

	simpleStep := steps[2] // second entry step
	assert.Equal(t, "entry-simple-2", simpleStep.ID)
	assert.Equal(t, "AAPL", simpleStep.Label)

	all := flattenChildren(simpleStep.Children)

	// Must have a radio_group with name = "entries[simple-2].mode".
	var radios []components.Component
	for _, c := range all {
		if c.Type == "radio_group" {
			radios = append(radios, c)
		}
	}
	require.Len(t, radios, 1, "non-complex step must have exactly one radio_group")
	rg := radios[0]
	assert.Equal(t, "entries[simple-2].mode", rg.Props["name"])

	opts, ok := rg.Props["options"].([]components.SelectOption)
	require.True(t, ok)
	require.Len(t, opts, 2)
	values := []string{opts[0].Value, opts[1].Value}
	assert.Contains(t, values, "price")
	assert.Contains(t, values, "override")

	// default_value must be "price".
	assert.Equal(t, "price", rg.Props["default_value"])

	// Must have two inputs with visible_when.
	var inputsWithVW []components.Component
	for _, c := range all {
		if c.Type == "input" {
			if _, ok := c.Props["visible_when"]; ok {
				inputsWithVW = append(inputsWithVW, c)
			}
		}
	}
	require.Len(t, inputsWithVW, 2, "non-complex step must have exactly 2 inputs with visible_when")

	// Verify names and visible_when fields.
	names := make(map[string]components.VisibleWhen)
	for _, inp := range inputsWithVW {
		name := inp.Props["name"].(string)
		vw := inp.Props["visible_when"].(components.VisibleWhen)
		names[name] = vw
	}

	priceVW, hasPriceInput := names["entries[simple-2].current_price"]
	require.True(t, hasPriceInput, "expected input for current_price")
	assert.Equal(t, "entries[simple-2].mode", priceVW.Field)
	assert.Equal(t, "eq", priceVW.Op)
	assert.Equal(t, "price", priceVW.Value)

	overrideVW, hasOverrideInput := names["entries[simple-2].current_value_override"]
	require.True(t, hasOverrideInput, "expected input for current_value_override")
	assert.Equal(t, "entries[simple-2].mode", overrideVW.Field)
	assert.Equal(t, "eq", overrideVW.Op)
	assert.Equal(t, "override", overrideVW.Value)
}

// Test 9: Last step is "summary", kind "summary", skippable false, include_default true.
func TestBuildCreateWizard_SummaryStep(t *testing.T) {
	catalog := mixedCatalog()
	w := BuildCreateWizard(catalog, defaultParams(), "en", "", "")
	steps := wizardSteps(t, w)

	last := steps[len(steps)-1]
	assert.Equal(t, "summary", last.ID)
	assert.Equal(t, "summary", last.Kind)
	assert.False(t, last.Skippable)
	assert.True(t, last.IncludeDefault)
	assert.NotEmpty(t, last.Children, "summary step must have at least one child")
}

// Test 10: inlineError sets banner with variant=error and message.
func TestBuildCreateWizard_BannerSet(t *testing.T) {
	w := BuildCreateWizard(nil, defaultParams(), "en", "boom", "")

	bannerRaw, ok := w.Props["banner"]
	require.True(t, ok, "expected banner prop to be set")
	banner, ok := bannerRaw.(*components.WizardBanner)
	require.True(t, ok, "expected banner to be *components.WizardBanner")
	assert.Equal(t, "error", banner.Variant)
	assert.Equal(t, "boom", banner.Message)
}

// Test 11: empty inlineError → no banner prop.
func TestBuildCreateWizard_NoBanner(t *testing.T) {
	w := BuildCreateWizard(nil, defaultParams(), "en", "", "")
	_, ok := w.Props["banner"]
	assert.False(t, ok, "expected no banner prop when inlineError is empty")
}

// Test 12: initialStepID propagation.
func TestBuildCreateWizard_InitialStepID(t *testing.T) {
	// When set, initial_step_id prop must be present.
	w := BuildCreateWizard(nil, defaultParams(), "en", "", "summary")
	val, ok := w.Props["initial_step_id"]
	assert.True(t, ok, "expected initial_step_id to be set")
	assert.Equal(t, "summary", val)

	// When empty, initial_step_id must be absent.
	w2 := BuildCreateWizard(nil, defaultParams(), "en", "", "")
	_, ok2 := w2.Props["initial_step_id"]
	assert.False(t, ok2, "expected no initial_step_id when empty")
}

// Test: submit endpoint carries no is_full_snapshot param when nil.
func TestBuildCreateWizard_SubmitAction_NoFilter(t *testing.T) {
	p := ListParams{Offset: 0}
	w := BuildCreateWizard(nil, p, "en", "", "")

	submit := w.Props["submit_action"].(components.Action)
	// No query string when no params.
	assert.False(t, strings.Contains(submit.Endpoint, "is_full_snapshot"),
		"expected no is_full_snapshot in endpoint when nil")
	assert.Equal(t, "/actions/snapshots/create", submit.Endpoint)
}

// Test: catalog order is preserved in steps.
func TestBuildCreateWizard_CatalogOrder(t *testing.T) {
	catalog := mixedCatalog()
	w := BuildCreateWizard(catalog, defaultParams(), "en", "", "")
	steps := wizardSteps(t, w)

	// steps[0] = info, steps[1] = entry-complex-1, steps[2] = entry-simple-2, steps[3] = summary
	assert.Equal(t, "entry-complex-1", steps[1].ID)
	assert.Equal(t, "entry-simple-2", steps[2].ID)
}

// ---- BuildEditWizard tests ----

// sampleSnapshot returns a snapshot with entries for both catalog assets.
func sampleSnapshot() *Snapshot {
	return &Snapshot{
		ID:         "snap-1",
		RecordedAt: "2024-03-15T10:30:00Z",
		Notes:      "my notes",
		Entries: []Entry{
			{AssetID: "complex-1", CurrentValueOverride: "9999.00"},
			{AssetID: "simple-2", CurrentPrice: "50.00", CurrentValueOverride: ""},
		},
	}
}

// Test E1: mode == "edit" and title interpolates the snapshot's date.
func TestBuildEditWizard_ModeAndTitle(t *testing.T) {
	s := sampleSnapshot()
	w := BuildEditWizard(s, nil, defaultParams(), "en", "", "", nil)
	assert.Equal(t, "edit", w.Props["mode"])
	title, ok := w.Props["title"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, title)
	// Title must contain the date portion of RecordedAt (YYYY-MM-DD).
	assert.Contains(t, title, "2024-03-15")
}

// Test E2: submit_action endpoint targets PATCH on /actions/snapshots/<id>? with list params.
func TestBuildEditWizard_SubmitAction(t *testing.T) {
	s := sampleSnapshot()
	isFull := true
	p := ListParams{IsFullSnapshot: &isFull, Offset: 20}
	w := BuildEditWizard(s, nil, p, "en", "", "", nil)

	submit, ok := w.Props["submit_action"].(components.Action)
	require.True(t, ok, "expected submit_action to be components.Action")

	assert.Contains(t, submit.Endpoint, "/actions/snapshots/snap-1?")
	assert.Contains(t, submit.Endpoint, "is_full_snapshot=true")
	assert.Contains(t, submit.Endpoint, "offset=20")
	assert.Equal(t, "PATCH", submit.Method)
	assert.Equal(t, ScreenID, submit.TargetID)
}

// Test E3: dismiss_action is the client-side Dismiss action (matches assets pattern).
func TestBuildEditWizard_DismissAction(t *testing.T) {
	s := sampleSnapshot()
	w := BuildEditWizard(s, nil, defaultParams(), "en", "", "", nil)

	dismiss, ok := w.Props["dismiss_action"].(components.Action)
	require.True(t, ok, "expected dismiss_action to be components.Action")

	assert.Equal(t, "click", dismiss.Trigger)
	assert.Equal(t, "dismiss", dismiss.Type)
}

// Test E4: Info step — first child is a text (not input) with the formatted date; second is notes textarea pre-filled.
func TestBuildEditWizard_InfoStep(t *testing.T) {
	s := sampleSnapshot()
	w := BuildEditWizard(s, nil, defaultParams(), "en", "", "", nil)
	steps := wizardSteps(t, w)

	require.NotEmpty(t, steps)
	info := steps[0]
	assert.Equal(t, "info", info.ID)
	assert.Equal(t, "info", info.Kind)
	assert.False(t, info.Skippable)
	assert.True(t, info.IncludeDefault)
	require.Len(t, info.Children, 2, "info step must have exactly 2 children")

	// Child 0: text (not input) containing the date.
	ra := info.Children[0]
	assert.Equal(t, "text", ra.Type, "recorded_at in edit mode must be a text element, not an input")
	content, _ := ra.Props["content"].(string)
	assert.Contains(t, content, "2024-03-15", "text must contain the formatted date")

	// Child 1: notes textarea pre-filled.
	notes := info.Children[1]
	assert.Equal(t, "textarea", notes.Type)
	assert.Equal(t, "notes", notes.Props["name"])
	assert.Equal(t, "my notes", notes.Props["default_value"])
}

// Test E5: Catalog with one complex asset (in snapshot) and one non-complex (not in snapshot).
func TestBuildEditWizard_MixedCatalogEntrySteps(t *testing.T) {
	catalog := mixedCatalog() // complex-1 (in snapshot), simple-2 (in snapshot)
	// Build with only complex-1 in snapshot; override simple-2 to be absent.
	sPartial := &Snapshot{
		ID:         "snap-1",
		RecordedAt: "2024-03-15T10:30:00Z",
		Notes:      "my notes",
		Entries: []Entry{
			{AssetID: "complex-1", CurrentValueOverride: "9999.00"},
			// simple-2 NOT in entries
		},
	}
	w := BuildEditWizard(sPartial, catalog, defaultParams(), "en", "", "", nil)
	steps := wizardSteps(t, w)

	// steps: info + entry-complex-1 + entry-simple-2 + summary = 4
	require.Len(t, steps, 4)

	// Step for complex-1 (in snapshot): skippable=false, include_default=true.
	complexStep := steps[1]
	assert.Equal(t, "entry-complex-1", complexStep.ID)
	assert.False(t, complexStep.Skippable, "asset in snapshot: skippable must be false")
	assert.True(t, complexStep.IncludeDefault, "asset in snapshot: include_default must be true")

	// current_value_override input must be pre-filled.
	all := flattenChildren(complexStep.Children)
	var overrideInput *components.Component
	for i := range all {
		if all[i].Type == "input" {
			name, _ := all[i].Props["name"].(string)
			if name == "entries[complex-1].current_value_override" {
				overrideInput = &all[i]
				break
			}
		}
	}
	require.NotNil(t, overrideInput, "expected current_value_override input in complex step")
	assert.Equal(t, "9999.00", overrideInput.Props["default_value"])

	// An "already_included" text must appear in the step (resolved via i18n).
	const alreadyIncludedText = "Already in snapshot, cannot be removed"
	foundAlready := false
	for _, c := range all {
		if c.Type == "text" {
			if content, ok := c.Props["content"].(string); ok && content == alreadyIncludedText {
				foundAlready = true
				break
			}
		}
	}
	assert.True(t, foundAlready, "expected already_included text in complex step")

	// Step for simple-2 (NOT in snapshot): skippable=true, include_default=false.
	simpleStep := steps[2]
	assert.Equal(t, "entry-simple-2", simpleStep.ID)
	assert.True(t, simpleStep.Skippable, "asset not in snapshot: skippable must be true")
	assert.False(t, simpleStep.IncludeDefault, "asset not in snapshot: include_default must be false")

	// No "already_included" text in simple step.
	allSimple := flattenChildren(simpleStep.Children)
	for _, c := range allSimple {
		if c.Type == "text" {
			assert.NotEqual(t, alreadyIncludedText, c.Props["content"],
				"non-included step must not have already_included text")
		}
	}
	// Radio default_value for simple-2 (not in snapshot) = "price".
	for _, c := range allSimple {
		if c.Type == "radio_group" {
			assert.Equal(t, "price", c.Props["default_value"])
		}
	}
}

// Test E6: Non-complex asset in snapshot with CurrentPrice="" and CurrentValueOverride="100":
// radio default_value="override", override input has default_value="100".
func TestBuildEditWizard_NonComplex_OverrideMode(t *testing.T) {
	catalog := []assetscatalog.Asset{
		{ID: "simple-2", Ticker: "AAPL", Name: "Apple Inc", AssetType: "STOCK", IsComplex: false},
	}
	s := &Snapshot{
		ID:         "snap-1",
		RecordedAt: "2024-03-15T10:30:00Z",
		Entries: []Entry{
			{AssetID: "simple-2", CurrentPrice: "", CurrentValueOverride: "100"},
		},
	}
	w := BuildEditWizard(s, catalog, defaultParams(), "en", "", "", nil)
	steps := wizardSteps(t, w)

	// info + entry-simple-2 + summary
	require.Len(t, steps, 3)
	entryStep := steps[1]
	all := flattenChildren(entryStep.Children)

	var rg *components.Component
	for i := range all {
		if all[i].Type == "radio_group" {
			rg = &all[i]
			break
		}
	}
	require.NotNil(t, rg, "expected radio_group")
	assert.Equal(t, "override", rg.Props["default_value"])

	var overrideInput *components.Component
	for i := range all {
		if all[i].Type == "input" {
			if name, _ := all[i].Props["name"].(string); name == "entries[simple-2].current_value_override" {
				overrideInput = &all[i]
				break
			}
		}
	}
	require.NotNil(t, overrideInput)
	assert.Equal(t, "100", overrideInput.Props["default_value"])
}

// Test E7: Non-complex asset in snapshot with CurrentPrice="50" and CurrentValueOverride="":
// radio default_value="price", price input has default_value="50".
func TestBuildEditWizard_NonComplex_PriceMode(t *testing.T) {
	catalog := []assetscatalog.Asset{
		{ID: "simple-2", Ticker: "AAPL", Name: "Apple Inc", AssetType: "STOCK", IsComplex: false},
	}
	s := &Snapshot{
		ID:         "snap-1",
		RecordedAt: "2024-03-15T10:30:00Z",
		Entries: []Entry{
			{AssetID: "simple-2", CurrentPrice: "50", CurrentValueOverride: ""},
		},
	}
	w := BuildEditWizard(s, catalog, defaultParams(), "en", "", "", nil)
	steps := wizardSteps(t, w)

	require.Len(t, steps, 3)
	entryStep := steps[1]
	all := flattenChildren(entryStep.Children)

	var rg *components.Component
	for i := range all {
		if all[i].Type == "radio_group" {
			rg = &all[i]
			break
		}
	}
	require.NotNil(t, rg)
	assert.Equal(t, "price", rg.Props["default_value"])

	var priceInput *components.Component
	for i := range all {
		if all[i].Type == "input" {
			if name, _ := all[i].Props["name"].(string); name == "entries[simple-2].current_price" {
				priceInput = &all[i]
				break
			}
		}
	}
	require.NotNil(t, priceInput)
	assert.Equal(t, "50", priceInput.Props["default_value"])
}

// Test E8: Caller passes a non-nil banner → that banner is used verbatim (even if inlineError != "").
func TestBuildEditWizard_BannerPrecedence_CallerWins(t *testing.T) {
	s := sampleSnapshot()
	callerBanner := &components.WizardBanner{Variant: "info", Message: "info from caller", Dismissible: true}
	w := BuildEditWizard(s, nil, defaultParams(), "en", "boom", "", callerBanner)

	bannerRaw, ok := w.Props["banner"]
	require.True(t, ok)
	b, ok := bannerRaw.(*components.WizardBanner)
	require.True(t, ok)
	assert.Equal(t, "info", b.Variant)
	assert.Equal(t, "info from caller", b.Message)
	assert.True(t, b.Dismissible)
}

// Test E9: Caller passes nil and inlineError="boom" → banner variant=error, message=boom.
func TestBuildEditWizard_BannerFallback_InlineError(t *testing.T) {
	s := sampleSnapshot()
	w := BuildEditWizard(s, nil, defaultParams(), "en", "boom", "", nil)

	bannerRaw, ok := w.Props["banner"]
	require.True(t, ok)
	b, ok := bannerRaw.(*components.WizardBanner)
	require.True(t, ok)
	assert.Equal(t, "error", b.Variant)
	assert.Equal(t, "boom", b.Message)
}

// Test E10: Caller passes nil and inlineError="" → no banner prop.
func TestBuildEditWizard_NoBanner(t *testing.T) {
	s := sampleSnapshot()
	w := BuildEditWizard(s, nil, defaultParams(), "en", "", "", nil)
	_, ok := w.Props["banner"]
	assert.False(t, ok, "expected no banner prop when no error and no caller banner")
}

// Test E11: initialStepID passes through.
func TestBuildEditWizard_InitialStepID(t *testing.T) {
	s := sampleSnapshot()
	w := BuildEditWizard(s, nil, defaultParams(), "en", "", "summary", nil)
	val, ok := w.Props["initial_step_id"]
	assert.True(t, ok)
	assert.Equal(t, "summary", val)

	w2 := BuildEditWizard(s, nil, defaultParams(), "en", "", "", nil)
	_, ok2 := w2.Props["initial_step_id"]
	assert.False(t, ok2, "expected no initial_step_id when empty")
}

// Test E12: Snapshot has entry whose asset is NOT in catalog (asset was deleted).
// An extra step appears: skippable=false, include_default=true.
func TestBuildEditWizard_DeletedAssetEntry(t *testing.T) {
	// Catalog has only simple-2; snapshot has complex-1 (deleted from catalog) + simple-2.
	catalog := []assetscatalog.Asset{
		{ID: "simple-2", Ticker: "AAPL", Name: "Apple Inc", AssetType: "STOCK", IsComplex: false},
	}
	s := &Snapshot{
		ID:         "snap-1",
		RecordedAt: "2024-03-15T10:30:00Z",
		Entries: []Entry{
			{AssetID: "complex-1", CurrentValueOverride: "9999.00"}, // not in catalog
			{AssetID: "simple-2", CurrentPrice: "50.00"},
		},
	}
	w := BuildEditWizard(s, catalog, defaultParams(), "en", "", "", nil)
	steps := wizardSteps(t, w)

	// Expected: info + entry-simple-2 (from catalog) + entry-complex-1 (orphan) + summary = 4
	require.Len(t, steps, 4, "expected extra step for entry whose asset is not in catalog")

	// Find the orphan step.
	var orphanStep *components.WizardStep
	for i := range steps {
		if steps[i].ID == "entry-complex-1" {
			orphanStep = &steps[i]
			break
		}
	}
	require.NotNil(t, orphanStep, "expected a step for the deleted-asset entry")
	assert.False(t, orphanStep.Skippable, "orphan entry step must be skippable=false (existing entry)")
	assert.True(t, orphanStep.IncludeDefault, "orphan entry step must be include_default=true")
}
