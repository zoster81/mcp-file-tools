package handler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zoster81/mcp-file-tools/internal/filesystem"
	"github.com/zoster81/mcp-file-tools/internal/operation"
)

func TestRestorePreviewStoreExpiresAndClosesRetainedIdentity(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target.txt")
	if err := os.WriteFile(target, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	identity, err := filesystem.OpenFileIdentity(target)
	if err != nil {
		t.Fatal(err)
	}
	store := newRestorePreviewStore(2, 1024, time.Minute)
	now := time.Unix(1_700_000_000, 0).UTC()
	store.now = func() time.Time { return now }
	preview, err := store.put(preparedRestore{
		backupID:          strings.Repeat("a", 64),
		requestedPath:     target,
		resolvedPath:      target,
		targetExisted:     true,
		targetIdentity:    identity,
		targetFingerprint: strings.Repeat("b", 64),
		resultFingerprint: strings.Repeat("c", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	if store.len() != 0 {
		t.Fatalf("expired restore preview remained live")
	}
	if _, err := identity.Matches(target); err == nil {
		t.Fatal("expired restore preview did not close its target identity")
	}
	if _, err := store.claim(preview.id); operation.KindOf(err) != operation.KindConflict {
		t.Fatalf("claim(expired) error=%v, want CONFLICT", err)
	}
}

func TestRestorePreviewStoreEvictsOldestAndClaimsOnce(t *testing.T) {
	store := newRestorePreviewStore(1, 4096, time.Minute)
	store.now = func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }
	first, err := store.put(preparedRestore{backupID: strings.Repeat("a", 64), requestedPath: "/first", resolvedPath: "/first", resultFingerprint: strings.Repeat("b", 64)})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.put(preparedRestore{backupID: strings.Repeat("c", 64), requestedPath: "/second", resolvedPath: "/second", resultFingerprint: strings.Repeat("d", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.claim(first.id); operation.KindOf(err) != operation.KindConflict {
		t.Fatalf("claim(evicted) error=%v, want CONFLICT", err)
	}
	claimed, err := store.claim(second.id)
	if err != nil || claimed.id != second.id {
		t.Fatalf("claim(second)=%+v err=%v", claimed, err)
	}
	if _, err := store.claim(second.id); operation.KindOf(err) != operation.KindConflict {
		t.Fatalf("claim(replay) error=%v, want CONFLICT", err)
	}
}

func TestRestorePreviewStoreRejectsRetainedByteOverflow(t *testing.T) {
	store := newRestorePreviewStore(2, 32, time.Minute)
	prepared := preparedRestore{
		backupID:          strings.Repeat("a", 64),
		requestedPath:     "/target",
		resolvedPath:      "/target",
		resultFingerprint: strings.Repeat("b", 64),
		diff:              strings.Repeat("x", 64),
	}
	if _, err := store.put(prepared); operation.KindOf(err) != operation.KindLimit {
		t.Fatalf("put(oversized) error=%v, want LIMIT", err)
	}
	if store.len() != 0 {
		t.Fatal("oversized restore preview changed cache state")
	}
}
