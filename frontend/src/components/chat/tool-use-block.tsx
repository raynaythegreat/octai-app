import {
  IconCheck,
  IconChevronDown,
  IconChevronRight,
  IconLoader2,
  IconTool,
  IconX,
} from "@tabler/icons-react"
import { useState } from "react"

import { type ToolUseBlock } from "@/store/chat"
import { cn } from "@/lib/utils"

interface ToolUseBlockProps {
  tool: ToolUseBlock
}

export function ToolUseBlockCard({ tool }: ToolUseBlockProps) {
  const [expanded, setExpanded] = useState(false)
  const hasDetail = tool.args_preview || tool.result_preview

  return (
    <div
      className={cn(
        "rounded-lg border-l-2 bg-muted/40 text-xs",
        tool.status === "running" && "border-violet-400",
        tool.status === "done" && "border-green-500",
        tool.status === "error" && "border-red-500",
      )}
    >
      <button
        type="button"
        className={cn(
          "flex w-full items-center gap-2 px-3 py-1.5 text-left",
          hasDetail && "cursor-pointer",
        )}
        onClick={() => hasDetail && setExpanded((v) => !v)}
        disabled={!hasDetail}
      >
        <IconTool className="size-3 shrink-0 text-muted-foreground" />
        <span className="font-mono font-medium text-foreground/80">{tool.tool_name}</span>

        <div className="ml-auto flex items-center gap-2">
          {tool.duration_ms !== undefined && (
            <span className="text-muted-foreground">{tool.duration_ms}ms</span>
          )}
          {tool.status === "running" && (
            <IconLoader2 className="size-3 animate-spin text-violet-400" />
          )}
          {tool.status === "done" && (
            <IconCheck className="size-3 text-green-500" />
          )}
          {tool.status === "error" && (
            <IconX className="size-3 text-red-500" />
          )}
          {hasDetail && (
            expanded
              ? <IconChevronDown className="size-3 text-muted-foreground" />
              : <IconChevronRight className="size-3 text-muted-foreground" />
          )}
        </div>
      </button>

      {expanded && hasDetail && (
        <div className="border-t border-border/40 px-3 py-2 space-y-1.5">
          {tool.args_preview && (
            <div>
              <div className="text-[10px] text-muted-foreground uppercase tracking-wider mb-1">Args</div>
              <pre className="font-mono text-[11px] text-foreground/70 whitespace-pre-wrap break-all">
                {tool.args_preview}
              </pre>
            </div>
          )}
          {tool.result_preview && (
            <div>
              <div className="text-[10px] text-muted-foreground uppercase tracking-wider mb-1">Result</div>
              <pre className="font-mono text-[11px] text-foreground/70 whitespace-pre-wrap break-all">
                {tool.result_preview}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
