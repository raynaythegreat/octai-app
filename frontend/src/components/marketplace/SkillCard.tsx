import {
  IconDownload,
  IconEye,
  IconLoader2,
  IconStar,
  IconStarFilled,
} from "@tabler/icons-react"

import type { MarketplaceSkill } from "./hooks/useMarketplace"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"

interface SkillCardProps {
  skill: MarketplaceSkill
  onPreview: (skill: MarketplaceSkill) => void
  onInstall: (skillId: string) => void
  isInstalling?: boolean
}

export function SkillCard({ skill, onPreview, onInstall, isInstalling }: SkillCardProps) {
  const formatDownloads = (count: number): string => {
    if (count >= 1000000) return `${(count / 1000000).toFixed(1)}M`
    if (count >= 1000) return `${(count / 1000).toFixed(1)}K`
    return String(count)
  }

  const formatPrice = (price: number): string => {
    return price === 0 ? "Free" : `$${price.toFixed(2)}`
  }

  const renderStars = (rating: number) => {
    const stars = []
    const fullStars = Math.floor(rating)
    for (let i = 0; i < 5; i++) {
      if (i < fullStars) {
        stars.push(
          <IconStarFilled key={i} className="size-3 text-yellow-500" />
        )
      } else {
        stars.push(
          <IconStar key={i} className="size-3 text-muted-foreground/30" />
        )
      }
    }
    return stars
  }

  const getCategoryColor = (category: string): string => {
    const colors: Record<string, string> = {
      Productivity: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
      Automation: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
      Analytics: "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400",
      Communication: "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400",
      Integration: "bg-pink-100 text-pink-700 dark:bg-pink-900/30 dark:text-pink-400",
    }
    return colors[category] || "bg-muted text-muted-foreground"
  }

  return (
    <Card className="border-border/60 bg-white/80 transition-all hover:shadow-md dark:bg-card/80" size="sm">
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <CardTitle className="truncate text-base font-semibold">
              {skill.name}
            </CardTitle>
            <p className="text-muted-foreground mt-1 truncate text-xs">
              by {skill.author}
            </p>
          </div>
          <span className="text-sm font-semibold text-foreground">
            {formatPrice(skill.price)}
          </span>
        </div>
      </CardHeader>

      <CardContent className="pb-3">
        <p className="text-muted-foreground line-clamp-2 text-sm leading-snug">
          {skill.description}
        </p>

        <div className="mt-3 flex flex-wrap gap-1.5">
          <span className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${getCategoryColor(skill.category)}`}>
            {skill.category}
          </span>
          {skill.tags.slice(0, 2).map((tag) => (
            <span
              key={tag}
              className="bg-muted/60 text-muted-foreground rounded px-1.5 py-0.5 text-[10px] font-medium"
            >
              {tag}
            </span>
          ))}
        </div>
      </CardContent>

      <CardFooter className="flex items-center justify-between border-t pt-3">
        <div className="flex items-center gap-3 text-xs">
          <div className="flex items-center gap-0.5">
            {renderStars(skill.rating)}
            <span className="text-muted-foreground ml-1">{skill.rating.toFixed(1)}</span>
          </div>
          <div className="text-muted-foreground flex items-center gap-1">
            <IconDownload className="size-3" />
            {formatDownloads(skill.downloads)}
          </div>
        </div>

        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onPreview(skill)}
            title="Preview"
          >
            <IconEye className="size-3.5" />
          </Button>
          <Button
            variant={skill.installed ? "outline" : "default"}
            size="sm"
            onClick={() => !skill.installed && onInstall(skill.id)}
            disabled={skill.installed || isInstalling}
          >
            {isInstalling ? (
              <IconLoader2 className="size-3.5 animate-spin" />
            ) : skill.installed ? (
              "Installed"
            ) : (
              <>
                <IconDownload className="size-3.5" />
                Install
              </>
            )}
          </Button>
        </div>
      </CardFooter>
    </Card>
  )
}
