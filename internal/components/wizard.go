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
