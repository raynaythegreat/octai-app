import {
  IconCircleCheck,
  IconAlertTriangle,
  IconLoader,
  IconPencil,
  IconRobot,
  IconTrash,
} from "@tabler/icons-react"
import { useState } from "react"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"

import type { TeamData, TeamMember } from "./TeamOverview"
import { TeamFormSheet } from "./TeamFormSheet"
import { useDeleteTeam } from "./hooks/useTeams"

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

interface InfoRowProps {
  label: string
  value: React.ReactNode
}

function InfoRow({ label, value }: InfoRowProps) {
  return (
    <div className="flex items-start justify-between gap-4 py-2">
      <span className="text-xs text-muted-foreground shrink-0">{label}</span>
      <span className="text-xs font-medium text-right break-all">{value}</span>
    </div>
  )
}

interface TeamDetailSheetProps {
  team: TeamData | null
  onClose: () => void
  onDeleted: () => void
}

export function TeamDetailSheet({ team, onClose, onDeleted }: TeamDetailSheetProps) {
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const deleteMutation = useDeleteTeam()

  const handleDelete = () => {
    if (!team) return
    deleteMutation.mutate(team.id, {
      onSuccess: () => {
        setDeleteOpen(false)
        onDeleted()
        onClose()
      },
    })
  }

  return (
    <>
      <Sheet open={team !== null} onOpenChange={(v) => !v && onClose()}>
        <SheetContent
          side="right"
          className="flex flex-col gap-0 p-0 data-[side=right]:!w-full data-[side=right]:sm:!w-[440px] data-[side=right]:sm:!max-w-[440px]"
        >
          <SheetHeader className="border-b border-b-muted px-6 py-5">
            <div className="flex items-center justify-between gap-3 pr-8">
              <div className="min-w-0">
                <SheetTitle className="truncate text-base">
                  {team?.name ?? ""}
                </SheetTitle>
                <p className="mt-0.5 font-mono text-xs text-muted-foreground">
                  {team?.id}
                </p>
              </div>
              <div className="flex shrink-0 gap-1">
                <Button
                  size="icon-sm"
                  variant="ghost"
                  onClick={() => setEditOpen(true)}
                  title="Edit team"
                >
                  <IconPencil className="size-4" />
                </Button>
                <Button
                  size="icon-sm"
                  variant="ghost"
                  className="text-destructive hover:text-destructive"
                  onClick={() => setDeleteOpen(true)}
                  title="Delete team"
                >
                  <IconTrash className="size-4" />
                </Button>
              </div>
            </div>
          </SheetHeader>

          {team && (
            <div className="min-h-0 flex-1 overflow-y-auto px-6 py-4 space-y-4">
              {/* Config */}
              <div>
                <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-1">
                  Configuration
                </p>
                <div className="divide-y divide-border/50">
                  <InfoRow
                    label="Orchestrator"
                    value={
                      <span className="font-mono">{team.orchestratorId || "—"}</span>
                    }
                  />
                  <InfoRow
                    label="Token budget"
                    value={team.tokenBudget > 0 ? team.tokenBudget.toLocaleString() : "unlimited"}
                  />
                  <InfoRow
                    label="Max concurrent"
                    value={team.maxConcurrent > 0 ? String(team.maxConcurrent) : "default"}
                  />
                  {team.sharedKbPath && (
                    <InfoRow
                      label="Shared KB"
                      value={<span className="font-mono">{team.sharedKbPath}</span>}
                    />
                  )}
                </div>
              </div>

              <Separator />

              {/* Members */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                    Members
                  </p>
                  <Badge variant="outline" className="text-[10px]">
                    {team.members.length + 1} agents
                  </Badge>
                </div>

                {/* Orchestrator row */}
                <div className="flex items-center gap-3 rounded-lg border border-border/50 bg-muted/30 px-3 py-2 mb-2">
                  <IconRobot className="size-4 text-muted-foreground shrink-0" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium">
                        {team.orchestratorId}
                      </span>
                      <span className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${roleColors.orchestrator}`}>
                        orchestrator
                      </span>
                    </div>
                  </div>
                </div>

                {team.members.length === 0 ? (
                  <p className="text-xs text-muted-foreground text-center py-4">
                    No members configured
                  </p>
                ) : (
                  <div className="space-y-1.5">
                    {team.members.map((m) => {
                      const roleClass = roleColors[m.role] ?? roleColors.custom
                      return (
                        <div
                          key={m.agentId}
                          className="flex items-center gap-3 rounded-lg border border-border/50 bg-muted/30 px-3 py-2"
                        >
                          <IconRobot className="size-4 text-muted-foreground shrink-0" />
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2">
                              <span className="truncate text-sm font-medium">
                                {m.agentId}
                              </span>
                              <span className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${roleClass}`}>
                                {m.role}
                              </span>
                            </div>
                            <div className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
                              <StateIcon state={m.state} />
                              <span>{m.state}</span>
                              {m.turnsTotal > 0 && (
                                <span>· {Math.round(m.successRate * 100)}% success</span>
                              )}
                              {m.activeTurns > 0 && (
                                <span>· {m.activeTurns} active</span>
                              )}
                            </div>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            </div>
          )}
        </SheetContent>
      </Sheet>

      {/* Edit sheet */}
      <TeamFormSheet
        open={editOpen}
        onClose={() => setEditOpen(false)}
        team={team}
      />

      {/* Delete confirm */}
      <AlertDialog open={deleteOpen} onOpenChange={(v) => !v && setDeleteOpen(false)}>
        <AlertDialogContent size="sm">
          <AlertDialogHeader>
            <AlertDialogTitle>Delete team?</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete{" "}
              <strong>{team?.name}</strong>? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel
              onClick={() => setDeleteOpen(false)}
              disabled={deleteMutation.isPending}
            >
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
