import { IconCloud, IconLogin, IconDownload } from "@tabler/icons-react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type TailscaleDevice,
  type TailscaleStatus,
  authenticateTailscale,
  getTailnetDevices,
  getTailscaleStatus,
  installTailscale,
  setTailscaleConfig,
} from "@/api/tailscale"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/page-header"

export function TailscaleVpnPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const { data: status, isLoading } = useQuery({
    queryKey: ["tailscale", "status"],
    queryFn: getTailscaleStatus,
  })

  const { data: devices } = useQuery({
    queryKey: ["tailscale", "devices"],
    queryFn: getTailnetDevices,
    enabled: status?.installed && status?.connected,
  })

  const [magicDns, setMagicDns] = useState(false)
  const [autoStart, setAutoStart] = useState(false)
  const [installing, setInstalling] = useState(false)
  const [authenticating, setAuthenticating] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!status) return
    setMagicDns(status.magicDns)
    setAutoStart(status.autoStart)
  }, [status])

  const handleInstall = async () => {
    try {
      setInstalling(true)
      const res = await installTailscale()
      if (res.success) {
        toast.success(t("settings.tailscale.install_success"))
        queryClient.invalidateQueries({ queryKey: ["tailscale"] })
      } else {
        toast.error(res.message)
      }
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : t("settings.tailscale.install_error"),
      )
    } finally {
      setInstalling(false)
    }
  }

  const handleAuthenticate = async () => {
    try {
      setAuthenticating(true)
      const res = await authenticateTailscale()
      if (res.url) {
        window.open(res.url, "_blank")
        toast.info(t("settings.tailscale.auth_started"))
      }
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : t("settings.tailscale.auth_error"),
      )
    } finally {
      setAuthenticating(false)
    }
  }

  const handleToggleMagicDns = async (checked: boolean) => {
    try {
      setSaving(true)
      await setTailscaleConfig({ magicDns: checked })
      setMagicDns(checked)
      queryClient.invalidateQueries({ queryKey: ["tailscale"] })
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.tailscale.toggle_error"),
      )
      setMagicDns(!checked)
    } finally {
      setSaving(false)
    }
  }

  const handleToggleAutoStart = async (checked: boolean) => {
    try {
      setSaving(true)
      await setTailscaleConfig({ autoStart: checked })
      setAutoStart(checked)
      queryClient.invalidateQueries({ queryKey: ["tailscale"] })
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.tailscale.toggle_error"),
      )
      setAutoStart(!checked)
    } finally {
      setSaving(false)
    }
  }

  const statusBadge = (s?: TailscaleStatus) => {
    if (!s?.installed) {
      return (
        <Badge className="bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400">
          {t("settings.tailscale.status_not_installed")}
        </Badge>
      )
    }
    if (s.connected) {
      return (
        <Badge className="bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">
          {t("settings.tailscale.status_connected")}
        </Badge>
      )
    }
    return (
      <Badge className="bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-400">
        {t("settings.tailscale.status_disconnected")}
      </Badge>
    )
  }

  if (isLoading) {
    return (
      <div className="flex h-full flex-col">
        <PageHeader title={t("settings.tailscale.title")} />
        <div className="flex-1 overflow-auto p-3 lg:p-6">
          <div className="text-muted-foreground py-6 text-sm">
            {t("labels.loading")}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("settings.tailscale.title")} />
      <div className="flex-1 overflow-auto p-3 lg:p-6">
        <div className="mx-auto w-full max-w-[1000px] space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <IconCloud className="size-5" />
                {t("settings.tailscale.title")}
              </CardTitle>
              <CardDescription>
                {t("settings.tailscale.description")}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between">
                <Label>{t("settings.tailscale.status")}</Label>
                {statusBadge(status)}
              </div>

              {!status?.installed && (
                <Button onClick={handleInstall} disabled={installing}>
                  <IconDownload className="size-4" />
                  {installing
                    ? t("settings.tailscale.installing")
                    : t("settings.tailscale.install")}
                </Button>
              )}

              {status?.installed && !status?.connected && (
                <Button
                  onClick={handleAuthenticate}
                  disabled={authenticating}
                >
                  <IconLogin className="size-4" />
                  {authenticating
                    ? t("settings.tailscale.authenticating")
                    : t("settings.tailscale.authenticate")}
                </Button>
              )}

              {status?.connected && (
                <>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-1">
                      <span className="text-muted-foreground text-sm">
                        {t("settings.tailscale.ip_address")}
                      </span>
                      <p className="font-mono text-sm">{status.ip || "—"}</p>
                    </div>
                    <div className="space-y-1">
                      <span className="text-muted-foreground text-sm">
                        {t("settings.tailscale.hostname")}
                      </span>
                      <p className="text-sm">{status.hostname || "—"}</p>
                    </div>
                  </div>

                  {status.tailnetUrl && (
                    <div className="space-y-1">
                      <span className="text-muted-foreground text-sm">
                        {t("settings.tailscale.tailnet_url")}
                      </span>
                      <p className="font-mono text-sm">{status.tailnetUrl}</p>
                    </div>
                  )}
                </>
              )}

              {status?.installed && (
                <>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="tailscale-magic-dns">
                      {t("settings.tailscale.magic_dns")}
                    </Label>
                    <Switch
                      id="tailscale-magic-dns"
                      checked={magicDns}
                      onCheckedChange={handleToggleMagicDns}
                      disabled={saving}
                    />
                  </div>

                  <div className="flex items-center justify-between">
                    <Label htmlFor="tailscale-auto-start">
                      {t("settings.tailscale.auto_start")}
                    </Label>
                    <Switch
                      id="tailscale-auto-start"
                      checked={autoStart}
                      onCheckedChange={handleToggleAutoStart}
                      disabled={saving}
                    />
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {status?.installed && status?.connected && (
            <Card>
              <CardHeader>
                <CardTitle>{t("settings.tailscale.devices")}</CardTitle>
                <CardDescription>
                  {t("settings.tailscale.devices_description")}
                </CardDescription>
              </CardHeader>
              <CardContent>
                {!devices || devices.length === 0 ? (
                  <div className="text-muted-foreground py-4 text-center text-sm">
                    {t("settings.tailscale.no_devices")}
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b text-left text-muted-foreground">
                          <th className="pb-2 pr-4 font-medium">
                            {t("settings.tailscale.device_hostname")}
                          </th>
                          <th className="pb-2 pr-4 font-medium">IP</th>
                          <th className="pb-2 pr-4 font-medium">OS</th>
                          <th className="pb-2 font-medium">
                            {t("settings.tailscale.device_status")}
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {devices.map((device: TailscaleDevice) => (
                          <tr
                            key={device.id}
                            className="border-b last:border-0"
                          >
                            <td className="py-2 pr-4">{device.hostname}</td>
                            <td className="py-2 pr-4 font-mono">{device.ip}</td>
                            <td className="py-2 pr-4">{device.os}</td>
                            <td className="py-2">
                              <Badge
                                className={
                                  device.online
                                    ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                                    : "bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-400"
                                }
                              >
                                {device.online
                                  ? t("settings.tailscale.online")
                                  : t("settings.tailscale.offline")}
                              </Badge>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}
