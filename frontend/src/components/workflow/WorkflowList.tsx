import {
  IconCircleCheck,
  IconCircleX,
  IconClock,
  IconGitBranch,
  IconLoader,
  IconPlayerPlay,
  IconPlus,
} from "@tabler/icons-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

export interface WorkflowSummary {
  id: string
  name: string
  description?: string
  nodeCount: number
  lastRunStatus?: "completed" | "failed" | "running" | "pending"
  lastRunAt?: string
  triggers: string[]
}

interface WorkflowListProps {
  workflows: WorkflowSummary[]
  onRun?: (workflowId: string) => void
  onCreate?: () => void
}

const statusConfig = {
  completed: { label: "Completed", icon: IconCircleCheck, color: "text-green-500" },
  failed:    { label: "Failed",    icon: IconCircleX,     color: "text-violet-500"   },
  running:   { label: "Running",   icon: IconLoader,       color: "text-blue-500 animate-spin" },
  pending:   { label: "Pending",   icon: IconClock,        color: "text-amber-500" },
} as const

export function WorkflowList({ workflows, onRun, onCreate }: WorkflowListProps) {
  if (workflows.length === 0) {
    return (
      <div className="flex h-48 flex-col items-center justify-center gap-3 text-center">
        <IconGitBranch className="size-10 text-muted-foreground/40" />
        <div>
          <p className="text-sm font-medium">No workflows yet</p>
          <p className="mt-1 text-xs text-muted-foreground">
            Create a workflow to automate multi-step agent tasks.
          </p>
        </div>
        {onCreate && (
          <Button size="sm" onClick={onCreate} className="mt-2">
            <IconPlus className="size-4 mr-1" />
            Create Workflow
          </Button>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        {onCreate && (
          <Button size="sm" onClick={onCreate}>
            <IconPlus className="size-4 mr-1" />
            New Workflow
          </Button>
        )}
      </div>
      {workflows.map((wf) => {
        const status = wf.lastRunStatus ? statusConfig[wf.lastRunStatus] : null
        const StatusIcon = status?.icon

        return (
          <Card key={wf.id}>
            <CardHeader className="pb-2">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <CardTitle className="flex items-center gap-2 text-sm">
                    <IconGitBranch className="size-4 shrink-0 text-primary" />
                    {wf.name}
                  </CardTitle>
                  {wf.description && (
                    <p className="mt-0.5 text-xs text-muted-foreground line-clamp-1">
                      {wf.description}
                    </p>
                  )}
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  className="h-7 shrink-0 text-xs"
                  onClick={() => onRun?.(wf.id)}
                >
                  <IconPlayerPlay className="size-3 mr-1" />
                  Run
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <span>{wf.nodeCount} nodes</span>
                {wf.triggers.map((t) => (
                  <Badge key={t} variant="secondary" className="text-[10px]">
                    {t}
                  </Badge>
                ))}
                {status && StatusIcon && (
                  <span className="flex items-center gap-1">
                    <StatusIcon className={`size-3 ${status.color}`} />
                    {status.label}
                    {wf.lastRunAt && ` · ${wf.lastRunAt}`}
                  </span>
                )}
              </div>
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
