import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import type { TeamData, TeamMember } from "../TeamOverview"

// --- API response types (matches web/backend/api/teams.go) ---

interface TeamMemberResponse {
  agent_id: string
  name?: string
  role?: string
}

interface TeamResponse {
  id: string
  name: string
  orchestrator_id: string
  members: TeamMemberResponse[]
  shared_kb_path?: string
  token_budget?: number
  max_concurrent?: number
}

interface TeamsListResponse {
  teams: TeamResponse[]
  total: number
}

export interface AgentSpec {
  role: string
  name?: string
  system_prompt_extra?: string
  tools?: string[]
  model?: string
}

export interface WorkflowSpec {
  name: string
  description?: string
  definition_json?: string
}

export interface TeamTemplate {
  id: string
  name: string
  description: string
  category: string
  agents: AgentSpec[]
  workflows?: WorkflowSpec[]
  author: string
  price: number
  rating: number
  downloads: number
  tags?: string[]
  created_at: string
  updated_at: string
}

interface TeamTemplatesResponse {
  templates: TeamTemplate[]
  total: number
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

async function fetchTeams(): Promise<TeamsListResponse> {
  return request<TeamsListResponse>("/api/v1/teams")
}

async function fetchTeamTemplates(): Promise<TeamTemplatesResponse> {
  return request<TeamTemplatesResponse>("/api/v1/teams/templates")
}

export interface TeamFormData {
  name: string
  orchestratorId: string
  memberIds: string[]
  tokenBudget: number
  maxConcurrent: number
}

async function createTeam(data: TeamFormData): Promise<TeamResponse> {
  return request<TeamResponse>("/api/v1/teams", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: data.name,
      orchestrator_id: data.orchestratorId,
      member_ids: data.memberIds,
      token_budget: data.tokenBudget,
      max_concurrent: data.maxConcurrent,
    }),
  })
}

async function updateTeam(id: string, data: TeamFormData): Promise<TeamResponse> {
  return request<TeamResponse>(`/api/v1/teams/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: data.name,
      orchestrator_id: data.orchestratorId,
      member_ids: data.memberIds,
      token_budget: data.tokenBudget,
      max_concurrent: data.maxConcurrent,
    }),
  })
}

async function deleteTeam(id: string): Promise<void> {
  await request<unknown>(`/api/v1/teams/${id}`, { method: "DELETE" })
}

// --- Map API response to component types ---

function toTeamData(r: TeamResponse): TeamData {
  const members: TeamMember[] = r.members.map((m) => ({
    agentId: m.agent_id,
    role: m.role ?? "custom",
    // Runtime metrics are not available from config — default to ready/zero
    state: "ready" as const,
    successRate: 0,
    activeTurns: 0,
    turnsTotal: 0,
  }))
  return {
    id: r.id,
    name: r.name,
    orchestratorId: r.orchestrator_id,
    memberIds: r.members.map((m) => m.agent_id),
    members,
    sharedKbPath: r.shared_kb_path,
    tokenBudget: r.token_budget ?? 0,
    maxConcurrent: r.max_concurrent ?? 0,
  }
}

// --- Hooks ---

export function useTeams() {
  return useQuery({
    queryKey: ["teams"],
    queryFn: async () => {
      const res = await fetchTeams()
      return res.teams.map(toTeamData)
    },
    staleTime: 30_000,
  })
}

export function useTeamTemplates() {
  return useQuery({
    queryKey: ["team-templates"],
    queryFn: async () => {
      const res = await fetchTeamTemplates()
      return res.templates
    },
    staleTime: 5 * 60_000,
  })
}

export function useCreateTeam() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: createTeam,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["teams"] })
      toast.success("Team created")
    },
    onError: (err: Error) => {
      toast.error(`Failed to create team: ${err.message}`)
    },
  })
}

export function useUpdateTeam() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: TeamFormData }) =>
      updateTeam(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["teams"] })
      toast.success("Team updated")
    },
    onError: (err: Error) => {
      toast.error(`Failed to update team: ${err.message}`)
    },
  })
}

export function useDeleteTeam() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deleteTeam,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["teams"] })
      toast.success("Team deleted")
    },
    onError: (err: Error) => {
      toast.error(`Failed to delete team: ${err.message}`)
    },
  })
}
