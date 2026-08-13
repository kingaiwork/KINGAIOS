package update

import "testing"

func TestABStageAndConfirm(t *testing.T) {
	s, err := NewSlotState(SlotA, "1.0.0")
	if err != nil { t.Fatal(err) }
	plan, err := s.PlanStage("1.0.1")
	if err != nil { t.Fatal(err) }
	if plan.TargetSlot != SlotB || plan.Executable { t.Fatalf("unexpected plan: %#v", plan) }
	s, err = s.MarkPending("1.0.1")
	if err != nil { t.Fatal(err) }
	if s.PendingSlot != SlotB || s.PreviousSlot != SlotA || s.Confirmed { t.Fatalf("unexpected pending state: %#v", s) }
	s, err = s.RecordBootAttempt()
	if err != nil { t.Fatal(err) }
	s, err = s.ConfirmPending()
	if err != nil { t.Fatal(err) }
	if s.ActiveSlot != SlotB || s.ActiveVersion != "1.0.1" || !s.Confirmed { t.Fatalf("unexpected confirmed state: %#v", s) }
	if err := s.Validate(); err != nil { t.Fatalf("confirmed state must validate: %v", err) }
}

func TestABRollbackThreshold(t *testing.T) {
	s, _ := NewSlotState(SlotA, "1.0.0")
	s, _ = s.MarkPending("1.0.1")
	for i := 0; i < s.MaxBootAttempts; i++ {
		var err error
		s, err = s.RecordBootAttempt()
		if err != nil { t.Fatal(err) }
	}
	if !s.RollbackRequired { t.Fatal("rollback must be required after boot attempt threshold") }
	if _, err := s.ConfirmPending(); err == nil { t.Fatal("rollback-marked update must not be confirmable") }
}

func TestABRejectsUnsafeTransitions(t *testing.T) {
	s, _ := NewSlotState(SlotA, "1.0.0")
	if _, err := s.PlanStage("1.0.0"); err == nil { t.Fatal("same version must be rejected") }

	bad := s
	bad.PendingSlot = SlotA
	bad.PendingVersion = "1.0.1"
	if err := bad.Validate(); err == nil { t.Fatal("pending slot cannot equal active slot") }

	bad = s
	bad.PendingVersion = "orphan"
	if err := bad.Validate(); err == nil { t.Fatal("pending version without pending slot must be rejected") }

	bad = s
	bad.BootAttempts = 1
	if err := bad.Validate(); err == nil { t.Fatal("idle state cannot retain boot attempts") }

	bad = s
	bad.RollbackRequired = true
	if err := bad.Validate(); err == nil { t.Fatal("idle state cannot require rollback") }

	bad = s
	bad.Confirmed = false
	if err := bad.Validate(); err == nil { t.Fatal("idle state must be confirmed") }

	bad, _ = s.MarkPending("1.0.1")
	bad.PreviousSlot = SlotB
	if err := bad.Validate(); err == nil { t.Fatal("pending state must retain active slot as previous slot") }

	bad, _ = s.MarkPending("1.0.1")
	bad.Confirmed = true
	if err := bad.Validate(); err == nil { t.Fatal("pending state cannot claim confirmation") }
}
