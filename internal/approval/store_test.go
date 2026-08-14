package approval

import (
	"testing"
	"time"
)

func TestApprovalLifecycle(t *testing.T) {
	s := Store{Root: t.TempDir()}
	r, err := s.Create("system-ops", "service.restart", "abc", 1000, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != StatusPending {
		t.Fatalf("status=%s", r.Status)
	}
	approved, err := s.Decide(r.ID, true, 0)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != StatusApproved {
		t.Fatalf("status=%s", approved.Status)
	}
	consumed, err := s.Consume(r.ID, "system-ops", "service.restart", "abc", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if consumed.Status != StatusConsumed {
		t.Fatalf("status=%s", consumed.Status)
	}
	if _, err := s.Consume(r.ID, "system-ops", "service.restart", "abc", 1000); err == nil {
		t.Fatal("expected second consume to fail")
	}
}

func TestApprovalBindingMismatch(t *testing.T) {
	s := Store{Root: t.TempDir()}
	r, err := s.Create("system-ops", "service.restart", "abc", 1000, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Decide(r.ID, true, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Consume(r.ID, "system-ops", "service.restart", "different", 1000); err == nil {
		t.Fatal("expected target binding mismatch")
	}
}

func TestApprovalExpiry(t *testing.T) {
	s := Store{Root: t.TempDir()}
	r, err := s.Create("system-ops", "service.restart", "abc", 1000, time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	got, err := s.Get(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusExpired {
		t.Fatalf("status=%s", got.Status)
	}
}
