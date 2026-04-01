import {
  IconChevronLeft,
  IconChevronRight,
  IconLayoutGrid,
  IconList,
  IconLoader2,
  IconPackage,
  IconRefresh,
  IconSearch,
} from "@tabler/icons-react"
import { useState } from "react"

import type { MarketplaceSkill, SkillCategory, SortOption } from "./hooks/useMarketplace"
import { useMarketplace } from "./hooks/useMarketplace"
import { SkillCard } from "./SkillCard"
import { SkillDetailSheet } from "./SkillDetailSheet"
import { InstalledSkills } from "./InstalledSkills"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Card, CardContent } from "@/components/ui/card"

const CATEGORIES: SkillCategory[] = [
  "Productivity",
  "Automation",
  "Analytics",
  "Communication",
  "Integration",
]

const SORT_OPTIONS: { value: SortOption; label: string }[] = [
  { value: "popularity", label: "Most Popular" },
  { value: "rating", label: "Highest Rated" },
  { value: "newest", label: "Newest" },
  { value: "price", label: "Price: Low to High" },
]

export function MarketplacePage() {
  const [view, setView] = useState<"grid" | "list">("grid")
  const [activeTab, setActiveTab] = useState<"browse" | "installed">("browse")
  const [selectedSkill, setSelectedSkill] = useState<MarketplaceSkill | null>(null)

  const {
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
    installSkill,
    uninstallSkill,
    updateSkill,
    submitRating,
    isInstalling,
    isUninstalling,
    isUpdating,
    isSubmittingRating,
  } = useMarketplace()

  const handleInstall = (skillId: string) => {
    installSkill(skillId)
  }

  const handleUninstall = (skillId: string) => {
    uninstallSkill(skillId)
  }

  const handleUpdate = (skillId: string) => {
    updateSkill(skillId)
  }

  const handleSubmitRating = (skillId: string, rating: number, comment?: string) => {
    submitRating({ skillId, rating, comment })
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title="Skills Marketplace"
        titleExtra={
          <div className="flex gap-1">
            <Button
              variant={activeTab === "browse" ? "default" : "outline"}
              size="sm"
              onClick={() => setActiveTab("browse")}
            >
              Browse
            </Button>
            <Button
              variant={activeTab === "installed" ? "default" : "outline"}
              size="sm"
              onClick={() => setActiveTab("installed")}
            >
              Installed
            </Button>
          </div>
        }
      >
        <Button variant="outline" size="icon-sm" onClick={() => refetch()}>
          <IconRefresh className="size-4" />
        </Button>
      </PageHeader>

      <div className="flex-1 overflow-auto px-6 py-3">
        <div className="w-full max-w-6xl space-y-6">
          {activeTab === "browse" ? (
            <>
              <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                <div className="relative w-full sm:w-72">
                  <IconSearch className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
                  <Input
                    placeholder="Search skills..."
                    value={filters.search ?? ""}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-9"
                  />
                </div>

                <div className="flex items-center gap-2">
                  <Select
                    value={filters.category ?? "all"}
                    onValueChange={(v) => setCategory(v === "all" ? undefined : (v as SkillCategory))}
                  >
                    <SelectTrigger className="w-40">
                      <SelectValue placeholder="Category" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Categories</SelectItem>
                      {CATEGORIES.map((cat) => (
                        <SelectItem key={cat} value={cat}>
                          {cat}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>

                  <Select
                    value={filters.sortBy ?? "popularity"}
                    onValueChange={(v) => setSortBy(v as SortOption)}
                  >
                    <SelectTrigger className="w-44">
                      <SelectValue placeholder="Sort by" />
                    </SelectTrigger>
                    <SelectContent>
                      {SORT_OPTIONS.map((opt) => (
                        <SelectItem key={opt.value} value={opt.value}>
                          {opt.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>

                  <div className="hidden gap-0.5 sm:flex">
                    <Button
                      variant={view === "grid" ? "default" : "ghost"}
                      size="icon-sm"
                      onClick={() => setView("grid")}
                    >
                      <IconLayoutGrid className="size-4" />
                    </Button>
                    <Button
                      variant={view === "list" ? "default" : "ghost"}
                      size="icon-sm"
                      onClick={() => setView("list")}
                    >
                      <IconList className="size-4" />
                    </Button>
                  </div>
                </div>
              </div>

              {isLoading ? (
                <div className="flex h-48 items-center justify-center">
                  <IconLoader2 className="text-muted-foreground size-8 animate-spin" />
                </div>
              ) : error ? (
                <Card className="border-destructive/50">
                  <CardContent className="text-destructive py-10 text-center text-sm">
                    Failed to load skills. Please try again.
                  </CardContent>
                </Card>
              ) : skills.length === 0 ? (
                <Card className="border-dashed">
                  <CardContent className="text-muted-foreground py-16 text-center">
                    <IconPackage className="mx-auto mb-4 size-12 opacity-50" />
                    <p className="text-sm">No skills found matching your criteria.</p>
                  </CardContent>
                </Card>
              ) : (
                <>
                  <div className="text-muted-foreground text-sm">
                    Showing {skills.length} of {total} skills
                  </div>

                  <div
                    className={
                      view === "grid"
                        ? "grid gap-4 sm:grid-cols-2 lg:grid-cols-3"
                        : "space-y-3"
                    }
                  >
                    {skills.map((skill) => (
                      <SkillCard
                        key={skill.id}
                        skill={skill}
                        onPreview={setSelectedSkill}
                        onInstall={handleInstall}
                        isInstalling={isInstalling}
                      />
                    ))}
                  </div>

                  {totalPages > 1 && (
                    <div className="flex items-center justify-center gap-2 pt-4">
                      <Button
                        variant="outline"
                        size="icon-sm"
                        onClick={() => setPage((filters.page ?? 1) - 1)}
                        disabled={(filters.page ?? 1) <= 1}
                      >
                        <IconChevronLeft className="size-4" />
                      </Button>
                      <span className="text-muted-foreground text-sm">
                        Page {filters.page ?? 1} of {totalPages}
                      </span>
                      <Button
                        variant="outline"
                        size="icon-sm"
                        onClick={() => setPage((filters.page ?? 1) + 1)}
                        disabled={(filters.page ?? 1) >= totalPages}
                      >
                        <IconChevronRight className="size-4" />
                      </Button>
                    </div>
                  )}
                </>
              )}
            </>
          ) : (
            <InstalledSkills />
          )}
        </div>
      </div>

      <SkillDetailSheet
        skill={selectedSkill}
        open={selectedSkill !== null}
        onOpenChange={(open) => !open && setSelectedSkill(null)}
        onInstall={handleInstall}
        onUninstall={handleUninstall}
        onUpdate={handleUpdate}
        onSubmitRating={handleSubmitRating}
        isInstalling={isInstalling}
        isUninstalling={isUninstalling}
        isUpdating={isUpdating}
        isSubmittingRating={isSubmittingRating}
      />
    </div>
  )
}

export default MarketplacePage
