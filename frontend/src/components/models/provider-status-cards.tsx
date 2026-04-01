import {
  IconAlertTriangle,
  IconCheck,
  IconExternalLink,
} from "@tabler/icons-react"

import type { ModelInfo } from "@/api/models"

import { MODEL_CATALOG } from "./catalog-data"
import { ProviderIcon } from "./provider-icon"
import { getProviderKey, getProviderLabel } from "./provider-label"

interface ProviderStatusCardsProps {
  models: ModelInfo[]
}

interface ProviderStatus {
  key: string
  label: string
  total: number
  configured: number
  hasDefault: boolean
  docsUrl?: string
}

export function ProviderStatusCards({ models }: ProviderStatusCardsProps) {
  // Build per-provider stats from configured models
  const providerMap = new Map<string, ProviderStatus>()

  for (const model of models) {
    const key = getProviderKey(model.model)
    if (!providerMap.has(key)) {
      const docsUrl = MODEL_CATALOG.find((p) => p.providerKey === key)?.docsUrl
      providerMap.set(key, {
        key,
        label: getProviderLabel(model.model),
        total: 0,
        configured: 0,
        hasDefault: false,
        docsUrl,
      })
    }
    const ps = providerMap.get(key)!
    ps.total++
    if (model.configured) ps.configured++
    if (model.is_default) ps.hasDefault = true
  }

  const providers = [...providerMap.values()].sort((a, b) => {
    if (a.hasDefault && !b.hasDefault) return -1
    if (!a.hasDefault && b.hasDefault) return 1
    if (b.configured !== a.configured) return b.configured - a.configured
    return a.label.localeCompare(b.label)
  })

  if (providers.length === 0) return null

  return (
    <div className="mb-6">
      <h3 className="text-muted-foreground mb-3 text-xs font-semibold tracking-wide uppercase">
        Provider Status
      </h3>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
        {providers.map((p) => {
          const allReady = p.configured === p.total && p.total > 0
          const anyReady = p.configured > 0
          return (
            <div
              key={p.key}
              className={[
                "relative flex flex-col gap-2 rounded-lg border p-3",
                allReady
                  ? "border-border/60 bg-card"
                  : anyReady
                    ? "border-yellow-500/30 bg-yellow-500/5"
                    : "border-orange-500/30 bg-orange-500/5",
              ].join(" ")}
            >
              <div className="flex items-center justify-between gap-1.5">
                <div className="flex min-w-0 items-center gap-1.5">
                  <ProviderIcon providerKey={p.key} providerLabel={p.label} />
                  <span className="text-foreground truncate text-xs font-medium">
                    {p.label}
                  </span>
                </div>
                <div className="shrink-0">
                  {allReady ? (
                    <IconCheck className="size-3.5 text-green-500" />
                  ) : (
                    <IconAlertTriangle className="size-3.5 text-orange-400" />
                  )}
                </div>
              </div>

              <div className="flex items-center justify-between">
                <span className="text-muted-foreground text-[11px]">
                  {p.configured}/{p.total} ready
                </span>
                {p.docsUrl && (
                  <a
                    href={p.docsUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-muted-foreground hover:text-foreground transition-colors"
                    title="Docs"
                  >
                    <IconExternalLink className="size-3" />
                  </a>
                )}
              </div>

              {p.hasDefault && (
                <span className="bg-primary/10 text-primary absolute top-2 right-2 rounded px-1 py-0.5 text-[9px] font-medium leading-none">
                  default
                </span>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
