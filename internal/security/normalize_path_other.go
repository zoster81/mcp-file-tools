//go:build !windows

package security

func normalizePlatformPath(path string) string {
	return path
}
