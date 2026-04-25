package trades

import (
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
)

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
