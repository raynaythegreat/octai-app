import { IconAlertTriangle, IconCheck, IconClock } from "@tabler/icons-react"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

export interface PerformanceData {
  avgResponseTime: number
  responseTimeTrend: number[]
  successRate: number
  successRateTrend: number[]
  errorRate: number
  errorRateTrend: number[]
  totalRequests: number
}

interface SparklineProps {
  data: number[]
  color: string
  height?: number
}

function Sparkline({ data, color, height = 24 }: SparklineProps) {
  if (!data.length) return null

  const min = Math.min(...data)
  const max = Math.max(...data)
  const range = max - min || 1
  const width = data.length * 4

  const points = data
    .map((value, index) => {
      const x = index * 4
      const y = height - ((value - min) / range) * (height - 4) - 2
      return `${x},${y}`
    })
    .join(" ")

  return (
    <svg width={width} height={height} className="overflow-visible">
      <polyline
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
        points={points}
      />
    </svg>
  )
}

interface MetricCardProps {
  title: string
  value: string | number
  unit?: string
  trend: number[]
  trendColor: string
  status?: "good" | "warning" | "critical"
  icon?: React.ReactNode
}

function MetricCard({
  title,
  value,
  unit,
  trend,
  trendColor,
  status,
  icon,
}: MetricCardProps) {
  const statusColors = {
    good: "text-emerald-600",
    warning: "text-amber-500",
    critical: "text-violet-500",
  }

  return (
    <Card size="sm">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center justify-between text-sm font-medium text-muted-foreground">
          {title}
          {icon && (
            <span className={status ? statusColors[status] : ""}>
              {icon}
            </span>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-end justify-between">
          <div>
            <span className="text-2xl font-semibold">{value}</span>
            {unit && (
              <span className="text-muted-foreground ml-1 text-sm">{unit}</span>
            )}
          </div>
          <Sparkline data={trend} color={trendColor} />
        </div>
      </CardContent>
    </Card>
  )
}

interface PerformanceMetricsProps {
  data: PerformanceData
  className?: string
}

export function PerformanceMetrics({ data, className }: PerformanceMetricsProps) {
  const responseTimeStatus =
    data.avgResponseTime < 1000
      ? "good"
      : data.avgResponseTime < 3000
        ? "warning"
        : "critical"

  const successStatus =
    data.successRate >= 99
      ? "good"
      : data.successRate >= 95
        ? "warning"
        : "critical"

  const errorStatus =
    data.errorRate < 1
      ? "good"
      : data.errorRate < 5
        ? "warning"
        : "critical"

  return (
    <div className={cn("grid gap-4 sm:grid-cols-2 lg:grid-cols-3", className)}>
      <MetricCard
        title="Avg Response Time"
        value={data.avgResponseTime}
        unit="ms"
        trend={data.responseTimeTrend}
        trendColor="#3b82f6"
        status={responseTimeStatus}
        icon={<IconClock className="size-4" />}
      />
      <MetricCard
        title="Success Rate"
        value={data.successRate.toFixed(1)}
        unit="%"
        trend={data.successRateTrend}
        trendColor="#10b981"
        status={successStatus}
        icon={<IconCheck className="size-4" />}
      />
      <MetricCard
        title="Error Rate"
        value={data.errorRate.toFixed(2)}
        unit="%"
        trend={data.errorRateTrend}
        trendColor="#ef4444"
        status={errorStatus}
        icon={<IconAlertTriangle className="size-4" />}
      />
    </div>
  )
}

interface PerformanceMetricsGridProps {
  metrics: PerformanceData
  className?: string
}

export function PerformanceMetricsGrid({
  metrics,
  className,
}: PerformanceMetricsGridProps) {
  return (
    <div className={cn("space-y-4", className)}>
      <PerformanceMetrics data={metrics} />
      <Card size="sm">
        <CardHeader>
          <CardTitle className="text-sm">Request Summary</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-3 gap-4 text-center">
            <div>
              <p className="text-2xl font-semibold">{metrics.totalRequests.toLocaleString()}</p>
              <p className="text-xs text-muted-foreground">Total Requests</p>
            </div>
            <div>
              <p className="text-2xl font-semibold text-emerald-600">
                {Math.round(metrics.totalRequests * (metrics.successRate / 100)).toLocaleString()}
              </p>
              <p className="text-xs text-muted-foreground">Successful</p>
            </div>
            <div>
              <p className="text-2xl font-semibold text-violet-500">
                {Math.round(metrics.totalRequests * (metrics.errorRate / 100)).toLocaleString()}
              </p>
              <p className="text-xs text-muted-foreground">Failed</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
