package snapshots

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// Stub — filled in in Task 6.1.
func BuildScreen(_ *ListResult, _ []assetscatalog.Asset, _ ListParams, _ string) components.Component {
	return components.Component{Type: "screen", ID: "snapshots-screen"}
}

// Stub — filled in in Task 6.1.
func BuildSnapshotsSection(_ *ListResult, _ []assetscatalog.Asset, _ ListParams, _ string) components.Component {
	return components.Component{Type: "column", ID: "snapshots-section"}
}
