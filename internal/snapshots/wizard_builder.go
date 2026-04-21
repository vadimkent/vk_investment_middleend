package snapshots

import (
	"fmt"
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
		steps = append(steps, buildEntryStep(a, lang))
	}
	steps = append(steps, buildSummaryStep(lang))

	var banner *components.WizardBanner
	if inlineError != "" {
		banner = &components.WizardBanner{Variant: "error", Message: inlineError}
	}

	title := i18n.T(lang, "snapshots.create.title")
	return components.Wizard(WizardID, "create", title, steps, submitAction, dismissAction, banner, initialStepID)
}

// buildDismissAction returns the action that clears the modal slot.
func buildDismissAction() components.Action {
	return components.Action{
		Trigger:  "click",
		Type:     "replace",
		TargetID: ModalSlotID,
	}
}

// buildInfoStep builds the first "info" step (recorded_at + notes).
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

// buildEntryStep builds one entry step for a catalog asset.
func buildEntryStep(asset assetscatalog.Asset, lang string) components.WizardStep {
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
		entryChildren = append(entryChildren, buildComplexEntryChildren(asset, lang)...)
	} else {
		entryChildren = append(entryChildren, buildNonComplexEntryChildren(asset, lang)...)
	}

	return components.WizardStep{
		ID:             "entry-" + asset.ID,
		Label:          asset.Ticker,
		Kind:           "entry",
		Skippable:      true,
		IncludeDefault: false,
		Children:       entryChildren,
	}
}

// buildComplexEntryChildren returns the single override input for a complex asset.
func buildComplexEntryChildren(asset assetscatalog.Asset, lang string) []components.Component {
	override := components.Component{
		Type: "input",
		ID:   fmt.Sprintf("snapshots-wizard-override-%s", asset.ID),
		Props: map[string]any{
			"name":       fmt.Sprintf("entries[%s].current_value_override", asset.ID),
			"input_type": "text",
			"label":      i18n.T(lang, "snapshots.form.current_value_override"),
		},
	}
	return []components.Component{override}
}

// buildNonComplexEntryChildren returns the radio group + two conditional inputs for a non-complex asset.
func buildNonComplexEntryChildren(asset assetscatalog.Asset, lang string) []components.Component {
	modeName := fmt.Sprintf("entries[%s].mode", asset.ID)

	radioGroup := components.Component{
		Type: "radio_group",
		ID:   fmt.Sprintf("snapshots-wizard-mode-%s", asset.ID),
		Props: map[string]any{
			"name": modeName,
			"options": []components.SelectOption{
				{Value: "price", Label: i18n.T(lang, "snapshots.form.toggle_price")},
				{Value: "override", Label: i18n.T(lang, "snapshots.form.toggle_override")},
			},
			"default_value": "price",
		},
	}

	priceInput := components.Component{
		Type: "input",
		ID:   fmt.Sprintf("snapshots-wizard-price-%s", asset.ID),
		Props: map[string]any{
			"name":         fmt.Sprintf("entries[%s].current_price", asset.ID),
			"input_type":   "text",
			"label":        i18n.T(lang, "snapshots.form.current_price"),
			"visible_when": components.VisibleWhen{Field: modeName, Op: "eq", Value: "price"},
		},
	}

	overrideInput := components.Component{
		Type: "input",
		ID:   fmt.Sprintf("snapshots-wizard-override-%s", asset.ID),
		Props: map[string]any{
			"name":         fmt.Sprintf("entries[%s].current_value_override", asset.ID),
			"input_type":   "text",
			"label":        i18n.T(lang, "snapshots.form.current_value_override"),
			"visible_when": components.VisibleWhen{Field: modeName, Op: "eq", Value: "override"},
		},
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
