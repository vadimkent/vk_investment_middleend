package trades

import "testing"

func TestParseListResponse(t *testing.T) {
	body := []byte(`{"trades":[{"id":"t1","asset_id":"a1","trade_type":"BUY","quantity":"10","price_per_unit":"100","fees":"0","date":"2024-01-10T10:00:00Z","source":"MANUAL","notes":"n","created_at":"2024-01-10T10:00:00Z"}],"total":1,"size":10,"offset":0}`)
	r, err := ParseListResponse(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if r.Total != 1 || len(r.Trades) != 1 {
		t.Fatalf("parsed wrong: %+v", r)
	}
	got := r.Trades[0]
	if got.ID != "t1" || got.AssetID != "a1" || got.TradeType != "BUY" || got.Quantity != "10" || got.Source != "MANUAL" {
		t.Errorf("unexpected trade: %+v", got)
	}
}

func TestParseListResponse_EmptyTrades(t *testing.T) {
	body := []byte(`{"trades":[],"total":0,"size":10,"offset":0}`)
	r, err := ParseListResponse(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if r.Trades == nil {
		t.Fatalf("expected non-nil Trades slice, got nil")
	}
	if len(r.Trades) != 0 {
		t.Errorf("expected empty Trades, got %d", len(r.Trades))
	}
	if r.Total != 0 {
		t.Errorf("expected Total=0, got %d", r.Total)
	}
}

func TestParseListResponse_Malformed(t *testing.T) {
	body := []byte(`{"trades":[`)
	if _, err := ParseListResponse(body); err == nil {
		t.Fatalf("expected error for malformed JSON, got nil")
	}
}

func TestParseListResponse_SellImport(t *testing.T) {
	body := []byte(`{"trades":[{"id":"t2","asset_id":"a2","trade_type":"SELL","quantity":"5","price_per_unit":"200","fees":"1","date":"2024-02-10T10:00:00Z","source":"IMPORT","notes":"","created_at":"2024-02-10T10:00:00Z"}],"total":1,"size":10,"offset":0}`)
	r, err := ParseListResponse(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(r.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(r.Trades))
	}
	got := r.Trades[0]
	if got.TradeType != "SELL" {
		t.Errorf("expected TradeType=SELL, got %q", got.TradeType)
	}
	if got.Source != "IMPORT" {
		t.Errorf("expected Source=IMPORT, got %q", got.Source)
	}
}
