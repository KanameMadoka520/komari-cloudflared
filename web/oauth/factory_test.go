package oauth

import (
	"github.com/komari-monitor/komari/web/oauth/factory"
	"testing"
)

func TestRetainedCloudflareAccessProviderIsAvailable(t *testing.T) {
	All()
	constructor, ok := factory.GetConstructor("CloudflareAccess")
	if !ok {
		t.Fatal("Cloudflare Access was removed from the provider registry")
	}
	provider := constructor()
	if provider.GetConfiguration() == nil {
		t.Fatal("missing provider configuration")
	}
	if err := provider.Init(); err == nil {
		t.Fatal("empty Cloudflare Access settings must be rejected")
	}
	if len(factory.GetProviderConfigs()["CloudflareAccess"]) != 2 {
		t.Fatal("missing team domain or policy AUD fields")
	}
}
