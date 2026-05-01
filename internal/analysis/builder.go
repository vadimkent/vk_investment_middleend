package analysis

import (
	"net/url"

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
			components.Text("analysis-title", i18n.T(lang, "analysis.title"), "lg", "bold"),
		},
	}
	content := BuildContentStart(lang, "", "")
	return components.Component{
		Type: "column",
		ID:   "analysis-root",
		Props: map[string]any{
			"gap":  "lg",
			"fill": true,
		},
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
			"fill":          true,
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
			"gap":  "md",
			"fill": true,
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
		"ANALYSIS_SESSION_NOT_FOUND": i18n.T(lang, "analysis.error.session_not_found"),
		"ANALYSIS_SESSION_EXPIRED":   i18n.T(lang, "analysis.error.session_expired"),
		"ANALYSIS_TOO_MANY_MESSAGES": i18n.T(lang, "analysis.error.too_many_messages"),
		"ANALYSIS_FOCUS_TOO_LONG":    i18n.T(lang, "analysis.error.focus_too_long"),
		"AI_PROVIDER_UNAVAILABLE":    i18n.T(lang, "analysis.error.provider_unavailable"),
		"AI_RATE_LIMITED":            i18n.T(lang, "analysis.error.rate_limited"),
		"AI_TIMEOUT":                 i18n.T(lang, "analysis.error.timeout"),
		"AI_CONTEXT_TOO_LARGE":       i18n.T(lang, "analysis.error.context_too_large"),
		"RATE_LIMITED":               i18n.T(lang, "analysis.error.rate_limited"),
		"INTERNAL_ERROR":             i18n.T(lang, "analysis.error.internal"),
		"default":                    i18n.T(lang, "analysis.error.default"),
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
