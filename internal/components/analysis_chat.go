package components

// AnalysisChatProps captures all configuration for an analysis_chat component.
// Required: InitialEndpoint, FollowupEndpoint, Placeholder, SubmitLabel,
// ErrorMessages, TerminalErrorCodes, TerminalCTALabel, ResetAction. Other
// fields are optional and omitted from the rendered Props map when zero.
type AnalysisChatProps struct {
	InitialEndpoint    string
	FollowupEndpoint   string
	Placeholder        string
	SubmitLabel        string
	StreamingLabel     string
	MaxInputLength     int
	ErrorMessages      map[string]string
	TerminalErrorCodes []string
	TerminalCTALabel   string
	ResetAction        Action
}

// AnalysisChat creates an analysis_chat custom component. See
// spec/sdui-custom-components.md §5 for the contract.
func AnalysisChat(id string, p AnalysisChatProps) Component {
	props := map[string]any{
		"initial_endpoint":     p.InitialEndpoint,
		"followup_endpoint":    p.FollowupEndpoint,
		"placeholder":          p.Placeholder,
		"submit_label":         p.SubmitLabel,
		"error_messages":       p.ErrorMessages,
		"terminal_error_codes": p.TerminalErrorCodes,
		"terminal_cta_label":   p.TerminalCTALabel,
		"reset_action":         p.ResetAction,
	}
	if p.StreamingLabel != "" {
		props["streaming_label"] = p.StreamingLabel
	}
	if p.MaxInputLength > 0 {
		props["max_input_length"] = p.MaxInputLength
	}
	return Component{
		Type:  "analysis_chat",
		ID:    id,
		Props: props,
	}
}
