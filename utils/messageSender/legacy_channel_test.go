package messageSender

import "testing"

func TestRegisterLegacyChannelsKeepsSavedProviders(t *testing.T) {
	if err := RegisterLegacyChannels(); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"telegram", "email", "bark", "Javascript", "Server酱³", "Server酱Turbo", "empty"} {
		if !NotificationChannelRegistered(id) {
			t.Errorf("notification channel %s is not registered", id)
		}
	}
}
