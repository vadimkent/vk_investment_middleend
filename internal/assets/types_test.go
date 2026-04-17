package assets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseListResponse_AllFieldsSet(t *testing.T) {
	raw := []byte(`{
	  "assets":[
	    {
	      "id":"a1","ticker":"AAPL","name":"Apple Inc.","asset_type":"STOCK","currency":"USD",
	      "is_complex":false,"price_provider":"TWELVE_DATA","external_ticker":"AAPL",
	      "created_at":"2024-01-10T10:00:00Z"
	    }
	  ],
	  "total":42,"size":10,"offset":0
	}`)

	r, err := ParseListResponse(raw)
	require.NoError(t, err)
	require.Len(t, r.Assets, 1)
	assert.Equal(t, 42, r.Total)
	assert.Equal(t, 10, r.Size)
	assert.Equal(t, 0, r.Offset)

	a := r.Assets[0]
	assert.Equal(t, "a1", a.ID)
	assert.Equal(t, "AAPL", a.Ticker)
	assert.Equal(t, "Apple Inc.", a.Name)
	assert.Equal(t, "STOCK", a.AssetType)
	assert.Equal(t, "USD", a.Currency)
	assert.False(t, a.IsComplex)
	require.NotNil(t, a.PriceProvider)
	assert.Equal(t, "TWELVE_DATA", *a.PriceProvider)
}

func TestParseListResponse_NullPriceProviderAndComplex(t *testing.T) {
	raw := []byte(`{
	  "assets":[
	    {
	      "id":"a2","ticker":"HOUSE","name":"Apartment","asset_type":"REAL_ESTATE","currency":"USD",
	      "is_complex":true,"price_provider":null,"external_ticker":null,
	      "created_at":"2024-01-11T10:00:00Z"
	    }
	  ],
	  "total":1,"size":10,"offset":0
	}`)

	r, err := ParseListResponse(raw)
	require.NoError(t, err)
	require.Len(t, r.Assets, 1)

	a := r.Assets[0]
	assert.True(t, a.IsComplex)
	assert.Nil(t, a.PriceProvider)
}

func TestParseListResponse_EmptyAssets(t *testing.T) {
	raw := []byte(`{"assets":[],"total":0,"size":10,"offset":0}`)
	r, err := ParseListResponse(raw)
	require.NoError(t, err)
	assert.Empty(t, r.Assets)
	assert.Equal(t, 0, r.Total)
}

func TestParseListResponse_InvalidJSON(t *testing.T) {
	_, err := ParseListResponse([]byte(`not json`))
	require.Error(t, err)
}
