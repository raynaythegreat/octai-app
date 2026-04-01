import { IconClock, IconCoin, IconTrendingUp } from "@tabler/icons-react"
import type { ReactNode } from "react"

import { cn } from "@/lib/utils"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"

interface NavItem {
  id: string
  label: string
  icon: React.ComponentType<{ className?: string }>
}

const navItems: NavItem[] = [
  { id: "usage", label: "Usage", icon: IconTrendingUp },
  { id: "performance", label: "Performance", icon: IconClock },
  { id: "costs", label: "Costs", icon: IconCoin },
]

interface DashboardLayoutProps {
  children: ReactNode
  activeTab: string
  onTabChange: (tab: string) => void
  summaryCards?: ReactNode
}

export function DashboardLayout({
  children,
  activeTab,
  onTabChange,
  summaryCards,
}: DashboardLayoutProps) {
  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="border-b-border/50 border-b px-6 py-3">
        <nav className="flex gap-1">
          {navItems.map((item) => {
            const Icon = item.icon
            const isActive = activeTab === item.id
            return (
              <button
                key={item.id}
                onClick={() => onTabChange(item.id)}
                className={cn(
                  "flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                )}
              >
                <Icon className="size-4" />
                {item.label}
              </button>
            )
          })}
        </nav>
      </div>

      {summaryCards && (
        <div className="grid grid-cols-1 gap-4 px-6 py-4 sm:grid-cols-2 lg:grid-cols-4">
          {summaryCards}
        </div>
      )}

      <ScrollArea className="flex-1 px-6">
        <div className="pb-6">{children}</div>
      </ScrollArea>
    </div>
  )
}

interface SummaryCardProps {
  title: string
  value: string | number
  subtitle?: string
  trend?: "up" | "down" | "neutral"
  trendValue?: string
  icon?: ReactNode
}

export function SummaryCard({
  title,
  value,
  subtitle,
  trend,
  trendValue,
  icon,
}: SummaryCardProps) {
  return (
    <Card size="sm">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center justify-between text-sm font-medium text-muted-foreground">
          {title}
          {icon}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-semibold">{value}</div>
        {(subtitle || trendValue) && (
          <div className="mt-1 flex items-center gap-2 text-xs">
            {trend && trendValue && (
              <span
                className={cn(
                  "font-medium",
                  trend === "up" && "text-emerald-600",
                  trend === "down" && "text-violet-500",
                  trend === "neutral" && "text-muted-foreground"
                )}
              >
                {trend === "up" && "+"}
                {trendValue}
              </span>
            )}
            {subtitle && (
              <span className="text-muted-foreground">{subtitle}</span>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
