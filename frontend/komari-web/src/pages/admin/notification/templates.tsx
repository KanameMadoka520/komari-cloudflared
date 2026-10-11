import Loading from "@/components/loading";
import { updateSettings } from "@/lib/api";
import { Button, Flex, Select, Text, TextArea, TextField } from "@radix-ui/themes";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { DEFAULT_NOTIFICATION_TIMEZONE, NOTIFICATION_TIMEZONES, timezoneMessageKey } from "./timezones";

type Preview = { title: string; text: string };
type TemplateItem = {
  id: string;
  title: string;
  body: string;
  node: string;
  has_node: boolean;
  customized: boolean;
  placeholders: string[];
  node_placeholders: string[];
  preview: Preview;
};

async function readJson<T>(response: Response): Promise<T> {
  const payload = await response.json().catch(() => ({}));
  if (!response.ok || payload.status === "error") {
    throw new Error(payload.message || response.statusText);
  }
  return payload.data as T;
}

export default function NotificationTemplatesPage() {
  const { t } = useTranslation();
  const [items, setItems] = useState<TemplateItem[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [node, setNode] = useState("");
  const [preview, setPreview] = useState<Preview>({ title: "", text: "" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [timezone, setTimezone] = useState(DEFAULT_NOTIFICATION_TIMEZONE);
  const [customZone, setCustomZone] = useState("");
  const [savingZone, setSavingZone] = useState(false);
  const [error, setError] = useState("");

  const selected = useMemo(
    () => items.find((item) => item.id === selectedId),
    [items, selectedId]
  );

  const load = async (preferId?: string) => {
    const list = await readJson<TemplateItem[]>(await fetch("/api/admin/notification/templates"));
    setItems(list);
    const next = list.find((item) => item.id === (preferId || selectedId)) || list[0];
    if (next) {
      setSelectedId(next.id);
      setTitle(next.title);
      setBody(next.body);
      setNode(next.node || "");
      setPreview(next.preview);
    }
    return next;
  };

  useEffect(() => {
    load().catch((err: Error) => setError(err.message)).finally(() => setLoading(false));
    fetch("/api/admin/settings")
      .then((response) => response.json())
      .then((payload) => {
        const stored = String(payload?.data?.notification_timezone || "").trim();
        if (stored) setTimezone(stored);
      })
      .catch(() => undefined);
    // initial load only
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!selectedId || loading) return;
    const timer = window.setTimeout(() => {
      fetch("/api/admin/notification/templates/preview", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: selectedId, title, body, node }),
      })
        .then((response) => readJson<Preview>(response))
        .then(setPreview)
        .catch(() => undefined);
    }, 250);
    return () => window.clearTimeout(timer);
  }, [selectedId, title, body, node, timezone, loading]);

  const saveTimezone = async (value: string) => {
    const next = value.trim() || DEFAULT_NOTIFICATION_TIMEZONE;
    setSavingZone(true);
    try {
      await updateSettings({ notification_timezone: next });
      setTimezone(next);
      setCustomZone("");
      toast.success(t("settings.settings_saved"));
    } catch (err) {
      const message = err instanceof Error ? err.message : "";
      toast.error(message.includes("invalid notification timezone") ? t("notification.templates.timezone_invalid") : (message || t("common.error")));
    } finally {
      setSavingZone(false);
    }
  };

  const choose = (id: string) => {
    const item = items.find((entry) => entry.id === id);
    if (!item) return;
    setSelectedId(id);
    setTitle(item.title);
    setBody(item.body);
    setNode(item.node || "");
    setPreview(item.preview);
  };

  const save = async () => {
    setSaving(true);
    try {
      await readJson(await fetch("/api/admin/notification/templates", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: selectedId, title, body, node }),
      }));
      await load(selectedId);
      toast.success(t("common.updated_successfully"));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setSaving(false);
    }
  };

  const reset = async () => {
    setSaving(true);
    try {
      await readJson(await fetch("/api/admin/notification/templates/reset", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: selectedId }),
      }));
      await load(selectedId);
      toast.success(t("notification.templates.reset_done"));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <Loading />;
  if (error) return <Text color="red">{error}</Text>;

  return (
    <div className="km-page-admin-notification-templates flex flex-col gap-4 p-1 md:p-4">
      <div>
        <h1 className="text-2xl font-semibold">{t("notification.templates.title")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t("notification.templates.description")}</p>
      </div>
      <div className="flex flex-col gap-2 rounded-xl border p-3">
        <div className="text-sm font-medium">{t("notification.templates.timezone")}</div>
        <p className="text-sm text-muted-foreground">{t("notification.templates.timezone_description")}</p>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Select.Root value={timezone} onValueChange={saveTimezone} disabled={savingZone}>
            <Select.Trigger className="sm:w-72" />
            <Select.Content>
              {(NOTIFICATION_TIMEZONES as readonly string[]).includes(timezone) ? null : (
                <Select.Item value={timezone}>{timezone}</Select.Item>
              )}
              {NOTIFICATION_TIMEZONES.map((zone) => (
                <Select.Item key={zone} value={zone}>
                  {t(timezoneMessageKey(zone), { defaultValue: zone })}
                </Select.Item>
              ))}
            </Select.Content>
          </Select.Root>
          <TextField.Root
            value={customZone}
            placeholder={t("notification.templates.timezone_custom_placeholder")}
            onChange={(event) => setCustomZone(event.target.value)}
          />
          <Button variant="soft" disabled={savingZone || customZone.trim() === ""} onClick={() => saveTimezone(customZone)}>
            {t("notification.templates.timezone_custom")}
          </Button>
        </div>
      </div>
      <div className="grid gap-4 lg:grid-cols-[240px_minmax(0,1fr)]">
        <div className="flex flex-col gap-2">
          <Select.Root value={selectedId} onValueChange={choose}>
            <Select.Trigger className="lg:hidden" />
            <Select.Content>
              {items.map((item) => (
                <Select.Item key={item.id} value={item.id}>
                  {t(`notification.templates.items.${item.id}`)}
                </Select.Item>
              ))}
            </Select.Content>
          </Select.Root>
          <div className="hidden flex-col gap-1 lg:flex">
            {items.map((item) => (
              <button
                key={item.id}
                type="button"
                onClick={() => choose(item.id)}
                className={`rounded-md px-3 py-2 text-left text-sm ${item.id === selectedId ? "bg-accent font-medium" : "hover:bg-accent/50"}`}
              >
                {t(`notification.templates.items.${item.id}`)}
                {item.customized ? <span className="ml-2 text-xs text-muted-foreground">{t("notification.templates.customized")}</span> : null}
              </button>
            ))}
          </div>
        </div>
        {selected ? (
          <div className="grid gap-4 xl:grid-cols-2">
            <Flex direction="column" gap="3">
              <label className="flex flex-col gap-1 text-sm">
                {t("notification.templates.field_title")}
                <TextField.Root value={title} onChange={(event) => setTitle(event.target.value)} />
              </label>
              <label className="flex flex-col gap-1 text-sm">
                {t("notification.templates.field_body")}
                <TextArea value={body} onChange={(event) => setBody(event.target.value)} rows={12} />
              </label>
              {selected.has_node ? (
                <label className="flex flex-col gap-1 text-sm">
                  {t("notification.templates.field_node")}
                  <TextArea value={node} onChange={(event) => setNode(event.target.value)} rows={5} />
                </label>
              ) : null}
              <Text size="1" color="gray">
                {t("notification.templates.placeholders")}
                {(selected.placeholders || []).concat(selected.node_placeholders || []).map((name) => ` {{${name}}}`).join("")}
              </Text>
              <Flex gap="2">
                <Button onClick={save} disabled={saving}>{t("common.save")}</Button>
                <Button variant="soft" color="gray" onClick={reset} disabled={saving || !selected.customized}>
                  {t("notification.templates.reset")}
                </Button>
              </Flex>
            </Flex>
            <div className="rounded-xl border bg-muted/40 p-4">
              <div className="mb-3 text-sm text-muted-foreground">{t("notification.templates.preview")}</div>
              <div className="max-w-xl rounded-2xl bg-background px-4 py-3 shadow-sm">
                <div className="whitespace-pre-wrap text-base font-semibold leading-6">{preview.title}</div>
                <div className="mt-3 whitespace-pre-wrap text-sm leading-6">{preview.text}</div>
              </div>
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}
