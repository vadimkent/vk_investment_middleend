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
		focusValue, 2, 500, false, false,
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
