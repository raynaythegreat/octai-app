import { IconCoin } from "@tabler/icons-react"
import {
  Bar,
  BarChart,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

export interface CostByProvider {
  provider: string
  cost: number
  percentage: number
}

export interface DailyCost {
  date: string
  cost: number
}

export interface CostBreakdownData {
  totalCost: number
  currency: string
  byProvider: CostByProvider[]
  dailyCosts: DailyCost[]
}

const PROVIDER_COLORS = [
  "#A855F7",
  "#8b5cf6",
  "#06b6d4",
  "#f59e0b",
  "#ec4899",
  "#10b981",
  "#f97316",
  "#6366f1",
]

const formatCurrency = (value: number, currency = "USD"): string => {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
  }).format(value)
}

const formatShortDate = (dateStr: string): string => {
  const date = new Date(dateStr)
  return date.toLocaleDateString([], { month: "short", day: "numeric" })
}

interface CostPieChartProps {
  data: CostByProvider[]
  currency: string
}

function CostPieChart({ data, currency }: CostPieChartProps) {
  const chartData = data.map((item) => ({
    name: item.provider,
    value: item.cost,
  }))

  return (
    <div className="flex h-[250px] items-center">
      <ResponsiveContainer width="50%" height="100%">
        <PieChart>
          <Pie
            data={chartData}
            cx="50%"
            cy="50%"
            innerRadius={50}
            outerRadius={80}
            paddingAngle={2}
            dataKey="value"
          >
            {chartData.map((_, index) => (
              <Cell
                key={`cell-${index}`}
                fill={PROVIDER_COLORS[index % PROVIDER_COLORS.length]}
              />
            ))}
          </Pie>
          <Tooltip
            formatter={(value) => formatCurrency(Number(value), currency)}
          />
        </PieChart>
      </ResponsiveContainer>
      <div className="flex-1 space-y-2">
        {data.map((item, index) => (
          <div key={item.provider} className="flex items-center gap-2">
            <div
              className="size-3 rounded-sm"
              style={{ backgroundColor: PROVIDER_COLORS[index % PROVIDER_COLORS.length] }}
            />
            <span className="flex-1 text-sm">{item.provider}</span>
            <span className="text-sm font-medium">
              {formatCurrency(item.cost, currency)}
            </span>
            <span className="w-12 text-right text-xs text-muted-foreground">
              {item.percentage.toFixed(1)}%
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

interface DailyCostChartProps {
  data: DailyCost[]
  currency: string
}

function DailyCostChart({ data, currency }: DailyCostChartProps) {
  return (
    <div className="h-[250px] w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data}>
          <XAxis
            dataKey="date"
            tickFormatter={formatShortDate}
            tick={{ fontSize: 11 }}
            className="text-muted-foreground"
          />
          <YAxis
            tickFormatter={(value) => `$${value}`}
            tick={{ fontSize: 11 }}
            className="text-muted-foreground"
          />
          <Tooltip
            labelFormatter={(label) => formatShortDate(String(label))}
            formatter={(value) => [formatCurrency(Number(value), currency), "Cost"]}
          />
          <Bar dataKey="cost" fill="#A855F7" radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}

interface CostBreakdownProps {
  data: CostBreakdownData
  className?: string
}

export function CostBreakdown({ data, className }: CostBreakdownProps) {
  return (
    <div className={cn("space-y-4", className)}>
      <Card size="sm">
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-sm font-medium">
            <IconCoin className="size-4 text-amber-500" />
            Total Cost
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-3xl font-bold">
            {formatCurrency(data.totalCost, data.currency)}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {data.byProvider.length} providers active
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Cost by Provider</CardTitle>
        </CardHeader>
        <CardContent>
          <CostPieChart data={data.byProvider} currency={data.currency} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Daily Costs</CardTitle>
        </CardHeader>
        <CardContent>
          <DailyCostChart data={data.dailyCosts} currency={data.currency} />
        </CardContent>
      </Card>
    </div>
  )
}

interface CostSummaryCardProps {
  totalCost: number
  currency: string
  period: string
  className?: string
}

export function CostSummaryCard({
  totalCost,
  currency,
  period,
  className,
}: CostSummaryCardProps) {
  return (
    <Card size="sm" className={cn("", className)}>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
          <IconCoin className="size-4" />
          Total Cost
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold">
          {formatCurrency(totalCost, currency)}
        </p>
        <p className="text-xs text-muted-foreground">{period}</p>
      </CardContent>
    </Card>
  )
}
