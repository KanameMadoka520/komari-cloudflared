package plugin

import (
	"github.com/komari-monitor/komari/utils"
	"testing"
)

func TestCheckKomariVersion(t *testing.T) {
	previous := utils.CurrentVersion
	utils.CurrentVersion = "0.0.1"
	t.Cleanup(func() { utils.CurrentVersion = previous })
	tests := []struct {
		constraint string
		wantErr    bool
	}{
		{"", false},
		{"0.0.1", false},
		{"v0.0.1", false},
		{">=0.0.1", false},
		{">0.0.1", true},
		{"<1.0.0", false},
		{"<=0.0.1", false},
		{">=99.0.0", true},
		{"0.1", true}, // 0.1.0 > 0.0.1
		{"1", true},   // 1.0.0 > 0.0.1
		{"1.2.3.4", true},
		{"abc", true},
		{">", true},
	}
	for _, tt := range tests {
		err := CheckKomariVersion(tt.constraint)
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckKomariVersion(%q) error = %v, wantErr %v", tt.constraint, err, tt.wantErr)
		}
	}
}

func TestForkVersionEnforcesPluginRequirements(t *testing.T) {
	previous := utils.CurrentVersion
	t.Cleanup(func() { utils.CurrentVersion = previous })
	for _, version := range []string{"1.5.0-fix1", "1.5.0-fix1-cloudflared.1", "1.5.0+custom.1"} {
		utils.CurrentVersion = version
		if err := CheckKomariVersion(">=1.5.0"); err != nil {
			t.Fatal(err)
		}
		if err := CheckKomariVersion(">=99.0.0"); err == nil {
			t.Fatalf("%s bypassed requirement", version)
		}
	}
}
