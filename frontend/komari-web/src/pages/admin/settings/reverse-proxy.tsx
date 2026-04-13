import Loading from "@/components/loading";
import { SettingCard, SettingCardLabel } from "@/components/admin/SettingCard";
import { useSettings } from "@/lib/api";
import {
  CLOUDFLARED_STOP_CONFIRM_TEXT,
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
  const [showToken, setShowToken] = React.useState(false);
  const [stopDialogOpen, setStopDialogOpen] = React.useState(false);
  const [currentPassword, setCurrentPassword] = React.useState("");
  const [confirmText, setConfirmText] = React.useState("");

  const refreshStatus = React.useCallback(async (silent = false) => {
    if (!silent) {
      setRefreshing(true);
    }
    try {
      const nextStatus = await getCloudflaredStatus();
      setStatus({
        ...nextStatus,
        logs: Array.isArray(nextStatus.logs) ? nextStatus.logs : [],
      });
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
  }, []);

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
      setToken("");
      setShowToken(false);
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
  const canStart = status.installed && (status.tokenStored || token.trim().length > 0);
  const stopConfirmSatisfied = disablePasswordDoubleCheck
    ? confirmText.trim() === CLOUDFLARED_STOP_CONFIRM_TEXT
    : currentPassword.trim().length > 0;

  return (
    <Flex direction="column" gap="4">
      <SettingCardLabel>
        {t("settings.reverse_proxy.title", "反向代理")}
      </SettingCardLabel>

      <SettingCard
        title={t("settings.reverse_proxy.cloudflare_title", "Cloudflare Tunnel")}
        description={t(
          "settings.reverse_proxy.cloudflare_description",
          "在 Komari-cloudflared 中直接启动和管理 cloudflared，交互体验参考 Uptime Kuma 的 Reverse Proxy 设置页。",
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
                "已检测到环境变量 KOMARI_CLOUDFLARED_TOKEN，Komari-cloudflared 重启后会优先用它恢复 cloudflared。",
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
              placeholder={
                status.tokenStored
                  ? t(
                      "settings.reverse_proxy.cloudflare_token_masked",
                      "••••••••••••••••（已保存，不会回显）",
                    )
                  : t(
                      "settings.reverse_proxy.cloudflare_token_placeholder",
                      "请输入 Cloudflare Tunnel Token",
                    )
              }
              onChange={(event) => {
                setToken(event.target.value);
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
                "已保存的 Token 只会在后端加密持久化，不会再明文回传到浏览器。未修改输入框时，点击 Start 会直接使用已保存的 Token。",
              )}
            </Text>
            {status.tokenStored && !status.running ? (
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
                  {t("settings.reverse_proxy.remove_token", "移除已保存的 Token")}
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
                disabled={submitting || !canStart}
                onClick={() =>
                  withSubmit(
                    async () => {
                      const nextToken = token.trim();
                      if (nextToken) {
                        await saveCloudflaredToken(nextToken);
                      }
                      return startCloudflared(nextToken);
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

            <Button variant="ghost" disabled={refreshing} onClick={() => refreshStatus()}>
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

          {status.logs.length > 0 ? (
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
                "非 Docker 环境需要先在宿主机安装 cloudflared；本项目的 komari-cloudflared Docker 镜像已内置 cloudflared。",
              )}
            </Text>
          ) : null}
        </Flex>
      </SettingCard>

      <Dialog.Root
        open={stopDialogOpen}
        onOpenChange={(open) => {
          setStopDialogOpen(open);
          if (!open) {
            setCurrentPassword("");
            setConfirmText("");
          }
        }}
      >
        <Dialog.Content maxWidth="520px">
          <Dialog.Title>
            {t("settings.reverse_proxy.stop", "Stop")} cloudflared
          </Dialog.Title>
          <Dialog.Description>
            {t(
              "settings.reverse_proxy.stop_warning",
              "如果你当前就是通过 Cloudflare Tunnel 访问这个页面，停止后本次连接可能会立即中断。",
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
          ) : (
            <Flex direction="column" gap="2" className="mt-4">
              <label className="text-sm font-medium" htmlFor="cloudflaredStopConfirmText">
                {t("settings.reverse_proxy.stop_confirm_label", "确认短语")}
              </label>
              <Text size="2" color="gray">
                {t(
                  "settings.reverse_proxy.stop_confirm_help",
                  "当前已禁用密码登录。请输入 STOP CLOUDFLARED 以确认停止 cloudflared。",
                )}
              </Text>
              <TextField.Root
                id="cloudflaredStopConfirmText"
                value={confirmText}
                onChange={(event) => setConfirmText(event.target.value)}
                autoComplete="off"
              />
            </Flex>
          )}

          <Flex justify="end" gap="2" className="mt-6">
            <Dialog.Close>
              <Button variant="soft">{t("cancel", "取消")}</Button>
            </Dialog.Close>
            <Button
              color="red"
              disabled={submitting || !stopConfirmSatisfied}
              onClick={async () => {
                await withSubmit(
                  () => stopCloudflared(currentPassword, confirmText),
                  t("settings.reverse_proxy.stop_success", "cloudflared 已停止"),
                );
                setCurrentPassword("");
                setConfirmText("");
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
