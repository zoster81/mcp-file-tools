package handler

import (
	"os"
	"testing"
)

func sameExistingTestFile(t *testing.T, first, second string) bool {
	t.Helper()

	firstInfo, err := os.Stat(first)
	if err != nil {
		t.Fatalf("stat first path %q: %v", first, err)
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		t.Fatalf("stat second path %q: %v", second, err)
	}
	return os.SameFile(firstInfo, secondInfo)
}
