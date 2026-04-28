package imports

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildRestoreCardSuccess_Shape(t *testing.T) {
	loadTestLocales(t)
	res := &RestoreResult{
		AssetsImported: 1, AssetsSkipped: 0,
		TradesImported: 2, TradesSkipped: 1,
		SnapshotsImported: 0, SnapshotsSkipped: 3,
		SnapshotEntriesImported: 4, SnapshotEntriesSkipped: 5,
	}
	c := BuildRestoreCardSuccess("en", res)
	b, _ := json.Marshal(c)
	js := string(b)

	for _, want := range []string{
		`"id":"restore-card"`,
		`"id":"restore-success-table"`,
		`Restored successfully`,
		`Restore another file`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q", want)
		}
	}
}
