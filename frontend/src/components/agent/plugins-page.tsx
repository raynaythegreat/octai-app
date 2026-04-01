import {
  IconLoader2,
  IconPuzzle,
  IconRefresh,
  IconSearch,
  IconTrash,
} from "@tabler/icons-react"
import * as React from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { PageHeader } from "@/components/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"

interface Plugin {
  name: string
  type: string
  description: string
  enabled: boolean
  source: string
}

type FilterTab = "all" | "mcp_server" | "skill" | "tool" | "other"

const TYPE_COLORS: Record<string, string> = {
  mcp_server: "bg-blue-500/15 text-blue-400 border-blue-500/20",
  skill: "bg-green-500/15 text-green-400 border-green-500/20",
  tool: "bg-yellow-500/15 text-yellow-400 border-yellow-500/20",
  plugin: "bg-purple-500/15 text-purple-400 border-purple-500/20",
  connection: "bg-orange-500/15 text-orange-400 border-orange-500/20",
}

const TYPE_LABELS: Record<string, string> = {
  mcp_server: "MCP Server",
  skill: "Skill",
  tool: "Tool",
  plugin: "Plugin",
  connection: "Connection",
}

function typeBadgeClass(type: string): string {
  return (
    TYPE_COLORS[type] ??
    "bg-gray-500/15 text-gray-400 border-gray-500/20"
  )
}

function typeLabel(type: string): string {
  return TYPE_LABELS[type] ?? type
}

function matchesTab(plugin: Plugin, tab: FilterTab): boolean {
  if (tab === "all") return true
  if (tab === "other")
    return !["mcp_server", "skill", "tool"].includes(plugin.type)
  return plugin.type === tab
}

export function PluginsPage() {
  const { t } = useTranslation()
  const [plugins, setPlugins] = React.useState<Plugin[]>([])
  const [loading, setLoading] = React.useState(true)
  const [search, setSearch] = React.useState("")
  const [activeTab, setActiveTab] = React.useState<FilterTab>("all")
  const [togglingName, setTogglingName] = React.useState<string | null>(null)
  const [deletingName, setDeletingName] = React.useState<string | null>(null)

  const fetchPlugins = React.useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch("/api/plugins")
      if (!res.ok) throw new Error(`Failed to load plugins: ${res.status}`)
      const data = await res.json()
      setPlugins(data.plugins ?? [])
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("pages.agent.plugins.load_error"),
      )
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void fetchPlugins()
  }, [fetchPlugins])

  const handleToggle = async (plugin: Plugin, enabled: boolean) => {
    setTogglingName(plugin.name)
    try {
      const res = await fetch(`/api/plugins/${encodeURIComponent(plugin.name)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled }),
      })
      if (!res.ok) throw new Error(`Failed to update plugin: ${res.status}`)
      setPlugins((prev) =>
        prev.map((p) =>
          p.name === plugin.name ? { ...p, enabled } : p,
        ),
      )
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("pages.agent.plugins.update_error"),
      )
    } finally {
      setTogglingName(null)
    }
  }

  const handleDelete = async (plugin: Plugin) => {
    setDeletingName(plugin.name)
    try {
      const res = await fetch(
        `/api/plugins/${encodeURIComponent(plugin.name)}?type=${encodeURIComponent(plugin.type)}`,
        { method: "DELETE" },
      )
      if (!res.ok) throw new Error(`Failed to delete plugin: ${res.status}`)
      setPlugins((prev) => prev.filter((p) => p.name !== plugin.name))
      toast.success(t("pages.agent.plugins.remove_success", { name: plugin.name }))
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("pages.agent.plugins.delete_error"),
      )
    } finally {
      setDeletingName(null)
    }
  }

  const filtered = plugins.filter((p) => {
    const q = search.toLowerCase()
    const matchesSearch =
      !q || p.name.toLowerCase().includes(q) || p.type.toLowerCase().includes(q)
    return matchesSearch && matchesTab(p, activeTab)
  })

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("pages.agent.plugins.title")} />
      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4 md:p-6">
        {/* Controls */}
        <div className="flex shrink-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="relative flex-1 sm:max-w-xs">
            <IconSearch className="text-muted-foreground absolute left-3 top-1/2 size-4 -translate-y-1/2" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("pages.agent.plugins.search_placeholder")}
              className="pl-9"
            />
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={fetchPlugins}
            disabled={loading}
          >
            {loading ? (
              <IconLoader2 className="mr-2 size-4 animate-spin" />
            ) : (
              <IconRefresh className="mr-2 size-4" />
            )}
            {t("pages.agent.plugins.refresh")}
          </Button>
        </div>

        {/* Filter tabs */}
        <Tabs
          value={activeTab}
          onValueChange={(v) => setActiveTab(v as FilterTab)}
          className="shrink-0"
        >
          <TabsList>
            <TabsTrigger value="all">{t("pages.agent.plugins.tab_all")}</TabsTrigger>
            <TabsTrigger value="mcp_server">{t("pages.agent.plugins.tab_mcp_servers")}</TabsTrigger>
            <TabsTrigger value="skill">{t("pages.agent.plugins.tab_skills")}</TabsTrigger>
            <TabsTrigger value="tool">{t("pages.agent.plugins.tab_tools")}</TabsTrigger>
            <TabsTrigger value="other">{t("pages.agent.plugins.tab_other")}</TabsTrigger>
          </TabsList>
        </Tabs>

        {/* Loading */}
        {loading && (
          <div className="flex flex-1 items-center justify-center">
            <IconLoader2 className="text-muted-foreground size-8 animate-spin" />
          </div>
        )}

        {/* Empty state */}
        {!loading && filtered.length === 0 && (
          <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center">
            <IconPuzzle className="text-muted-foreground/40 size-12" />
            <p className="text-muted-foreground text-sm">
              {plugins.length === 0
                ? t("pages.agent.plugins.empty_no_installed")
                : t("pages.agent.plugins.empty_no_match")}
            </p>
          </div>
        )}

        {/* Plugin grid */}
        {!loading && filtered.length > 0 && (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {filtered.map((plugin) => (
              <Card key={plugin.name} size="sm" className="flex flex-col">
                <CardHeader className="border-b border-border/40 pb-3">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-semibold">
                        {plugin.name}
                      </p>
                      <Badge
                        variant="outline"
                        className={`mt-1 text-xs ${typeBadgeClass(plugin.type)}`}
                      >
                        {typeLabel(plugin.type)}
                      </Badge>
                    </div>
                    <Switch
                      checked={plugin.enabled}
                      disabled={togglingName === plugin.name}
                      onCheckedChange={(checked) =>
                        void handleToggle(plugin, checked)
                      }
                      size="sm"
                    />
                  </div>
                </CardHeader>
                <CardContent className="flex flex-1 flex-col justify-between gap-3 pt-3">
                  {plugin.description ? (
                    <p className="text-muted-foreground line-clamp-3 text-xs">
                      {plugin.description}
                    </p>
                  ) : (
                    <p className="text-muted-foreground/50 text-xs italic">
                      {t("pages.agent.plugins.no_description")}
                    </p>
                  )}
                  <div className="flex items-center justify-between gap-2">
                    {plugin.source && (
                      <p className="text-muted-foreground/60 truncate text-[10px]">
                        {plugin.source}
                      </p>
                    )}
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-destructive hover:bg-destructive/10 ml-auto shrink-0"
                      disabled={deletingName === plugin.name}
                      onClick={() => void handleDelete(plugin)}
                      title={t("pages.agent.plugins.remove")}
                    >
                      {deletingName === plugin.name ? (
                        <IconLoader2 className="size-4 animate-spin" />
                      ) : (
                        <IconTrash className="size-4" />
                      )}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
