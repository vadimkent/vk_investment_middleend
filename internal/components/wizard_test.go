package components

import (
	"encoding/json"
	"testing"
)

func TestWizard_BasicShape(t *testing.T) {
	submit := Submit("/actions/snapshots/create", "POST", "snapshot-wizard")
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
	submit := Submit("/x", "POST", "w")
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
	submit := Submit("/x", "POST", "w")
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
