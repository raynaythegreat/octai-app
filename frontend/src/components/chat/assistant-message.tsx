import { IconCheck, IconCopy } from "@tabler/icons-react"
import { useState } from "react"
import ReactMarkdown from "react-markdown"
import rehypeRaw from "rehype-raw"
import rehypeSanitize, { defaultSchema } from "rehype-sanitize"
import remarkGfm from "remark-gfm"

import { AgentBranchView } from "@/components/chat/agent-branch-view"
import { ThinkingIndicator } from "@/components/chat/thinking-indicator"
import { ThinkingReferences, type Reference } from "@/components/chat/thinking-references"
import { ToolUseBlockCard } from "@/components/chat/tool-use-block"
import { Button } from "@/components/ui/button"
import { formatMessageTime } from "@/hooks/use-pico-chat"
import { type MessageMeta, type ThinkingPhase } from "@/store/chat"
import { getShowDetailedSteps } from "@/store/chat"

const sanitizeSchema = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), "img", "video", "source"],
  attributes: {
    ...defaultSchema.attributes,
    img: ["src", "alt", "title", "width", "height", "loading"],
    video: ["src", "controls", "width", "height", "poster", "autoplay", "loop", "muted"],
    source: ["src", "type"],
  },
}

interface AssistantMessageProps {
  content: string
  timestamp?: string | number
  meta?: MessageMeta
  thinkingPhase?: ThinkingPhase
  thinkingStartTime?: number | null
  thinkingReferences?: Reference[]
}

export function AssistantMessage({
  content,
  timestamp = "",
  meta,
  thinkingPhase = null,
  thinkingStartTime = null,
  thinkingReferences = [],
}: AssistantMessageProps) {
  const [isCopied, setIsCopied] = useState(false)
  const showDetailedSteps = getShowDetailedSteps()
  const formattedTimestamp =
    timestamp !== "" ? formatMessageTime(timestamp) : ""

  const handleCopy = () => {
    navigator.clipboard.writeText(content).then(() => {
      setIsCopied(true)
      setTimeout(() => setIsCopied(false), 2000)
    })
  }

  // Show thinking indicator during thinking phase
  if (thinkingPhase === "thinking") {
    return (
      <ThinkingIndicator
        startTime={thinkingStartTime ?? Date.now()}
      />
    )
  }

  // Show references during references or response phase
  const showReferences = thinkingPhase === "references" || thinkingPhase === "response"

  // Show content only during response phase or when no thinking phase (backward compatibility)
  const showContent = thinkingPhase === "response" || thinkingPhase === null

  return (
    <div className="group flex w-full flex-col gap-1.5">
      <div className="text-muted-foreground flex items-center justify-between gap-2 px-1 text-xs opacity-70">
        <div className="flex items-center gap-2">
          <span>OctAi</span>
          {formattedTimestamp && (
            <>
              <span className="opacity-50">•</span>
              <span>{formattedTimestamp}</span>
            </>
          )}
        </div>
      </div>

      <div className="bg-card text-card-foreground relative overflow-hidden rounded-xl border">
        {/* Skill chips */}
        {showDetailedSteps && meta?.active_skills && meta.active_skills.length > 0 && (
          <div className="flex flex-wrap gap-1.5 border-b border-border/50 px-4 py-2">
            {meta.active_skills.map((skill) => (
              <span
                key={skill}
                className="rounded-full bg-violet-500/10 px-2 py-0.5 text-[11px] font-medium text-violet-400"
              >
                {skill}
              </span>
            ))}
          </div>
        )}

        {/* Agent branches */}
        {showDetailedSteps && meta?.agents && meta.agents.length > 0 && (
          <div className="border-b border-border/50 px-4 py-2">
            <AgentBranchView agents={meta.agents} />
          </div>
        )}

        {/* Tool use blocks */}
        {showDetailedSteps && meta?.tool_uses && meta.tool_uses.length > 0 && (
          <div className="flex flex-col gap-1 border-b border-border/50 px-4 py-2">
            {meta.tool_uses.map((tool, i) => (
              <ToolUseBlockCard key={`${tool.tool_name}-${i}`} tool={tool} />
            ))}
          </div>
        )}

        {/* Thinking References - visible during references and response phases */}
        {showReferences && thinkingReferences.length > 0 && (
          <div className="border-b border-border/50 px-4 py-2">
            <ThinkingReferences
              references={thinkingReferences}
              defaultExpanded={thinkingPhase === "references"}
            />
          </div>
        )}

        {/* Content - visible during response phase or when no thinking phase */}
        {showContent && (
          <div className="prose dark:prose-invert prose-p:my-2 prose-pre:my-2 prose-pre:rounded-lg prose-pre:border prose-pre:bg-zinc-950 prose-pre:p-3 max-w-none p-4 text-[15px] leading-relaxed">
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeRaw, [rehypeSanitize, sanitizeSchema]]}
            >
              {content}
            </ReactMarkdown>
          </div>
        )}

        {/* Copy button - only show when content is visible */}
        {showContent && (
          <Button
            variant="ghost"
            size="icon"
            className="bg-background/50 hover:bg-background/80 absolute top-2 right-2 h-7 w-7 opacity-0 transition-opacity group-hover:opacity-100"
            onClick={handleCopy}
          >
            {isCopied ? (
              <IconCheck className="h-4 w-4 text-green-500" />
            ) : (
              <IconCopy className="text-muted-foreground h-4 w-4" />
            )}
          </Button>
        )}
      </div>
    </div>
  )
}
