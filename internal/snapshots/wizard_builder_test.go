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

// Test 4: dismiss_action targets ModalSlotID with type "replace".
func TestBuildCreateWizard_DismissAction(t *testing.T) {
	w := BuildCreateWizard(nil, defaultParams(), "en", "", "")

	dismiss, ok := w.Props["dismiss_action"].(components.Action)
	require.True(t, ok, "expected dismiss_action to be components.Action")

	assert.Equal(t, ModalSlotID, dismiss.TargetID)
	assert.Equal(t, "replace", dismiss.Type)
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
