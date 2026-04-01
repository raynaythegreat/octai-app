import {
  IconCheck,
  IconLoader2,
  IconRobot,
  IconX,
} from "@tabler/icons-react"

import { type AgentBlock } from "@/store/chat"
import { ToolUseBlockCard } from "@/components/chat/tool-use-block"
import { cn } from "@/lib/utils"

interface AgentBranchViewProps {
  agents: AgentBlock[]
}

export function AgentBranchView({ agents }: AgentBranchViewProps) {
  if (!agents.length) return null

  return (
    <div className="space-y-1 text-xs">
      {agents.map((agent, i) => (
        <div key={agent.agent_id} className="flex gap-2">
          {/* Connector lines */}
          <div className="flex flex-col items-center">
            <div className={cn("mt-2 h-2 w-px", i === 0 ? "bg-transparent" : "bg-border")} />
            <div className="size-5 shrink-0 flex items-center justify-center rounded-full border bg-background">
              {agent.status === "running" ? (
                <IconLoader2 className="size-3 animate-spin text-violet-400" />
              ) : agent.status === "done" ? (
                <IconCheck className="size-3 text-green-500" />
              ) : (
                <IconX className="size-3 text-red-500" />
              )}
            </div>
            {i < agents.length - 1 && (
              <div className="flex-1 w-px bg-border min-h-[8px]" />
            )}
          </div>

          {/* Agent card */}
          <div className="flex-1 mb-2">
            <div className="flex items-center gap-2 py-1">
              <IconRobot className="size-3 text-muted-foreground shrink-0" />
              <span className="font-medium text-foreground/80">{agent.agent_name}</span>
              {agent.model && (
                <span className="rounded bg-violet-500/10 px-1.5 py-0.5 text-[10px] text-violet-400">
                  {agent.model}
                </span>
              )}
              {agent.tokens && (
                <span className="text-muted-foreground text-[10px]">
                  {agent.tokens.input.toLocaleString()}↑ {agent.tokens.output.toLocaleString()}↓
                </span>
              )}
            </div>

            {agent.tool_uses && agent.tool_uses.length > 0 && (
              <div className="ml-5 space-y-1">
                {agent.tool_uses.map((tool, j) => (
                  <ToolUseBlockCard key={`${tool.tool_name}-${j}`} tool={tool} />
                ))}
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}
