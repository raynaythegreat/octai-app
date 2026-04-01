import {
  Area,
  AreaChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

export interface UsageDataPoint {
  timestamp: string
  messages?: number
  tokens?: number
  inputTokens?: number
  outputTokens?: number
}

type Granularity = "hour" | "day" | "week"
type Variant = "area" | "line"

interface UsageChartProps {
  data: UsageDataPoint[]
  title?: string
  granularity?: Granularity
  variant?: Variant
  className?: string
  showLegend?: boolean
  metrics?: ("messages" | "tokens" | "inputTokens" | "outputTokens")[]
}

const formatTimestamp = (timestamp: string, granularity: Granularity): string => {
  const date = new Date(timestamp)
  switch (granularity) {
    case "hour":
      return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    case "day":
      return date.toLocaleDateString([], { month: "short", day: "numeric" })
    case "week":
      return date.toLocaleDateString([], { month: "short", day: "numeric" })
    default:
      return timestamp
  }
}

const formatNumber = (value: number): string => {
  if (value >= 1000000) {
    return `${(value / 1000000).toFixed(1)}M`
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}K`
  }
  return value.toString()
}

const COLORS = {
  messages: "#A855F7",
  tokens: "#8b5cf6",
  inputTokens: "#06b6d4",
  outputTokens: "#f59e0b",
}

const ChartTooltip = ({
  active,
  payload,
  label,
  granularity,
}: {
  active?: boolean
  payload?: Array<{ value?: number; dataKey?: string; color?: string }>
  label?: string
  granularity: Granularity
}) => {
  if (!active || !payload?.length) return null

  return (
    <div className="rounded-lg border border-border bg-background px-3 py-2 shadow-lg">
      <p className="mb-1 text-xs text-muted-foreground">
        {formatTimestamp(label ?? "", granularity)}
      </p>
      {payload.map((entry, index) => (
        <p
          key={index}
          className="text-sm"
          style={{ color: entry.color }}
        >
          {entry.dataKey}: {entry.value?.toLocaleString()}
        </p>
      ))}
    </div>
  )
}

export function UsageChart({
  data,
  title = "Usage Over Time",
  granularity = "day",
  variant = "area",
  className,
  showLegend = true,
  metrics = ["messages", "tokens"],
}: UsageChartProps) {
  const hasMessages = metrics.includes("messages") && data.some((d) => d.messages)
  const hasTokens = metrics.includes("tokens") && data.some((d) => d.tokens)
  const hasInputTokens = metrics.includes("inputTokens") && data.some((d) => d.inputTokens)
  const hasOutputTokens = metrics.includes("outputTokens") && data.some((d) => d.outputTokens)

  const ChartComponent = variant === "area" ? AreaChart : LineChart

  return (
    <Card className={cn("w-full", className)}>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-[300px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <ChartComponent data={data}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-border/50" />
              <XAxis
                dataKey="timestamp"
                tickFormatter={(value) => formatTimestamp(value, granularity)}
                tick={{ fontSize: 12 }}
                className="text-muted-foreground"
              />
              <YAxis
                tickFormatter={formatNumber}
                tick={{ fontSize: 12 }}
                className="text-muted-foreground"
              />
              <Tooltip content={<ChartTooltip granularity={granularity} />} />
              {showLegend && <Legend />}
              {variant === "area" ? (
                <>
                  {hasMessages && (
                    <Area
                      type="monotone"
                      dataKey="messages"
                      stroke={COLORS.messages}
                      fill={COLORS.messages}
                      fillOpacity={0.2}
                    />
                  )}
                  {hasTokens && (
                    <Area
                      type="monotone"
                      dataKey="tokens"
                      stroke={COLORS.tokens}
                      fill={COLORS.tokens}
                      fillOpacity={0.2}
                    />
                  )}
                  {hasInputTokens && (
                    <Area
                      type="monotone"
                      dataKey="inputTokens"
                      stroke={COLORS.inputTokens}
                      fill={COLORS.inputTokens}
                      fillOpacity={0.2}
                    />
                  )}
                  {hasOutputTokens && (
                    <Area
                      type="monotone"
                      dataKey="outputTokens"
                      stroke={COLORS.outputTokens}
                      fill={COLORS.outputTokens}
                      fillOpacity={0.2}
                    />
                  )}
                </>
              ) : (
                <>
                  {hasMessages && (
                    <Line
                      type="monotone"
                      dataKey="messages"
                      stroke={COLORS.messages}
                      strokeWidth={2}
                      dot={false}
                    />
                  )}
                  {hasTokens && (
                    <Line
                      type="monotone"
                      dataKey="tokens"
                      stroke={COLORS.tokens}
                      strokeWidth={2}
                      dot={false}
                    />
                  )}
                  {hasInputTokens && (
                    <Line
                      type="monotone"
                      dataKey="inputTokens"
                      stroke={COLORS.inputTokens}
                      strokeWidth={2}
                      dot={false}
                    />
                  )}
                  {hasOutputTokens && (
                    <Line
                      type="monotone"
                      dataKey="outputTokens"
                      stroke={COLORS.outputTokens}
                      strokeWidth={2}
                      dot={false}
                    />
                  )}
                </>
              )}
            </ChartComponent>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  )
}
