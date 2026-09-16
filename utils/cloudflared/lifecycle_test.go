package cloudflared_test

import (
	"github.com/komari-monitor/komari/internal/config"
	"github.com/komari-monitor/komari/utils/cloudflared"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStoredTokenSurvivesProcessRestart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX helper process")
	}
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "config.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	config.SetDb(db)
	t.Setenv("KOMARI_SECRET_KEY", "lifecycle-test-secret-key-32-chars")
	t.Setenv("KOMARI_CLOUDFLARED_TOKEN", "")
	helper := filepath.Join(dir, "cloudflared")
	script := "#!/bin/sh\n[ \"$TUNNEL_TOKEN\" = \"lifecycle-test-token\" ] || exit 3\ntrap 'exit 0' TERM\necho ready\nwhile :; do sleep 0.1; done\n"
	if err := os.WriteFile(helper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KOMARI_CLOUDFLARED_BIN", helper)
	t.Cleanup(cloudflared.Shutdown)
	const token = "lifecycle-test-token"
	if err := cloudflared.SaveToken(token); err != nil {
		t.Fatal(err)
	}
	encrypted, err := config.GetAs[string](config.CloudflareTunnelTokenKey)
	if err != nil || encrypted == token || encrypted == "" {
		t.Fatal("token was not encrypted")
	}
	for i := 0; i < 2; i++ {
		if err := cloudflared.AutoStart(""); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(3 * time.Second)
		for strings.Count(strings.Join(cloudflared.Status().Logs, "\n"), "ready") < i+1 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		status := cloudflared.Status()
		if !status.Running || !status.TokenStored || strings.Count(strings.Join(status.Logs, "\n"), "ready") < i+1 {
			t.Fatal("stored token did not restart the tunnel process")
		}
		if strings.Contains(strings.Join(status.Logs, "\n"), token) {
			t.Fatal("raw token leaked to status logs")
		}
		if err := cloudflared.RemoveToken(); err == nil {
			t.Fatal("removed token while tunnel running")
		}
		cloudflared.Shutdown()
		if cloudflared.Status().Running {
			t.Fatal("tunnel process survived shutdown")
		}
	}
	if err := cloudflared.RemoveToken(); err != nil {
		t.Fatal(err)
	}
	stored, err := config.GetAs[string](config.CloudflareTunnelTokenKey)
	if err != nil || stored != "" {
		t.Fatal("token removal failed")
	}
}
