package analysis

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/project/vk-investment-middleend/internal/i18n"
)

func loadAnalysisLocales(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	en := `{
		"analysis": {
			"title": "Analysis",
			"start": {
				"description": "Get an AI-powered analysis of your portfolio — positions, allocation, risk, and opportunities.",
				"focus_label": "Focus area (optional)",
				"focus_placeholder": "e.g. risk exposure, crypto allocation, dividend potential",
				"submit": "Analyze my portfolio"
			},
			"new_analysis": "New analysis",
			"chat": {
				"placeholder": "Ask a follow-up question…",
				"submit_label": "Send",
				"streaming_label": "AI is thinking…",
				"terminal_cta": "Start a new analysis"
			},
			"error": {
				"session_not_found": "Session not found.",
				"session_expired": "Session expired. Start a new analysis.",
				"too_many_messages": "Conversation length limit reached. Start a new analysis.",
				"focus_too_long": "Focus area is too long.",
				"provider_unavailable": "AI provider unavailable. Please retry.",
				"rate_limited": "AI rate limit reached. Please retry shortly.",
				"timeout": "AI request timed out. Please retry.",
				"context_too_large": "Portfolio context is too large for the AI.",
				"internal": "Connection lost. Please try again.",
				"default": "Something went wrong. Please retry."
			},
			"feedback": { "start_failed": "Could not start analysis. Please retry." }
		}
	}`
	if err := os.WriteFile(dir+"/en.json", []byte(en), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := i18n.Load(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBuildRoot_Shape(t *testing.T) {
	loadAnalysisLocales(t)
	root := BuildRoot("en")
	b, _ := json.MarshalIndent(root, "", "  ")
	js := string(b)
	for _, want := range []string{
		`"id": "analysis-screen"`,
		`"id": "analysis-root"`,
		`"id": "analysis-header"`,
		`"id": "analysis-content"`,
		`"id": "analysis-start-card"`,
		`"id": "analysis-start-form"`,
		`"id": "analysis-focus"`,
		`"id": "analysis-start-submit"`,
		`"title": "Analysis"`,
		`Analyze my portfolio`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in root tree", want)
		}
	}
}

func TestBuildStartState_PrefillAndError(t *testing.T) {
	loadAnalysisLocales(t)
	c := BuildStartState("en", "risk exposure", "Focus area is too long.")
	b, _ := json.Marshal(c)
	js := string(b)
	if !strings.Contains(js, `risk exposure`) {
		t.Error("missing prefilled focus value")
	}
	if !strings.Contains(js, `Focus area is too long.`) {
		t.Error("missing error message")
	}
}

func TestBuildContentChat_Shape(t *testing.T) {
	loadAnalysisLocales(t)
	c := BuildContentChat("en", "risk exposure")
	b, _ := json.Marshal(c)
	js := string(b)

	for _, want := range []string{
		`"id":"analysis-content"`,
		`"id":"analysis-new-btn"`,
		`"id":"analysis-chat"`,
		`"type":"analysis_chat"`,
		`"initial_endpoint":"/actions/analysis/stream?focus=risk+exposure"`,
		`"followup_endpoint":"/actions/analysis/sessions/{session_id}/messages"`,
		`"endpoint":"/actions/analysis/reset"`,
		`AI is thinking…`,
		`Start a new analysis`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in chat state", want)
		}
	}
}

func TestBuildContentChat_EmptyFocusOmitsQueryParam(t *testing.T) {
	loadAnalysisLocales(t)
	c := BuildContentChat("en", "")
	b, _ := json.Marshal(c)
	js := string(b)
	if strings.Contains(js, `?focus=`) {
		t.Errorf("expected no ?focus= when empty, got: %s", js)
	}
	if !strings.Contains(js, `"initial_endpoint":"/actions/analysis/stream"`) {
		t.Errorf("expected initial_endpoint without query, got: %s", js)
	}
}
