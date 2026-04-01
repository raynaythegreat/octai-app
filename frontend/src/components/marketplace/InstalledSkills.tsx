import {
  IconLoader2,
  IconSettings,
  IconTrash,
  IconUpload,
} from "@tabler/icons-react"
import { useState } from "react"

import type { MarketplaceSkill } from "./hooks/useMarketplace"
import { useInstalledSkills } from "./hooks/useMarketplace"
import SkillConfiguration from "./SkillConfiguration"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"

export function InstalledSkills() {
  const {
    skills,
    isLoading,
    error,
    uninstallSkill,
    toggleSkill,
    updateSkill,
    isUninstalling,
    isUpdating,
  } = useInstalledSkills()

  const [skillToUninstall, setSkillToUninstall] = useState<MarketplaceSkill | null>(null)
  const [configuringSkill, setConfiguringSkill] = useState<MarketplaceSkill | null>(null)

  const handleToggle = (skill: MarketplaceSkill) => {
    toggleSkill({ skillId: skill.id, enabled: !skill.enabled })
  }

  const handleUninstall = () => {
    if (skillToUninstall) {
      uninstallSkill(skillToUninstall.id)
      setSkillToUninstall(null)
    }
  }

  const handleUpdate = (skill: MarketplaceSkill) => {
    updateSkill(skill.id)
  }

  if (isLoading) {
    return (
      <div className="flex h-32 items-center justify-center">
        <IconLoader2 className="text-muted-foreground size-6 animate-spin" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="text-destructive text-sm">
        Failed to load installed skills
      </div>
    )
  }

  if (skills.length === 0) {
    return (
      <Card className="border-dashed">
        <CardContent className="text-muted-foreground py-10 text-center text-sm">
          No skills installed yet. Browse the marketplace to find skills.
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <div className="space-y-3">
        {skills.map((skill) => (
          <Card
            key={skill.id}
            className="border-border/60 bg-white/80 transition-colors hover:shadow-xs dark:bg-card/80"
            size="sm"
          >
            <CardHeader className="pb-2">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <CardTitle className="text-base font-semibold">
                      {skill.name}
                    </CardTitle>
                    {skill.hasUpdate && (
                      <span className="bg-primary/10 text-primary rounded px-1.5 py-0.5 text-[10px] font-medium">
                        Update Available
                      </span>
                    )}
                  </div>
                  <CardDescription className="mt-1">
                    {skill.description}
                  </CardDescription>
                </div>
                <div className="flex items-center gap-3">
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground text-xs">Enabled</span>
                    <Switch
                      checked={skill.enabled ?? true}
                      onCheckedChange={() => handleToggle(skill)}
                      size="sm"
                    />
                  </div>
                </div>
              </div>
            </CardHeader>
            <CardContent className="flex items-center justify-between">
              <div className="text-muted-foreground flex items-center gap-4 text-xs">
                <span>v{skill.version}</span>
                <span>by {skill.author}</span>
                <span className="bg-muted rounded px-1.5 py-0.5">{skill.category}</span>
              </div>
              <div className="flex items-center gap-1">
                {skill.hasUpdate && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleUpdate(skill)}
                    disabled={isUpdating}
                  >
                    {isUpdating ? (
                      <IconLoader2 className="size-3.5 animate-spin" />
                    ) : (
                      <IconUpload className="size-3.5" />
                    )}
                    Update
                  </Button>
                )}
                {skill.configSchema && skill.configSchema.length > 0 && (
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => setConfiguringSkill(skill)}
                    title="Configure"
                  >
                    <IconSettings className="size-4" />
                  </Button>
                )}
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => setSkillToUninstall(skill)}
                  title="Uninstall"
                  className="text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                >
                  <IconTrash className="size-4" />
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <AlertDialog
        open={skillToUninstall !== null}
        onOpenChange={(open) => !open && setSkillToUninstall(null)}
      >
        <AlertDialogContent size="sm">
          <AlertDialogHeader>
            <AlertDialogTitle>Uninstall Skill</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to uninstall "{skillToUninstall?.name}"? This
              action cannot be undone and any configuration will be lost.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isUninstalling}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={handleUninstall}
              disabled={isUninstalling}
            >
              {isUninstalling ? (
                <IconLoader2 className="size-4 animate-spin" />
              ) : (
                <IconTrash className="size-4" />
              )}
              Uninstall
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Sheet
        open={configuringSkill !== null}
        onOpenChange={(open) => !open && setConfiguringSkill(null)}
      >
        <SheetContent side="right" className="w-full sm:!w-[500px] sm:!max-w-[500px]">
          <SheetHeader>
            <SheetTitle>Configure {configuringSkill?.name}</SheetTitle>
            <SheetDescription>
              Adjust settings for this skill
            </SheetDescription>
          </SheetHeader>
          {configuringSkill && (
            <SkillConfiguration skill={configuringSkill} />
          )}
        </SheetContent>
      </Sheet>
    </>
  )
}

export default InstalledSkills
