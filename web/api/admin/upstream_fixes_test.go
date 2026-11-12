package admin

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestThemeExtractionMakesArchiveDirectoriesTraversable(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "theme.zip")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	manifest, _ := w.Create("komari-theme.json")
	manifest.Write([]byte(`{"name":"Permission test","short":"permission-test","author":"test","version":"1.0.0"}`))
	header := &zip.FileHeader{Name: "assets/"}
	header.SetMode(os.ModeDir | 0664)
	if _, err = w.CreateHeader(header); err != nil {
		t.Fatal(err)
	}
	asset, _ := w.Create("assets/test.txt")
	asset.Write([]byte("test"))
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if _, err = extractAndValidateTheme(archive); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat("data/theme/permission-test/assets")
	if err != nil || info.Mode().Perm()&0111 != 0111 {
		t.Fatalf("directory is not traversable: %v %v", info, err)
	}
	if data, err := os.ReadFile("data/theme/permission-test/assets/test.txt"); err != nil || string(data) != "test" {
		t.Fatalf("asset missing: %q %v", data, err)
	}
}

func TestBackupIncludesPluginData(t *testing.T) {
	source, dest := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "plugin-data", "test"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "plugin-data", "test", "state.json"), []byte(`{"saved":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := copyWhitelistedFilesFrom(source, dest); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(dest, "plugin-data", "test", "state.json")); err != nil || string(data) != `{"saved":true}` {
		t.Fatalf("plugin data missing: %q %v", data, err)
	}
}
