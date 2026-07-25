//go:build windows

package filesystem

import (
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func sameTestPath(t *testing.T, first, second string) bool {
	t.Helper()
	return strings.EqualFold(
		windowsLongPathForTest(t, first),
		windowsLongPathForTest(t, second),
	)
}

func windowsLongPathForTest(t *testing.T, path string) string {
	t.Helper()
	pathPtr, err := windows.UTF16PtrFromString(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}

	bufferSize := uint32(260)
	for bufferSize <= 1<<15 {
		buffer := make([]uint16, bufferSize)
		length, err := windows.GetLongPathName(pathPtr, &buffer[0], bufferSize)
		if err != nil {
			t.Fatalf("expand long path %q: %v", path, err)
		}
		if length < bufferSize {
			return filepath.Clean(windows.UTF16ToString(buffer[:length]))
		}
		bufferSize = length + 1
	}
	t.Fatalf("long path %q exceeded supported test buffer", path)
	return ""
}
