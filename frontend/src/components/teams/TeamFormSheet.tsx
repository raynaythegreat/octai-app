import { IconLoader2 } from "@tabler/icons-react"
import { useEffect, useState } from "react"

import { Field } from "@/components/shared-form"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Textarea } from "@/components/ui/textarea"

import type { TeamData } from "./TeamOverview"
import {
  type TeamFormData,
  useCreateTeam,
  useUpdateTeam,
} from "./hooks/useTeams"

interface TeamFormSheetProps {
  open: boolean
  onClose: () => void
  /** If provided, the sheet is in edit mode for this team. */
  team?: TeamData | null
}

interface FormState {
  name: string
  orchestratorId: string
  memberIds: string
  tokenBudget: string
  maxConcurrent: string
}

const EMPTY: FormState = {
  name: "",
  orchestratorId: "",
  memberIds: "",
  tokenBudget: "",
  maxConcurrent: "",
}

function teamToForm(team: TeamData): FormState {
  return {
    name: team.name,
    orchestratorId: team.orchestratorId,
    memberIds: team.memberIds.join("\n"),
    tokenBudget: team.tokenBudget > 0 ? String(team.tokenBudget) : "",
    maxConcurrent: team.maxConcurrent > 0 ? String(team.maxConcurrent) : "",
  }
}

export function TeamFormSheet({ open, onClose, team }: TeamFormSheetProps) {
  const isEdit = !!team
  const [form, setForm] = useState<FormState>(EMPTY)
  const [nameError, setNameError] = useState("")

  const createMutation = useCreateTeam()
  const updateMutation = useUpdateTeam()
  const saving = createMutation.isPending || updateMutation.isPending

  useEffect(() => {
    if (open) {
      setForm(team ? teamToForm(team) : EMPTY)
      setNameError("")
    }
  }, [open, team])

  const setField =
    (key: keyof FormState) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      setForm((f) => ({ ...f, [key]: e.target.value }))
      if (key === "name") setNameError("")
    }

  const buildPayload = (): TeamFormData => ({
    name: form.name.trim(),
    orchestratorId: form.orchestratorId.trim(),
    memberIds: form.memberIds
      .split("\n")
      .map((s) => s.trim())
      .filter(Boolean),
    tokenBudget: form.tokenBudget ? Number(form.tokenBudget) : 0,
    maxConcurrent: form.maxConcurrent ? Number(form.maxConcurrent) : 0,
  })

  const handleSave = () => {
    if (!form.name.trim()) {
      setNameError("Name is required")
      return
    }
    const payload = buildPayload()
    if (isEdit && team) {
      updateMutation.mutate(
        { id: team.id, data: payload },
        { onSuccess: onClose },
      )
    } else {
      createMutation.mutate(payload, { onSuccess: onClose })
    }
  }

  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent
        side="right"
        className="flex flex-col gap-0 p-0 data-[side=right]:!w-full data-[side=right]:sm:!w-[480px] data-[side=right]:sm:!max-w-[480px]"
      >
        <SheetHeader className="border-b border-b-muted px-6 py-5">
          <SheetTitle className="text-base">
            {isEdit ? `Edit team: ${team?.name}` : "Create team"}
          </SheetTitle>
          <SheetDescription className="text-xs">
            {isEdit
              ? "Update team configuration."
              : "Configure a new agent team."}
          </SheetDescription>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="space-y-5 px-6 py-5">
            <Field label="Name" required error={nameError}>
              <Input
                value={form.name}
                onChange={setField("name")}
                placeholder="e.g. Sales Team"
                aria-invalid={!!nameError}
              />
            </Field>

            <Field
              label="Orchestrator ID"
              hint="Agent ID that coordinates this team."
            >
              <Input
                value={form.orchestratorId}
                onChange={setField("orchestratorId")}
                placeholder="e.g. agent-sales-orchestrator"
                className="font-mono text-sm"
              />
            </Field>

            <Field
              label="Member IDs"
              hint="One agent ID per line. The orchestrator is not included here."
            >
              <Textarea
                value={form.memberIds}
                onChange={setField("memberIds")}
                placeholder={"agent-sales-rep\nagent-crm-writer"}
                rows={4}
                className="font-mono text-sm"
              />
            </Field>

            <Field
              label="Token budget"
              hint="Maximum tokens for the team per conversation. Leave blank for unlimited."
            >
              <Input
                value={form.tokenBudget}
                onChange={setField("tokenBudget")}
                placeholder="e.g. 200000"
                type="number"
                min={0}
              />
            </Field>

            <Field
              label="Max concurrent"
              hint="Maximum parallel agent turns. Leave blank for default."
            >
              <Input
                value={form.maxConcurrent}
                onChange={setField("maxConcurrent")}
                placeholder="e.g. 4"
                type="number"
                min={0}
              />
            </Field>
          </div>
        </div>

        <SheetFooter className="border-t border-t-muted px-6 py-4">
          <Button variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving && <IconLoader2 className="size-4 animate-spin" />}
            {isEdit ? "Save changes" : "Create team"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
