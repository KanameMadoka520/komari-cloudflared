package cloudflared_test

import (
	"testing"

	"github.com/komari-monitor/komari/utils/cloudflared"
)

func TestStatusReturnsSafeDefaults(t *testing.T) {
	status := cloudflared.Status(false)
	if status.Running {
		t.Fatalf("expected cloudflared to be stopped in test environment")
	}
	if status.Installed && status.BinaryPath == "" {
		t.Fatalf("expected binary path when cloudflared is installed")
	}
}
