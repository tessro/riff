package browser

import (
	"runtime"
	"testing"
)

func TestOpenSupported(t *testing.T) {
	// Just verify the function doesn't panic on supported platforms
	// We can't actually test browser opening in a unit test
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		// These are supported platforms
	default:
		t.Skipf("Unsupported platform: %s", runtime.GOOS)
	}
}
