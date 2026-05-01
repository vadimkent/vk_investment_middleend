package components

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnalysisChat_RequiredProps(t *testing.T) {
	c := AnalysisChat("analysis-chat", AnalysisChatProps{
		InitialEndpoint:    "/actions/analysis/stream?focus=x",
		FollowupEndpoint:   "/actions/analysis/sessions/{session_id}/messages",
		Placeholder:        "Ask a follow-up question…",
		SubmitLabel:        "Send",
		ErrorMessages:      map[string]string{"default": "Something went wrong."},
		TerminalErrorCodes: []string{"ANALYSIS_SESSION_EXPIRED"},
		TerminalCTALabel:   "Start a new analysis",
		ResetAction: Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: "/actions/analysis/reset",
			TargetID: "analysis-content",
			Loading:  "section",
		},
	})
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	for _, want := range []string{
		`"type":"analysis_chat"`,
		`"id":"analysis-chat"`,
		`"initial_endpoint":"/actions/analysis/stream?focus=x"`,
		`"followup_endpoint":"/actions/analysis/sessions/{session_id}/messages"`,
		`"placeholder":"Ask a follow-up question…"`,
		`"submit_label":"Send"`,
		`"terminal_cta_label":"Start a new analysis"`,
		`"terminal_error_codes":["ANALYSIS_SESSION_EXPIRED"]`,
		`"reset_action":{`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON, got: %s", want, got)
		}
	}
	for _, omitted := range []string{`"streaming_label"`, `"max_input_length"`} {
		if strings.Contains(got, omitted) {
			t.Errorf("expected %q to be omitted when zero, got: %s", omitted, got)
		}
	}
}

func TestAnalysisChat_OptionalProps(t *testing.T) {
	c := AnalysisChat("analysis-chat", AnalysisChatProps{
		InitialEndpoint:    "/x",
		FollowupEndpoint:   "/y",
		Placeholder:        "p",
		SubmitLabel:        "s",
		StreamingLabel:     "AI is thinking…",
		MaxInputLength:     2000,
		ErrorMessages:      map[string]string{"default": "x"},
		TerminalErrorCodes: []string{},
		TerminalCTALabel:   "t",
		ResetAction:        Action{Trigger: "click", Type: "reload", Endpoint: "/r", TargetID: "t"},
	})
	b, _ := json.Marshal(c)
	got := string(b)
	for _, want := range []string{
		`"streaming_label":"AI is thinking…"`,
		`"max_input_length":2000`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q, got: %s", want, got)
		}
	}
}
