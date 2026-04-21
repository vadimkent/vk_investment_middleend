package snapshots

import "testing"

func TestParseListResponse_Basic(t *testing.T) {
	body := []byte(`{
		"snapshots": [
			{"id":"s1","recorded_at":"2024-01-10T10:00:00Z","is_full_snapshot":true,"notes":"hi",
			 "entries":[{"asset_id":"a1","quantity":"10.5","current_price":"150.00","current_value_override":null,"source":"MANUAL"}],
			 "created_at":"2024-01-10T10:00:00Z"}
		],
		"total": 1, "size": 10, "offset": 0
	}`)
	res, err := ParseListResponse(body)
	if err != nil {
		t.Fatalf("ParseListResponse err: %v", err)
	}
	if res.Total != 1 || res.Size != 10 || res.Offset != 0 {
		t.Fatalf("pagination wrong: %+v", res)
	}
	if len(res.Snapshots) != 1 {
		t.Fatalf("want 1 snapshot, got %d", len(res.Snapshots))
	}
	s := res.Snapshots[0]
	if s.ID != "s1" || !s.IsFullSnapshot || s.Notes != "hi" {
		t.Fatalf("header wrong: %+v", s)
	}
	if len(s.Entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(s.Entries))
	}
	e := s.Entries[0]
	if e.AssetID != "a1" || e.Quantity != "10.5" || e.CurrentPrice != "150.00" || e.CurrentValueOverride != "" || e.Source != "MANUAL" {
		t.Fatalf("entry wrong: %+v", e)
	}
}

func TestParseListResponse_NullQuantity(t *testing.T) {
	body := []byte(`{
		"snapshots":[
			{"id":"s1","recorded_at":"2024-01-10T10:00:00Z","is_full_snapshot":false,"notes":"",
			 "entries":[{"asset_id":"a1","quantity":null,"current_price":null,"current_value_override":"1000.00","source":"MANUAL"}],
			 "created_at":"2024-01-10T10:00:00Z"}
		],"total":1,"size":10,"offset":0
	}`)
	res, err := ParseListResponse(body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	e := res.Snapshots[0].Entries[0]
	if e.Quantity != "" {
		t.Fatalf("null quantity should parse to empty string, got %q", e.Quantity)
	}
	if e.CurrentPrice != "" {
		t.Fatalf("null current_price should parse to empty string, got %q", e.CurrentPrice)
	}
	if e.CurrentValueOverride != "1000.00" {
		t.Fatalf("current_value_override wrong: %q", e.CurrentValueOverride)
	}
}

func TestParseSnapshot_Single(t *testing.T) {
	body := []byte(`{"id":"s1","recorded_at":"2024-01-10T10:00:00Z","is_full_snapshot":true,"notes":"x",
		"entries":[{"asset_id":"a1","quantity":"1","current_price":"100","current_value_override":null,"source":"COINGECKO"}],
		"created_at":"2024-01-10T10:00:00Z"}`)
	s, err := ParseSnapshot(body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if s.ID != "s1" || len(s.Entries) != 1 || s.Entries[0].Source != "COINGECKO" {
		t.Fatalf("parsed wrong: %+v", s)
	}
}
