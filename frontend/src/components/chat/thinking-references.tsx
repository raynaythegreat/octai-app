import { IconChevronDown, IconChevronRight, IconLink } from "@tabler/icons-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"

import { cn } from "@/lib/utils"

export interface Reference {
  id: string
  title: string
  url?: string
  snippet?: string
}

interface ThinkingReferencesProps {
  references: Reference[]
  className?: string
  defaultExpanded?: boolean
}

export function ThinkingReferences({
  references,
  className,
  defaultExpanded = false,
}: ThinkingReferencesProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(defaultExpanded)

  if (references.length === 0) return null

  return (
    <div className={cn("rounded-lg border border-violet-500/20 bg-violet-500/5", className)}>
      {/* Header - always visible */}
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left transition-colors hover:bg-violet-500/10 rounded-lg"
      >
        <IconLink className="size-3.5 shrink-0 text-violet-400" />
        <span className="text-xs font-medium text-violet-300">
          {t("chat.references.label", "Sources")}
        </span>
        <span className="text-muted-foreground ml-1 text-[10px]">
          ({references.length})
        </span>
        <div className="ml-auto flex items-center gap-1.5">
          {references.slice(0, 3).map((ref, i) => (
            <span
              key={ref.id}
              className="max-w-[80px] truncate text-[10px] text-muted-foreground/70"
              title={ref.title}
            >
              {ref.title}
              {i < Math.min(references.length, 3) - 1 && (
                <span className="ml-1 text-muted-foreground/40">•</span>
              )}
            </span>
          ))}
          {references.length > 3 && (
            <span className="text-[10px] text-muted-foreground/50">
              +{references.length - 3}
            </span>
          )}
          {expanded ? (
            <IconChevronDown className="size-3.5 text-violet-400" />
          ) : (
            <IconChevronRight className="size-3.5 text-violet-400" />
          )}
        </div>
      </button>

      {/* Expanded list */}
      {expanded && (
        <div className="border-t border-violet-500/10 px-3 py-2">
          <ul className="space-y-1.5">
            {references.map((ref, index) => (
              <li key={ref.id} className="flex items-start gap-2">
                <span className="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-violet-500/20 text-[9px] font-medium text-violet-300">
                  {index + 1}
                </span>
                <div className="flex-1 min-w-0">
                  {ref.url ? (
                    <a
                      href={ref.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="block truncate text-xs text-violet-300 hover:text-violet-200 hover:underline transition-colors"
                      title={ref.title}
                    >
                      {ref.title}
                    </a>
                  ) : (
                    <span
                      className="block truncate text-xs text-violet-300"
                      title={ref.title}
                    >
                      {ref.title}
                    </span>
                  )}
                  {ref.snippet && (
                    <p className="mt-0.5 line-clamp-2 text-[10px] text-muted-foreground/70">
                      {ref.snippet}
                    </p>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
