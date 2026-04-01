import { IconLoader } from "@tabler/icons-react"
import { useState } from "react"

import { PageHeader } from "@/components/page-header"

import { CreateWorkflowSheet } from "./CreateWorkflowSheet"
import { WorkflowList } from "./WorkflowList"
import { useTriggerWorkflow, useWorkflows } from "./hooks/useWorkflows"

export function WorkflowsPage() {
  const { data: workflows, isLoading, error } = useWorkflows()
  const triggerMutation = useTriggerWorkflow()
  const [createOpen, setCreateOpen] = useState(false)

  const handleRun = (workflowId: string) => {
    triggerMutation.mutate(workflowId)
  }

  const handleCreate = () => {
    setCreateOpen(true)
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <PageHeader title="Workflows" />
      <div className="flex-1 overflow-auto px-6 py-4">
        {isLoading && (
          <div className="flex items-center justify-center py-12 text-muted-foreground">
            <IconLoader className="size-5 animate-spin mr-2" />
            <span className="text-sm">Loading workflows…</span>
          </div>
        )}
        {error && (
          <div className="py-12 text-center text-sm text-destructive">
            Failed to load workflows: {error.message}
          </div>
        )}
        {!isLoading && !error && (
          <WorkflowList
            workflows={workflows ?? []}
            onRun={handleRun}
            onCreate={handleCreate}
          />
        )}
      </div>

      <CreateWorkflowSheet
        open={createOpen}
        onClose={() => setCreateOpen(false)}
      />
    </div>
  )
}
