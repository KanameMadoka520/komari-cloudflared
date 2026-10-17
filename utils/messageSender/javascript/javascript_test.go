package javascript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/komari-monitor/komari/database/models"
)

func TestNodeCompatibleNotificationPreservesLegacyIDAndConfinesFiles(t *testing.T) {
	oldDir := DataDir
	DataDir = filepath.Join(t.TempDir(), "javascript")
	t.Cleanup(func() { DataDir = oldDir })
	sender := &JavaScriptSender{Addition: Addition{Script: `
   async function sendMessage(message, title) {
     const fs = require("node:fs");
     let denied = false;
     try { fs.writeFileSync("../escape.txt", "bad"); } catch (_) { denied = true; }
     if (!denied) throw new Error("escaped script directory");
     await fs.promises.writeFile("message.txt", title + ":" + message);
     return true;
   }
 `}}
	if sender.GetName() != "Javascript" {
		t.Fatal("stored provider ID changed")
	}
	defer sender.Destroy()
	if err := sender.SendTextMessage("hello", "test"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(DataDir, "message.txt"))
	if err != nil || string(data) != "test:hello" {
		t.Fatalf("notification output: %q, %v", data, err)
	}
}

func TestStructuredNotificationEventAndRequiredLegacyEntryPoint(t *testing.T) {
	oldDir := DataDir
	DataDir = t.TempDir()
	t.Cleanup(func() { DataDir = oldDir })
	sender := &JavaScriptSender{Addition: Addition{Script: `
   function sendMessage() { throw new Error("unexpected fallback"); }
   function sendEvent(event) { return event.message === "structured"; }
 `}}
	defer sender.Destroy()
	if err := sender.SendEvent(models.EventMessage{Message: "structured"}); err != nil {
		t.Fatal(err)
	}
	invalid := &JavaScriptSender{Addition: Addition{Script: `function sendEvent() { return true; }`}}
	if err := invalid.Init(); err == nil {
		invalid.Destroy()
		t.Fatal("missing sendMessage must be rejected")
	}
}
