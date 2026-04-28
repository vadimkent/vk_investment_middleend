package imports

// Session is the backend's import session response, mirroring the JSON body
// returned by POST /v1/import/sessions and friends.
type Session struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	CreatedAt   string    `json:"created_at"`
	ExpiresAt   string    `json:"expires_at"`
	AISummary   string    `json:"ai_summary"`
	Assumptions []string  `json:"assumptions"`
	Preview     Preview   `json:"preview"`
	Gaps        []Gap     `json:"gaps"`
	GapCounts   GapCounts `json:"gap_counts"`
}

type GapCounts struct {
	Blocking int `json:"blocking"`
	Warnings int `json:"warnings"`
}

type Gap struct {
	ID           string  `json:"id"`
	Severity     string  `json:"severity"`
	Type         string  `json:"type"`
	Description  string  `json:"description"`
	AffectedRows []int   `json:"affected_rows"`
	Suggestion   string  `json:"suggestion"`
	Resolution   *string `json:"resolution"`
}

type Preview struct {
	Assets    []PreviewAsset    `json:"assets"`
	Trades    []PreviewTrade    `json:"trades"`
	Snapshots []PreviewSnapshot `json:"snapshots"`
}

type PreviewAsset struct {
	Ticker    string `json:"ticker"`
	Name      string `json:"name"`
	AssetType string `json:"asset_type"`
	Currency  string `json:"currency"`
	Action    string `json:"action"`
}

type PreviewTrade struct {
	Row          int     `json:"row"`
	Ticker       string  `json:"ticker"`
	TradeType    string  `json:"trade_type"`
	Date         string  `json:"date"`
	Quantity     *string `json:"quantity"`
	PricePerUnit *string `json:"price_per_unit"`
	Fees         string  `json:"fees"`
	Status       string  `json:"status"`
	GapID        *string `json:"gap_id"`
}

type PreviewSnapshot struct {
	Rows       []int                  `json:"rows"`
	RecordedAt string                 `json:"recorded_at"`
	Entries    []PreviewSnapshotEntry `json:"entries"`
	Status     string                 `json:"status"`
}

type PreviewSnapshotEntry struct {
	Ticker     string `json:"ticker"`
	TotalValue string `json:"total_value"`
	Status     string `json:"status"`
}

// ConfirmResult mirrors POST /v1/import/sessions/:id/confirm.
type ConfirmResult struct {
	AssetsCreated     int `json:"assets_created"`
	TradesImported    int `json:"trades_imported"`
	SnapshotsImported int `json:"snapshots_imported"`
	Warnings          int `json:"warnings"`
}

// RestoreResult mirrors POST /v1/restore.
type RestoreResult struct {
	AssetsImported          int `json:"assets_imported"`
	AssetsSkipped           int `json:"assets_skipped"`
	TradesImported          int `json:"trades_imported"`
	TradesSkipped           int `json:"trades_skipped"`
	SnapshotsImported       int `json:"snapshots_imported"`
	SnapshotsSkipped        int `json:"snapshots_skipped"`
	SnapshotEntriesImported int `json:"snapshot_entries_imported"`
	SnapshotEntriesSkipped  int `json:"snapshot_entries_skipped"`
}

// GapResolution is the request shape for PATCH /v1/import/sessions/:id/gaps.
type GapResolution struct {
	GapID string `json:"gap_id"`
	Value string `json:"value"`
}

// BackendError carries a code + message that the BE returns on validation
// failures (typically 400 / 422). The middleend surfaces .Message to the user
// directly (no re-translation) and uses .Code for routing decisions.
type BackendError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *BackendError) Error() string {
	return e.Code + ": " + e.Message
}
