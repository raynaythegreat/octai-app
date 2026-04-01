import { IconCalendar, IconRefresh } from "@tabler/icons-react"
import { useMemo, useState } from "react"

import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { cn } from "@/lib/utils"
import { CostBreakdown } from "./CostBreakdown"
import { DashboardLayout, SummaryCard } from "./DashboardLayout"
import { PerformanceMetricsGrid } from "./PerformanceMetrics"
import { UsageChart } from "./UsageChart"
import { useAnalytics } from "./hooks/useAnalytics"

type TabId = "usage" | "performance" | "costs"

interface DateRange {
  start: Date
  end: Date
}

interface AnalyticsPageProps {
  className?: string
}

const DATE_RANGE_OPTIONS = [
  { label: "24h", days: 1 },
  { label: "7d", days: 7 },
  { label: "30d", days: 30 },
  { label: "90d", days: 90 },
]

export function AnalyticsPage({ className }: AnalyticsPageProps) {
  const [activeTab, setActiveTab] = useState<TabId>("usage")
  const [selectedRangeDays, setSelectedRangeDays] = useState(7)
  const [autoRefresh, setAutoRefresh] = useState(true)

  const dateRange = useMemo<DateRange>(() => {
    const end = new Date()
    const start = new Date()
    start.setDate(start.getDate() - selectedRangeDays)
    return { start, end }
  }, [selectedRangeDays])

  const { data, isLoading, error, refetch } = useAnalytics({
    startDate: dateRange.start,
    endDate: dateRange.end,
    autoRefresh,
    refreshInterval: 30000,
  })

  const handleRangeChange = (days: number) => {
    setSelectedRangeDays(days)
  }

  const renderContent = () => {
    if (isLoading && !data) {
      return (
        <div className="flex h-[400px] items-center justify-center">
          <div className="text-muted-foreground">Loading analytics...</div>
        </div>
      )
    }

    if (error) {
      return (
        <div className="flex h-[400px] items-center justify-center">
          <div className="text-violet-500">
            Error loading analytics: {error.message}
          </div>
        </div>
      )
    }

    if (!data) {
      return (
        <div className="flex h-[400px] items-center justify-center">
          <div className="text-muted-foreground">No data available</div>
        </div>
      )
    }

    switch (activeTab) {
      case "usage":
        return (
          <div className="space-y-4">
            <UsageChart
              data={data.usage.hourly}
              title="Hourly Usage"
              granularity="hour"
              variant="area"
              metrics={["messages", "tokens"]}
            />
            <UsageChart
              data={data.usage.daily}
              title="Daily Usage"
              granularity="day"
              variant="line"
              metrics={["messages", "tokens", "inputTokens", "outputTokens"]}
            />
          </div>
        )
      case "performance":
        return <PerformanceMetricsGrid metrics={data.performance} />
      case "costs":
        return <CostBreakdown data={data.costs} />
      default:
        return null
    }
  }

  const renderSummaryCards = () => {
    if (!data) return null

    switch (activeTab) {
      case "usage":
        return (
          <>
            <SummaryCard
              title="Total Messages"
              value={data.summary.totalMessages.toLocaleString()}
              trend="up"
              trendValue={`${data.summary.messagesChange}%`}
              subtitle="vs last period"
            />
            <SummaryCard
              title="Total Tokens"
              value={data.summary.totalTokens.toLocaleString()}
              trend="up"
              trendValue={`${data.summary.tokensChange}%`}
              subtitle="vs last period"
            />
            <SummaryCard
              title="Input Tokens"
              value={data.summary.inputTokens.toLocaleString()}
              trend="neutral"
            />
            <SummaryCard
              title="Output Tokens"
              value={data.summary.outputTokens.toLocaleString()}
              trend="neutral"
            />
          </>
        )
      case "performance":
        return (
          <>
            <SummaryCard
              title="Avg Response"
              value={`${data.performance.avgResponseTime}ms`}
              trend={
                data.performance.avgResponseTime < 1000
                  ? "up"
                  : "down"
              }
            />
            <SummaryCard
              title="Success Rate"
              value={`${data.performance.successRate.toFixed(1)}%`}
              trend="up"
            />
            <SummaryCard
              title="Error Rate"
              value={`${data.performance.errorRate.toFixed(2)}%`}
              trend="down"
            />
            <SummaryCard
              title="Total Requests"
              value={data.performance.totalRequests.toLocaleString()}
            />
          </>
        )
      case "costs":
        return (
          <>
            <SummaryCard
              title="Total Cost"
              value={`$${data.costs.totalCost.toFixed(2)}`}
              trend="up"
              trendValue={`${data.summary.costChange}%`}
              subtitle="vs last period"
            />
            <SummaryCard
              title="Top Provider"
              value={data.costs.byProvider[0]?.provider || "N/A"}
              subtitle={
                data.costs.byProvider[0]
                  ? `$${data.costs.byProvider[0].cost.toFixed(2)}`
                  : undefined
              }
            />
            <SummaryCard
              title="Avg Daily Cost"
              value={`$${(data.costs.totalCost / data.costs.dailyCosts.length).toFixed(2)}`}
            />
            <SummaryCard
              title="Active Providers"
              value={data.costs.byProvider.length}
            />
          </>
        )
      default:
        return null
    }
  }

  return (
    <div className={cn("flex h-full flex-col", className)}>
      <div className="flex items-center justify-between border-b-border/50 border-b px-6 py-3">
        <div className="flex items-center gap-2">
          <IconCalendar className="text-muted-foreground size-4" />
          <div className="flex gap-1">
            {DATE_RANGE_OPTIONS.map((option) => (
              <button
                key={option.days}
                onClick={() => handleRangeChange(option.days)}
                className={cn(
                  "rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
                  selectedRangeDays === option.days
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                )}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <Switch
              checked={autoRefresh}
              onCheckedChange={setAutoRefresh}
              size="sm"
            />
            <span className="text-sm text-muted-foreground">Auto-refresh</span>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => refetch()}
            disabled={isLoading}
          >
            <IconRefresh
              className={cn("size-4", isLoading && "animate-spin")}
            />
            Refresh
          </Button>
        </div>
      </div>

      <DashboardLayout
        activeTab={activeTab}
        onTabChange={(tab) => setActiveTab(tab as TabId)}
        summaryCards={renderSummaryCards()}
      >
        {renderContent()}
      </DashboardLayout>
    </div>
  )
}

export type { AnalyticsPageProps, DateRange }
