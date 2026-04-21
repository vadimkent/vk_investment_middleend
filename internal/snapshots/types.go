// Package snapshots implements the Snapshots SDUI screen (browse with
// expandable rows, wizard create/edit, auto-snapshot, delete).
package snapshots

import "encoding/json"

// Entry is a single asset entry within a snapshot.
// Quantity, CurrentPrice, CurrentValueOverride are kept as strings to preserve
// decimal precision; empty string encodes BE null.
type Entry struct {
	AssetID              string
	Quantity             string
	CurrentPrice         string
	CurrentValueOverride string
	Source               string
}

// Snapshot is a timestamped portfolio capture.
type Snapshot struct {
	ID             string
	RecordedAt     string
	IsFullSnapshot bool
	Notes          string
	Entries        []Entry
	CreatedAt      string
}

// ListParams captures the list endpoint query parameters.
type ListParams struct {
	IsFullSnapshot *bool // nil = no filter; pointer distinguishes "false" from "unset"
	Offset         int
}

// ListResult wraps the parsed backend list response.
type ListResult struct {
	Snapshots []Snapshot
	Total     int
	Size      int
	Offset    int
}

type rawEntry struct {
	AssetID              string  `json:"asset_id"`
	Quantity             *string `json:"quantity"`
	CurrentPrice         *string `json:"current_price"`
	CurrentValueOverride *string `json:"current_value_override"`
	Source               string  `json:"source"`
}

func (r rawEntry) toDomain() Entry {
	return Entry{
		AssetID:              r.AssetID,
		Quantity:             deref(r.Quantity),
		CurrentPrice:         deref(r.CurrentPrice),
		CurrentValueOverride: deref(r.CurrentValueOverride),
		Source:               r.Source,
	}
}

type rawSnapshot struct {
	ID             string     `json:"id"`
	RecordedAt     string     `json:"recorded_at"`
	IsFullSnapshot bool       `json:"is_full_snapshot"`
	Notes          string     `json:"notes"`
	Entries        []rawEntry `json:"entries"`
	CreatedAt      string     `json:"created_at"`
}

func (r rawSnapshot) toDomain() Snapshot {
	entries := make([]Entry, 0, len(r.Entries))
	for _, e := range r.Entries {
		entries = append(entries, e.toDomain())
	}
	return Snapshot{
		ID:             r.ID,
		RecordedAt:     r.RecordedAt,
		IsFullSnapshot: r.IsFullSnapshot,
		Notes:          r.Notes,
		Entries:        entries,
		CreatedAt:      r.CreatedAt,
	}
}

type rawListResponse struct {
	Snapshots []rawSnapshot `json:"snapshots"`
	Total     int           `json:"total"`
	Size      int           `json:"size"`
	Offset    int           `json:"offset"`
}

// ParseListResponse parses the backend GET /v1/snapshots body.
func ParseListResponse(body []byte) (*ListResult, error) {
	var r rawListResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := &ListResult{Total: r.Total, Size: r.Size, Offset: r.Offset}
	out.Snapshots = make([]Snapshot, 0, len(r.Snapshots))
	for _, rs := range r.Snapshots {
		out.Snapshots = append(out.Snapshots, rs.toDomain())
	}
	return out, nil
}

// ParseSnapshot parses a single-snapshot backend response body.
func ParseSnapshot(body []byte) (*Snapshot, error) {
	var r rawSnapshot
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	s := r.toDomain()
	return &s, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
