package memory

import (
	"encoding/json"
	"testing"
)

func TestFileStoreRoundTrip(t *testing.T) {
	s := FileStore{Root:t.TempDir()}
	r, err := s.Put("user-1", "semantic", "private", json.RawMessage(`{"text":"hello"}`))
	if err != nil { t.Fatal(err) }
	got, err := s.List("user-1", 10); if err != nil { t.Fatal(err) }
	if len(got) != 1 || got[0].ID != r.ID { t.Fatalf("unexpected records: %#v", got) }
	if err := s.Delete("user-1", r.ID); err != nil { t.Fatal(err) }
}

func TestOwnerTraversalRejected(t *testing.T) {
	s := FileStore{Root:t.TempDir()}
	if _, err := s.Put("../root", "semantic", "private", json.RawMessage(`{}`)); err == nil { t.Fatal("path traversal must be rejected") }
}
