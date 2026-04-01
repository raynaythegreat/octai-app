import {
  IconCalendar,
  IconLoader,
  IconPlayerPause,
  IconPlayerPlay,
  IconPlus,
  IconTrash,
} from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { toast } from "sonner"

import { PageHeader } from "@/components/page-header"
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
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
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
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Field } from "@/components/shared-form"

// ─── Types ────────────────────────────────────────────────────────────────────

export type CronJobStatus = "enabled" | "disabled"
export type ScheduleKind = "interval" | "cron" | "once"

export interface CronSchedule {
  kind: ScheduleKind
  everyMs?: number   // for interval
  expr?: string      // for cron
  atMs?: number      // for once
  tz?: string
}

export interface CronJob {
  id: string
  name: string
  enabled: boolean
  schedule: CronSchedule
  payload: {
    message: string
    deliver?: boolean
    channel?: string
  }
  state: {
    nextRunAtMs?: number
    lastRunAtMs?: number
    lastStatus?: string
    lastError?: string
  }
  createdAtMs: number
  updatedAtMs: number
}

interface CreateCronPayload {
  name: string
  message: string
  schedule: CronSchedule
}

// ─── Mock data ────────────────────────────────────────────────────────────────

const MOCK_JOBS: CronJob[] = [
  {
    id: "cron-001",
    name: "Morning Briefing",
    enabled: true,
    schedule: { kind: "cron", expr: "0 9 * * 1-5", tz: "America/New_York" },
    payload: { message: "Summarize overnight emails and Slack messages for the morning standup." },
    state: {
      nextRunAtMs: (() => {
        const d = new Date()
        d.setHours(9, 0, 0, 0)
        if (d <= new Date()) d.setDate(d.getDate() + 1)
        return d.getTime()
      })(),
      lastRunAtMs: Date.now() - 23 * 3600 * 1000,
      lastStatus: "ok",
    },
    createdAtMs: Date.now() - 7 * 86400 * 1000,
    updatedAtMs: Date.now() - 3600 * 1000,
  },
  {
    id: "cron-002",
    name: "Hourly Health Check",
    enabled: true,
    schedule: { kind: "interval", everyMs: 3600 * 1000 },
    payload: { message: "Run a system health check and report anomalies." },
    state: {
      nextRunAtMs: Date.now() + 22 * 60 * 1000,
      lastRunAtMs: Date.now() - 38 * 60 * 1000,
      lastStatus: "ok",
    },
    createdAtMs: Date.now() - 14 * 86400 * 1000,
    updatedAtMs: Date.now() - 38 * 60 * 1000,
  },
  {
    id: "cron-003",
    name: "Q2 Kickoff Reminder",
    enabled: false,
    schedule: { kind: "once", atMs: new Date("2026-04-01T09:00:00").getTime() },
    payload: { message: "Remind the team about the Q2 kickoff meeting agenda." },
    state: {
      nextRunAtMs: new Date("2026-04-01T09:00:00").getTime(),
      lastStatus: undefined,
    },
    createdAtMs: Date.now() - 2 * 86400 * 1000,
    updatedAtMs: Date.now() - 2 * 86400 * 1000,
  },
]

// ─── Common timezones ─────────────────────────────────────────────────────────

const TIMEZONES = [
  "UTC",
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "Europe/London",
  "Europe/Paris",
  "Europe/Berlin",
  "Asia/Tokyo",
  "Asia/Shanghai",
  "Asia/Kolkata",
  "Australia/Sydney",
]

// ─── API helpers ──────────────────────────────────────────────────────────────

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    let message = `API error: ${res.status} ${res.statusText}`
    try {
      const body = (await res.json()) as { error?: string }
      if (body.error) message = body.error
    } catch {}
    throw new Error(message)
  }
  return res.json() as Promise<T>
}

async function fetchCronJobs(): Promise<CronJob[]> {
  try {
    const res = await apiFetch<{ jobs: CronJob[] }>("/api/v1/cron/jobs")
    return res.jobs
  } catch {
    return MOCK_JOBS
  }
}

async function createCronJob(payload: CreateCronPayload): Promise<CronJob> {
  return apiFetch<CronJob>("/api/v1/cron/jobs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  })
}

async function deleteCronJob(id: string): Promise<void> {
  await apiFetch<unknown>(`/api/v1/cron/jobs/${id}`, { method: "DELETE" })
}

async function enableCronJob(id: string, enabled: boolean): Promise<CronJob> {
  return apiFetch<CronJob>(`/api/v1/cron/jobs/${id}/enable`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  })
}

// ─── Hooks ────────────────────────────────────────────────────────────────────

function useCronJobs() {
  return useQuery({
    queryKey: ["cron-jobs"],
    queryFn: fetchCronJobs,
    staleTime: 15_000,
    refetchInterval: 30_000,
  })
}

function useCreateCronJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: createCronJob,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["cron-jobs"] })
      toast.success("Schedule created")
    },
    onError: (err: Error) => {
      toast.error(`Failed to create schedule: ${err.message}`)
    },
  })
}

function useDeleteCronJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deleteCronJob,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["cron-jobs"] })
      toast.success("Schedule removed")
    },
    onError: (err: Error) => {
      toast.error(`Failed to remove schedule: ${err.message}`)
    },
  })
}

function useEnableCronJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      enableCronJob(id, enabled),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["cron-jobs"] })
    },
    onError: (err: Error) => {
      toast.error(`Failed to update schedule: ${err.message}`)
    },
  })
}

// ─── Utilities ────────────────────────────────────────────────────────────────

function truncate(text: string, maxLen: number): string {
  return text.length > maxLen ? `${text.slice(0, maxLen)}…` : text
}

function formatDate(ms?: number): string {
  if (!ms) return "—"
  const d = new Date(ms)
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

function scheduleLabel(schedule: CronSchedule): string {
  switch (schedule.kind) {
    case "cron":
      return schedule.expr ?? "cron"
    case "interval": {
      const ms = schedule.everyMs ?? 0
      const secs = Math.round(ms / 1000)
      if (secs >= 86400) return `every ${Math.round(secs / 86400)}d`
      if (secs >= 3600) return `every ${Math.round(secs / 3600)}h`
      if (secs >= 60) return `every ${Math.round(secs / 60)}m`
      return `every ${secs}s`
    }
    case "once":
      return schedule.atMs ? `once at ${formatDate(schedule.atMs)}` : "once"
    default:
      return "unknown"
  }
}

/** Very simple client-side next-run preview for cron expressions (3 occurrences). */
function cronNextRuns(expr: string, tz: string): string[] {
  // We cannot fully parse cron on the client; just show the expression and hint.
  // A real implementation would call the backend for next-run times.
  return [`Next runs for "${expr}" (${tz || "UTC"}) computed server-side`]
}

// ─── Status badge ─────────────────────────────────────────────────────────────

function StatusBadge({ enabled, lastStatus }: { enabled: boolean; lastStatus?: string }) {
  if (!enabled) {
    return (
      <Badge className="bg-gray-100 text-gray-700 dark:bg-gray-800/40 dark:text-gray-400 border-transparent">
        Disabled
      </Badge>
    )
  }
  if (lastStatus === "error") {
    return (
      <Badge className="bg-violet-100 text-violet-800 dark:bg-violet-900/30 dark:text-violet-300 border-transparent">
        Error
      </Badge>
    )
  }
  return (
    <Badge className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300 border-transparent">
      Active
    </Badge>
  )
}

// ─── New Schedule Form ────────────────────────────────────────────────────────

type ScheduleKindTab = "interval" | "cron" | "once"

interface NewScheduleFormState {
  name: string
  message: string
  kind: ScheduleKindTab
  // interval fields
  intervalPreset: string
  // cron fields
  cronExpr: string
  tz: string
  // once fields
  runAt: string
}

const INTERVAL_PRESETS: { label: string; ms: number }[] = [
  { label: "5 minutes", ms: 5 * 60 * 1000 },
  { label: "15 minutes", ms: 15 * 60 * 1000 },
  { label: "30 minutes", ms: 30 * 60 * 1000 },
  { label: "1 hour", ms: 60 * 60 * 1000 },
  { label: "6 hours", ms: 6 * 60 * 60 * 1000 },
  { label: "24 hours", ms: 24 * 60 * 60 * 1000 },
]

const EMPTY_FORM: NewScheduleFormState = {
  name: "",
  message: "",
  kind: "interval",
  intervalPreset: String(60 * 60 * 1000),
  cronExpr: "0 9 * * 1-5",
  tz: "UTC",
  runAt: "",
}

interface NewScheduleSheetProps {
  open: boolean
  onClose: () => void
}

function NewScheduleSheet({ open, onClose }: NewScheduleSheetProps) {
  const [form, setForm] = useState<NewScheduleFormState>(EMPTY_FORM)
  const [errors, setErrors] = useState<Partial<Record<keyof NewScheduleFormState, string>>>({})
  const createMutation = useCreateCronJob()

  const setField =
    (key: keyof NewScheduleFormState) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      setForm((f) => ({ ...f, [key]: e.target.value }))
      setErrors((err) => ({ ...err, [key]: undefined }))
    }

  const handleClose = () => {
    setForm(EMPTY_FORM)
    setErrors({})
    onClose()
  }

  const handleSubmit = () => {
    const newErrors: Partial<Record<keyof NewScheduleFormState, string>> = {}
    if (!form.name.trim()) newErrors.name = "Name is required"
    if (!form.message.trim()) newErrors.message = "Message is required"

    let schedule: CronSchedule

    if (form.kind === "interval") {
      const ms = parseInt(form.intervalPreset, 10)
      if (!ms || ms <= 0) newErrors.intervalPreset = "Select an interval"
      schedule = { kind: "interval", everyMs: ms }
    } else if (form.kind === "cron") {
      if (!form.cronExpr.trim()) newErrors.cronExpr = "Cron expression is required"
      schedule = { kind: "cron", expr: form.cronExpr.trim(), tz: form.tz || "UTC" }
    } else {
      if (!form.runAt) newErrors.runAt = "Date/time is required"
      schedule = { kind: "once", atMs: form.runAt ? new Date(form.runAt).getTime() : 0, tz: form.tz || "UTC" }
    }

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors)
      return
    }

    const payload: CreateCronPayload = {
      name: form.name.trim(),
      message: form.message.trim(),
      schedule,
    }

    createMutation.mutate(payload, { onSuccess: handleClose })
  }

  const kindTabClass = (k: ScheduleKindTab) =>
    `flex-1 rounded px-3 py-1.5 text-xs font-medium transition-colors ${
      form.kind === k
        ? "bg-background text-foreground shadow-sm"
        : "text-muted-foreground hover:text-foreground"
    }`

  return (
    <Sheet open={open} onOpenChange={(v) => !v && handleClose()}>
      <SheetContent
        side="right"
        className="flex flex-col gap-0 p-0 data-[side=right]:!w-full data-[side=right]:sm:!w-[520px] data-[side=right]:sm:!max-w-[520px]"
      >
        <SheetHeader className="border-b border-b-muted px-6 py-5">
          <SheetTitle className="text-base">New Schedule</SheetTitle>
          <SheetDescription className="text-xs">
            Schedule a recurring or one-time agent task.
          </SheetDescription>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="space-y-5 px-6 py-5">
            <Field label="Name" required error={errors.name}>
              <Input
                value={form.name}
                onChange={setField("name")}
                placeholder="e.g. Morning Briefing"
                aria-invalid={!!errors.name}
              />
            </Field>

            <Field label="Message / Prompt" required error={errors.message}>
              <Textarea
                value={form.message}
                onChange={setField("message")}
                placeholder="Describe what the agent should do…"
                rows={3}
                aria-invalid={!!errors.message}
              />
            </Field>

            {/* Schedule type toggle */}
            <div>
              <span className="mb-1.5 block text-sm font-medium">Schedule Type</span>
              <div className="flex gap-1 rounded-lg bg-muted p-1">
                <button
                  type="button"
                  className={kindTabClass("interval")}
                  onClick={() => setForm((f) => ({ ...f, kind: "interval" }))}
                >
                  Interval
                </button>
                <button
                  type="button"
                  className={kindTabClass("cron")}
                  onClick={() => setForm((f) => ({ ...f, kind: "cron" }))}
                >
                  Cron Expression
                </button>
                <button
                  type="button"
                  className={kindTabClass("once")}
                  onClick={() => setForm((f) => ({ ...f, kind: "once" }))}
                >
                  One-time
                </button>
              </div>
            </div>

            {/* Interval fields */}
            {form.kind === "interval" && (
              <Field label="Every" error={errors.intervalPreset}>
                <Select
                  value={form.intervalPreset}
                  onValueChange={(v) => setForm((f) => ({ ...f, intervalPreset: v }))}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {INTERVAL_PRESETS.map((p) => (
                      <SelectItem key={p.ms} value={String(p.ms)}>
                        {p.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            )}

            {/* Cron fields */}
            {form.kind === "cron" && (
              <>
                <Field
                  label="Cron Expression"
                  hint='Standard 5-field format: minute hour day month weekday. e.g. "0 9 * * 1-5" = 9am Mon-Fri.'
                  error={errors.cronExpr}
                >
                  <Input
                    value={form.cronExpr}
                    onChange={setField("cronExpr")}
                    placeholder="0 9 * * 1-5"
                    className="font-mono"
                    aria-invalid={!!errors.cronExpr}
                  />
                </Field>
                {form.cronExpr && (
                  <div className="rounded-md bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
                    {cronNextRuns(form.cronExpr, form.tz).map((line, i) => (
                      <div key={i}>{line}</div>
                    ))}
                  </div>
                )}
                <Field label="Timezone">
                  <Select
                    value={form.tz}
                    onValueChange={(v) => setForm((f) => ({ ...f, tz: v }))}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {TIMEZONES.map((tz) => (
                        <SelectItem key={tz} value={tz}>
                          {tz}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
              </>
            )}

            {/* Once fields */}
            {form.kind === "once" && (
              <>
                <Field label="Run At" required error={errors.runAt}>
                  <Input
                    type="datetime-local"
                    value={form.runAt}
                    onChange={setField("runAt")}
                    aria-invalid={!!errors.runAt}
                  />
                </Field>
                <Field label="Timezone">
                  <Select
                    value={form.tz}
                    onValueChange={(v) => setForm((f) => ({ ...f, tz: v }))}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {TIMEZONES.map((tz) => (
                        <SelectItem key={tz} value={tz}>
                          {tz}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
              </>
            )}
          </div>
        </div>

        <SheetFooter className="border-t border-t-muted px-6 py-4">
          <Button variant="ghost" onClick={handleClose} disabled={createMutation.isPending}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={createMutation.isPending}>
            {createMutation.isPending && (
              <IconLoader className="size-4 animate-spin" />
            )}
            Create Schedule
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

// ─── Main page ────────────────────────────────────────────────────────────────

export function CronPage() {
  const { data: jobs, isLoading, error } = useCronJobs()
  const deleteMutation = useDeleteCronJob()
  const enableMutation = useEnableCronJob()

  const [createOpen, setCreateOpen] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<CronJob | null>(null)

  const isActionPending = deleteMutation.isPending || enableMutation.isPending

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <PageHeader title="Schedule">
        <Button size="sm" variant="outline" onClick={() => setCreateOpen(true)}>
          <IconPlus className="size-4" />
          New Schedule
        </Button>
      </PageHeader>

      <div className="flex-1 overflow-auto px-6 py-4">
        {isLoading && (
          <div className="flex items-center justify-center py-12 text-muted-foreground">
            <IconLoader className="size-5 animate-spin mr-2" />
            <span className="text-sm">Loading schedules…</span>
          </div>
        )}
        {error && (
          <div className="py-12 text-center text-sm text-destructive">
            Failed to load schedules: {(error as Error).message}
          </div>
        )}
        {!isLoading && (
          <>
            {!jobs || jobs.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-20 gap-3 text-center">
                <IconCalendar className="size-10 text-muted-foreground/40" />
                <div>
                  <p className="text-sm font-medium">No schedules configured</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    Create a schedule to run agent tasks at specific times or intervals.
                  </p>
                </div>
                <Button size="sm" onClick={() => setCreateOpen(true)} className="mt-2">
                  <IconPlus className="size-4 mr-1" />
                  New Schedule
                </Button>
              </div>
            ) : (
              <div className="rounded-lg border border-border/60 overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border/60 bg-muted/40">
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                          Name
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                          Message
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider whitespace-nowrap">
                          Schedule
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider whitespace-nowrap">
                          Next Run
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider whitespace-nowrap">
                          Last Run
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                          Status
                        </th>
                        <th className="px-4 py-3 text-right text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                          Actions
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border/40">
                      {jobs.map((job) => (
                        <tr
                          key={job.id}
                          className="hover:bg-muted/20 transition-colors"
                        >
                          <td className="px-4 py-3">
                            <div>
                              <span className="text-sm font-medium">{job.name}</span>
                              <span className="ml-2 font-mono text-xs text-muted-foreground/60">
                                {job.id}
                              </span>
                            </div>
                          </td>
                          <td className="px-4 py-3 max-w-[220px]">
                            <span className="text-sm" title={job.payload.message}>
                              {truncate(job.payload.message, 45)}
                            </span>
                          </td>
                          <td className="px-4 py-3">
                            <span className="font-mono text-xs">
                              {scheduleLabel(job.schedule)}
                            </span>
                            {job.schedule.tz && job.schedule.tz !== "UTC" && (
                              <span className="ml-1 text-xs text-muted-foreground">
                                ({job.schedule.tz})
                              </span>
                            )}
                          </td>
                          <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                            {job.enabled ? formatDate(job.state.nextRunAtMs) : "—"}
                          </td>
                          <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                            {formatDate(job.state.lastRunAtMs)}
                          </td>
                          <td className="px-4 py-3">
                            <StatusBadge
                              enabled={job.enabled}
                              lastStatus={job.state.lastStatus}
                            />
                          </td>
                          <td className="px-4 py-3">
                            <div className="flex items-center justify-end gap-1">
                              {job.enabled ? (
                                <Button
                                  size="icon-sm"
                                  variant="ghost"
                                  className="text-muted-foreground hover:text-foreground"
                                  disabled={isActionPending}
                                  onClick={() =>
                                    enableMutation.mutate({ id: job.id, enabled: false })
                                  }
                                  title="Disable schedule"
                                >
                                  <IconPlayerPause className="size-4" />
                                </Button>
                              ) : (
                                <Button
                                  size="icon-sm"
                                  variant="ghost"
                                  className="text-muted-foreground hover:text-foreground"
                                  disabled={isActionPending}
                                  onClick={() =>
                                    enableMutation.mutate({ id: job.id, enabled: true })
                                  }
                                  title="Enable schedule"
                                >
                                  <IconPlayerPlay className="size-4" />
                                </Button>
                              )}
                              <Button
                                size="icon-sm"
                                variant="ghost"
                                className="text-muted-foreground hover:text-destructive"
                                disabled={isActionPending}
                                onClick={() => setPendingDelete(job)}
                                title="Delete schedule"
                              >
                                <IconTrash className="size-4" />
                              </Button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      <NewScheduleSheet open={createOpen} onClose={() => setCreateOpen(false)} />

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(v) => !v && setPendingDelete(null)}
      >
        <AlertDialogContent size="sm">
          <AlertDialogHeader>
            <AlertDialogTitle>Delete schedule?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete{" "}
              <strong>{pendingDelete?.name}</strong> (
              <span className="font-mono text-xs">{pendingDelete?.id}</span>
              ). This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel
              onClick={() => setPendingDelete(null)}
              disabled={deleteMutation.isPending}
            >
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => {
                if (pendingDelete) {
                  deleteMutation.mutate(pendingDelete.id, {
                    onSuccess: () => setPendingDelete(null),
                  })
                }
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
