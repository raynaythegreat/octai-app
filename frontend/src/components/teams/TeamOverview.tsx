import {
  IconBolt,
  IconRobot,
  IconUsers,
  IconCircleCheck,
  IconAlertTriangle,
  IconLoader,
} from "@tabler/icons-react"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

export interface TeamMember {
  agentId: string
  role: string
  state: "ready" | "busy" | "degraded" | "retired" | "initializing"
  successRate: number
  activeTurns: number
  turnsTotal: number
}

export interface TeamData {
  id: string
  name: string
  orchestratorId: string
  memberIds: string[]
  members: TeamMember[]
  sharedKbPath?: string
  tokenBudget: number
  maxConcurrent: number
}

interface TeamOverviewProps {
  teams: TeamData[]
  onSelectTeam?: (teamId: string) => void
}

const roleColors: Record<string, string> = {
  orchestrator: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
  sales:        "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
  support:      "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  research:     "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300",
  content:      "bg-pink-100 text-pink-800 dark:bg-pink-900/30 dark:text-pink-300",
  analytics:    "bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-300",
  admin:        "bg-slate-100 text-slate-800 dark:bg-slate-900/30 dark:text-slate-300",
  custom:       "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300",
}

function StateIcon({ state }: { state: TeamMember["state"] }) {
  switch (state) {
    case "ready":
      return <IconCircleCheck className="size-3.5 text-green-500" />
    case "busy":
      return <IconLoader className="size-3.5 animate-spin text-blue-500" />
    case "degraded":
      return <IconAlertTriangle className="size-3.5 text-amber-500" />
    case "retired":
      return <IconCircleCheck className="size-3.5 text-slate-400" />
    default:
      return <IconLoader className="size-3.5 text-slate-400" />
  }
}

function MemberCard({ member }: { member: TeamMember }) {
  const roleClass = roleColors[member.role] ?? roleColors.custom

  return (
    <div className="flex items-center gap-3 rounded-lg border border-border/50 bg-muted/30 px-3 py-2">
      <IconRobot className="size-4 text-muted-foreground shrink-0" />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate text-sm font-medium">{member.agentId}</span>
          <span className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${roleClass}`}>
            {member.role}
          </span>
        </div>
        <div className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
          <StateIcon state={member.state} />
          <span>{member.state}</span>
          {member.turnsTotal > 0 && (
            <span>· {Math.round(member.successRate * 100)}% success</span>
          )}
          {member.activeTurns > 0 && (
            <span>· {member.activeTurns} active</span>
          )}
        </div>
      </div>
    </div>
  )
}

export function TeamOverview({ teams, onSelectTeam }: TeamOverviewProps) {
  if (teams.length === 0) {
    return (
      <div className="flex h-48 flex-col items-center justify-center gap-3 text-center">
        <IconUsers className="size-10 text-muted-foreground/40" />
        <div>
          <p className="text-sm font-medium">No teams configured</p>
          <p className="mt-1 text-xs text-muted-foreground">
            Add a <code className="rounded bg-muted px-1">teams</code> section to your config.json to create agent teams.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {teams.map((team) => (
        <Card
          key={team.id}
          className="cursor-pointer transition-shadow hover:shadow-md"
          onClick={() => onSelectTeam?.(team.id)}
        >
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base">
                <IconBolt className="size-4 text-primary" />
                {team.name}
              </CardTitle>
              <Badge variant="outline" className="text-xs">
                {team.members.length + 1} agents
              </Badge>
            </div>
            <p className="text-xs text-muted-foreground">
              Orchestrator: <span className="font-mono">{team.orchestratorId}</span>
            </p>
          </CardHeader>
          <CardContent className="space-y-2">
            {team.members.slice(0, 4).map((m) => (
              <MemberCard key={m.agentId} member={m} />
            ))}
            {team.members.length > 4 && (
              <p className="text-center text-xs text-muted-foreground">
                +{team.members.length - 4} more members
              </p>
            )}
            {(team.tokenBudget > 0 || team.maxConcurrent > 0 || team.sharedKbPath) && (
              <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 border-t border-border/40 pt-2 text-xs text-muted-foreground">
                {team.tokenBudget > 0 && (
                  <span>Budget: {team.tokenBudget.toLocaleString()} tokens</span>
                )}
                {team.maxConcurrent > 0 && (
                  <span>Concurrency: {team.maxConcurrent}</span>
                )}
                {team.sharedKbPath && (
                  <span className="truncate font-mono">KB: {team.sharedKbPath}</span>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
