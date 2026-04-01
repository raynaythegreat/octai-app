import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import type { WorkflowSummary } from "../WorkflowList"

// --- API response types (matches web/backend/api/workflows.go) ---

interface WorkflowRunResponse {
  id: string
  workflow_id: string
  status: "pending" | "running" | "completed" | "failed" | "canceled"
  started_at?: string
  completed_at?: string
  error?: string
}

interface WorkflowSummaryResponse {
  id: string
  name: string
  description?: string
  team_id?: string
  trigger_kind?: string
  node_count: number
  recent_runs?: WorkflowRunResponse[]
  created_at: string
  updated_at: string
}

interface WorkflowsListResponse {
  workflows: WorkflowSummaryResponse[]
  total: number
}

interface TriggerRunResponse {
  run_id: string
  status: string
  message: string
}

// --- Fetch helpers ---

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    let message = `API error: ${res.status} ${res.statusText}`
    try {
      const body = (await res.json()) as { error?: string }
      if (body.error) message = body.error
    } catch {}
    throw new Error(message)
  }
  return res.json() as Promise<T>
}

async function fetchWorkflows(teamId?: string): Promise<WorkflowsListResponse> {
  const params = new URLSearchParams()
  if (teamId) params.set("team_id", teamId)
  const qs = params.toString()
  return request<WorkflowsListResponse>(`/api/v1/workflows${qs ? `?${qs}` : ""}`)
}

async function triggerWorkflow(workflowId: string): Promise<TriggerRunResponse> {
  return request<TriggerRunResponse>(`/api/v1/workflows/${workflowId}/run`, {
    method: "POST",
  })
}

export interface WorkflowCreatePayload {
  name: string
  description?: string
  trigger_kind?: string
  definition_json?: string
}

async function createWorkflow(payload: WorkflowCreatePayload): Promise<WorkflowSummaryResponse> {
  return request<WorkflowSummaryResponse>("/api/v1/workflows", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  })
}

// --- Map API response to component types ---

function formatRelativeTime(isoString?: string): string | undefined {
  if (!isoString) return undefined
  try {
    const diff = Date.now() - new Date(isoString).getTime()
    const mins = Math.floor(diff / 60_000)
    if (mins < 1) return "just now"
    if (mins < 60) return `${mins}m ago`
    const hrs = Math.floor(mins / 60)
    if (hrs < 24) return `${hrs}h ago`
    return `${Math.floor(hrs / 24)}d ago`
  } catch {
    return undefined
  }
}

function toWorkflowSummary(r: WorkflowSummaryResponse): WorkflowSummary {
  const lastRun = r.recent_runs?.[0]
  return {
    id: r.id,
    name: r.name,
    description: r.description,
    nodeCount: r.node_count,
    lastRunStatus: lastRun?.status as WorkflowSummary["lastRunStatus"],
    lastRunAt: formatRelativeTime(lastRun?.started_at),
    triggers: r.trigger_kind ? [r.trigger_kind] : [],
  }
}

// --- Hooks ---

export function useWorkflows(teamId?: string) {
  return useQuery({
    queryKey: ["workflows", teamId],
    queryFn: async () => {
      const res = await fetchWorkflows(teamId)
      return res.workflows.map(toWorkflowSummary)
    },
    staleTime: 30_000,
  })
}

export function useTriggerWorkflow() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: triggerWorkflow,
    onSuccess: (data) => {
      void queryClient.invalidateQueries({ queryKey: ["workflows"] })
      toast.success(`Workflow run queued (${data.run_id.slice(-8)})`)
    },
    onError: (err: Error) => {
      toast.error(`Failed to trigger workflow: ${err.message}`)
    },
  })
}

export function useCreateWorkflow() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: createWorkflow,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["workflows"] })
      toast.success("Workflow created")
    },
    onError: (err: Error) => {
      toast.error(`Failed to create workflow: ${err.message}`)
    },
  })
}
