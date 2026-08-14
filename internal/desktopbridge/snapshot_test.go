package desktopbridge

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/agent"
	"github.com/kingaiwork/KINGAIOS/internal/memory"
	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

func TestBuildPrivateSnapshotDropsRawTaskStepData(t *testing.T) {
	now := time.Now().UTC()
	longGoal := strings.Repeat("界", 600)
	tasks := []taskgraph.Task{{
		ID: "task-1",
		Goal: longGoal,
		Agent: "main",
		PeerUID: 1000,
		Status: taskgraph.StatusRunning,
		Steps: []taskgraph.Step{{
			ID: "step-1",
			Title: "private step title",
			Capability: "filesystem.write",
			Target: "/home/user/private-output",
			ApprovalID: "approval-private",
			Status: taskgraph.StatusCompleted,
			Result: json.RawMessage(`{"token":"step-result-secret"}`),
		}},
		CreatedAt: now.Add(-time.Minute),
		UpdatedAt: now,
	}}
	mem := memory.Summary{Total: 3, ByLayer: map[string]int{"M1": 2, "M4": 1}, BySensitivity: map[string]int{"private": 3}}

	snapshot := Build(1000, "0.1-test", tasks, mem, now)
	if snapshot.Schema != Schema || snapshot.UserUID != 1000 || snapshot.Memory.Total != 3 { t.Fatalf("unexpected snapshot: %#v", snapshot) }
	if snapshot.Agents == nil { t.Fatal("agents must serialize as an empty array, not null") }
	if len(snapshot.Tasks) != 1 || snapshot.Tasks[0].DoneSteps != 1 || snapshot.Tasks[0].StepCount != 1 { t.Fatalf("unexpected tasks: %#v", snapshot.Tasks) }
	if len([]rune(snapshot.Tasks[0].Goal)) != 513 || !strings.HasSuffix(snapshot.Tasks[0].Goal, "…") { t.Fatalf("goal was not bounded: %d", len([]rune(snapshot.Tasks[0].Goal))) }

	b, err := json.Marshal(snapshot)
	if err != nil { t.Fatal(err) }
	text := strings.ToLower(string(b))
	for _, forbidden := range []string{"private step title", "filesystem.write", "/home/user/private-output", "approval-private", "step-result-secret", "peer_uid", "target", "approval_id", "result"} {
		if strings.Contains(text, strings.ToLower(forbidden)) { t.Fatalf("private snapshot leaked raw task field %q: %s", forbidden, text) }
	}
}

func TestSummarizeAgentsExposesRoleAndCountNotCapabilityNames(t *testing.T) {
	defs := []agent.Definition{
		{ID:"main", Role:"assistant", Capabilities:[]string{"filesystem.read","network.read"}},
		{ID:"system-ops", Role:"privileged-system", Capabilities:[]string{"service.restart","package.install"}},
	}
	summary := SummarizeAgents(defs, func(id string) bool { return id == "main" })
	if len(summary) != 2 { t.Fatalf("len=%d", len(summary)) }
	if summary[0].ID != "main" || summary[0].Role != "assistant" || summary[0].CapabilityCount != 2 || !summary[0].Authorized { t.Fatalf("unexpected main summary: %#v", summary[0]) }
	if summary[1].ID != "system-ops" || summary[1].CapabilityCount != 2 || summary[1].Authorized { t.Fatalf("unexpected system summary: %#v", summary[1]) }
	b, err := json.Marshal(summary)
	if err != nil { t.Fatal(err) }
	text := strings.ToLower(string(b))
	for _, forbidden := range []string{"filesystem.read","network.read","service.restart","package.install"} {
		if strings.Contains(text, forbidden) { t.Fatalf("Agent summary leaked capability name %q: %s", forbidden, text) }
	}
}

func TestBuildNormalizesTimeAndVersion(t *testing.T) {
	zone := time.FixedZone("test", -7*60*60)
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, zone)
	snapshot := Build(42, " 1.2.3 ", nil, memory.Summary{}, now)
	if snapshot.Version != "1.2.3" { t.Fatalf("version=%q", snapshot.Version) }
	if snapshot.UpdatedAt.Location() != time.UTC { t.Fatalf("updated_at location=%v", snapshot.UpdatedAt.Location()) }
	if snapshot.Tasks == nil { t.Fatal("tasks must serialize as an empty array, not null") }
	if snapshot.Agents == nil { t.Fatal("agents must serialize as an empty array, not null") }
}
