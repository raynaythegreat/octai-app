import { useQuery } from "@tanstack/react-query"

import type { CostBreakdownData } from "../CostBreakdown"
import type { PerformanceData } from "../PerformanceMetrics"
import type { UsageDataPoint } from "../UsageChart"

interface UsageData {
  hourly: UsageDataPoint[]
  daily: UsageDataPoint[]
}

interface AnalyticsSummary {
  totalMessages: number
  totalTokens: number
  inputTokens: number
  outputTokens: number
  messagesChange: number
  tokensChange: number
  costChange: number
}

interface AnalyticsData {
  usage: UsageData
  performance: PerformanceData
  costs: CostBreakdownData
  summary: AnalyticsSummary
}

interface UseAnalyticsParams {
  startDate: Date
  endDate: Date
  autoRefresh?: boolean
  refreshInterval?: number
}

interface AnalyticsApiResponse {
  usage: {
    hourly: Array<{
      timestamp: string
      messages: number
      tokens: number
      inputTokens: number
      outputTokens: number
    }>
    daily: Array<{
      timestamp: string
      messages: number
      tokens: number
      inputTokens: number
      outputTokens: number
    }>
  }
  performance: {
    avgResponseTime: number
    responseTimeTrend: number[]
    successRate: number
    successRateTrend: number[]
    errorRate: number
    errorRateTrend: number[]
    totalRequests: number
  }
  costs: {
    totalCost: number
    currency: string
    byProvider: Array<{
      provider: string
      cost: number
      percentage: number
    }>
    dailyCosts: Array<{
      date: string
      cost: number
    }>
  }
  summary: {
    totalMessages: number
    totalTokens: number
    inputTokens: number
    outputTokens: number
    messagesChange: number
    tokensChange: number
    costChange: number
  }
}

async function fetchAnalytics(
  startDate: Date,
  endDate: Date
): Promise<AnalyticsData> {
  const params = new URLSearchParams({
    start: startDate.toISOString(),
    end: endDate.toISOString(),
  })

  const response = await fetch(`/api/analytics?${params.toString()}`)

  if (!response.ok) {
    throw new Error(`Failed to fetch analytics: ${response.statusText}`)
  }

  const data: AnalyticsApiResponse = await response.json()

  return {
    usage: {
      hourly: data.usage.hourly.map((item) => ({
        timestamp: item.timestamp,
        messages: item.messages,
        tokens: item.tokens,
        inputTokens: item.inputTokens,
        outputTokens: item.outputTokens,
      })),
      daily: data.usage.daily.map((item) => ({
        timestamp: item.timestamp,
        messages: item.messages,
        tokens: item.tokens,
        inputTokens: item.inputTokens,
        outputTokens: item.outputTokens,
      })),
    },
    performance: {
      avgResponseTime: data.performance.avgResponseTime,
      responseTimeTrend: data.performance.responseTimeTrend,
      successRate: data.performance.successRate,
      successRateTrend: data.performance.successRateTrend,
      errorRate: data.performance.errorRate,
      errorRateTrend: data.performance.errorRateTrend,
      totalRequests: data.performance.totalRequests,
    },
    costs: {
      totalCost: data.costs.totalCost,
      currency: data.costs.currency,
      byProvider: data.costs.byProvider,
      dailyCosts: data.costs.dailyCosts,
    },
    summary: data.summary,
  }
}

export function useAnalytics({
  startDate,
  endDate,
  autoRefresh = true,
  refreshInterval = 30000,
}: UseAnalyticsParams) {
  return useQuery({
    queryKey: ["analytics", startDate.toISOString(), endDate.toISOString()],
    queryFn: () => fetchAnalytics(startDate, endDate),
    refetchInterval: autoRefresh ? refreshInterval : false,
    refetchIntervalInBackground: false,
    staleTime: 10000,
  })
}

export function useAnalyticsSummary(startDate: Date, endDate: Date) {
  return useQuery({
    queryKey: ["analytics-summary", startDate.toISOString(), endDate.toISOString()],
    queryFn: async () => {
      const params = new URLSearchParams({
        start: startDate.toISOString(),
        end: endDate.toISOString(),
      })

      const response = await fetch(`/api/analytics/summary?${params.toString()}`)

      if (!response.ok) {
        throw new Error(`Failed to fetch analytics summary: ${response.statusText}`)
      }

      return response.json() as Promise<AnalyticsSummary>
    },
    staleTime: 30000,
  })
}

export function useUsageData(
  startDate: Date,
  endDate: Date,
  granularity: "hour" | "day" = "day"
) {
  return useQuery({
    queryKey: [
      "analytics-usage",
      startDate.toISOString(),
      endDate.toISOString(),
      granularity,
    ],
    queryFn: async () => {
      const params = new URLSearchParams({
        start: startDate.toISOString(),
        end: endDate.toISOString(),
        granularity,
      })

      const response = await fetch(`/api/analytics/usage?${params.toString()}`)

      if (!response.ok) {
        throw new Error(`Failed to fetch usage data: ${response.statusText}`)
      }

      return response.json() as Promise<UsageDataPoint[]>
    },
    staleTime: 60000,
  })
}

export function usePerformanceMetrics(startDate: Date, endDate: Date) {
  return useQuery({
    queryKey: [
      "analytics-performance",
      startDate.toISOString(),
      endDate.toISOString(),
    ],
    queryFn: async () => {
      const params = new URLSearchParams({
        start: startDate.toISOString(),
        end: endDate.toISOString(),
      })

      const response = await fetch(
        `/api/analytics/performance?${params.toString()}`
      )

      if (!response.ok) {
        throw new Error(
          `Failed to fetch performance metrics: ${response.statusText}`
        )
      }

      return response.json() as Promise<PerformanceData>
    },
    staleTime: 30000,
  })
}

export function useCostBreakdown(startDate: Date, endDate: Date) {
  return useQuery({
    queryKey: [
      "analytics-costs",
      startDate.toISOString(),
      endDate.toISOString(),
    ],
    queryFn: async () => {
      const params = new URLSearchParams({
        start: startDate.toISOString(),
        end: endDate.toISOString(),
      })

      const response = await fetch(`/api/analytics/costs?${params.toString()}`)

      if (!response.ok) {
        throw new Error(`Failed to fetch cost breakdown: ${response.statusText}`)
      }

      return response.json() as Promise<CostBreakdownData>
    },
    staleTime: 60000,
  })
}

export type {
  AnalyticsData,
  AnalyticsSummary,
  UsageData,
  UseAnalyticsParams,
}
