import { IconRefresh, IconSettings } from "@tabler/icons-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

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

export function AdvancedPage() {
  const { t } = useTranslation()

  const [terminalEnabled, setTerminalEnabled] = useState(false)
  const [shellPath, setShellPath] = useState("/bin/bash")
  const [portableMode, setPortableMode] = useState(false)
  const [autoUpdaterEnabled, setAutoUpdaterEnabled] = useState(true)
  const [updateChannel, setUpdateChannel] = useState("stable")
  const [checkingUpdates, setCheckingUpdates] = useState(false)

  const handleCheckUpdates = async () => {
    setCheckingUpdates(true)
    try {
      await new Promise((resolve) => setTimeout(resolve, 2000))
      toast.info(t("settings.advanced.update_check_info"))
    } catch {
      toast.error(t("settings.advanced.update_check_error"))
    } finally {
      setCheckingUpdates(false)
    }
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("settings.advanced.title")} />
      <div className="flex-1 overflow-auto p-3 lg:p-6">
        <div className="mx-auto w-full max-w-[1000px] space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <IconSettings className="size-5" />
                {t("settings.advanced.title")}
              </CardTitle>
              <CardDescription>
                {t("settings.advanced.description")}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-4">
                <h3 className="text-foreground text-sm font-medium">
                  {t("settings.advanced.terminal_access")}
                </h3>
                <div className="flex items-center justify-between">
                  <Label htmlFor="terminal-enabled">
                    {t("settings.advanced.terminal_enabled")}
                  </Label>
                  <Switch
                    id="terminal-enabled"
                    checked={terminalEnabled}
                    onCheckedChange={setTerminalEnabled}
                  />
                </div>
                {terminalEnabled && (
                  <div className="space-y-2">
                    <Label>{t("settings.advanced.shell_path")}</Label>
                    <Input
                      value={shellPath}
                      onChange={(e) => setShellPath(e.target.value)}
                      placeholder="/bin/bash"
                      className="font-mono text-sm"
                    />
                  </div>
                )}
              </div>

              <div className="border-t pt-6">
                <div className="space-y-4">
                  <h3 className="text-foreground text-sm font-medium">
                    {t("settings.advanced.portable_mode")}
                  </h3>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="portable-mode">
                      {t("settings.advanced.portable_enabled")}
                    </Label>
                    <Switch
                      id="portable-mode"
                      checked={portableMode}
                      onCheckedChange={setPortableMode}
                    />
                  </div>
                </div>
              </div>

              <div className="border-t pt-6">
                <div className="space-y-4">
                  <h3 className="text-foreground text-sm font-medium">
                    {t("settings.advanced.auto_updater")}
                  </h3>
                  <div className="flex items-center justify-between">
                    <Label htmlFor="auto-updater">
                      {t("settings.advanced.auto_updater_enabled")}
                    </Label>
                    <Switch
                      id="auto-updater"
                      checked={autoUpdaterEnabled}
                      onCheckedChange={setAutoUpdaterEnabled}
                    />
                  </div>

                  {autoUpdaterEnabled && (
                    <>
                      <div className="space-y-2">
                        <Label>
                          {t("settings.advanced.update_channel")}
                        </Label>
                        <Select
                          value={updateChannel}
                          onValueChange={setUpdateChannel}
                        >
                          <SelectTrigger className="w-full">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="stable">
                              {t("settings.advanced.channel_stable")}
                            </SelectItem>
                            <SelectItem value="beta">
                              {t("settings.advanced.channel_beta")}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                      </div>

                      <div className="flex items-center justify-between">
                        <div>
                          <span className="text-sm">
                            {t("settings.advanced.current_version")}
                          </span>
                          <span className="ml-2 font-mono text-sm text-muted-foreground">
                            v0.0.0
                          </span>
                        </div>
                        <Button
                          variant="outline"
                          onClick={handleCheckUpdates}
                          disabled={checkingUpdates}
                        >
                          <IconRefresh className="size-4" />
                          {checkingUpdates
                            ? t("settings.advanced.checking")
                            : t("settings.advanced.check_now")}
                        </Button>
                      </div>
                    </>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
