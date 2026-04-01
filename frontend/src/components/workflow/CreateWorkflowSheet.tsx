import { IconLoader2 } from "@tabler/icons-react"
import { useEffect, useState } from "react"

import { Field } from "@/components/shared-form"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Textarea } from "@/components/ui/textarea"

import { useCreateWorkflow } from "./hooks/useWorkflows"

const DEFAULT_DEFINITION = JSON.stringify(
  {
    nodes: [],
    edges: [],
  },
  null,
  2,
)

interface FormState {
  name: string
  description: string
  triggerKind: string
  definitionJson: string
}

const EMPTY: FormState = {
  name: "",
  description: "",
  triggerKind: "manual",
  definitionJson: DEFAULT_DEFINITION,
}

interface CreateWorkflowSheetProps {
  open: boolean
  onClose: () => void
}

export function CreateWorkflowSheet({ open, onClose }: CreateWorkflowSheetProps) {
  const [form, setForm] = useState<FormState>(EMPTY)
  const [nameError, setNameError] = useState("")
  const [jsonError, setJsonError] = useState("")
  const createMutation = useCreateWorkflow()

  useEffect(() => {
    if (open) {
      setForm(EMPTY)
      setNameError("")
      setJsonError("")
    }
  }, [open])

  const setField =
    (key: keyof FormState) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      setForm((f) => ({ ...f, [key]: e.target.value }))
      if (key === "name") setNameError("")
      if (key === "definitionJson") setJsonError("")
    }

  const handleSave = () => {
    if (!form.name.trim()) {
      setNameError("Name is required")
      return
    }

    // Validate JSON
    try {
      JSON.parse(form.definitionJson)
    } catch {
      setJsonError("Invalid JSON — please fix the definition before saving")
      return
    }

    createMutation.mutate(
      {
        name: form.name.trim(),
        description: form.description.trim() || undefined,
        trigger_kind: form.triggerKind,
        definition_json: form.definitionJson.trim() || undefined,
      },
      { onSuccess: onClose },
    )
  }

  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent
        side="right"
        className="flex flex-col gap-0 p-0 data-[side=right]:!w-full data-[side=right]:sm:!w-[520px] data-[side=right]:sm:!max-w-[520px]"
      >
        <SheetHeader className="border-b border-b-muted px-6 py-5">
          <SheetTitle className="text-base">Create workflow</SheetTitle>
          <SheetDescription className="text-xs">
            Define a new automated workflow for your agent team.
          </SheetDescription>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="space-y-5 px-6 py-5">
            <Field label="Name" required error={nameError}>
              <Input
                value={form.name}
                onChange={setField("name")}
                placeholder="e.g. Daily report generation"
                aria-invalid={!!nameError}
              />
            </Field>

            <Field
              label="Description"
              hint="Optional summary of what this workflow does."
            >
              <Textarea
                value={form.description}
                onChange={setField("description")}
                placeholder="Describe the workflow purpose…"
                rows={2}
              />
            </Field>

            <Field
              label="Trigger type"
              hint="How this workflow is initiated."
            >
              <Select
                value={form.triggerKind}
                onValueChange={(v) => setForm((f) => ({ ...f, triggerKind: v }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="manual">Manual</SelectItem>
                  <SelectItem value="schedule">Schedule</SelectItem>
                  <SelectItem value="webhook">Webhook</SelectItem>
                </SelectContent>
              </Select>
            </Field>

            <Field
              label="Workflow definition (JSON)"
              hint="Paste or edit the workflow node/edge graph. Must be valid JSON."
              error={jsonError}
            >
              <Textarea
                value={form.definitionJson}
                onChange={setField("definitionJson")}
                rows={10}
                className="font-mono text-xs"
                aria-invalid={!!jsonError}
              />
            </Field>
          </div>
        </div>

        <SheetFooter className="border-t border-t-muted px-6 py-4">
          <Button
            variant="ghost"
            onClick={onClose}
            disabled={createMutation.isPending}
          >
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={createMutation.isPending}>
            {createMutation.isPending && (
              <IconLoader2 className="size-4 animate-spin" />
            )}
            Create workflow
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
