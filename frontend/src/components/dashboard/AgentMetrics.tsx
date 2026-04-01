import {
  IconRobot,
  IconTrendingUp,
  IconClock,
  IconCoins,
} from "@tabler/icons-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

export interface AgentMetricData {
  agentId: string
  role?: string
  successRate: number
  turnsTotal: number
  avgLatencyMs: number
  tokensUsed: number
  costUSD: number
  state: "ready" | "busy" | "degraded" | "retired"
}

interface AgentMetricsProps {
  agents: AgentMetricData[]
}

const stateVariant = {
  ready:        "outline" as const,
  busy:         "default" as const,
  degraded:     "destructive" as const,
  retired:      "secondary" as const,
}

function formatLatency(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}


export function AgentMetrics({ agents }: AgentMetricsProps) {
  if (agents.length === 0) {
    return (
      <div className="flex h-32 items-center justify-center text-sm text-muted-foreground">
        No agent metrics available yet.
      </div>
    )
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {agents.map((agent) => (
        <Card key={agent.agentId} className="text-sm">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center justify-between text-sm">
              <span className="flex items-center gap-1.5">
                <IconRobot className="size-4 text-muted-foreground" />
                <span className="font-mono text-xs">{agent.agentId}</span>
              </span>
              <Badge variant={stateVariant[agent.state]} className="text-[10px]">
                {agent.state}
              </Badge>
            </CardTitle>
            {agent.role && (
              <p className="text-xs text-muted-foreground capitalize">{agent.role}</p>
            )}
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-2">
            <div className="flex items-center gap-1.5">
              <IconTrendingUp className="size-3.5 text-green-500 shrink-0" />
              <div>
                <p className="text-[10px] text-muted-foreground">Success</p>
                <p className="text-xs font-semibold">
                  {Math.round(agent.successRate * 100)}%
                </p>
              </div>
            </div>
            <div className="flex items-center gap-1.5">
              <IconClock className="size-3.5 text-blue-500 shrink-0" />
              <div>
                <p className="text-[10px] text-muted-foreground">Avg latency</p>
                <p className="text-xs font-semibold">{formatLatency(agent.avgLatencyMs)}</p>
              </div>
            </div>
            <div className="flex items-center gap-1.5">
              <IconRobot className="size-3.5 text-purple-500 shrink-0" />
              <div>
                <p className="text-[10px] text-muted-foreground">Turns</p>
                <p className="text-xs font-semibold">{agent.turnsTotal}</p>
              </div>
            </div>
            <div className="flex items-center gap-1.5">
              <IconCoins className="size-3.5 text-amber-500 shrink-0" />
              <div>
                <p className="text-[10px] text-muted-foreground">Cost</p>
                <p className="text-xs font-semibold">${agent.costUSD.toFixed(3)}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
