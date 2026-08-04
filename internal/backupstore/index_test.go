package backupstore

import (
	"strings"
	"testing"
)

func TestPersistedIndexSizeDoesNotScaleWithDetailedProjection(t *testing.T) {
	index := Index{
		FormatVersion:    IndexVersion,
		StoreID:          strings.Repeat("a", 64),
		GeneratedAt:      "2026-08-04T17:00:00Z",
		Generation:       strings.Repeat("b", 64),
		TotalObjectBytes: 123,
		ObjectCount:      1000,
		ManifestCount:    1000,
		PinnedCount:      100,
		Manifests:        make([]ManifestSummary, 1000),
		Objects:          make([]ObjectSummary, 1000),
		Targets:          make([]TargetSummary, 1000),
	}
	largePath := "/" + strings.Repeat("very-long-component/", 200)
	for position := range index.Manifests {
		index.Manifests[position].TargetPath = largePath
		index.Targets[position].TargetPath = largePath
	}

	data, err := encodeIndex(index)
	if err != nil {
		t.Fatalf("encodeIndex() error = %v", err)
	}
	if len(data) >= 2048 {
		t.Fatalf("persisted compact index size = %d bytes", len(data))
	}
	if strings.Contains(string(data), "very-long-component") {
		t.Fatal("persisted index included detailed target paths")
	}
}
