import { useEffect, useState } from "react";
import { Button, Flex, Text, TextField } from "@radix-ui/themes";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { formatBytes } from "@/utils/unitHelper";
import { SettingCardCollapse } from "./SettingCard";

// Explicit units avoid treating invalid input as zero or evaluating expressions.
function parseUsage(input: string): number | null {
  const match = input.trim().match(/^(\d+(?:\.\d+)?)\s*(B|KB|MB|GB|TB|KiB|MiB|GiB|TiB)?$/i);
  if (!match) return null;
  const unit = (match[2] ?? "B").toUpperCase();
  const powers: Record<string, number> = { B: 0, KB: 1, MB: 2, GB: 3, TB: 4, KIB: 1, MIB: 2, GIB: 3, TIB: 4 };
  const bytes = Math.round(Number(match[1]) * (unit.includes("I") ? 1024 : 1000) ** powers[unit]);
  return Number.isSafeInteger(bytes) && bytes >= 0 ? bytes : null;
}

type Usage = { upload: number; download: number; reset_at: string | null };

async function request(uuid: string, action = "", values?: object): Promise<Usage> {
  const response = await fetch(`/api/admin/client/${encodeURIComponent(uuid)}/traffic${action}`, {
    method: action ? "POST" : "GET",
    headers: { "Content-Type": "application/json" },
    body: action ? JSON.stringify(values ?? {}) : undefined,
  });
  const payload = await response.json();
  if (!response.ok || payload.status === "error") throw new Error(payload.message || `HTTP ${response.status}`);
  return payload.data ?? payload;
}

export default function TrafficCycleEditor({ uuid, open, onSaved }: { uuid: string; open: boolean; onSaved: () => void }) {
  const { t } = useTranslation();
  const [usage, setUsage] = useState<Usage | null>(null);
  const [upload, setUpload] = useState("");
  const [download, setDownload] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [confirmReset, setConfirmReset] = useState(false);
  const apply = (data: Usage) => {
    setUsage(data);
    // Keep exact bytes when opening/saving: rounded display values must never
    // silently change the accounting baseline.
    setUpload(`${data.upload} B`);
    setDownload(`${data.download} B`);
  };
  useEffect(() => {
    setUsage(null); setError(""); setConfirmReset(false);
    if (!open) return;
    let cancelled = false;
    request(uuid).then((data) => { if (!cancelled) apply(data); })
      .catch((err) => { if (!cancelled) setError(err.message); });
    return () => { cancelled = true; };
  }, [open, uuid]);

  const save = async (reset: boolean) => {
    const up = reset ? 0 : parseUsage(upload);
    const down = reset ? 0 : parseUsage(download);
    if (up === null || down === null) { setError(t("trafficCycle.invalid")); return; }
    setBusy(true); setError("");
    try {
      await request(uuid, reset ? "/reset" : "/set", { upload: up, download: down });
      // The write has succeeded, even if a subsequent refresh fails.
      apply({ upload: up, download: down, reset_at: new Date().toISOString() });
      setConfirmReset(false); onSaved(); toast.success(t("trafficCycle.saved"));
    } catch (err) { setError(err instanceof Error ? err.message : t("trafficCycle.failed")); }
    finally { setBusy(false); }
  };
  return <SettingCardCollapse title={t("trafficCycle.title")}>
    <Flex direction="column" gap="3">
      <Text size="2" color="gray">{t("trafficCycle.description")}</Text>
      {usage && <Text size="2">{t("trafficCycle.current")}: ↑ {formatBytes(usage.upload)} / ↓ {formatBytes(usage.download)}</Text>}
      {usage?.reset_at && <Text size="1" color="gray">{t("trafficCycle.since")}: {new Date(usage.reset_at).toLocaleString()}</Text>}
      <label><Text size="2">{t("trafficCycle.upload")}</Text><TextField.Root aria-label={t("trafficCycle.upload")} value={upload} onChange={(e) => setUpload(e.target.value)} disabled={busy || !usage} placeholder="12.5 GB" /></label>
      <label><Text size="2">{t("trafficCycle.download")}</Text><TextField.Root aria-label={t("trafficCycle.download")} value={download} onChange={(e) => setDownload(e.target.value)} disabled={busy || !usage} placeholder="80 GB" /></label>
      <Text size="1" color="gray">{t("trafficCycle.units")}</Text>
      {error && <Text role="alert" color="red" size="2">{error}</Text>}
      {confirmReset ? <Flex direction="column" gap="2">
        <Text size="2">{t("trafficCycle.confirm")}</Text>
        <Flex gap="2"><Button type="button" color="red" disabled={busy} onClick={() => void save(true)}>{t("trafficCycle.confirmButton")}</Button><Button type="button" variant="soft" disabled={busy} onClick={() => setConfirmReset(false)}>{t("common.cancel")}</Button></Flex>
      </Flex> : <Flex gap="2" wrap="wrap">
        <Button type="button" variant="soft" color="red" disabled={busy || !usage} onClick={() => setConfirmReset(true)}>{t("trafficCycle.reset")}</Button>
        <Button type="button" disabled={busy || !usage} onClick={() => void save(false)}>{t("trafficCycle.save")}</Button>
      </Flex>}
    </Flex>
  </SettingCardCollapse>;
}
