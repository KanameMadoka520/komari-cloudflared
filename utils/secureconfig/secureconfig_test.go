package secureconfig

import "testing"

func TestEncryptDecryptString(t *testing.T) {
	t.Setenv("KOMARI_SECRET_KEY", "komari-secureconfig-test-key-32-chars")

	encrypted, err := EncryptString("hello-komari")
	if err != nil {
		t.Fatalf("EncryptString failed: %v", err)
	}
	if encrypted == "" || encrypted == "hello-komari" {
		t.Fatalf("EncryptString returned unexpected value: %q", encrypted)
	}

	decrypted, err := DecryptString(encrypted)
	if err != nil {
		t.Fatalf("DecryptString failed: %v", err)
	}
	if decrypted != "hello-komari" {
		t.Fatalf("unexpected decrypted value: %q", decrypted)
	}
}
