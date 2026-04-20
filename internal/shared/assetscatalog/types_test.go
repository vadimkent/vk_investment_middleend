package assetscatalog

import "testing"

func TestParseListResponse(t *testing.T) {
	body := []byte(`{"assets":[{"id":"a","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD","is_complex":false}],"total":1,"size":100,"offset":0}`)
	r, err := ParseListResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Total != 1 || len(r.Assets) != 1 || r.Assets[0].Ticker != "AAPL" || r.Assets[0].IsComplex {
		t.Errorf("parsed wrong: %+v", r)
	}
	if r.Assets[0].ID != "a" || r.Assets[0].Name != "Apple" || r.Assets[0].AssetType != "STOCK" || r.Assets[0].Currency != "USD" {
		t.Errorf("field mapping wrong: %+v", r.Assets[0])
	}
	if r.Size != 100 || r.Offset != 0 {
		t.Errorf("page meta wrong: size=%d offset=%d", r.Size, r.Offset)
	}
}

func TestParseListResponseComplexTrue(t *testing.T) {
	body := []byte(`{"assets":[{"id":"b","ticker":"XYZ","name":"Complex","asset_type":"BOND","currency":"EUR","is_complex":true}],"total":1,"size":100,"offset":0}`)
	r, err := ParseListResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.Assets[0].IsComplex {
		t.Errorf("is_complex=true not mapped: %+v", r.Assets[0])
	}
}

func TestParseListResponseEmptyAssets(t *testing.T) {
	body := []byte(`{"assets":[],"total":0,"size":100,"offset":0}`)
	r, err := ParseListResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Total != 0 || len(r.Assets) != 0 {
		t.Errorf("empty case wrong: %+v", r)
	}
}

func TestParseListResponseMalformed(t *testing.T) {
	body := []byte(`{not json`)
	if _, err := ParseListResponse(body); err == nil {
		t.Fatalf("expected error on malformed JSON, got nil")
	}
}
