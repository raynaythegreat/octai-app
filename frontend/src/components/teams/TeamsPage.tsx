import { IconLoader, IconPlus } from "@tabler/icons-react"
import { useState } from "react"

import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"

import { TeamDetailSheet } from "./TeamDetailSheet"
import { TeamFormSheet } from "./TeamFormSheet"
import { TeamOverview, type TeamData } from "./TeamOverview"
import { useTeams } from "./hooks/useTeams"

export function TeamsPage() {
  const { data: teams, isLoading, error } = useTeams()
  const [selectedTeam, setSelectedTeam] = useState<TeamData | null>(null)
  const [createOpen, setCreateOpen] = useState(false)

  const handleSelectTeam = (teamId: string) => {
    const team = teams?.find((t) => t.id === teamId) ?? null
    setSelectedTeam(team)
  }

  const handleDetailClosed = () => setSelectedTeam(null)
  const handleDeleted = () => setSelectedTeam(null)

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <PageHeader title="Agent Teams">
        <Button size="sm" variant="outline" onClick={() => setCreateOpen(true)}>
          <IconPlus className="size-4" />
          Create Team
        </Button>
      </PageHeader>

      <div className="flex-1 overflow-auto px-6 py-4">
        {isLoading && (
          <div className="flex items-center justify-center py-12 text-muted-foreground">
            <IconLoader className="size-5 animate-spin mr-2" />
            <span className="text-sm">Loading teams…</span>
          </div>
        )}
        {error && (
          <div className="py-12 text-center text-sm text-destructive">
            Failed to load teams: {error.message}
          </div>
        )}
        {!isLoading && !error && (
          <>
            {teams && teams.length === 0 ? (
              <div className="py-12 text-center text-sm text-muted-foreground">
                No teams configured yet.{" "}
                <button
                  onClick={() => setCreateOpen(true)}
                  className="underline underline-offset-4 hover:text-foreground transition-colors"
                >
                  Create your first team
                </button>{" "}
                or add a{" "}
                <code className="font-mono text-xs bg-muted px-1 rounded">teams</code>{" "}
                section to your{" "}
                <code className="font-mono text-xs bg-muted px-1 rounded">config.json</code>.
              </div>
            ) : (
              <TeamOverview
                teams={teams ?? []}
                onSelectTeam={handleSelectTeam}
              />
            )}
          </>
        )}
      </div>

      {/* Team detail side panel */}
      <TeamDetailSheet
        team={selectedTeam}
        onClose={handleDetailClosed}
        onDeleted={handleDeleted}
      />

      {/* Create team sheet */}
      <TeamFormSheet
        open={createOpen}
        onClose={() => setCreateOpen(false)}
      />
    </div>
  )
}
