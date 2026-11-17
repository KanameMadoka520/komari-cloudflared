import { useEffect, useState } from "react";
import { Badge, Button, Dialog, Flex, Select, Text, TextField } from "@radix-ui/themes";
import { Copy, Eye, RefreshCw, Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

interface Log {
  id: number;
  ip: string;
  uuid: string;
  message: string;
  msg_type: string;
  time: string;
}
const TYPES = ["info", "warn", "error", "login", "logout", "terminal", "renewal"];

export default function LogPage() {
  const { t } = useTranslation();
  const [logs, setLogs] = useState<Log[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const [total, setTotal] = useState(0);
  const [msgType, setMsgType] = useState("");
  const [search, setSearch] = useState("");
  const [appliedSearch, setAppliedSearch] = useState("");
  const [refresh, setRefresh] = useState(0);
  const [selected, setSelected] = useState<Log | null>(null);

  useEffect(() => {
    const timer = setTimeout(() => {
      setAppliedSearch(search.trim());
      setPage(1);
    }, 280);
    return () => clearTimeout(timer);
  }, [search]);

  useEffect(() => {
    const controller = new AbortController();
    const query = new URLSearchParams({ limit: String(limit), page: String(page) });
    if (msgType) query.set("msg_type", msgType);
    if (appliedSearch) query.set("q", appliedSearch);
    setLoading(true);
    setError(null);
    fetch(`/api/admin/logs?${query}`, { signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error(t("logs.load_failed"));
        return response.json();
      })
      .then(({ data }) => {
        if (controller.signal.aborted) return;
        setLogs(data.logs ?? []);
        setTotal(data.total ?? 0);
        const pages = Math.max(1, Math.ceil((data.total ?? 0) / limit));
        if (page > pages) setPage(pages);
      })
      .catch((err: unknown) => {
        if (!controller.signal.aborted) setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, [limit, page, msgType, appliedSearch, refresh, t]);

  const typeLabel = (type: string) => t(`logs.type.${type}`, type || "-");
  const formatTime = (value: string) => {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
  };
  const copy = async (log: Log) => {
    try {
      await navigator.clipboard.writeText([
        `ID: ${log.id}`, `${t("logs.field_type")}: ${typeLabel(log.msg_type)}`,
        `${t("logs.field_time")}: ${formatTime(log.time)}`, `IP: ${log.ip || "-"}`,
        `UUID: ${log.uuid || "-"}`, "", log.message,
      ].join("\n"));
      toast.success(t("common.copied", "Copied"));
    } catch { toast.error(t("common.copy_failed", "Copy failed")); }
  };
  const pages = Math.max(1, Math.ceil(total / limit));
  return (
    <div className="km-page km-page-admin-log space-y-4">
      <Flex justify="between" align="center">
        <Text size="5" weight="bold">{t("logs.title")}</Text>
        <Button variant="soft" disabled={loading} onClick={() => setRefresh((value) => value + 1)}>
          <RefreshCw size={15} className={loading ? "animate-spin" : undefined} />{t("common.refresh", "Refresh")}
        </Button>
      </Flex>
      <Flex gap="3" wrap="wrap" align="center">
        <TextField.Root value={search} onChange={(event) => setSearch(event.target.value)}
          placeholder={t("logs.search_placeholder")} aria-label={t("logs.search_placeholder")} className="w-full sm:max-w-80">
          <TextField.Slot><Search size={15} /></TextField.Slot>
        </TextField.Root>
        <Select.Root value={msgType || "all"} onValueChange={(value) => { setMsgType(value === "all" ? "" : value); setPage(1); }}>
          <Select.Trigger aria-label={t("logs.filter_type")} />
          <Select.Content>
            <Select.Item value="all">{t("logs.filter_all_types")}</Select.Item>
            {TYPES.map((type) => <Select.Item key={type} value={type}>{typeLabel(type)}</Select.Item>)}
          </Select.Content>
        </Select.Root>
        {(search || msgType) && <Button variant="ghost" onClick={() => { setSearch(""); setAppliedSearch(""); setMsgType(""); setPage(1); }}>{t("logs.clear_filters")}</Button>}
      </Flex>
      {error && <div role="alert" className="text-red-500">{error}</div>}
      <div className="overflow-x-auto" aria-busy={loading}>
        <Table>
          <TableHeader><TableRow>
            <TableHead>{t("logs.field_type")}</TableHead><TableHead>{t("logs.field_message")}</TableHead>
            <TableHead>IP</TableHead><TableHead>UUID</TableHead><TableHead>{t("logs.field_time")}</TableHead><TableHead />
          </TableRow></TableHeader>
          <TableBody>
            {logs.map((log) => <TableRow key={log.id}>
              <TableCell><Badge color={log.msg_type === "error" ? "red" : log.msg_type === "warn" ? "amber" : "blue"}>{typeLabel(log.msg_type)}</Badge></TableCell>
              <TableCell className="max-w-md"><div className="truncate" title={log.message}>{log.message}</div></TableCell>
              <TableCell>{log.ip || "-"}</TableCell><TableCell title={log.uuid}>{log.uuid ? `${log.uuid.slice(0, 8)}…` : "-"}</TableCell>
              <TableCell className="whitespace-nowrap">{formatTime(log.time)}</TableCell>
              <TableCell><Flex gap="2">
                <Button variant="ghost" aria-label={t("logs.detail_title")} onClick={() => setSelected(log)}><Eye size={15} /></Button>
                <Button variant="ghost" aria-label={t("common.copy", "Copy")} onClick={() => void copy(log)}><Copy size={15} /></Button>
              </Flex></TableCell>
            </TableRow>)}
            {!logs.length && <TableRow><TableCell colSpan={6} className="py-8 text-center">{loading ? t("common.loading", "Loading…") : t(appliedSearch || msgType ? "logs.no_matches" : "logs.empty")}</TableCell></TableRow>}
          </TableBody>
        </Table>
      </div>
      <Flex gap="3" align="center" justify="between" wrap="wrap">
        <Text size="2">{t("logs.showing", { from: total ? (page - 1) * limit + 1 : 0, to: Math.min(page * limit, total), total })}</Text>
        <Flex gap="3" align="center">
          <Select.Root value={String(limit)} onValueChange={(value) => { setLimit(Number(value)); setPage(1); }}>
            <Select.Trigger aria-label={t("logs.per_page")} />
            <Select.Content>{[10, 20, 50, 100].map((value) => <Select.Item key={value} value={String(value)}>{value}</Select.Item>)}</Select.Content>
          </Select.Root>
          <Button variant="soft" disabled={page <= 1 || loading} onClick={() => setPage((value) => value - 1)}>‹</Button>
          <Text size="2">{page} / {pages}</Text>
          <Button variant="soft" disabled={page >= pages || loading} onClick={() => setPage((value) => value + 1)}>›</Button>
        </Flex>
      </Flex>
      <Dialog.Root open={selected !== null} onOpenChange={(open) => { if (!open) setSelected(null); }}>
        <Dialog.Content maxWidth="720px"><Dialog.Title>{t("logs.detail_title")}</Dialog.Title>
          <Dialog.Description>{selected ? `${typeLabel(selected.msg_type)} · ${formatTime(selected.time)} · ${selected.ip || "-"} · ${selected.uuid || "-"}` : ""}</Dialog.Description>
          <pre className="my-4 max-h-96 overflow-auto whitespace-pre-wrap break-words">{selected?.message}</pre>
          <Flex justify="end" gap="3"><Button variant="soft" onClick={() => { if (selected) void copy(selected); }}>{t("common.copy", "Copy")}</Button><Dialog.Close><Button>{t("common.close", "Close")}</Button></Dialog.Close></Flex>
        </Dialog.Content>
      </Dialog.Root>
    </div>
  );
}
