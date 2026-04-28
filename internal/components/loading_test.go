package components

import (
	"encoding/json"
	"testing"
)

func TestSubmit_LoadingDefaultsToSectionString(t *testing.T) {
	a := Submit("/actions/foo", "POST", "foo-id")
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["loading"] != "section" {
		t.Fatalf("expected loading=\"section\", got %#v", got["loading"])
	}
}

func TestAction_LoadingFullStringRoundTrips(t *testing.T) {
	a := Action{Trigger: "click", Type: "submit", Endpoint: "/x", Method: "POST", TargetID: "y", Loading: "full"}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !containsStr(string(b), `"loading":"full"`) {
		t.Fatalf("expected loading=\"full\" string token, got: %s", b)
	}
}

func TestAction_LoadingFullWithMessages(t *testing.T) {
	a := Action{
		Trigger:  "click",
		Type:     "submit",
		Endpoint: "/actions/import/analyze",
		Method:   "POST",
		TargetID: "import-modal-slot",
		Loading: LoadingFullWithMessages([]string{
			"Detecting columns…",
			"Mapping tickers…",
		}),
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `"loading":{"scope":"full","messages":["Detecting columns…","Mapping tickers…"]}`
	if !containsStr(string(b), want) {
		t.Fatalf("expected loading object form. got: %s", b)
	}
}

func TestAction_LoadingOmittedWhenEmpty(t *testing.T) {
	a := Action{Trigger: "click", Type: "submit", Endpoint: "/x", Method: "POST", TargetID: "y"}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if containsStr(string(b), `"loading"`) {
		t.Fatalf("expected loading to be omitted when empty. got: %s", b)
	}
}

func containsStr(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOfStr(haystack, needle) >= 0)
}

func indexOfStr(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
