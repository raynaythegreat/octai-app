import {
  IconPencil,
  IconPlug,
  IconPlus,
  IconTrash,
} from "@tabler/icons-react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { getAppConfig, patchAppConfig } from "@/api/channels"
import { MCPServerSheet } from "@/components/mcp/mcp-server-sheet"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"

export interface MCPServerConfig {
  enabled: boolean
  command?: string
  args?: string[]
  env?: Record<string, string>
  env_file?: string
  type?: string
  url?: string
  deferred?: boolean
}

function deepSet<T extends Record<string, unknown>>(
  obj: T,
  path: string,
  value: unknown,
): T {
  const keys = path.split(".")
  const result = { ...obj } as Record<string, unknown>
  let current = result
  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i]
    current[key] = { ...(current[key] as Record<string, unknown>) }
    current = current[key] as Record<string, unknown>
  }
  current[keys[keys.length - 1]] = value
  return result as T
}

function MCPServerCard({
  name,
  cfg,
  onEdit,
  onDelete,
  onToggle,
}: {
  name: string
  cfg: MCPServerConfig
  onEdit: () => void
  onDelete: () => void
  onToggle: (enabled: boolean) => void
}) {
  const { t } = useTranslation()
  const isStdio = !cfg.type || cfg.type === "stdio"

  return (
    <div className="bg-card border-border/60 flex items-center gap-3 rounded-lg border p-4 shadow-sm">
      <div className="bg-primary/10 flex size-9 shrink-0 items-center justify-center rounded-md">
        <IconPlug className="text-primary size-4" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-foreground truncate text-sm font-medium">
            {name}
          </span>
          <span className="bg-muted text-muted-foreground rounded-full px-2 py-0.5 text-xs">
            {cfg.type ?? "stdio"}
          </span>
        </div>
        <p className="text-muted-foreground mt-0.5 truncate text-xs">
          {isStdio
            ? `${cfg.command ?? ""} ${(cfg.args ?? []).join(" ")}`.trim()
            : (cfg.url ?? "")}
        </p>
      </div>
      <div className="flex items-center gap-2">
        <Switch
          checked={cfg.enabled}
          onCheckedChange={onToggle}
          className="scale-90"
          aria-label={t("mcp.enabled")}
        />
        <Button
          size="icon"
          variant="ghost"
          className="size-8"
          onClick={onEdit}
          aria-label={t("mcp.editServer")}
        >
          <IconPencil className="size-3.5" />
        </Button>
        <Button
          size="icon"
          variant="ghost"
          className="size-8 text-destructive hover:text-destructive"
          onClick={onDelete}
          aria-label={t("mcp.deleteServer")}
        >
          <IconTrash className="size-3.5" />
        </Button>
      </div>
    </div>
  )
}

export function MCPPage() {
  const { t } = useTranslation()
  const [config, setConfig] = useState<Record<string, unknown>>({})
  const [loading, setLoading] = useState(true)
  const [isSheetOpen, setIsSheetOpen] = useState(false)
  const [editingServer, setEditingServer] = useState<{
    name: string
    cfg: MCPServerConfig
  } | null>(null)

  const fetchConfig = async () => {
    try {
      const cfg = await getAppConfig()
      setConfig(cfg)
    } catch {
      toast.error(t("pages.config.load_error"))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchConfig()
  }, [])

  const getMCPConfig = () => {
    return (config?.tools as Record<string, unknown>)
      ?.mcp as Record<string, unknown> | undefined
  }

  const servers = (getMCPConfig()?.servers as Record<string, MCPServerConfig>) ?? {}

  const handleToggleMCPEnabled = async (checked: boolean) => {
    try {
      await patchAppConfig({ tools: { mcp: { enabled: checked } } })
      setConfig((prev) =>
        deepSet(prev as Record<string, unknown>, "tools.mcp.enabled", checked),
      )
    } catch {
      toast.error(t("mcp.toggle.enableError"))
    }
  }

  const handleToggleDiscovery = async (checked: boolean) => {
    try {
      await patchAppConfig({
        tools: { mcp: { discovery: { enabled: checked } } },
      })
      setConfig((prev) =>
        deepSet(
          prev as Record<string, unknown>,
          "tools.mcp.discovery.enabled",
          checked,
        ),
      )
    } catch {
      toast.error(t("mcp.toggle.enableError"))
    }
  }

  const handleAddServer = () => {
    setEditingServer(null)
    setIsSheetOpen(true)
  }

  const handleEditServer = (name: string, cfg: MCPServerConfig) => {
    setEditingServer({ name, cfg })
    setIsSheetOpen(true)
  }

  const handleSaveServer = async (
    name: string,
    cfg: MCPServerConfig,
    originalName?: string,
  ) => {
    try {
      if (originalName && originalName !== name) {
        await patchAppConfig({
          tools: { mcp: { servers: { [originalName]: null } } },
        })
      }
      await patchAppConfig({ tools: { mcp: { servers: { [name]: cfg } } } })
      await fetchConfig()
      toast.success(t("mcp.server.saved"))
    } catch {
      toast.error(t("mcp.server.saveError"))
      throw new Error("Save failed")
    }
  }

  const handleDeleteServer = async (name: string) => {
    try {
      await patchAppConfig({
        tools: { mcp: { servers: { [name]: null } } },
      })
      await fetchConfig()
      toast.success(t("mcp.server.deleted"))
    } catch {
      toast.error(t("mcp.server.deleteError"))
    }
  }

  const handleToggleServer = async (name: string, enabled: boolean) => {
    try {
      await patchAppConfig({
        tools: { mcp: { servers: { [name]: { enabled } } } },
      })
      setConfig((prev) =>
        deepSet(
          prev as Record<string, unknown>,
          `tools.mcp.servers.${name}.enabled`,
          enabled,
        ),
      )
    } catch {
      toast.error(t("mcp.toggle.enableError"))
    }
  }

  if (loading) {
    return (
      <div className="flex h-full flex-col">
        <PageHeader title={t("mcp.title")} />
        <div className="flex flex-1 items-center justify-center">
          <p className="text-muted-foreground text-sm">{t("labels.loading")}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("mcp.title")} />
      <div className="flex-1 overflow-y-auto px-4 py-6 sm:px-6">
        <div className="mx-auto max-w-4xl space-y-6">
          {/* Global toggles card */}
          <div className="bg-card border-border/60 rounded-xl border p-5 shadow-sm">
            <h2 className="text-foreground mb-4 text-sm font-semibold">
              {t("mcp.enabled")}
            </h2>
            <div className="space-y-4">
              {/* MCP Enabled toggle */}
              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-sm font-medium">
                    {t("mcp.enabled")}
                  </Label>
                  <p className="text-muted-foreground mt-0.5 text-xs">
                    {t("mcp.description")}
                  </p>
                </div>
                <Switch
                  checked={Boolean(getMCPConfig()?.enabled)}
                  onCheckedChange={(checked) =>
                    void handleToggleMCPEnabled(checked)
                  }
                />
              </div>

              {/* Discovery toggle */}
              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-sm font-medium">
                    {t("mcp.discovery.title")}
                  </Label>
                  <p className="text-muted-foreground mt-0.5 text-xs">
                    {t("mcp.discovery.description")}
                  </p>
                </div>
                <Switch
                  checked={Boolean(
                    (
                      getMCPConfig()?.discovery as
                        | Record<string, unknown>
                        | undefined
                    )?.enabled,
                  )}
                  onCheckedChange={(checked) =>
                    void handleToggleDiscovery(checked)
                  }
                />
              </div>
            </div>
          </div>

          {/* Servers section */}
          <div>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-foreground text-sm font-semibold">
                Servers
              </h2>
              <Button
                size="sm"
                onClick={handleAddServer}
                className="gap-1.5"
              >
                <IconPlus className="size-4" />
                {t("mcp.addServer")}
              </Button>
            </div>

            {Object.keys(servers).length === 0 ? (
              <div className="text-muted-foreground flex flex-col items-center gap-3 py-16 text-sm">
                <IconPlug className="size-10 opacity-30" />
                <p>{t("mcp.noServers")}</p>
              </div>
            ) : (
              <div className="space-y-3">
                {Object.entries(servers).map(([name, cfg]) => (
                  <MCPServerCard
                    key={name}
                    name={name}
                    cfg={cfg}
                    onEdit={() => handleEditServer(name, cfg)}
                    onDelete={() => void handleDeleteServer(name)}
                    onToggle={(enabled) => void handleToggleServer(name, enabled)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      <MCPServerSheet
        open={isSheetOpen}
        editingServer={editingServer}
        onOpenChange={setIsSheetOpen}
        onSave={handleSaveServer}
      />
    </div>
  )
}
