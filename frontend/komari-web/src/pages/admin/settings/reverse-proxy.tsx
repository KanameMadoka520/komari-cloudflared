import Loading from "@/components/loading";
import { SettingCard, SettingCardLabel } from "@/components/admin/SettingCard";
import { useSettings } from "@/lib/api";
import {
  getCloudflaredStatus,
  removeCloudflaredToken,
  saveCloudflaredToken,
  startCloudflared,
  stopCloudflared,
  type CloudflaredStatus,
} from "@/lib/cloudflared";
import {
  Badge,
  Button,
  Dialog,
  Flex,
  IconButton,
  Text,
  TextArea,
  TextField,
} from "@radix-ui/themes";
import { Eye, EyeOff, Play, RefreshCw, Square } from "lucide-react";
import React from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

const emptyStatus: CloudflaredStatus = {
  installed: false,
  running: false,
  message: "",
  errorMessage: "",
  logs: [],
  token: "",
  tokenStored: false,
  envTokenPresent: false,
};

export default function ReverseProxySettings() {
  const { t } = useTranslation();
  const { settings, loading: settingsLoading, error: settingsError } = useSettings();
  const [status, setStatus] = React.useState<CloudflaredStatus>(emptyStatus);
  const [loading, setLoading] = React.useState(true);
  const [refreshing, setRefreshing] = React.useState(false);
  const [submitting, setSubmitting] = React.useState(false);
  const [token, setToken] = React.useState("");
  const [tokenDirty, setTokenDirty] = React.useState(false);
  const [showToken, setShowToken] = React.useState(false);
  const [stopDialogOpen, setStopDialogOpen] = React.useState(false);
  const [currentPassword, setCurrentPassword] = React.useState("");

  const refreshStatus = React.useCallback(
    async (silent = false) => {
      if (!silent) {
        setRefreshing(true);
      }
      try {
        const nextStatus = await getCloudflaredStatus();
        setStatus({
          ...nextStatus,
          logs: Array.isArray(nextStatus.logs) ? nextStatus.logs : [],
        });
        if (!tokenDirty || nextStatus.running) {
          setToken(nextStatus.token || "");
          setTokenDirty(false);
        }
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to fetch cloudflared status";
        if (!silent) {
          toast.error(message);
        }
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [],
  );

  React.useEffect(() => {
    refreshStatus();
    const timer = window.setInterval(() => {
      refreshStatus(true);
    }, 5000);
    return () => {
      window.clearInterval(timer);
    };
  }, [refreshStatus]);

  const withSubmit = async (task: () => Promise<CloudflaredStatus>, successMessage: string) => {
    setSubmitting(true);
    try {
      const nextStatus = await task();
      setStatus({
        ...nextStatus,
        logs: Array.isArray(nextStatus.logs) ? nextStatus.logs : [],
      });
      setToken(nextStatus.token || "");
      setTokenDirty(false);
      toast.success(successMessage);
      return nextStatus;
    } catch (error) {
      const message =
        error instanceof Error ? error.message : t("settings.settings_save_failed");
      toast.error(message);
      throw error;
    } finally {
      setSubmitting(false);
    }
  };

  if (settingsLoading || loading) {
    return <Loading />;
  }

  if (settingsError) {
    return <Text color="red">{settingsError}</Text>;
  }

  const disablePasswordDoubleCheck = Boolean(settings.disable_password_login);

  return (
    <Flex direction="column" gap="4">
      <SettingCardLabel>
        {t("settings.reverse_proxy.title", "Reverse Proxy")}
      </SettingCardLabel>

      <SettingCard
        title={t("settings.reverse_proxy.cloudflare_title", "Cloudflare Tunnel")}
        description={t(
          "settings.reverse_proxy.cloudflare_description",
          "Start and manage cloudflared inside Komari, similar to Uptime Kuma's reverse proxy settings page.",
        )}
        direction="column"
      >
        <Flex direction="column" gap="3" className="w-full pt-3">
          <Flex gap="3" wrap="wrap">
            <StatusLine
              label="cloudflared"
              ok={status.installed}
              okText={t("settings.reverse_proxy.installed", "已安装")}
              failText={t("settings.reverse_proxy.not_installed", "未安装")}
            />
            <StatusLine
              label={t("settings.reverse_proxy.status", "状态")}
              ok={status.running}
              okText={t("common.running", "运行中")}
              failText={t("settings.reverse_proxy.not_running", "已停止")}
            />
            {status.pid ? (
              <Badge variant="soft" color="gray">
                PID: {status.pid}
              </Badge>
            ) : null}
          </Flex>

          {status.binaryPath ? (
            <Text size="2" color="gray">
              Binary: <code>{status.binaryPath}</code>
            </Text>
          ) : null}

          {status.envTokenPresent ? (
            <Text size="2" color="gray">
              {t(
                "settings.reverse_proxy.env_token_hint",
                "检测到 KOMARI_CLOUDFLARED_TOKEN 环境变量，重启 Komari 后会按该 Token 自动恢复启动。",
              )}
            </Text>
          ) : null}

          <div>
            <label className="mb-2 block text-sm font-medium" htmlFor="cloudflareTunnelToken">
              {t("settings.reverse_proxy.cloudflare_token", "Cloudflare Tunnel 令牌")}
            </label>
            <TextField.Root
              id="cloudflareTunnelToken"
              type={showToken ? "text" : "password"}
              value={token}
              onChange={(event) => {
                setToken(event.target.value);
                setTokenDirty(true);
              }}
              autoComplete="new-password"
              disabled={status.running}
            >
              <TextField.Slot side="right">
                <IconButton
                  type="button"
                  variant="ghost"
                  onClick={() => setShowToken((prev) => !prev)}
                >
                  {showToken ? <EyeOff size={16} /> : <Eye size={16} />}
                </IconButton>
              </TextField.Slot>
            </TextField.Root>
            <Text size="2" color="gray" className="mt-2 block">
              {t(
                "settings.reverse_proxy.cloudflare_token_help",
                "默认情况下，配置 Token 后重新启动 Komari 会自动恢复 cloudflared。",
              )}
            </Text>
            {token && !status.running ? (
              <Text size="2" color="gray" className="mt-1 block">
                <button
                  type="button"
                  className="cursor-pointer underline"
                  onClick={() =>
                    withSubmit(
                      () => removeCloudflaredToken(),
                      t("settings.reverse_proxy.remove_token_success", "Cloudflare Tunnel Token 已移除"),
                    )
                  }
                >
                  {t("settings.reverse_proxy.remove_token", "Remove Token")}
                </button>
              </Text>
            ) : null}
            <Text size="2" color="gray" className="mt-1 block">
              {t("settings.reverse_proxy.guide_prefix", "不知道如何获取 Token？请阅读指南：")}{" "}
              <a
                href="https://github.com/louislam/uptime-kuma/wiki/Reverse-Proxy-with-Cloudflare-Tunnel"
                target="_blank"
                rel="noopener noreferrer"
              >
                https://github.com/louislam/uptime-kuma/wiki/Reverse-Proxy-with-Cloudflare-Tunnel
              </a>
            </Text>
          </div>

          <Flex gap="2" wrap="wrap">
            {!status.running ? (
              <Button
                disabled={submitting || !status.installed || !token.trim()}
                onClick={() =>
                  withSubmit(
                    async () => {
                      await saveCloudflaredToken(token);
                      return startCloudflared(token);
                    },
                    t("settings.reverse_proxy.start_success", "cloudflared 已启动"),
                  )
                }
              >
                <Play size={16} />
                {t("settings.reverse_proxy.start", "Start")} cloudflared
              </Button>
            ) : (
              <Button color="red" disabled={submitting} onClick={() => setStopDialogOpen(true)}>
                <Square size={16} />
                {t("settings.reverse_proxy.stop", "Stop")} cloudflared
              </Button>
            )}

            <Button
              variant="ghost"
              disabled={refreshing}
              onClick={() => refreshStatus()}
            >
              <RefreshCw size={16} className={refreshing ? "animate-spin" : ""} />
              {t("common.refresh", "刷新")}
            </Button>
          </Flex>

          {status.message ? (
            <Text size="2" color="gray">
              {t("settings.reverse_proxy.latest_message", "最近状态")}: {status.message}
            </Text>
          ) : null}

          {status.errorMessage ? (
            <div>
              <label className="mb-2 block text-sm font-medium">
                {t("settings.reverse_proxy.error_message", "错误信息")}
              </label>
              <TextArea value={status.errorMessage} readOnly rows={4} />
            </div>
          ) : null}

          {Array.isArray(status.logs) && status.logs.length > 0 ? (
            <div>
              <label className="mb-2 block text-sm font-medium">
                {t("settings.reverse_proxy.recent_logs", "近期日志")}
              </label>
              <TextArea value={status.logs.join("\n")} readOnly rows={10} />
            </div>
          ) : null}

          {!status.installed ? (
            <Text size="2" color="gray">
              {t(
                "settings.reverse_proxy.install_hint",
                "非 Docker 环境需要先在宿主机安装 cloudflared；Docker 镜像版 komari-cloudflared 已内置 cloudflared。",
              )}
            </Text>
          ) : null}
        </Flex>
      </SettingCard>

      <Dialog.Root open={stopDialogOpen} onOpenChange={setStopDialogOpen}>
        <Dialog.Content maxWidth="520px">
          <Dialog.Title>
            {t("settings.reverse_proxy.stop", "Stop")} cloudflared
          </Dialog.Title>
          <Dialog.Description>
            {t(
              "settings.reverse_proxy.stop_warning",
              "如果你当前就是通过 Cloudflare Tunnel 访问这个页面，停止后当前连接可能会中断。",
            )}
          </Dialog.Description>

          {!disablePasswordDoubleCheck ? (
            <Flex direction="column" gap="2" className="mt-4">
              <label className="text-sm font-medium" htmlFor="cloudflaredCurrentPassword">
                {t("account.current_password", "当前密码")}
              </label>
              <TextField.Root
                id="cloudflaredCurrentPassword"
                type="password"
                value={currentPassword}
                onChange={(event) => setCurrentPassword(event.target.value)}
              />
            </Flex>
          ) : null}

          <Flex justify="end" gap="2" className="mt-6">
            <Dialog.Close>
              <Button variant="soft">{t("cancel", "取消")}</Button>
            </Dialog.Close>
            <Button
              color="red"
              disabled={submitting}
              onClick={async () => {
                await withSubmit(
                  () => stopCloudflared(currentPassword),
                  t("settings.reverse_proxy.stop_success", "cloudflared 已停止"),
                );
                setCurrentPassword("");
                setStopDialogOpen(false);
              }}
            >
              {t("settings.reverse_proxy.stop", "Stop")} cloudflared
            </Button>
          </Flex>
        </Dialog.Content>
      </Dialog.Root>
    </Flex>
  );
}

function StatusLine({
  label,
  ok,
  okText,
  failText,
}: {
  label: string;
  ok: boolean;
  okText: string;
  failText: string;
}) {
  return (
    <Flex gap="2" align="center">
      <Text size="2" color="gray">
        {label}:
      </Text>
      <Badge variant="soft" color={ok ? "green" : "red"}>
        {ok ? okText : failText}
      </Badge>
    </Flex>
  );
}
