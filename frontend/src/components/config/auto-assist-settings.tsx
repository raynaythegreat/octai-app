import * as React from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"

interface AutoAssistConfig {
  enabled: boolean
  excluded_skills: string[]
  excluded_tools: string[]
  excluded_mcp_servers: string[]
  max_auto_skills: number
}

const DEFAULT_CONFIG: AutoAssistConfig = {
  enabled: false,
  excluded_skills: [],
  excluded_tools: [],
  excluded_mcp_servers: [],
  max_auto_skills: 3,
}

function useDebouncedCallback<T extends unknown[]>(
  fn: (...args: T) => void,
  delay: number,
) {
  const timerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null)
  return React.useCallback(
    (...args: T) => {
      if (timerRef.current) clearTimeout(timerRef.current)
      timerRef.current = setTimeout(() => fn(...args), delay)
    },
    [fn, delay],
  )
}

interface ExclusionListProps {
  title: string
  items: string[]
  excluded: string[]
  onToggle: (name: string, excluded: boolean) => void
}

function ExclusionList({ title, items, excluded, onToggle }: ExclusionListProps) {
  const { t } = useTranslation()
  if (items.length === 0) {
    return (
      <div>
        <p className="mb-2 text-sm font-medium">{title}</p>
        <p className="text-muted-foreground text-xs">{t("pages.config.autoAssist.none_available")}</p>
      </div>
    )
  }

  return (
    <div>
      <p className="mb-2 text-sm font-medium">{title}</p>
      <div className="border-border/50 max-h-48 overflow-y-auto rounded-lg border p-2">
        <div className="flex flex-col gap-1.5">
          {items.map((name) => {
            const isExcluded = excluded.includes(name)
            return (
              <label
                key={name}
                className="hover:bg-muted/50 flex cursor-pointer items-center gap-2.5 rounded px-2 py-1.5 transition-colors"
              >
                <Checkbox
                  checked={isExcluded}
                  onCheckedChange={(checked) =>
                    onToggle(name, checked === true)
                  }
                />
                <span className="text-sm">{name}</span>
                {isExcluded && (
                  <span className="text-muted-foreground ml-auto text-xs">
                    {t("pages.config.autoAssist.excluded")}
                  </span>
                )}
              </label>
            )
          })}
        </div>
      </div>
    </div>
  )
}

export function AutoAssistSettings() {
  const { t } = useTranslation()
  const [config, setConfig] = React.useState<AutoAssistConfig>(DEFAULT_CONFIG)
  const [availableSkills, setAvailableSkills] = React.useState<string[]>([])
  const [availableTools, setAvailableTools] = React.useState<string[]>([])
  const [availableMcpServers, setAvailableMcpServers] = React.useState<string[]>([])
  const [loading, setLoading] = React.useState(true)
  const [savedIndicator, setSavedIndicator] = React.useState(false)

  // Load initial data
  React.useEffect(() => {
    const fetchAll = async () => {
      setLoading(true)
      try {
        await Promise.all([
          fetch("/api/config/auto-assist")
            .then((r) => (r.ok ? r.json() : null))
            .then((d) => d && setConfig(d)),
          fetch("/api/skills")
            .then((r) => (r.ok ? r.json() : null))
            .then((d) => {
              if (d?.skills) {
                setAvailableSkills(
                  (d.skills as { name: string }[]).map((s) => s.name),
                )
              }
            }),
          fetch("/api/tools")
            .then((r) => (r.ok ? r.json() : null))
            .then((d) => {
              if (d?.tools) {
                setAvailableTools(
                  (d.tools as { name: string }[]).map((t) => t.name),
                )
              }
            }),
          fetch("/api/config")
            .then((r) => (r.ok ? r.json() : null))
            .then((d) => {
              const servers = d?.tools?.mcp?.servers as
                | Record<string, unknown>
                | undefined
              if (servers) {
                setAvailableMcpServers(Object.keys(servers))
              }
            }),
        ])
      } finally {
        setLoading(false)
      }
    }
    void fetchAll()
  }, [])

  const showSaved = React.useCallback(() => {
    setSavedIndicator(true)
    const timer = setTimeout(() => setSavedIndicator(false), 2000)
    return () => clearTimeout(timer)
  }, [])

  const saveConfig = React.useCallback(
    async (next: AutoAssistConfig) => {
      try {
        const res = await fetch("/api/config/auto-assist", {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(next),
        })
        if (!res.ok) throw new Error(`Save failed: ${res.status}`)
        showSaved()
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : t("pages.config.autoAssist.save_error"),
        )
      }
    },
    [showSaved],
  )

  const debouncedSave = useDebouncedCallback(saveConfig, 500)

  const updateConfig = React.useCallback(
    (updater: (prev: AutoAssistConfig) => AutoAssistConfig) => {
      setConfig((prev) => {
        const next = updater(prev)
        debouncedSave(next)
        return next
      })
    },
    [debouncedSave],
  )

  const handleEnabledChange = (enabled: boolean) => {
    updateConfig((prev) => ({ ...prev, enabled }))
  }

  const handleMaxSkillsChange = (value: string) => {
    const num = parseInt(value, 10)
    if (!isNaN(num) && num >= 1 && num <= 10) {
      updateConfig((prev) => ({ ...prev, max_auto_skills: num }))
    }
  }

  const toggleExclusion = (
    field: "excluded_skills" | "excluded_tools" | "excluded_mcp_servers",
    name: string,
    exclude: boolean,
  ) => {
    updateConfig((prev) => {
      const current = prev[field]
      const next = exclude
        ? current.includes(name)
          ? current
          : [...current, name]
        : current.filter((n) => n !== name)
      return { ...prev, [field]: next }
    })
  }

  if (loading) {
    return (
      <div className="text-muted-foreground p-4 text-sm">
        {t("pages.config.autoAssist.loading")}
      </div>
    )
  }

  return (
    <Card>
      <CardHeader className="border-b border-border/40 pb-4">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>{t("pages.config.autoAssist.title")}</CardTitle>
            <p className="text-muted-foreground mt-1 text-sm">
              {t("pages.config.autoAssist.description")}
            </p>
          </div>
          <div className="flex items-center gap-2">
            {savedIndicator && (
              <span className="text-muted-foreground animate-in fade-in text-xs">
                {t("pages.config.autoAssist.saved")}
              </span>
            )}
            <Switch
              checked={config.enabled}
              onCheckedChange={handleEnabledChange}
            />
          </div>
        </div>
      </CardHeader>

      {config.enabled && (
        <CardContent className="flex flex-col gap-6 pt-6">
          {/* Max auto skills */}
          <div className="flex items-center gap-4">
            <Label htmlFor="max-auto-skills" className="min-w-max text-sm">
              {t("pages.config.autoAssist.max_auto_skills")}
            </Label>
            <Input
              id="max-auto-skills"
              type="number"
              min={1}
              max={10}
              value={config.max_auto_skills}
              onChange={(e) => handleMaxSkillsChange(e.target.value)}
              className="w-20"
            />
            <span className="text-muted-foreground text-xs">(1–10)</span>
          </div>

          {/* Exclusion lists */}
          <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
            <ExclusionList
              title={t("pages.config.autoAssist.exclude_skills")}
              items={availableSkills}
              excluded={config.excluded_skills}
              onToggle={(name, excl) =>
                toggleExclusion("excluded_skills", name, excl)
              }
            />
            <ExclusionList
              title={t("pages.config.autoAssist.exclude_tools")}
              items={availableTools}
              excluded={config.excluded_tools}
              onToggle={(name, excl) =>
                toggleExclusion("excluded_tools", name, excl)
              }
            />
            <ExclusionList
              title={t("pages.config.autoAssist.exclude_mcp_servers")}
              items={availableMcpServers}
              excluded={config.excluded_mcp_servers}
              onToggle={(name, excl) =>
                toggleExclusion("excluded_mcp_servers", name, excl)
              }
            />
          </div>
        </CardContent>
      )}
    </Card>
  )
}
