import { IconCopy, IconDeviceMobile } from "@tabler/icons-react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"
import { QRCodeSVG } from "qrcode.react"

import {
  type ConnectedDevice,
  getConnectedDevices,
  getMobileAccessStatus,
  setMobileAccessConfig,
} from "@/api/mobile"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"

import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/page-header"

export function MobileAccessPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const { data: config, isLoading } = useQuery({
    queryKey: ["mobile", "status"],
    queryFn: getMobileAccessStatus,
  })

  const { data: devices } = useQuery({
    queryKey: ["mobile", "devices"],
    queryFn: getConnectedDevices,
  })

  const [enabled, setEnabled] = useState(false)
  const [connectionMethod, setConnectionMethod] = useState<
    "tailscale" | "lan"
  >("tailscale")
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!config) return
    setEnabled(config.enabled)
    setConnectionMethod(config.connectionMethod)
  }, [config])

  const handleToggleEnabled = async (checked: boolean) => {
    try {
      setSaving(true)
      await setMobileAccessConfig({ enabled: checked })
      setEnabled(checked)
      queryClient.invalidateQueries({ queryKey: ["mobile"] })
      toast.success(
        checked
          ? t("settings.mobile.enabled_success")
          : t("settings.mobile.disabled_success"),
      )
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.mobile.toggle_error"),
      )
      setEnabled(!checked)
    } finally {
      setSaving(false)
    }
  }

  const handleConnectionMethodChange = async (value: string) => {
    const method = value as "tailscale" | "lan"
    try {
      setSaving(true)
      await setMobileAccessConfig({ connectionMethod: method })
      setConnectionMethod(method)
      queryClient.invalidateQueries({ queryKey: ["mobile"] })
      toast.success(t("settings.mobile.method_changed"))
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.mobile.toggle_error"),
      )
    } finally {
      setSaving(false)
    }
  }

  const handleCopyUrl = () => {
    if (!config?.accessUrl) return
    navigator.clipboard.writeText(config.accessUrl)
    toast.success(t("settings.mobile.url_copied"))
  }

  if (isLoading) {
    return (
      <div className="flex h-full flex-col">
        <PageHeader title={t("settings.mobile.title")} />
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
      <PageHeader title={t("settings.mobile.title")} />
      <div className="flex-1 overflow-auto p-3 lg:p-6">
        <div className="mx-auto w-full max-w-[1000px] space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <IconDeviceMobile className="size-5" />
                {t("settings.mobile.title")}
              </CardTitle>
              <CardDescription>
                {t("settings.mobile.description")}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between">
                <Label htmlFor="mobile-enabled">
                  {t("settings.mobile.enable")}
                </Label>
                <Switch
                  id="mobile-enabled"
                  checked={enabled}
                  onCheckedChange={handleToggleEnabled}
                  disabled={saving}
                />
              </div>

              {enabled && (
                <>
                  <div className="space-y-2">
                    <Label>{t("settings.mobile.connection_method")}</Label>
                    <Select
                      value={connectionMethod}
                      onValueChange={handleConnectionMethodChange}
                      disabled={saving}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="tailscale">
                          {t("settings.mobile.method_tailscale")}
                        </SelectItem>
                        <SelectItem value="lan">
                          {t("settings.mobile.method_lan")}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label>{t("settings.mobile.access_url")}</Label>
                    <div className="flex gap-2">
                      <Input
                        readOnly
                        value={config?.accessUrl ?? ""}
                        className="flex-1 font-mono text-sm"
                      />
                      <Button
                        variant="outline"
                        size="icon"
                        onClick={handleCopyUrl}
                      >
                        <IconCopy className="size-4" />
                      </Button>
                    </div>
                  </div>

                  {config?.accessUrl && (
                    <div className="flex flex-col items-center gap-4">
                      <Label>{t("settings.mobile.qr_code")}</Label>
                      <div className="rounded-lg bg-white p-4">
                        <QRCodeSVG
                          value={config.accessUrl}
                          size={180}
                          level="M"
                        />
                      </div>
                    </div>
                  )}
                </>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("settings.mobile.devices")}</CardTitle>
              <CardDescription>
                {t("settings.mobile.devices_description")}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {!enabled ? (
                <div className="text-muted-foreground py-4 text-center text-sm">
                  {t("settings.mobile.enable_first")}
                </div>
              ) : !devices || devices.length === 0 ? (
                <div className="text-muted-foreground py-4 text-center text-sm">
                  {t("settings.mobile.no_devices")}
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b text-left text-muted-foreground">
                        <th className="pb-2 pr-4 font-medium">
                          {t("settings.mobile.device_name")}
                        </th>
                        <th className="pb-2 pr-4 font-medium">
                          {t("settings.mobile.device_type")}
                        </th>
                        <th className="pb-2 pr-4 font-medium">IP</th>
                        <th className="pb-2 font-medium">
                          {t("settings.mobile.last_connected")}
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {devices.map((device: ConnectedDevice) => (
                        <tr
                          key={device.id}
                          className="border-b last:border-0"
                        >
                          <td className="py-2 pr-4">{device.name}</td>
                          <td className="py-2 pr-4">{device.type}</td>
                          <td className="py-2 pr-4 font-mono">{device.ip}</td>
                          <td className="py-2 text-muted-foreground">
                            {device.lastConnected}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
