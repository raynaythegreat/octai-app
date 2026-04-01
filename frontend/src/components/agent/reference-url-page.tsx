import {
  IconBookmark,
  IconExternalLink,
  IconLoader2,
  IconPlus,
  IconSearch,
  IconTag,
  IconTrash,
  IconX,
} from "@tabler/icons-react"
import * as React from "react"
import { toast } from "sonner"

import {
  type ReferenceURL,
  addReferenceURL,
  deleteReferenceURL,
  getReferenceURLs,
} from "@/api/reference-urls"
import { PageHeader } from "@/components/page-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

const CATEGORY_COLORS: Record<string, string> = {
  documentation: "bg-blue-500/10 text-blue-400 border-blue-500/20",
  "api-reference": "bg-violet-500/10 text-violet-400 border-violet-500/20",
  tutorial: "bg-green-500/10 text-green-400 border-green-500/20",
  tool: "bg-orange-500/10 text-orange-400 border-orange-500/20",
  library: "bg-cyan-500/10 text-cyan-400 border-cyan-500/20",
  blog: "bg-pink-500/10 text-pink-400 border-pink-500/20",
  example: "bg-yellow-500/10 text-yellow-400 border-yellow-500/20",
  specification: "bg-red-500/10 text-red-400 border-red-500/20",
  dataset: "bg-teal-500/10 text-teal-400 border-teal-500/20",
  video: "bg-rose-500/10 text-rose-400 border-rose-500/20",
  other: "bg-muted text-muted-foreground border-border",
}

const ALL_CATEGORIES = [
  "documentation",
  "api-reference",
  "tutorial",
  "tool",
  "library",
  "blog",
  "example",
  "specification",
  "dataset",
  "video",
  "other",
]

function getFaviconUrl(url: string): string {
  try {
    const { hostname } = new URL(url)
    return `https://www.google.com/s2/favicons?domain=${hostname}&sz=32`
  } catch {
    return ""
  }
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
    })
  } catch {
    return iso
  }
}

export function ReferenceUrlPage() {
  const [references, setReferences] = React.useState<ReferenceURL[]>([])
  const [loading, setLoading] = React.useState(true)
  const [adding, setAdding] = React.useState(false)
  const [urlInput, setUrlInput] = React.useState("")
  const [notesInput, setNotesInput] = React.useState("")
  const [showNotes, setShowNotes] = React.useState(false)
  const [search, setSearch] = React.useState("")
  const [activeCategory, setActiveCategory] = React.useState<string | null>(null)

  const loadReferences = React.useCallback(async () => {
    try {
      const data = await getReferenceURLs()
      setReferences(data.references)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to load references")
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    void loadReferences()
  }, [loadReferences])

  const handleAdd = async () => {
    const trimmed = urlInput.trim()
    if (!trimmed) return
    setAdding(true)
    try {
      const ref = await addReferenceURL(trimmed, notesInput.trim() || undefined)
      setReferences((prev) => [ref, ...prev])
      setUrlInput("")
      setNotesInput("")
      setShowNotes(false)
      toast.success(`Saved: ${ref.title || trimmed}`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add reference")
    } finally {
      setAdding(false)
    }
  }

  const handleDelete = async (id: string, title: string) => {
    try {
      await deleteReferenceURL(id)
      setReferences((prev) => prev.filter((r) => r.id !== id))
      toast.success(`Removed: ${title}`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete reference")
    }
  }

  const filtered = React.useMemo(() => {
    let result = references
    if (activeCategory) {
      result = result.filter((r) => r.category === activeCategory)
    }
    if (search.trim()) {
      const q = search.toLowerCase()
      result = result.filter(
        (r) =>
          r.title.toLowerCase().includes(q) ||
          r.description.toLowerCase().includes(q) ||
          r.url.toLowerCase().includes(q) ||
          r.tags.some((tag) => tag.toLowerCase().includes(q)),
      )
    }
    return result
  }, [references, search, activeCategory])

  const categoryCounts = React.useMemo(() => {
    const counts: Record<string, number> = {}
    for (const r of references) {
      counts[r.category] = (counts[r.category] || 0) + 1
    }
    return counts
  }, [references])

  return (
    <div className="flex h-full flex-col">
      <PageHeader title="Reference URLs" />
      <div className="flex min-h-0 flex-1 flex-col gap-4 p-4 md:p-6">
        <p className="text-muted-foreground shrink-0 text-sm">
          Save useful URLs for your agents. AI automatically categorizes and describes each link so agents can find relevant references faster.
        </p>

        {/* Add URL input */}
        <div className="shrink-0 space-y-2">
          <div className="flex gap-2">
            <Input
              value={urlInput}
              onChange={(e) => setUrlInput(e.target.value)}
              placeholder="https://docs.example.com/api-reference"
              className="flex-1"
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) void handleAdd()
              }}
              disabled={adding}
            />
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setShowNotes((v) => !v)}
              title="Add notes"
              className="text-muted-foreground hover:text-foreground"
            >
              <IconTag className="size-4" />
            </Button>
            <Button onClick={handleAdd} disabled={adding || !urlInput.trim()}>
              {adding ? (
                <IconLoader2 className="size-4 animate-spin" />
              ) : (
                <IconPlus className="size-4" />
              )}
              {adding ? "Analyzing…" : "Add"}
            </Button>
          </div>
          {showNotes && (
            <Input
              value={notesInput}
              onChange={(e) => setNotesInput(e.target.value)}
              placeholder="Optional notes (e.g. 'Use for OAuth implementation')"
              disabled={adding}
            />
          )}
          {adding && (
            <p className="text-muted-foreground flex items-center gap-2 text-xs">
              <IconLoader2 className="size-3 animate-spin" />
              Fetching page and running AI categorization…
            </p>
          )}
        </div>

        {/* Search + category filter */}
        {references.length > 0 && (
          <div className="flex shrink-0 flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative flex-1">
              <IconSearch className="text-muted-foreground absolute left-2.5 top-2.5 size-4" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search references…"
                className="pl-8"
              />
              {search && (
                <button
                  onClick={() => setSearch("")}
                  className="text-muted-foreground hover:text-foreground absolute right-2.5 top-2.5"
                >
                  <IconX className="size-4" />
                </button>
              )}
            </div>
            <div className="flex flex-wrap gap-1">
              {ALL_CATEGORIES.filter((c) => categoryCounts[c]).map((cat) => (
                <button
                  key={cat}
                  onClick={() => setActiveCategory(activeCategory === cat ? null : cat)}
                  className={`rounded-full border px-2.5 py-0.5 text-xs transition-colors ${
                    activeCategory === cat
                      ? (CATEGORY_COLORS[cat] ?? CATEGORY_COLORS.other)
                      : "border-border/60 text-muted-foreground hover:border-border"
                  }`}
                >
                  {cat} ({categoryCounts[cat]})
                </button>
              ))}
            </div>
          </div>
        )}

        {/* List */}
        {loading ? (
          <div className="flex flex-1 items-center justify-center">
            <IconLoader2 className="text-muted-foreground size-6 animate-spin" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="text-muted-foreground flex flex-1 flex-col items-center justify-center gap-3">
            <IconBookmark className="size-10 opacity-20" />
            {references.length === 0 ? (
              <>
                <p className="text-sm font-medium">No references saved yet</p>
                <p className="max-w-xs text-center text-xs">
                  Add URLs like API docs, tutorials, or tools. AI will categorize them so your agents can reference them during tasks.
                </p>
              </>
            ) : (
              <p className="text-sm">No results match your filter.</p>
            )}
          </div>
        ) : (
          <div className="min-h-0 flex-1 overflow-y-auto">
            <div className="flex flex-col gap-2 pb-4">
              {filtered.map((ref) => (
                <div
                  key={ref.id}
                  className="bg-card border-border/60 hover:bg-muted/30 group flex items-start gap-3 rounded-xl border p-3 transition-colors"
                >
                  {/* Favicon */}
                  <div className="mt-0.5 size-6 shrink-0 overflow-hidden rounded">
                    {getFaviconUrl(ref.url) ? (
                      <img
                        src={getFaviconUrl(ref.url)}
                        alt=""
                        className="size-6 object-contain"
                        onError={(e) => {
                          ;(e.target as HTMLImageElement).style.display = "none"
                        }}
                      />
                    ) : (
                      <IconBookmark className="text-muted-foreground size-6" />
                    )}
                  </div>

                  {/* Content */}
                  <div className="min-w-0 flex-1">
                    <div className="flex items-start gap-2">
                      <a
                        href={ref.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="hover:text-primary line-clamp-1 flex-1 text-sm font-medium transition-colors"
                      >
                        {ref.title || ref.url}
                        <IconExternalLink className="ml-1 inline size-3 opacity-50" />
                      </a>
                      <Badge
                        variant="outline"
                        className={`shrink-0 text-[10px] ${CATEGORY_COLORS[ref.category] ?? CATEGORY_COLORS.other}`}
                      >
                        {ref.category}
                      </Badge>
                    </div>

                    {ref.description && (
                      <p className="text-muted-foreground mt-0.5 line-clamp-2 text-xs">
                        {ref.description}
                      </p>
                    )}

                    {ref.notes && (
                      <p className="text-foreground/70 mt-1 text-xs italic">
                        Note: {ref.notes}
                      </p>
                    )}

                    <div className="mt-1.5 flex flex-wrap items-center gap-1">
                      {ref.tags.map((tag) => (
                        <span
                          key={tag}
                          className="bg-muted text-muted-foreground rounded px-1.5 py-0.5 text-[10px]"
                        >
                          {tag}
                        </span>
                      ))}
                      <span className="text-muted-foreground/50 ml-auto text-[10px]">
                        {formatDate(ref.added_at)}
                      </span>
                    </div>
                  </div>

                  {/* Delete */}
                  <Button
                    variant="ghost"
                    size="icon"
                    className="text-muted-foreground hover:text-destructive hover:bg-destructive/10 size-7 shrink-0 opacity-0 transition-opacity group-hover:opacity-100"
                    onClick={() => handleDelete(ref.id, ref.title || ref.url)}
                    title="Remove reference"
                  >
                    <IconTrash className="size-3.5" />
                  </Button>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
