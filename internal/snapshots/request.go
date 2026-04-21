package snapshots

import (
	"encoding/json"
	"io"
	"regexp"
	"sort"

	"github.com/gin-gonic/gin"
)

// entryKeyRe matches flat wizard-form keys of the shape entries[<asset_id>].<field>.
// Capture group 1: asset_id (one or more non-] characters).
// Capture group 2: field name (word characters).
var entryKeyRe = regexp.MustCompile(`^entries\[([^\]]+)\]\.(\w+)$`)

// wizardEntry is a single parsed asset entry from the wizard submission.
// Mode is "price" / "override" / "". CurrentPrice and CurrentValueOverride
// carry the submitted string values (unparsed). An entry is considered
// "included" by the handler if any of Mode / CurrentPrice / CurrentValueOverride
// has a non-empty value.
type wizardEntry struct {
	AssetID              string
	Mode                 string
	CurrentPrice         string
	CurrentValueOverride string
}

// parseJSONBody reads the request body and unmarshals it as a JSON object.
// Empty or absent bodies return an empty map with no error, matching the
// assets mutation handler convention. Invalid JSON returns an error so the
// caller can respond with 400 BAD_REQUEST.
func parseJSONBody(c *gin.Context) (map[string]any, error) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil || len(raw) == 0 {
		return map[string]any{}, nil
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, nil
}

// asString returns the string value at key if present and a string, else "".
func asString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// parseWizardEntries extracts the flat "entries[<id>].<field>" keys from body
// into a slice of wizardEntry, keyed by asset_id. Keys that don't match the
// regex are silently ignored. Result is sorted by AssetID ascending
// (lexicographic) for deterministic output.
func parseWizardEntries(body map[string]any) []wizardEntry {
	byID := map[string]*wizardEntry{}

	for k, _ := range body {
		m := entryKeyRe.FindStringSubmatch(k)
		if m == nil {
			continue
		}
		assetID, field := m[1], m[2]
		e, ok := byID[assetID]
		if !ok {
			e = &wizardEntry{AssetID: assetID}
			byID[assetID] = e
		}
		val := asString(body, k)
		switch field {
		case "mode":
			e.Mode = val
		case "current_price":
			e.CurrentPrice = val
		case "current_value_override":
			e.CurrentValueOverride = val
		}
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	entries := make([]wizardEntry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, *byID[id])
	}
	return entries
}
