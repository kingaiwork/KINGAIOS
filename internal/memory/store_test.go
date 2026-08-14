package memory

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFileStoreRoundTrip(t *testing.T) {
	s := FileStore{Root:t.TempDir()}
	r, err := s.Put("user-1", "semantic", "private", json.RawMessage(`{"text":"hello"}`))
	if err != nil { t.Fatal(err) }
	got, err := s.List("user-1", 10); if err != nil { t.Fatal(err) }
	if len(got) != 1 || got[0].ID != r.ID || got[0].Layer != LayerSemantic { t.Fatalf("unexpected records: %#v", got) }
	if err := s.Delete("user-1", r.ID); err != nil { t.Fatal(err) }
}

func TestOwnerTraversalRejected(t *testing.T) {
	s := FileStore{Root:t.TempDir()}
	if _, err := s.Put("../root", "semantic", "private", json.RawMessage(`{}`)); err == nil { t.Fatal("path traversal must be rejected") }
}

func TestPutInfersLayerAndSearches(t *testing.T) {
	s := FileStore{Root: t.TempDir()}
	r, err := s.PutWithMetadata("uid-1000", "task", "private", Metadata{Agent: "main", Namespace: "project-a", Importance: 0.8, Source: "user"}, json.RawMessage(`{"note":"deploy nginx"}`))
	if err != nil { t.Fatal(err) }
	if r.Layer != LayerTask { t.Fatalf("layer=%s", r.Layer) }
	items, err := s.Search("uid-1000", Query{Text: "nginx", Agent: "main", Layer: LayerTask, Limit: 10})
	if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].ID != r.ID { t.Fatalf("items=%#v", items) }
}

func TestPromotionMustBeAdjacent(t *testing.T) {
	s := FileStore{Root: t.TempDir()}
	r, err := s.PutWithMetadata("uid-1000", "working", "private", Metadata{Layer: LayerWorking}, json.RawMessage(`{"x":1}`))
	if err != nil { t.Fatal(err) }
	if _, err := s.Promote("uid-1000", r.ID, LayerSemantic); err == nil { t.Fatal("expected non-adjacent promotion failure") }
	r, err = s.Promote("uid-1000", r.ID, LayerTask)
	if err != nil { t.Fatal(err) }
	if r.Layer != LayerTask { t.Fatalf("layer=%s", r.Layer) }
}

func TestExpiredRecordsAreHiddenAndPurged(t *testing.T) {
	s := FileStore{Root: t.TempDir()}
	expires := time.Now().UTC().Add(-time.Minute)
	r, err := s.PutWithMetadata("uid-1000", "context", "private", Metadata{ExpiresAt: &expires}, json.RawMessage(`{"x":1}`))
	if err != nil { t.Fatal(err) }
	items, err := s.List("uid-1000", 10)
	if err != nil { t.Fatal(err) }
	if len(items) != 0 { t.Fatalf("items=%#v", items) }
	if _, err := s.Get("uid-1000", r.ID); err == nil { t.Fatal("expected expired record to be hidden") }
	removed, err := s.PurgeExpired("uid-1000")
	if err != nil { t.Fatal(err) }
	if removed != 1 { t.Fatalf("removed=%d", removed) }
}

func TestSearchSortsByImportanceThenRecency(t *testing.T) {
	s := FileStore{Root: t.TempDir()}
	low, err := s.PutWithMetadata("uid-1000", "semantic", "private", Metadata{Importance: 0.2}, json.RawMessage(`{"name":"low"}`))
	if err != nil { t.Fatal(err) }
	high, err := s.PutWithMetadata("uid-1000", "semantic", "private", Metadata{Importance: 0.9}, json.RawMessage(`{"name":"high"}`))
	if err != nil { t.Fatal(err) }
	items, err := s.Search("uid-1000", Query{Layer: LayerSemantic, Limit: 10})
	if err != nil { t.Fatal(err) }
	if len(items) != 2 || items[0].ID != high.ID || items[1].ID != low.ID { t.Fatalf("items=%#v", items) }
}

func TestMetadataValidation(t *testing.T) {
	s := FileStore{Root: t.TempDir()}
	if _, err := s.PutWithMetadata("uid-1000", "semantic", "private", Metadata{Confidence: 1.1}, json.RawMessage(`{"x":1}`)); err == nil { t.Fatal("expected confidence validation") }
	if _, err := s.PutWithMetadata("uid-1000", "semantic", "private", Metadata{Layer: Layer("M9")}, json.RawMessage(`{"x":1}`)); err == nil { t.Fatal("expected layer validation") }
	if _, err := s.Put("uid-1000", "semantic", "private", json.RawMessage(`not-json`)); err == nil { t.Fatal("expected JSON validation") }
}
