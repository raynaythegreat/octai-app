import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState, useMemo, useCallback } from "react"
import { toast } from "sonner"

export type SkillCategory = "Productivity" | "Automation" | "Analytics" | "Communication" | "Integration"
export type SortOption = "popularity" | "rating" | "newest" | "price"

export interface MarketplaceSkill {
  id: string
  name: string
  author: string
  description: string
  longDescription?: string
  category: SkillCategory
  tags: string[]
  rating: number
  downloads: number
  price: number
  version: string
  versionHistory?: VersionInfo[]
  screenshots?: string[]
  documentationUrl?: string
  installed?: boolean
  enabled?: boolean
  hasUpdate?: boolean
  configSchema?: ConfigField[]
}

export interface VersionInfo {
  version: string
  date: string
  changes: string
}

export interface ConfigField {
  name: string
  label: string
  type: "text" | "password" | "number" | "boolean" | "select"
  required?: boolean
  default?: string | number | boolean
  options?: { label: string; value: string }[]
  description?: string
}

export interface SkillReview {
  id: string
  author: string
  rating: number
  comment: string
  date: string
}

interface MarketplaceFilters {
  category?: SkillCategory
  search?: string
  sortBy?: SortOption
  page?: number
  limit?: number
}

interface MarketplaceResponse {
  skills: MarketplaceSkill[]
  total: number
  page: number
  limit: number
}

interface InstalledSkillsResponse {
  skills: MarketplaceSkill[]
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    throw new Error(await extractErrorMessage(res))
  }
  return res.json() as Promise<T>
}

async function extractErrorMessage(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as { error?: string; errors?: string[] }
    if (Array.isArray(body.errors) && body.errors.length > 0) {
      return body.errors.join("; ")
    }
    if (typeof body.error === "string" && body.error.trim() !== "") {
      return body.error
    }
  } catch {}
  return `API error: ${res.status} ${res.statusText}`
}

async function fetchMarketplaceSkills(filters: MarketplaceFilters): Promise<MarketplaceResponse> {
  const params = new URLSearchParams()
  if (filters.category) params.set("category", filters.category)
  if (filters.search) params.set("search", filters.search)
  if (filters.sortBy) params.set("sort", filters.sortBy)
  if (filters.page) params.set("page", String(filters.page))
  if (filters.limit) params.set("limit", String(filters.limit))
  
  return request<MarketplaceResponse>(`/api/marketplace?${params.toString()}`)
}

async function fetchInstalledSkills(): Promise<InstalledSkillsResponse> {
  return request<InstalledSkillsResponse>("/api/marketplace/installed")
}

async function installSkill(skillId: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/marketplace/${skillId}/install`, {
    method: "POST",
  })
}

async function uninstallSkill(skillId: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/marketplace/${skillId}/uninstall`, {
    method: "DELETE",
  })
}

async function toggleSkill(skillId: string, enabled: boolean): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/marketplace/${skillId}/toggle`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  })
}

async function updateSkill(skillId: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/marketplace/${skillId}/update`, {
    method: "POST",
  })
}

async function submitRating(skillId: string, rating: number, comment?: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/marketplace/${skillId}/rate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ rating, comment }),
  })
}

async function fetchSkillReviews(skillId: string): Promise<{ reviews: SkillReview[] }> {
  return request<{ reviews: SkillReview[] }>(`/api/marketplace/${skillId}/reviews`)
}

async function testSkillConnection(skillId: string, config: Record<string, unknown>): Promise<{ success: boolean; message?: string }> {
  return request<{ success: boolean; message?: string }>(`/api/marketplace/${skillId}/test`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  })
}

async function saveSkillConfig(skillId: string, config: Record<string, unknown>): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/marketplace/${skillId}/config`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  })
}

export function useMarketplace() {
  const queryClient = useQueryClient()
  const [filters, setFilters] = useState<MarketplaceFilters>({
    sortBy: "popularity",
    page: 1,
    limit: 12,
  })

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["marketplace", filters],
    queryFn: () => fetchMarketplaceSkills(filters),
  })

  const skills = useMemo(() => data?.skills ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / (filters.limit ?? 12))

  const setSearch = useCallback((search: string) => {
    setFilters((prev) => ({ ...prev, search, page: 1 }))
  }, [])

  const setCategory = useCallback((category: SkillCategory | undefined) => {
    setFilters((prev) => ({ ...prev, category, page: 1 }))
  }, [])

  const setSortBy = useCallback((sortBy: SortOption) => {
    setFilters((prev) => ({ ...prev, sortBy, page: 1 }))
  }, [])

  const setPage = useCallback((page: number) => {
    setFilters((prev) => ({ ...prev, page }))
  }, [])

  const installMutation = useMutation({
    mutationFn: installSkill,
    onSuccess: () => {
      toast.success("Skill installed successfully")
      void queryClient.invalidateQueries({ queryKey: ["marketplace"] })
      void queryClient.invalidateQueries({ queryKey: ["installedSkills"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to install skill")
    },
  })

  const uninstallMutation = useMutation({
    mutationFn: uninstallSkill,
    onSuccess: () => {
      toast.success("Skill uninstalled successfully")
      void queryClient.invalidateQueries({ queryKey: ["marketplace"] })
      void queryClient.invalidateQueries({ queryKey: ["installedSkills"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to uninstall skill")
    },
  })

  const toggleMutation = useMutation({
    mutationFn: ({ skillId, enabled }: { skillId: string; enabled: boolean }) =>
      toggleSkill(skillId, enabled),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["installedSkills"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to toggle skill")
    },
  })

  const updateMutation = useMutation({
    mutationFn: updateSkill,
    onSuccess: () => {
      toast.success("Skill updated successfully")
      void queryClient.invalidateQueries({ queryKey: ["marketplace"] })
      void queryClient.invalidateQueries({ queryKey: ["installedSkills"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to update skill")
    },
  })

  const ratingMutation = useMutation({
    mutationFn: ({ skillId, rating, comment }: { skillId: string; rating: number; comment?: string }) =>
      submitRating(skillId, rating, comment),
    onSuccess: () => {
      toast.success("Rating submitted successfully")
      void queryClient.invalidateQueries({ queryKey: ["marketplace"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to submit rating")
    },
  })

  return {
    skills,
    total,
    totalPages,
    isLoading,
    error,
    filters,
    setSearch,
    setCategory,
    setSortBy,
    setPage,
    refetch,
    installSkill: installMutation.mutate,
    uninstallSkill: uninstallMutation.mutate,
    toggleSkill: toggleMutation.mutate,
    updateSkill: updateMutation.mutate,
    submitRating: ratingMutation.mutate,
    isInstalling: installMutation.isPending,
    isUninstalling: uninstallMutation.isPending,
    isUpdating: updateMutation.isPending,
    isSubmittingRating: ratingMutation.isPending,
  }
}

export function useInstalledSkills() {
  const queryClient = useQueryClient()

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["installedSkills"],
    queryFn: fetchInstalledSkills,
  })

  const uninstallMutation = useMutation({
    mutationFn: uninstallSkill,
    onSuccess: () => {
      toast.success("Skill uninstalled successfully")
      void queryClient.invalidateQueries({ queryKey: ["installedSkills"] })
      void queryClient.invalidateQueries({ queryKey: ["marketplace"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to uninstall skill")
    },
  })

  const toggleMutation = useMutation({
    mutationFn: ({ skillId, enabled }: { skillId: string; enabled: boolean }) =>
      toggleSkill(skillId, enabled),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["installedSkills"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to toggle skill")
    },
  })

  const updateMutation = useMutation({
    mutationFn: updateSkill,
    onSuccess: () => {
      toast.success("Skill updated successfully")
      void queryClient.invalidateQueries({ queryKey: ["installedSkills"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to update skill")
    },
  })

  return {
    skills: data?.skills ?? [],
    isLoading,
    error,
    refetch,
    uninstallSkill: uninstallMutation.mutate,
    toggleSkill: toggleMutation.mutate,
    updateSkill: updateMutation.mutate,
    isUninstalling: uninstallMutation.isPending,
    isUpdating: updateMutation.isPending,
  }
}

export function useSkillReviews(skillId: string) {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["skillReviews", skillId],
    queryFn: () => fetchSkillReviews(skillId),
    enabled: !!skillId,
  })

  return {
    reviews: data?.reviews ?? [],
    isLoading,
    error,
    refetch,
  }
}

export function useSkillConfig(skillId: string) {
  const queryClient = useQueryClient()
  const [isTesting, setIsTesting] = useState(false)
  const [testResult, setTestResult] = useState<{ success: boolean; message?: string } | null>(null)

  const saveMutation = useMutation({
    mutationFn: ({ config }: { config: Record<string, unknown> }) =>
      saveSkillConfig(skillId, config),
    onSuccess: () => {
      toast.success("Configuration saved successfully")
      void queryClient.invalidateQueries({ queryKey: ["installedSkills"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to save configuration")
    },
  })

  const testConnection = useCallback(
    async (config: Record<string, unknown>) => {
      setIsTesting(true)
      setTestResult(null)
      try {
        const result = await testSkillConnection(skillId, config)
        setTestResult(result)
        if (result.success) {
          toast.success("Connection test successful")
        } else {
          toast.error(result.message || "Connection test failed")
        }
        return result
      } catch (err) {
        const errorResult = { success: false, message: err instanceof Error ? err.message : "Connection test failed" }
        setTestResult(errorResult)
        toast.error(errorResult.message)
        return errorResult
      } finally {
        setIsTesting(false)
      }
    },
    [skillId]
  )

  return {
    saveConfig: saveMutation.mutate,
    isSaving: saveMutation.isPending,
    testConnection,
    isTesting,
    testResult,
    clearTestResult: () => setTestResult(null),
  }
}

export type { MarketplaceFilters, MarketplaceResponse, InstalledSkillsResponse }
