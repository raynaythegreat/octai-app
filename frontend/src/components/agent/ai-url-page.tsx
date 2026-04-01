import {
  IconBookmark,
  IconCheck,
  IconLink,
  IconLoader2,
  IconPlug,
  IconPuzzle,
  IconSparkles,
  IconTools,
  IconWand,
} from "@tabler/icons-react"
import * as React from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { type DiscoveredItem, analyzeURL } from "@/api/scanner"
import { PageHeader } from "@/components/page-header"
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface ConflictItem {
  name: string
  type: string
}

interface ConflictState {
  conflicts: ConflictItem[]
  message: string
  originalItems: DiscoveredItem[]
  decisions: Record<string, "replace" | "skip">
}

interface ManualConfigState {
  queue: DiscoveredItem[]   // items still needing config
  current: DiscoveredItem   // item being configured right now
  readyItems: DiscoveredItem[] // items already configured + ready to integrate
  // form values
  command: string
  args: string
  url: string
  serverType: "stdio" | "sse" | "http"
}

const TYPE_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  mcp_server: IconPlug,
  skill: IconSparkles,
  tool: IconTools,
  plugin: IconPuzzle,
  connection: IconLink,
  reference_url: IconBookmark,
}

/** Returns true when an MCP server item is missing both command and url */
function needsManualConfig(item: DiscoveredItem): boolean {
  if (item.type !== "mcp_server" && item.type !== "tool" && item.type !== "plugin") return false
  const hasCmd = typeof item.config?.command === "string" && (item.config.command as string).trim() !== ""
  const hasUrl = typeof item.config?.url === "string" && (item.config.url as string).trim() !== ""
  return !hasCmd && !hasUrl
}

export function AiUrlPage() {
  const { t } = useTranslation()
  const [url, setUrl] = React.useState("")
  const [crawlDepth, setCrawlDepth] = React.useState(1)
  const [maxPages, setMaxPages] = React.useState(10)
  const [sameDomain, setSameDomain] = React.useState(true)
  const [scanning, setScanning] = React.useState(false)
  const [items, setItems] = React.useState<DiscoveredItem[] | null>(null)
  const [selected, setSelected] = React.useState<Set<number>>(new Set())
  const [integrating, setIntegrating] = React.useState(false)
  const [conflictState, setConflictState] = React.useState<ConflictState | null>(null)
  const [manualState, setManualState] = React.useState<ManualConfigState | null>(null)

  const handleScan = async () => {
    const trimmed = url.trim()
    if (!trimmed) {
      toast.error(t("aiUrl.errors.emptyUrl"))
      return
    }
    setScanning(true)
    setItems(null)
    setSelected(new Set())
    try {
      const result = await analyzeURL(trimmed, crawlDepth, maxPages, sameDomain)
      setItems(result.items)
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("aiUrl.errors.analyzeFailed"),
      )
    } finally {
      setScanning(false)
    }
  }

  const toggleItem = (index: number) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(index)) {
        next.delete(index)
      } else {
        next.add(index)
      }
      return next
    })
  }

  const toggleAll = () => {
    if (!items) return
    if (selected.size === items.length) {
      setSelected(new Set())
    } else {
      setSelected(new Set(items.map((_, i) => i)))
    }
  }

  const runIntegration = React.useCallback(
    async (
      toIntegrate: DiscoveredItem[],
      resolve?: Record<string, "replace" | "skip">,
    ) => {
      setIntegrating(true)
      try {
        const body = resolve
          ? JSON.stringify({ items: toIntegrate, resolve })
          : JSON.stringify(toIntegrate)

        const res = await fetch("/api/scanner/integrate", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body,
        })

        if (res.status === 409) {
          const data = await res.json()
          const conflicts = (data.conflicts ?? []) as ConflictItem[]
          if (conflicts.length > 0) {
            const initialDecisions: Record<string, "replace" | "skip"> = {}
            for (const c of conflicts) {
              initialDecisions[c.name] = "skip"
            }
            setConflictState({
              conflicts,
              message: data.message ?? "Some items already exist.",
              originalItems: toIntegrate,
              decisions: initialDecisions,
            })
            return
          }
        }

        if (!res.ok) {
          const text = await res.text()
          throw new Error(text || `Integration failed: ${res.status}`)
        }

        const results = await res.json()
        const succeeded = (results as { success: boolean; name: string; error?: string }[]).filter((r) => r.success).length
        const failed = (results as { success: boolean; name: string; error?: string }[]).filter((r) => !r.success)
        if (succeeded > 0) {
          toast.success(t("aiUrl.integrateSuccess", { count: succeeded }))
        }
        for (const f of failed) {
          toast.error(`${f.name}: ${f.error ?? t("aiUrl.errors.integrateFailed")}`)
        }
        if (succeeded > 0) {
          setSelected(new Set())
          window.dispatchEvent(new CustomEvent("skills-updated"))
        }
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : t("aiUrl.errors.integrateFailed"),
        )
      } finally {
        setIntegrating(false)
      }
    },
    [t],
  )

  /** Called when user clicks Integrate — check for items needing manual config first */
  const handleIntegrate = async () => {
    if (!items || selected.size === 0) return
    const toIntegrate = [...selected].map((i) => items[i])

    const incomplete = toIntegrate.filter(needsManualConfig)
    const complete = toIntegrate.filter((item) => !needsManualConfig(item))

    if (incomplete.length > 0) {
      // Open manual config dialog for first incomplete item
      const [first, ...rest] = incomplete
      setManualState({
        queue: rest,
        current: first,
        readyItems: complete,
        command: "",
        args: "",
        url: "",
        serverType: "stdio",
      })
      return
    }

    await runIntegration(toIntegrate)
  }

  const handleManualCreate = async () => {
    if (!manualState) return
    const { current, command, args, url: mcpUrl, serverType, queue, readyItems } = manualState

    // Build completed item with user-provided config
    const config: Record<string, unknown> = { type: serverType }
    if (serverType === "stdio") {
      config.command = command.trim()
      if (args.trim()) {
        config.args = args.trim().split(/\s+/)
      }
    } else {
      config.url = mcpUrl.trim()
    }
    const completedItem: DiscoveredItem = { ...current, config }

    const nextReady = [...readyItems, completedItem]

    if (queue.length > 0) {
      // Move to next incomplete item
      const [next, ...remaining] = queue
      setManualState({
        queue: remaining,
        current: next,
        readyItems: nextReady,
        command: "",
        args: "",
        url: "",
        serverType: "stdio",
      })
    } else {
      // All done — integrate everything
      setManualState(null)
      await runIntegration(nextReady)
    }
  }

  const handleManualSkip = async () => {
    if (!manualState) return
    const { queue, readyItems } = manualState

    if (queue.length > 0) {
      const [next, ...remaining] = queue
      setManualState({
        queue: remaining,
        current: next,
        readyItems,
        command: "",
        args: "",
        url: "",
        serverType: "stdio",
      })
    } else {
      // Skip this one, integrate the rest
      setManualState(null)
      if (readyItems.length > 0) {
        await runIntegration(readyItems)
      }
    }
  }

  const handleManualCancel = () => {
    setManualState(null)
    setIntegrating(false)
  }

  const handleConflictDecision = (name: string, decision: "replace" | "skip") => {
    setConflictState((prev) =>
      prev ? { ...prev, decisions: { ...prev.decisions, [name]: decision } } : null,
    )
  }

  const handleConflictSubmit = async () => {
    if (!conflictState) return
    const { originalItems, decisions } = conflictState
    setConflictState(null)
    await runIntegration(originalItems, decisions)
  }

  const handleConflictCancel = () => {
    setConflictState(null)
    setIntegrating(false)
  }

  const allSelected = items !== null && items.length > 0 && selected.size === items.length

  return (
    <div className="flex h-full flex-col">
      {/* Manual config dialog */}
      <AlertDialog open={manualState !== null}>
        <AlertDialogContent className="max-w-lg">
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <IconWand className="size-4" />
              {t("pages.agent.aiUrl.manual_config_title")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              <strong>{manualState?.current.name}</strong> {t("pages.agent.aiUrl.manual_config_description")}
              {manualState && manualState.queue.length > 0 && (
                <span className="text-muted-foreground ml-1">
                  ({t("pages.agent.aiUrl.manual_config_more", { count: manualState.queue.length })})
                </span>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>

          {manualState?.current.description && (
            <p className="text-muted-foreground -mt-1 text-xs">{manualState.current.description}</p>
          )}

          <div className="space-y-3">
            <div className="space-y-1.5">
              <Label>{t("pages.agent.aiUrl.connection_type")}</Label>
              <Select
                value={manualState?.serverType ?? "stdio"}
                onValueChange={(v) =>
                  setManualState((prev) => prev ? { ...prev, serverType: v as "stdio" | "sse" | "http" } : prev)
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="stdio">{t("pages.agent.aiUrl.connection_stdio")}</SelectItem>
                  <SelectItem value="sse">{t("pages.agent.aiUrl.connection_sse")}</SelectItem>
                  <SelectItem value="http">{t("pages.agent.aiUrl.connection_http")}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {manualState?.serverType === "stdio" ? (
              <>
                <div className="space-y-1.5">
                  <Label>{t("pages.agent.aiUrl.command_label")}</Label>
                  <Input
                    placeholder={t("pages.agent.aiUrl.command_placeholder")}
                    value={manualState?.command ?? ""}
                    onChange={(e) =>
                      setManualState((prev) => prev ? { ...prev, command: e.target.value } : prev)
                    }
                  />
                </div>
                <div className="space-y-1.5">
                  <Label>{t("pages.agent.aiUrl.args_label")} <span className="text-muted-foreground">({t("pages.agent.aiUrl.args_hint")})</span></Label>
                  <Input
                    placeholder={t("pages.agent.aiUrl.args_placeholder")}
                    value={manualState?.args ?? ""}
                    onChange={(e) =>
                      setManualState((prev) => prev ? { ...prev, args: e.target.value } : prev)
                    }
                  />
                </div>
              </>
            ) : (
              <div className="space-y-1.5">
                <Label>{t("pages.agent.aiUrl.url_label")}</Label>
                <Input
                  placeholder={t("pages.agent.aiUrl.url_placeholder")}
                  value={manualState?.url ?? ""}
                  onChange={(e) =>
                    setManualState((prev) => prev ? { ...prev, url: e.target.value } : prev)
                  }
                />
              </div>
            )}
          </div>

          <AlertDialogFooter>
            <Button variant="outline" onClick={handleManualCancel}>
              {t("pages.agent.aiUrl.cancel_all")}
            </Button>
            <Button variant="ghost" onClick={() => void handleManualSkip()}>
              {t("pages.agent.aiUrl.skip_this")}
            </Button>
            <Button
              onClick={() => void handleManualCreate()}
              disabled={
                manualState?.serverType === "stdio"
                  ? !manualState?.command.trim()
                  : !manualState?.url.trim()
              }
            >
              <IconCheck className="mr-1 size-4" />
              {t("pages.agent.aiUrl.add")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Conflict resolution dialog */}
      <AlertDialog open={conflictState !== null}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("pages.agent.aiUrl.conflict_title")}</AlertDialogTitle>
            <AlertDialogDescription>
              {conflictState?.message ?? t("pages.agent.aiUrl.conflict_description")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex flex-col gap-2">
            {conflictState?.conflicts.map((conflict) => (
              <div
                key={conflict.name}
                className="flex items-center justify-between rounded-lg border border-border/50 px-3 py-2"
              >
                <div className="min-w-0 flex-1">
                  <span className="text-sm font-medium">{conflict.name}</span>
                  <Badge variant="secondary" className="ml-2 text-xs">
                    {conflict.type}
                  </Badge>
                  <p className="text-muted-foreground text-xs">{t("pages.agent.aiUrl.already_exists")}</p>
                </div>
                <div className="ml-3 flex shrink-0 gap-2">
                  <Button
                    size="sm"
                    variant={
                      conflictState.decisions[conflict.name] === "skip"
                        ? "default"
                        : "outline"
                    }
                    onClick={() => handleConflictDecision(conflict.name, "skip")}
                  >
                    {t("pages.agent.aiUrl.skip")}
                  </Button>
                  <Button
                    size="sm"
                    variant={
                      conflictState.decisions[conflict.name] === "replace"
                        ? "default"
                        : "outline"
                    }
                    onClick={() => handleConflictDecision(conflict.name, "replace")}
                  >
                    {t("pages.agent.aiUrl.replace")}
                  </Button>
                </div>
              </div>
            ))}
          </div>
          <AlertDialogFooter>
            <Button variant="outline" onClick={handleConflictCancel}>
              {t("common.cancel")}
            </Button>
            <Button onClick={() => void handleConflictSubmit()}>
              {t("pages.agent.aiUrl.continue")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <PageHeader title={t("aiUrl.title")} />
      <div className="flex min-h-0 flex-1 flex-col gap-4 p-4 md:p-6">
        <p className="text-muted-foreground shrink-0 text-sm">
          {t("aiUrl.description")}
        </p>

        {/* URL Input */}
        <div className="flex shrink-0 gap-2">
          <Input
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder={t("aiUrl.inputPlaceholder")}
            className="flex-1"
            onKeyDown={(e) => {
              if (e.key === "Enter") void handleScan()
            }}
            disabled={scanning}
          />
          <Button onClick={handleScan} disabled={scanning || !url.trim()}>
            {scanning ? (
              <IconLoader2 className="size-4 animate-spin" />
            ) : (
              t("aiUrl.scanButton")
            )}
          </Button>
        </div>

        {/* Crawl Options */}
        <div className="flex flex-wrap gap-4 shrink-0 items-center">
          <div className="flex items-center gap-2">
            <Label className="text-xs">{t("aiUrl.crawlDepth")}</Label>
            <Input
              type="number"
              className="w-16"
              value={crawlDepth}
              onChange={(e) => setCrawlDepth(Number(e.target.value))}
            />
          </div>
          <div className="flex items-center gap-2">
            <Label className="text-xs">{t("aiUrl.maxPages")}</Label>
            <Input
              type="number"
              className="w-16"
              value={maxPages}
              onChange={(e) => setMaxPages(Number(e.target.value))}
            />
          </div>
          <div className="flex items-center gap-2">
            <Checkbox
              checked={sameDomain}
              onCheckedChange={(checked) => setSameDomain(!!checked)}
              id="same-domain"
            />
            <Label htmlFor="same-domain" className="text-xs">{t("aiUrl.sameDomain")}</Label>
          </div>
        </div>

        {/* Scanning state */}
        {scanning && (
          <div className="text-muted-foreground flex shrink-0 items-center gap-2 text-sm">
            <IconLoader2 className="size-4 animate-spin" />
            {t("aiUrl.scanning")}
          </div>
        )}

        {/* Results */}
        {items !== null && !scanning && (
          <div className="flex min-h-0 flex-1 flex-col gap-3">
            {items.length === 0 ? (
              <div className="flex flex-1 items-center justify-center text-muted-foreground">
                <p>{t("aiUrl.noResults")}</p>
              </div>
            ) : (
              <>
                {/* Header row */}
                <div className="flex shrink-0 items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Checkbox
                      checked={allSelected}
                      onCheckedChange={toggleAll}
                      id="select-all"
                    />
                    <label htmlFor="select-all" className="cursor-pointer text-sm">
                      {t("aiUrl.selectAll")} •{" "}
                      <span className="text-muted-foreground">
                        {t("aiUrl.resultsTitle", { count: items.length })}
                      </span>
                    </label>
                  </div>
                  {selected.size > 0 && (
                    <Button
                      size="sm"
                      onClick={handleIntegrate}
                      disabled={integrating}
                    >
                      {integrating ? (
                        <IconLoader2 className="mr-2 size-4 animate-spin" />
                      ) : (
                        <IconCheck className="mr-2 size-4" />
                      )}
                      {t("aiUrl.integrateButton", { count: selected.size })}
                    </Button>
                  )}
                </div>

                {/* Item list */}
                <ScrollArea className="min-h-0 flex-1">
                  <div className="flex flex-col gap-2 pb-2 pr-4">
                    {items.map((item, i) => {
                      const Icon = TYPE_ICONS[item.type] ?? IconLink
                      const isSelected = selected.has(i)
                      const incomplete = needsManualConfig(item)
                      return (
                        <div
                          key={i}
                          className={`flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors ${
                            isSelected
                              ? "border-primary/50 bg-primary/5"
                              : "hover:bg-muted/50"
                          }`}
                          onClick={() => toggleItem(i)}
                        >
                          <Checkbox
                            checked={isSelected}
                            onCheckedChange={() => toggleItem(i)}
                            onClick={(e) => e.stopPropagation()}
                            className="mt-0.5"
                          />
                          <Icon className="mt-0.5 size-5 shrink-0 text-muted-foreground" />
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2 flex-wrap">
                              <span className="text-sm font-medium">{item.name}</span>
                              <Badge variant="secondary" className="text-xs">
                                {t(`aiUrl.types.${item.type}`, { defaultValue: item.type })}
                              </Badge>
                              {incomplete && (
                                <Badge variant="outline" className="border-amber-400 text-xs text-amber-600">
                                  {t("pages.agent.aiUrl.needs_config")}
                                </Badge>
                              )}
                            </div>
                            {item.description && (
                              <p className="text-muted-foreground mt-0.5 line-clamp-2 text-xs">
                                {item.description}
                              </p>
                            )}
                            {item.type === "reference_url" && typeof item.config?.url === "string" && (
                              <a
                                href={item.config.url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="mt-0.5 block truncate text-xs text-violet-500 hover:underline"
                                onClick={(e) => e.stopPropagation()}
                              >
                                {item.config.url}
                              </a>
                            )}
                          </div>
                        </div>
                      )
                    })}
                  </div>
                </ScrollArea>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
