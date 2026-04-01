// OctAi - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 OctAi contributors

package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ScheduleType determines how the schedule is specified.
type ScheduleType string

const (
	ScheduleTypeInterval ScheduleType = "interval" // e.g., every 5 minutes
	ScheduleTypeCron     ScheduleType = "cron"     // e.g., "0 9 * * 1-5" (9am weekdays)
	ScheduleTypeOnce     ScheduleType = "once"     // run once at a specific time
)

// LoopTask defines a recurring agent task.
type LoopTask struct {
	ID           string        `json:"id"`
	Prompt       string        `json:"prompt"`
	Interval     time.Duration `json:"interval"`              // e.g. 5*time.Minute (interval type)
	ScheduleType ScheduleType  `json:"schedule_type"`         // "interval", "cron", "once"
	CronExpr     string        `json:"cron_expr,omitempty"`   // for cron type
	RunAt        *time.Time    `json:"run_at,omitempty"`      // for once type
	Timezone     string        `json:"timezone,omitempty"`    // default UTC
	SessionMode  string        `json:"session_mode,omitempty"` // "main", "isolated"
	Tags         []string      `json:"tags,omitempty"`
	MaxRuns      int           `json:"max_runs"`   // 0 = unlimited
	CreatedAt    time.Time     `json:"created_at"`
	ExpiresAt    time.Time     `json:"expires_at"` // zero = never expires (default: 72h)
	RunCount     int           `json:"run_count"`
	LastRunAt    time.Time     `json:"last_run_at"`
	NextRunAt    time.Time     `json:"next_run_at"`
	Status       string        `json:"status"` // "active", "paused", "expired", "completed"
}

// ── Pure-Go cron parser ────────────────────────────────────────────────────────

// cronField holds the parsed set of valid values for one cron position.
type cronField struct {
	values []int // sorted list of valid values for this field
}

// parsedCron holds the five parsed cron fields.
type parsedCron struct {
	minute     cronField // 0-59
	hour       cronField // 0-23
	dayOfMonth cronField // 1-31
	month      cronField // 1-12
	dayOfWeek  cronField // 0-6 (Sunday=0)
}

// parseCronExpr parses a cron expression and returns a matcher function.
// Supports: * * * * * format, ranges (1-5), lists (1,3,5), steps (*/5, 1-5/2).
// Shortcuts: @hourly, @daily, @weekly, @monthly.
func parseCronExpr(expr string) (func(time.Time) bool, error) {
	pc, err := parseCron(expr)
	if err != nil {
		return nil, err
	}
	return func(t time.Time) bool {
		return pc.matches(t)
	}, nil
}

// nextCronTime returns the next time a cron expression fires after 'after'.
func nextCronTime(expr string, after time.Time, tz *time.Location) (time.Time, error) {
	if tz == nil {
		tz = time.UTC
	}
	pc, err := parseCron(expr)
	if err != nil {
		return time.Time{}, err
	}
	return pc.next(after.In(tz)), nil
}

func parseCron(expr string) (*parsedCron, error) {
	expr = strings.TrimSpace(expr)

	// Handle shortcuts
	switch expr {
	case "@hourly":
		expr = "0 * * * *"
	case "@daily", "@midnight":
		expr = "0 0 * * *"
	case "@weekly":
		expr = "0 0 * * 0"
	case "@monthly":
		expr = "0 0 1 * *"
	case "@yearly", "@annually":
		expr = "0 0 1 1 *"
	}

	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d", len(parts))
	}

	minute, err := parseCronField(parts[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field %q: %w", parts[0], err)
	}
	hour, err := parseCronField(parts[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field %q: %w", parts[1], err)
	}
	dom, err := parseCronField(parts[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-month field %q: %w", parts[2], err)
	}
	month, err := parseCronField(parts[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field %q: %w", parts[3], err)
	}
	dow, err := parseCronField(parts[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-week field %q: %w", parts[4], err)
	}

	return &parsedCron{
		minute:     minute,
		hour:       hour,
		dayOfMonth: dom,
		month:      month,
		dayOfWeek:  dow,
	}, nil
}

// parseCronField parses a single cron field spec within [min, max].
func parseCronField(spec string, min, max int) (cronField, error) {
	var values []int
	seen := make(map[int]bool)

	add := func(v int) {
		if !seen[v] {
			seen[v] = true
			values = append(values, v)
		}
	}

	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return cronField{}, fmt.Errorf("empty field part")
		}

		// Check for step: value/step
		stepPart := 1
		if idx := strings.Index(part, "/"); idx != -1 {
			stepStr := part[idx+1:]
			s, err := strconv.Atoi(stepStr)
			if err != nil || s <= 0 {
				return cronField{}, fmt.Errorf("invalid step %q", stepStr)
			}
			stepPart = s
			part = part[:idx]
		}

		// Determine the range
		rangeMin, rangeMax := min, max
		if part != "*" {
			if idx := strings.Index(part, "-"); idx != -1 {
				lo, err1 := strconv.Atoi(part[:idx])
				hi, err2 := strconv.Atoi(part[idx+1:])
				if err1 != nil || err2 != nil {
					return cronField{}, fmt.Errorf("invalid range %q", part)
				}
				if lo < min || hi > max || lo > hi {
					return cronField{}, fmt.Errorf("range %d-%d out of bounds [%d,%d]", lo, hi, min, max)
				}
				rangeMin, rangeMax = lo, hi
			} else {
				v, err := strconv.Atoi(part)
				if err != nil {
					return cronField{}, fmt.Errorf("invalid value %q", part)
				}
				if v < min || v > max {
					return cronField{}, fmt.Errorf("value %d out of bounds [%d,%d]", v, min, max)
				}
				rangeMin, rangeMax = v, v
			}
		}

		for v := rangeMin; v <= rangeMax; v += stepPart {
			add(v)
		}
	}

	// Sort values
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}

	return cronField{values: values}, nil
}

// contains returns true if v is in the field's valid values.
func (f *cronField) contains(v int) bool {
	for _, x := range f.values {
		if x == v {
			return true
		}
	}
	return false
}

// nextOrEqual returns the smallest value >= v in the field, or -1 if none.
func (f *cronField) nextOrEqual(v int) int {
	for _, x := range f.values {
		if x >= v {
			return x
		}
	}
	return -1
}

// matches returns true if t matches this cron expression.
func (pc *parsedCron) matches(t time.Time) bool {
	// day-of-week: Sunday=0 in cron, Sunday=0 in time.Weekday
	dow := int(t.Weekday())
	return pc.minute.contains(t.Minute()) &&
		pc.hour.contains(t.Hour()) &&
		pc.dayOfMonth.contains(t.Day()) &&
		pc.month.contains(int(t.Month())) &&
		pc.dayOfWeek.contains(dow)
}

// next returns the next time after 'after' when the cron fires.
// It advances minute-by-minute until a match is found (up to ~4 years).
func (pc *parsedCron) next(after time.Time) time.Time {
	// Start from the next whole minute after 'after'
	t := after.Truncate(time.Minute).Add(time.Minute)

	// Safety: don't loop more than ~2 years worth of minutes
	limit := t.Add(2 * 366 * 24 * time.Hour)
	for t.Before(limit) {
		// Check month
		if !pc.month.contains(int(t.Month())) {
			// Advance to first valid month
			next := pc.month.nextOrEqual(int(t.Month()))
			if next == -1 {
				// Roll over to next year
				t = time.Date(t.Year()+1, 1, 1, 0, 0, 0, 0, t.Location())
			} else {
				t = time.Date(t.Year(), time.Month(next), 1, 0, 0, 0, 0, t.Location())
			}
			continue
		}
		// Check day-of-month
		if !pc.dayOfMonth.contains(t.Day()) {
			t = t.AddDate(0, 0, 1)
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			continue
		}
		// Check day-of-week
		if !pc.dayOfWeek.contains(int(t.Weekday())) {
			t = t.AddDate(0, 0, 1)
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			continue
		}
		// Check hour
		if !pc.hour.contains(t.Hour()) {
			next := pc.hour.nextOrEqual(t.Hour())
			if next == -1 {
				// Roll to next day
				t = t.AddDate(0, 0, 1)
				t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			} else {
				t = time.Date(t.Year(), t.Month(), t.Day(), next, 0, 0, 0, t.Location())
			}
			continue
		}
		// Check minute
		if !pc.minute.contains(t.Minute()) {
			next := pc.minute.nextOrEqual(t.Minute())
			if next == -1 {
				// Roll to next hour
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
			} else {
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), next, 0, 0, t.Location())
			}
			continue
		}
		return t
	}
	return time.Time{} // should never happen
}

// LoopScheduler manages recurring loop tasks, each firing an agent turn on a
// ticker-based interval.
type LoopScheduler struct {
	mu       sync.RWMutex
	tasks    map[string]*LoopTask
	cancels  map[string]context.CancelFunc
	maxLoops int // default 50
	// runFn is called each time a loop fires. Runs in its own goroutine.
	runFn func(ctx context.Context, taskID, prompt string)
}

// NewLoopScheduler creates a scheduler. runFn is invoked for every tick of
// every active task; it should execute one agent turn.
func NewLoopScheduler(runFn func(ctx context.Context, taskID, prompt string)) *LoopScheduler {
	return &LoopScheduler{
		tasks:    make(map[string]*LoopTask),
		cancels:  make(map[string]context.CancelFunc),
		maxLoops: 50,
		runFn:    runFn,
	}
}

// Add registers and starts a loop task. Returns an error if the scheduler is
// already at maxLoops capacity or if the schedule is invalid.
func (ls *LoopScheduler) Add(task LoopTask) error {
	// Default to interval mode for backward compatibility
	if task.ScheduleType == "" {
		task.ScheduleType = ScheduleTypeInterval
	}

	switch task.ScheduleType {
	case ScheduleTypeInterval:
		if task.Interval <= 0 {
			return fmt.Errorf("loop task interval must be positive")
		}
	case ScheduleTypeCron:
		if task.CronExpr == "" {
			return fmt.Errorf("loop task cron expression must not be empty")
		}
		if _, err := parseCronExpr(task.CronExpr); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	case ScheduleTypeOnce:
		if task.RunAt == nil {
			return fmt.Errorf("loop task run_at must be set for once schedule")
		}
		if task.RunAt.Before(time.Now()) {
			return fmt.Errorf("loop task run_at must be in the future")
		}
	default:
		return fmt.Errorf("unknown schedule type %q", task.ScheduleType)
	}

	ls.mu.Lock()
	defer ls.mu.Unlock()

	if len(ls.tasks) >= ls.maxLoops {
		return fmt.Errorf("loop scheduler at capacity (%d tasks)", ls.maxLoops)
	}

	if task.ID == "" {
		task.ID = fmt.Sprintf("loop-%d", time.Now().UnixNano())
	}

	now := time.Now()
	task.Status = "active"
	task.CreatedAt = now

	// Compute initial NextRunAt
	task.NextRunAt = ls.computeNextRun(&task, now)

	if task.ExpiresAt.IsZero() && task.ScheduleType != ScheduleTypeOnce {
		task.ExpiresAt = now.Add(72 * time.Hour)
	}

	ls.tasks[task.ID] = &task
	ls.start(&task)

	return nil
}

// computeNextRun returns the next scheduled run time for a task after 'after'.
func (ls *LoopScheduler) computeNextRun(task *LoopTask, after time.Time) time.Time {
	switch task.ScheduleType {
	case ScheduleTypeCron:
		tz := time.UTC
		if task.Timezone != "" {
			if loc, err := time.LoadLocation(task.Timezone); err == nil {
				tz = loc
			}
		}
		next, err := nextCronTime(task.CronExpr, after, tz)
		if err != nil {
			return time.Time{}
		}
		return next
	case ScheduleTypeOnce:
		if task.RunAt != nil {
			return *task.RunAt
		}
		return time.Time{}
	default: // ScheduleTypeInterval
		return after.Add(task.Interval)
	}
}

// Remove stops and removes a task. Returns an error if the task is not found.
func (ls *LoopScheduler) Remove(taskID string) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if _, ok := ls.tasks[taskID]; !ok {
		return fmt.Errorf("loop task %q not found", taskID)
	}

	if cancel, ok := ls.cancels[taskID]; ok {
		cancel()
		delete(ls.cancels, taskID)
	}
	delete(ls.tasks, taskID)

	return nil
}

// Pause suspends a task's ticker without removing it from the scheduler.
func (ls *LoopScheduler) Pause(taskID string) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	tsk, ok := ls.tasks[taskID]
	if !ok {
		return fmt.Errorf("loop task %q not found", taskID)
	}
	if tsk.Status == "paused" {
		return nil // already paused, no-op
	}

	// Cancel the running goroutine.
	if cancel, ok := ls.cancels[taskID]; ok {
		cancel()
		delete(ls.cancels, taskID)
	}
	tsk.Status = "paused"

	return nil
}

// Resume restarts a previously paused task.
func (ls *LoopScheduler) Resume(taskID string) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	tsk, ok := ls.tasks[taskID]
	if !ok {
		return fmt.Errorf("loop task %q not found", taskID)
	}
	if tsk.Status != "paused" {
		return fmt.Errorf("loop task %q is not paused (status: %s)", taskID, tsk.Status)
	}

	tsk.Status = "active"
	tsk.NextRunAt = ls.computeNextRun(tsk, time.Now())
	ls.start(tsk)

	return nil
}

// List returns a snapshot copy of all current tasks.
func (ls *LoopScheduler) List() []LoopTask {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	out := make([]LoopTask, 0, len(ls.tasks))
	for _, t := range ls.tasks {
		out = append(out, *t)
	}
	return out
}

// Get returns a single task by ID. The second return value is false if the
// task does not exist.
func (ls *LoopScheduler) Get(taskID string) (*LoopTask, bool) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	tsk, ok := ls.tasks[taskID]
	if !ok {
		return nil, false
	}
	copy := *tsk
	return &copy, true
}

// Stop cancels all running task goroutines.
func (ls *LoopScheduler) Stop() {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	for id, cancel := range ls.cancels {
		cancel()
		delete(ls.cancels, id)
	}
}

// start launches the scheduling goroutine for a task. Must be called with
// ls.mu held (write lock). Returns the cancel function that was stored.
func (ls *LoopScheduler) start(task *LoopTask) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	ls.cancels[task.ID] = cancel

	// Capture the task ID and schedule type; we re-read the task under lock
	// on each iteration so we always have fresh state.
	taskID := task.ID

	go func() {
		// Use a timer instead of a ticker so we can recalculate the delay
		// after each fire (needed for cron and once schedules).
		ls.mu.RLock()
		tsk := ls.tasks[taskID]
		if tsk == nil {
			ls.mu.RUnlock()
			return
		}
		delay := time.Until(tsk.NextRunAt)
		if delay < 0 {
			delay = 0
		}
		ls.mu.RUnlock()

		timer := time.NewTimer(delay)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case t := <-timer.C:
				ls.mu.Lock()
				tsk := ls.tasks[taskID]
				if tsk == nil || tsk.Status != "active" {
					ls.mu.Unlock()
					return
				}

				// Check expiry.
				if !tsk.ExpiresAt.IsZero() && t.After(tsk.ExpiresAt) {
					tsk.Status = "expired"
					ls.mu.Unlock()
					cancel()
					return
				}

				// Update run tracking.
				tsk.RunCount++
				tsk.LastRunAt = t

				if tsk.MaxRuns > 0 && tsk.RunCount >= tsk.MaxRuns {
					tsk.Status = "completed"
				}

				// For once tasks, mark completed after the single run.
				if tsk.ScheduleType == ScheduleTypeOnce {
					tsk.Status = "completed"
				} else {
					// Compute next run and set the timer.
					tsk.NextRunAt = ls.computeNextRun(tsk, t)
				}

				prompt := tsk.Prompt
				nextDelay := time.Until(tsk.NextRunAt)
				if nextDelay < 0 {
					nextDelay = 0
				}
				completed := tsk.Status == "completed"
				ls.mu.Unlock()

				// Run agent in a separate goroutine so we never block the timer.
				go ls.runFn(ctx, taskID, prompt)

				if completed {
					cancel()
					return
				}

				timer.Reset(nextDelay)
			}
		}
	}()

	return cancel
}
