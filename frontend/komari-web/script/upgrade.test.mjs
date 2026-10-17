import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import ts from "typescript";

async function load(path) {
  const source = await readFile(new URL(`../src/${path}`, import.meta.url), "utf8");
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  });
  return import(`data:text/javascript;base64,${Buffer.from(outputText).toString("base64")}`);
}

test("traffic limits accept units without evaluating expressions or overflowing", async () => {
  const { stringToBytes } = await load("utils/unitHelper.ts");
  for (const [value, expected] of [["", 0], ["1.5 GiB", 1610612736], ["512 MiB", 536870912], ["2t", 2199023255552], ["kb", 1024], ["0", 0]]) {
    assert.equal(stringToBytes(value), expected, value);
  }
  for (const value of ["1+1GB", "-1", "Infinity", "9007199254740992", "999PB", "globalThis.pwned=true", "abc"]) {
    assert.equal(stringToBytes(value), null, value);
  }
  assert.equal(globalThis.pwned, undefined);
});

test("login preserves internal destinations and rejects external/self redirects", async () => {
  globalThis.window = { location: { origin: "https://komari.example" } };
  const { resolveLoginRedirect, loginPath } = await load("utils/loginRedirect.ts");
  assert.equal(resolveLoginRedirect("/terminal?id=node-1"), "/terminal?id=node-1");
  assert.equal(loginPath("/admin/servers", "?group=London"), "/admin/login?redirect=%2Fadmin%2Fservers%3Fgroup%3DLondon");
  for (const value of [null, "https://evil.example", "//evil.example", "/\\evil.example", "/admin/login", "/admin/login/?redirect=/admin/login"]) {
    assert.equal(resolveLoginRedirect(value), "/admin/dashboard", String(value));
  }
});

test("dashboard layouts repair invalid records while preserving an intentionally empty layout", async () => {
  const { parseDashboardLayout, defaultLayout } = await load("components/admin/dashboard/layout.ts");
  assert.deepEqual(parseDashboardLayout([]), []);
  assert.deepEqual(parseDashboardLayout([{ id: "unknown" }]), defaultLayout());
  const layout = parseDashboardLayout([{ id: "traffic", width: "bad", height: "bad", limit: -1 }, { id: "traffic" }, null]);
  assert.deepEqual(layout, [{ id: "traffic", width: "wide", height: "normal", limit: 5 }]);
});

test("theme, guide and dashboard writes preserve each other's settings, including after a failed write", async () => {
  const { saveThemeSettings } = await load("utils/saveThemeSettings.ts");
  let settings = { background: "keep", _komari_dashboard_v1: [{ id: "traffic" }] };
  let failRead = false;
  let writes = 0;
  globalThis.fetch = async (_url, options = {}) => {
    if (options.method === "POST") {
      writes++;
      settings = JSON.parse(options.body);
      return { ok: true };
    }
    return { ok: !failRead, status: 503, json: async () => ({ data: { theme: "default", theme_settings: settings } }) };
  };
  await Promise.all([
    saveThemeSettings("default", { _komari_onboarding_v1: { seen: ["install"] } }),
    saveThemeSettings("default", { background: "updated" }),
  ]);
  assert.equal(settings.background, "updated");
  assert.deepEqual(settings._komari_dashboard_v1, [{ id: "traffic" }]);
  assert.deepEqual(settings._komari_onboarding_v1, { seen: ["install"] });
  failRead = true;
  await assert.rejects(saveThemeSettings("default", { background: "lost" }));
  assert.equal(writes, 2);
  failRead = false;
  await assert.rejects(saveThemeSettings("other-theme", { background: "lost" }));
  assert.equal(writes, 2);
  await saveThemeSettings("default", { background: "final" });
  assert.equal(settings.background, "final");
});
