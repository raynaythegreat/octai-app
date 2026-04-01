import {
  IconCheck,
  IconExternalLink,
  IconLoader2,
  IconPlus,
  IconSearch,
  IconX,
} from "@tabler/icons-react"
import { useMemo, useState } from "react"

import { addModel } from "@/api/models"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

import {
  CATALOG_CATEGORIES,
  CATEGORY_COLORS,
  MODEL_CATALOG,
  type CatalogCategory,
} from "./catalog-data"
import { ProviderIcon } from "./provider-icon"

interface CatalogTabProps {
  configuredModelIds: Set<string>
  existingModelNames: string[]
  onModelAdded: () => void
}

export function CatalogTab({
  configuredModelIds,
  existingModelNames,
  onModelAdded,
}: CatalogTabProps) {
  const [search, setSearch] = useState("")
  const [activeCategory, setActiveCategory] = useState<CatalogCategory | null>(
    null,
  )
  const [addingId, setAddingId] = useState<string | null>(null)
  const [addedIds, setAddedIds] = useState<Set<string>>(new Set())

  const isConfigured = (modelId: string) =>
    configuredModelIds.has(modelId) || addedIds.has(modelId)

  const filteredProviders = useMemo(() => {
    const q = search.toLowerCase().trim()
    return MODEL_CATALOG.filter((p) => {
      if (activeCategory && p.category !== activeCategory) return false
      if (!q) return true
      if (
        p.provider.toLowerCase().includes(q) ||
        p.category.toLowerCase().includes(q)
      )
        return true
      return p.models.some(
        (m) =>
          m.name.toLowerCase().includes(q) ||
          m.id.toLowerCase().includes(q) ||
          m.desc.toLowerCase().includes(q),
      )
    }).map((p) => ({
      ...p,
      models: q
        ? p.models.filter(
            (m) =>
              m.name.toLowerCase().includes(q) ||
              m.id.toLowerCase().includes(q) ||
              m.desc.toLowerCase().includes(q) ||
              p.provider.toLowerCase().includes(q) ||
              p.category.toLowerCase().includes(q),
          )
        : p.models,
    }))
  }, [search, activeCategory])

  const handleAdd = async (modelId: string, modelName: string) => {
    if (isConfigured(modelId) || addingId) return
    setAddingId(modelId)
    try {
      // Generate a unique display name if needed
      let displayName = modelName
      let suffix = 1
      while (existingModelNames.includes(displayName)) {
        displayName = `${modelName}-${suffix}`
        suffix++
      }
      await addModel({
        model_name: displayName,
        model: modelId,
      })
      setAddedIds((prev) => new Set([...prev, modelId]))
      onModelAdded()
    } catch {
      // ignore
    } finally {
      setAddingId(null)
    }
  }

  const totalModels = filteredProviders.reduce(
    (sum, p) => sum + p.models.length,
    0,
  )

  return (
    <div className="space-y-4">
      {/* Search + Filters */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <IconSearch className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search models, providers, descriptions..."
            className="pl-9"
          />
          {search && (
            <button
              type="button"
              onClick={() => setSearch("")}
              className="text-muted-foreground hover:text-foreground absolute top-1/2 right-3 -translate-y-1/2 transition-colors"
            >
              <IconX className="size-4" />
            </button>
          )}
        </div>
        <p className="text-muted-foreground shrink-0 text-xs">
          {totalModels} model{totalModels !== 1 ? "s" : ""}
        </p>
      </div>

      {/* Category filter chips */}
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => setActiveCategory(null)}
          className={[
            "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
            activeCategory === null
              ? "border-primary bg-primary text-primary-foreground"
              : "border-border text-muted-foreground hover:border-foreground/30 hover:text-foreground",
          ].join(" ")}
        >
          All
        </button>
        {CATALOG_CATEGORIES.map((cat) => (
          <button
            key={cat}
            type="button"
            onClick={() =>
              setActiveCategory((prev) => (prev === cat ? null : cat))
            }
            className={[
              "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
              activeCategory === cat
                ? "border-primary bg-primary text-primary-foreground"
                : "border-border text-muted-foreground hover:border-foreground/30 hover:text-foreground",
            ].join(" ")}
          >
            {cat}
          </button>
        ))}
      </div>

      {/* Catalog grid */}
      {filteredProviders.length === 0 ? (
        <div className="text-muted-foreground py-12 text-center text-sm">
          No models match your search.
        </div>
      ) : (
        <div className="space-y-8 pb-8">
          {filteredProviders.map((providerGroup) => (
            <section key={`${providerGroup.providerKey}-${providerGroup.category}`}>
              {/* Provider header */}
              <div className="mb-3 flex items-center gap-3">
                <div className="border-border/40 border-t flex-1" />
                <div className="inline-flex items-center gap-2 px-1">
                  <ProviderIcon
                    providerKey={providerGroup.providerKey}
                    providerLabel={providerGroup.provider}
                  />
                  <span className="text-foreground/80 text-xs font-semibold tracking-wide uppercase">
                    {providerGroup.provider}
                  </span>
                  <span
                    className={[
                      "rounded-full px-2 py-0.5 text-[10px] font-medium",
                      CATEGORY_COLORS[providerGroup.category],
                    ].join(" ")}
                  >
                    {providerGroup.category}
                  </span>
                  {providerGroup.docsUrl && (
                    <a
                      href={providerGroup.docsUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-muted-foreground hover:text-foreground transition-colors"
                      title="Documentation"
                    >
                      <IconExternalLink className="size-3" />
                    </a>
                  )}
                </div>
                <div className="border-border/40 border-t flex-1" />
              </div>

              {/* Models grid */}
              <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-3">
                {providerGroup.models.map((model) => {
                  const configured = isConfigured(model.id)
                  const adding = addingId === model.id

                  return (
                    <div
                      key={model.id}
                      className={[
                        "group relative flex items-start justify-between gap-3 rounded-xl border p-3.5 transition-colors",
                        configured
                          ? "border-green-500/30 bg-green-500/5"
                          : "border-border/60 bg-card hover:bg-muted/30",
                      ].join(" ")}
                    >
                      <div className="min-w-0 flex-1">
                        <div className="mb-1 flex items-center gap-1.5">
                          <span className="text-foreground text-sm font-medium truncate">
                            {model.name}
                          </span>
                        </div>
                        <p className="text-muted-foreground mb-1 text-xs leading-snug">
                          {model.desc}
                        </p>
                        <p className="text-muted-foreground/60 font-mono text-[10px] truncate">
                          {model.id}
                        </p>
                      </div>

                      <div className="shrink-0 pt-0.5">
                        {configured ? (
                          <span className="inline-flex items-center gap-1 rounded-full border border-green-500/30 bg-green-500/10 px-2 py-0.5 text-[10px] font-medium text-green-600 dark:text-green-400">
                            <IconCheck className="size-3" />
                            Configured
                          </span>
                        ) : (
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-7 px-2.5 text-xs"
                            onClick={() => handleAdd(model.id, model.name)}
                            disabled={adding || !!addingId}
                          >
                            {adding ? (
                              <IconLoader2 className="size-3 animate-spin" />
                            ) : (
                              <IconPlus className="size-3" />
                            )}
                            Add
                          </Button>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            </section>
          ))}
        </div>
      )}
    </div>
  )
}
