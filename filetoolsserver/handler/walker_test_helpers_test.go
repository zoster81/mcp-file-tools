package handler

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func createDirectoryLinkForTest(t *testing.T, target, link string) {
	t.Helper()

	if err := os.Symlink(target, link); err == nil {
		return
	} else if runtime.GOOS != "windows" {
		t.Skipf("directory symlink creation is not supported in this environment: %v", err)
	}

	output, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		t.Skipf("directory junction creation is not supported in this environment: %v (%s)", err, output)
	}
}
