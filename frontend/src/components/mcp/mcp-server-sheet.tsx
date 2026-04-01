import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import type { MCPServerConfig } from "@/components/mcp/mcp-page"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"

interface MCPServerSheetProps {
  open: boolean
  editingServer: { name: string; cfg: MCPServerConfig } | null
  onOpenChange: (open: boolean) => void
  onSave: (
    name: string,
    cfg: MCPServerConfig,
    originalName?: string,
  ) => Promise<void>
}

export function MCPServerSheet({
  open,
  editingServer,
  onOpenChange,
  onSave,
}: MCPServerSheetProps) {
  const { t } = useTranslation()

  const [name, setName] = useState("")
  const [serverType, setServerType] = useState<"stdio" | "sse" | "http">(
    "stdio",
  )
  const [command, setCommand] = useState("")
  const [args, setArgs] = useState("")
  const [url, setUrl] = useState("")
  const [envText, setEnvText] = useState("")
  const [enabled, setEnabled] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (editingServer) {
      const { name: serverName, cfg } = editingServer
      setName(serverName)
      setServerType((cfg.type as "stdio" | "sse" | "http") ?? "stdio")
      setCommand(cfg.command ?? "")
      setArgs((cfg.args ?? []).join(" "))
      setUrl(cfg.url ?? "")
      setEnvText(
        cfg.env
          ? Object.entries(cfg.env)
              .map(([k, v]) => `${k}=${v}`)
              .join("\n")
          : "",
      )
      setEnabled(cfg.enabled)
    } else {
      setName("")
      setServerType("stdio")
      setCommand("")
      setArgs("")
      setUrl("")
      setEnvText("")
      setEnabled(true)
    }
  }, [editingServer])

  const handleSave = async () => {
    if (!name.trim() || saving) return

    setSaving(true)
    try {
      const cfg: MCPServerConfig = {
        enabled,
        type: serverType,
        ...(serverType === "stdio"
          ? {
              command,
              args: args.split(/\s+/).filter(Boolean),
            }
          : { url }),
        env: Object.fromEntries(
          envText
            .split("\n")
            .filter((line) => line.includes("="))
            .map((line) => {
              const idx = line.indexOf("=")
              return [line.slice(0, idx).trim(), line.slice(idx + 1).trim()]
            }),
        ),
      }
      await onSave(name.trim(), cfg, editingServer?.name)
      onOpenChange(false)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>
            {editingServer ? t("mcp.editServer") : t("mcp.addServer")}
          </SheetTitle>
        </SheetHeader>

        <div className="space-y-5 px-1 py-4">
          {/* Name */}
          <div className="space-y-1.5">
            <Label>{t("mcp.fields.name")}</Label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-server"
            />
          </div>

          {/* Type */}
          <div className="space-y-1.5">
            <Label>{t("mcp.fields.type")}</Label>
            <Select
              value={serverType}
              onValueChange={(v) =>
                setServerType(v as "stdio" | "sse" | "http")
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="stdio">{t("mcp.types.stdio")}</SelectItem>
                <SelectItem value="sse">{t("mcp.types.sse")}</SelectItem>
                <SelectItem value="http">{t("mcp.types.http")}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Command and Args (stdio only) */}
          {serverType === "stdio" && (
            <>
              <div className="space-y-1.5">
                <Label>{t("mcp.fields.command")}</Label>
                <Input
                  value={command}
                  onChange={(e) => setCommand(e.target.value)}
                  placeholder="npx"
                />
              </div>
              <div className="space-y-1.5">
                <Label>{t("mcp.fields.args")}</Label>
                <Input
                  value={args}
                  onChange={(e) => setArgs(e.target.value)}
                  placeholder="-y @modelcontextprotocol/server-filesystem /path"
                />
                <p className="text-muted-foreground text-xs">
                  {t("mcp.fields.argsHint")}
                </p>
              </div>
            </>
          )}

          {/* URL (sse/http only) */}
          {(serverType === "sse" || serverType === "http") && (
            <div className="space-y-1.5">
              <Label>{t("mcp.fields.url")}</Label>
              <Input
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="http://localhost:3000/sse"
              />
            </div>
          )}

          {/* Env vars (stdio only) */}
          {serverType === "stdio" && (
            <div className="space-y-1.5">
              <Label>{t("mcp.fields.env")}</Label>
              <Textarea
                value={envText}
                onChange={(e) => setEnvText(e.target.value)}
                placeholder={"API_KEY=abc123\nDEBUG=true"}
                rows={4}
              />
              <p className="text-muted-foreground text-xs">
                {t("mcp.fields.envHint")}
              </p>
            </div>
          )}

          {/* Enabled toggle */}
          <div className="flex items-center gap-3">
            <Switch checked={enabled} onCheckedChange={setEnabled} />
            <Label>{t("mcp.enabled")}</Label>
          </div>
        </div>

        <SheetFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button
            onClick={() => void handleSave()}
            disabled={!name.trim() || saving}
          >
            {saving ? t("common.saving") : t("common.save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
