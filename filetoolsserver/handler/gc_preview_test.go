package handler

import (
	"strings"
	"testing"
	"time"

	"github.com/zoster81/scripthold/internal/backupstore"
	"github.com/zoster81/scripthold/internal/operation"
)

func TestGCPreviewStoreIsBoundedExpiringAndOneShot(t *testing.T) {
	now := time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)
	store := newGCPreviewStore(1, 4096, time.Minute)
	store.now = func() time.Time { return now }
	store.random = strings.NewReader(strings.Repeat("a", 32) + strings.Repeat("b", 32) + strings.Repeat("c", 32))

	first, err := store.put(gcPreviewTestPlan("a"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.put(gcPreviewTestPlan("b"))
	if err != nil {
		t.Fatal(err)
	}
	if store.len() != 1 {
		t.Fatalf("cache length=%d, want 1", store.len())
	}
	if _, err := store.claim(first.id); operation.KindOf(err) != operation.KindConflict {
		t.Fatalf("evicted claim error=%v, want CONFLICT", err)
	}
	claimed, err := store.claim(second.id)
	if err != nil || claimed.plan.Generation != second.plan.Generation {
		t.Fatalf("claim=%#v error=%v", claimed, err)
	}
	if _, err := store.claim(second.id); operation.KindOf(err) != operation.KindConflict {
		t.Fatalf("replay error=%v, want CONFLICT", err)
	}

	third, err := store.put(gcPreviewTestPlan("c"))
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := store.claim(third.id); operation.KindOf(err) != operation.KindConflict {
		t.Fatalf("expired claim error=%v, want CONFLICT", err)
	}
}

func TestGCPreviewStoreRejectsOversizedPlan(t *testing.T) {
	store := newGCPreviewStore(1, 16, time.Minute)
	plan := gcPreviewTestPlan("a")
	plan.Manifests[0].TargetPath = strings.Repeat("x", 128)
	if _, err := store.put(plan); operation.KindOf(err) != operation.KindLimit {
		t.Fatalf("put error=%v, want LIMIT", err)
	}
}

func gcPreviewTestPlan(seed string) backupstore.GCPlan {
	return backupstore.GCPlan{
		PlannedAt:                "2026-08-05T08:00:00Z",
		Generation:               strings.Repeat(seed, 64),
		RetentionDays:            30,
		MinimumVersionsPerTarget: 1,
		ManifestCount:            1,
		ObjectCount:              1,
		ReclaimableBytes:         1,
		Manifests: []backupstore.GCManifestCandidate{{
			BackupID:     strings.Repeat(seed, 64),
			CreatedAt:    "2026-07-01T08:00:00Z",
			TargetPath:   `C:\target.txt`,
			ObjectDigest: strings.Repeat(seed, 64),
			ObjectBytes:  1,
			Reasons:      []backupstore.GCReason{backupstore.GCReasonRetention},
		}},
		Objects: []backupstore.GCObjectCandidate{{Digest: strings.Repeat(seed, 64), Bytes: 1, ReferencesBefore: 1}},
	}
}
