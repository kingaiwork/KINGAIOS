package memory

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSummarizeDoesNotExposeMemoryData(t *testing.T) {
	store := FileStore{Root: t.TempDir()}
	owner := "uid-1000"
	future := time.Now().UTC().Add(time.Hour)
	past := time.Now().UTC().Add(-time.Hour)

	if _, err := store.PutWithMetadata(owner, "working", "private", Metadata{Layer: LayerWorking}, json.RawMessage(`{"secret":"private-memory-body"}`)); err != nil { t.Fatal(err) }
	if _, err := store.PutWithMetadata(owner, "task", "sensitive", Metadata{Layer: LayerTask, ExpiresAt: &future}, json.RawMessage(`{"token":"never-publish"}`)); err != nil { t.Fatal(err) }
	if _, err := store.PutWithMetadata(owner, "semantic", "private", Metadata{Layer: LayerSemantic, ExpiresAt: &past}, json.RawMessage(`{"expired":"hidden"}`)); err != nil { t.Fatal(err) }

	summary, err := store.Summarize(owner)
	if err != nil { t.Fatal(err) }
	if summary.Total != 2 || summary.ByLayer["M1"] != 1 || summary.ByLayer["M2"] != 1 || summary.Expiring != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if summary.BySensitivity["private"] != 1 || summary.BySensitivity["sensitive"] != 1 {
		t.Fatalf("unexpected sensitivity summary: %#v", summary.BySensitivity)
	}
	b, err := json.Marshal(summary)
	if err != nil { t.Fatal(err) }
	text := strings.ToLower(string(b))
	for _, forbidden := range []string{"private-memory-body", "never-publish", "hidden", "data", "token", "secret"} {
		if strings.Contains(text, forbidden) { t.Fatalf("summary leaked memory content token %q: %s", forbidden, text) }
	}
}

func TestSummarizeMissingOwnerReturnsEmptyMaps(t *testing.T) {
	store := FileStore{Root: t.TempDir()}
	summary, err := store.Summarize("uid-404")
	if err != nil { t.Fatal(err) }
	if summary.Total != 0 || len(summary.ByLayer) != 0 || len(summary.BySensitivity) != 0 {
		t.Fatalf("unexpected empty summary: %#v", summary)
	}
}
