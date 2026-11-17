import React, { useState } from "react";
import { createRoot } from "react-dom/client";
import { Theme } from "@radix-ui/themes";
import "@radix-ui/themes/styles.css";
import "@xterm/xterm/css/xterm.css";
import "../../src/i18n/config";
import i18n from "i18next";
import TerminalSession from "../../src/pages/terminal/TerminalSession";
import LogPage from "../../src/pages/admin/log";
import { defaultXtermjsSettings } from "../../src/hooks/useXtermjsSettings";

function Harness() {
  const [settings, setSettings] = useState(defaultXtermjsSettings);
  const [visible, setVisible] = useState(true);
  Object.assign(window, { changeSettings: () => setSettings((s) => ({ ...s, terminalOptions: { ...s.terminalOptions, fontSize: 20 } })), changeLanguage: () => i18n.changeLanguage("en"), hideTerminal: () => setVisible(false) });
  if (new URLSearchParams(location.search).has("logs")) return <Theme><LogPage /></Theme>;
  return <div style={{ width: 900, height: 600, position: "relative" }}>{visible && <TerminalSession uuid="test-node" active settings={settings} twoFaEnabled={false} disconnectMessage="TEST DISCONNECTED" onApiChange={(api) => Object.assign(window, { terminalApi: api })} onLatencyChange={(latency) => Object.assign(window, { latency })} />}</div>;
}
createRoot(document.getElementById("root")!).render(<Harness />);
