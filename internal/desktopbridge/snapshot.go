package desktopbridge

import (
	"strings"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/memory"
	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

const Schema = 1

type TaskSummary struct {
	ID          string           `json:"id"`
	Goal        string           `json:"goal"`
	Agent       string           `json:"agent"`
	Status      taskgraph.Status `json:"status"`
	StepCount   int              `json:"step_count"`
	DoneSteps   int              `json:"done_steps"`
	FailedSteps int              `json:"failed_steps"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// Snapshot is a per-UID desktop payload. It may contain the user's task goal,
// so it is private rather than world-readable, but it intentionally excludes
// raw task step targets/capabilities/results and all Memory Data.
type Snapshot struct {
	Schema    int            `json:"schema"`
	Product   string         `json:"product"`
	Version   string         `json:"version"`
	UserUID   uint32         `json:"user_uid"`
	Tasks     []TaskSummary  `json:"tasks"`
	Memory    memory.Summary `json:"memory"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func Build(userUID uint32, version string, tasks []taskgraph.Task, memorySummary memory.Summary, now time.Time) Snapshot {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	return Snapshot{
		Schema:    Schema,
		Product:   "KINGAI OS Desktop",
		Version:   strings.TrimSpace(version),
		UserUID:   userUID,
		Tasks:     summarizeTasks(tasks),
		Memory:    memorySummary,
		UpdatedAt: now,
	}
}

func summarizeTasks(tasks []taskgraph.Task) []TaskSummary {
	out := make([]TaskSummary, 0, len(tasks))
	for _, task := range tasks {
		item := TaskSummary{
			ID:        truncate(strings.TrimSpace(task.ID), 96),
			Goal:      truncate(strings.TrimSpace(task.Goal), 512),
			Agent:     truncate(strings.TrimSpace(task.Agent), 64),
			Status:    task.Status,
			StepCount: len(task.Steps),
			CreatedAt: task.CreatedAt,
			UpdatedAt: task.UpdatedAt,
		}
		for _, step := range task.Steps {
			switch step.Status {
			case taskgraph.StatusCompleted:
				item.DoneSteps++
			case taskgraph.StatusFailed, taskgraph.StatusBlocked:
				item.FailedSteps++
			}
		}
		out = append(out, item)
	}
	return out
}

func truncate(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "…"
}
