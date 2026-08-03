//go:build !windows

package filesystem

import "os"

func openIdentityFile(path string) (*os.File, error) {
	return os.Open(path)
}
