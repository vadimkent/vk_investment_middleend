package snapshots

import (
	"fmt"
	"strings"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// WizardID is the DOM id of the create/edit wizard root.
const WizardID = "snapshots-wizard"

// BuildCreateWizard returns the wizard component tree for the snapshot create flow.
// catalog is the full asset catalog; one entry step is emitted per asset.
// p is the current list context; it is passed through to the submit endpoint.
// inlineError, when non-empty, is shown as an error banner.
// initialStepID, when non-empty, is passed as initial_step_id to focus a specific step.
func BuildCreateWizard(catalog []assetscatalog.Asset, p ListParams, lang, inlineError, initialStepID string) components.Component {
	endpoint := buildListURL("/actions/snapshots/create", p.IsFullSnapshot, p.Offset)

	submitAction := components.Submit(endpoint, "POST", ScreenID)
	dismissAction := buildDismissAction()

	steps := make([]components.WizardStep, 0, 1+len(catalog)+1)
	steps = append(steps, buildInfoStep(lang))
	for _, a := range catalog {
		steps = append(steps, buildEntryStep(a, lang, nil))
	}
	steps = append(steps, buildSummaryStep(lang))

	var banner *components.WizardBanner
	if inlineError != "" {
		banner = &components.WizardBanner{Variant: "error", Message: inlineError}
	}

	title := i18n.T(lang, "snapshots.create.title")
	return components.Wizard(WizardID, "create", title, steps, submitAction, dismissAction, banner, initialStepID)
}

// BuildEditWizard returns the wizard component tree for the snapshot edit flow.
// s is the snapshot to edit; catalog is the full asset catalog.
// p is the current list context; it is passed through to the submit endpoint.
// inlineError, when non-empty, is shown as an error banner (overridden by a non-nil banner arg).
// initialStepID, when non-empty, is passed as initial_step_id to focus a specific step.
// banner, when non-nil, takes precedence over any inlineError-derived banner.
func BuildEditWizard(s *Snapshot, catalog []assetscatalog.Asset, p ListParams, lang, inlineError, initialStepID string, banner *components.WizardBanner) components.Component {
	endpoint := buildListURL("/actions/snapshots/"+s.ID, p.IsFullSnapshot, p.Offset)

	submitAction := components.Submit(endpoint, "PATCH", ScreenID)
	dismissAction := buildDismissAction()

	// Build entry index for O(1) lookup.
	entryIndex := indexEntries(s.Entries)

	// Track which asset IDs are present in the catalog.
	catalogAssetIDs := make(map[string]bool, len(catalog))
	for _, a := range catalog {
		catalogAssetIDs[a.ID] = true
	}

	steps := make([]components.WizardStep, 0, 1+len(catalog)+1)
	steps = append(steps, buildEditInfoStep(s, lang))

	// Emit one step per catalog asset.
	for _, a := range catalog {
		var entry *Entry
		if e, ok := entryIndex[a.ID]; ok {
			entry = &e
		}
		steps = append(steps, buildEntryStep(a, lang, entry))
	}

	// Emit extra steps for entries whose asset is no longer in the catalog.
	for _, e := range s.Entries {
		if catalogAssetIDs[e.AssetID] {
			continue
		}
		// Synthesize a minimal asset from the entry data.
		orphanAsset := assetscatalog.Asset{
			ID:        e.AssetID,
			Ticker:    e.AssetID,
			Name:      e.AssetID,
			AssetType: "",
			IsComplex: true, // treat as complex: only override input makes sense for orphan entries
		}
		eCopy := e
		steps = append(steps, buildEntryStep(orphanAsset, lang, &eCopy))
	}

	steps = append(steps, buildSummaryStep(lang))

	// Banner precedence: caller-supplied > inlineError > nil.
	if banner == nil && inlineError != "" {
		banner = &components.WizardBanner{Variant: "error", Message: inlineError}
	}

	// Build title with formatted date — uses {date} placeholder like the delete modal.
	formattedDate := formatRecordedAt(s.RecordedAt)
	titleTemplate := i18n.T(lang, "snapshots.edit.title")
	title := strings.ReplaceAll(titleTemplate, "{date}", formattedDate)

	return components.Wizard(WizardID, "edit", title, steps, submitAction, dismissAction, banner, initialStepID)
}

// indexEntries returns a map of AssetID → Entry for O(1) lookup.
func indexEntries(entries []Entry) map[string]Entry {
	m := make(map[string]Entry, len(entries))
	for _, e := range entries {
		m[e.AssetID] = e
	}
	return m
}

// formatRecordedAt parses a RecordedAt string and returns the YYYY-MM-DD portion.
// Falls back to the original string if parsing fails.
func formatRecordedAt(recordedAt string) string {
	// RecordedAt may be RFC3339 or YYYY-MM-DDTHH:MM, truncate at 'T'.
	if idx := strings.IndexByte(recordedAt, 'T'); idx >= 0 {
		return recordedAt[:idx]
	}
	// Try parsing as RFC3339.
	if t, err := time.Parse(time.RFC3339, recordedAt); err == nil {
		return t.UTC().Format("2006-01-02")
	}
	return recordedAt
}

// buildDismissAction returns the action that closes the wizard. Matches the
// same pattern used by the assets screen's cancel buttons.
func buildDismissAction() components.Action {
	return components.Dismiss()
}

// buildInfoStep builds the first "info" step for create mode (recorded_at input + notes).
func buildInfoStep(lang string) components.WizardStep {
	now := time.Now().UTC().Format("2006-01-02T15:04")

	recordedAt := components.Component{
		Type: "input",
		ID:   "snapshots-wizard-recorded-at",
		Props: map[string]any{
			"name":       "recorded_at",
			"input_type": "datetime-local",
			"label":      i18n.T(lang, "snapshots.form.recorded_at"),
			"required":   true,
			"max":        now,
		},
	}

	notes := components.Component{
		Type: "textarea",
		ID:   "snapshots-wizard-notes",
		Props: map[string]any{
			"name":        "notes",
			"label":       i18n.T(lang, "snapshots.form.notes"),
			"placeholder": i18n.T(lang, "snapshots.form.notes_placeholder"),
			"max_length":  500,
		},
	}

	return components.WizardStep{
		ID:             "info",
		Label:          i18n.T(lang, "snapshots.wizard.info_label"),
		Kind:           "info",
		Skippable:      false,
		IncludeDefault: true,
		Children:       []components.Component{recordedAt, notes},
	}
}

// buildEditInfoStep builds the "info" step for edit mode: recorded_at is static text, notes is pre-filled.
func buildEditInfoStep(s *Snapshot, lang string) components.WizardStep {
	formattedDate := formatRecordedAt(s.RecordedAt)

	recordedAtText := components.Text(
		"snapshots-wizard-recorded-at",
		i18n.T(lang, "snapshots.form.recorded_at_readonly")+": "+formattedDate,
		"sm", "normal",
	)

	notes := components.Component{
		Type: "textarea",
		ID:   "snapshots-wizard-notes",
		Props: map[string]any{
			"name":          "notes",
			"label":         i18n.T(lang, "snapshots.form.notes"),
			"placeholder":   i18n.T(lang, "snapshots.form.notes_placeholder"),
			"max_length":    500,
			"default_value": s.Notes,
		},
	}

	return components.WizardStep{
		ID:             "info",
		Label:          i18n.T(lang, "snapshots.wizard.info_label"),
		Kind:           "info",
		Skippable:      false,
		IncludeDefault: true,
		Children:       []components.Component{recordedAtText, notes},
	}
}

// buildEntryStep builds one entry step for a catalog asset.
// entry is non-nil when the asset is already in the snapshot (edit mode pre-fill).
// When entry is nil the step behaves as a create-mode step (skippable, empty inputs).
func buildEntryStep(asset assetscatalog.Asset, lang string, entry *Entry) components.WizardStep {
	header := components.RowWithGap(
		fmt.Sprintf("snapshots-wizard-entry-%s-header", asset.ID),
		[]string{"auto", "auto", "auto"},
		"sm",
		components.Text(fmt.Sprintf("snapshots-wizard-entry-%s-ticker", asset.ID), asset.Ticker, "md", "bold"),
		components.Text(fmt.Sprintf("snapshots-wizard-entry-%s-name", asset.ID), asset.Name, "sm", "normal"),
		components.Text(fmt.Sprintf("snapshots-wizard-entry-%s-type", asset.ID), asset.AssetType, "xs", "normal"),
	)

	var entryChildren []components.Component
	entryChildren = append(entryChildren, header)
	if asset.IsComplex {
		entryChildren = append(entryChildren, buildComplexEntryChildren(asset, lang, entry)...)
	} else {
		entryChildren = append(entryChildren, buildNonComplexEntryChildren(asset, lang, entry)...)
	}

	// If this is an existing entry, append the "already included" indicator.
	if entry != nil {
		alreadyIncluded := components.Text(
			fmt.Sprintf("snapshots-wizard-entry-%s-already-included", asset.ID),
			i18n.T(lang, "snapshots.wizard.already_included"),
			"sm", "normal",
		)
		entryChildren = append(entryChildren, alreadyIncluded)
	}

	skippable := entry == nil
	includeDefault := entry != nil

	return components.WizardStep{
		ID:             "entry-" + asset.ID,
		Label:          asset.Ticker,
		Kind:           "entry",
		Skippable:      skippable,
		IncludeDefault: includeDefault,
		Children:       entryChildren,
	}
}

// buildComplexEntryChildren returns the single override input for a complex asset.
// entry is non-nil in edit mode to pre-fill the input.
func buildComplexEntryChildren(asset assetscatalog.Asset, lang string, entry *Entry) []components.Component {
	props := map[string]any{
		"name":       fmt.Sprintf("entries[%s].current_value_override", asset.ID),
		"input_type": "text",
		"label":      i18n.T(lang, "snapshots.form.current_value_override"),
	}
	if entry != nil && entry.CurrentValueOverride != "" {
		props["default_value"] = entry.CurrentValueOverride
	}
	override := components.Component{
		Type:  "input",
		ID:    fmt.Sprintf("snapshots-wizard-override-%s", asset.ID),
		Props: props,
	}
	return []components.Component{override}
}

// buildNonComplexEntryChildren returns the radio group + two conditional inputs for a non-complex asset.
// entry is non-nil in edit mode to set radio default and pre-fill inputs.
func buildNonComplexEntryChildren(asset assetscatalog.Asset, lang string, entry *Entry) []components.Component {
	modeName := fmt.Sprintf("entries[%s].mode", asset.ID)

	// Determine radio default and input default values.
	radioDefault := "price"
	priceDefault := ""
	overrideDefault := ""
	if entry != nil {
		if entry.CurrentPrice != "" {
			radioDefault = "price"
			priceDefault = entry.CurrentPrice
		} else {
			radioDefault = "override"
			overrideDefault = entry.CurrentValueOverride
		}
	}

	radioGroup := components.Component{
		Type: "radio_group",
		ID:   fmt.Sprintf("snapshots-wizard-mode-%s", asset.ID),
		Props: map[string]any{
			"name": modeName,
			"options": []components.SelectOption{
				{Value: "price", Label: i18n.T(lang, "snapshots.form.toggle_price")},
				{Value: "override", Label: i18n.T(lang, "snapshots.form.toggle_override")},
			},
			"default_value": radioDefault,
		},
	}

	priceProps := map[string]any{
		"name":         fmt.Sprintf("entries[%s].current_price", asset.ID),
		"input_type":   "text",
		"label":        i18n.T(lang, "snapshots.form.current_price"),
		"visible_when": components.VisibleWhen{Field: modeName, Op: "eq", Value: "price"},
	}
	if priceDefault != "" {
		priceProps["default_value"] = priceDefault
	}
	priceInput := components.Component{
		Type:  "input",
		ID:    fmt.Sprintf("snapshots-wizard-price-%s", asset.ID),
		Props: priceProps,
	}

	overrideProps := map[string]any{
		"name":         fmt.Sprintf("entries[%s].current_value_override", asset.ID),
		"input_type":   "text",
		"label":        i18n.T(lang, "snapshots.form.current_value_override"),
		"visible_when": components.VisibleWhen{Field: modeName, Op: "eq", Value: "override"},
	}
	if overrideDefault != "" {
		overrideProps["default_value"] = overrideDefault
	}
	overrideInput := components.Component{
		Type:  "input",
		ID:    fmt.Sprintf("snapshots-wizard-override-%s", asset.ID),
		Props: overrideProps,
	}

	return []components.Component{radioGroup, priceInput, overrideInput}
}

// buildSummaryStep builds the final "summary" step.
func buildSummaryStep(lang string) components.WizardStep {
	instructions := components.Text(
		"snapshots-wizard-summary-instructions",
		i18n.T(lang, "snapshots.wizard.summary_instructions"),
		"sm", "normal",
	)
	return components.WizardStep{
		ID:             "summary",
		Kind:           "summary",
		Skippable:      false,
		IncludeDefault: true,
		Children:       []components.Component{instructions},
	}
}
