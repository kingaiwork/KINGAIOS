package update

import (
	"errors"
	"fmt"
	"time"
)

type Slot string

const (
	SlotA Slot = "A"
	SlotB Slot = "B"
)

type SlotState struct {
	ActiveSlot       Slot      `json:"active_slot"`
	PreviousSlot     Slot      `json:"previous_slot,omitempty"`
	PendingSlot      Slot      `json:"pending_slot,omitempty"`
	ActiveVersion    string    `json:"active_version"`
	PendingVersion   string    `json:"pending_version,omitempty"`
	BootAttempts     int       `json:"boot_attempts"`
	MaxBootAttempts  int       `json:"max_boot_attempts"`
	Confirmed        bool      `json:"confirmed"`
	RollbackRequired bool      `json:"rollback_required"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type StagePlan struct {
	FromSlot      Slot   `json:"from_slot"`
	TargetSlot    Slot   `json:"target_slot"`
	CurrentVersion string `json:"current_version"`
	TargetVersion string `json:"target_version"`
	Destructive   bool   `json:"destructive"`
	Executable    bool   `json:"executable"`
	Reason        string `json:"reason"`
}

func NewSlotState(active Slot, version string) (SlotState, error) {
	if !validSlot(active) {
		return SlotState{}, errors.New("active slot must be A or B")
	}
	if version == "" {
		return SlotState{}, errors.New("active version is required")
	}
	return SlotState{
		ActiveSlot: active,
		ActiveVersion: version,
		MaxBootAttempts: 3,
		Confirmed: true,
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (s SlotState) Validate() error {
	if !validSlot(s.ActiveSlot) {
		return errors.New("invalid active slot")
	}
	if s.ActiveVersion == "" {
		return errors.New("active version is required")
	}
	if s.MaxBootAttempts <= 0 || s.MaxBootAttempts > 10 {
		return errors.New("max boot attempts must be between 1 and 10")
	}
	if s.PendingSlot != "" {
		if !validSlot(s.PendingSlot) {
			return errors.New("invalid pending slot")
		}
		if s.PendingSlot == s.ActiveSlot {
			return errors.New("pending slot must be inactive")
		}
		if s.PendingVersion == "" {
			return errors.New("pending version is required when a pending slot exists")
		}
	}
	if s.BootAttempts < 0 || s.BootAttempts > s.MaxBootAttempts {
		return errors.New("boot attempt counter is outside policy")
	}
	return nil
}

func (s SlotState) PlanStage(targetVersion string) (StagePlan, error) {
	if err := s.Validate(); err != nil {
		return StagePlan{}, err
	}
	if targetVersion == "" {
		return StagePlan{}, errors.New("target version is required")
	}
	if s.PendingSlot != "" {
		return StagePlan{}, errors.New("an update is already pending")
	}
	if targetVersion == s.ActiveVersion {
		return StagePlan{}, errors.New("target version equals active version")
	}
	return StagePlan{
		FromSlot: s.ActiveSlot,
		TargetSlot: otherSlot(s.ActiveSlot),
		CurrentVersion: s.ActiveVersion,
		TargetVersion: targetVersion,
		Destructive: true,
		Executable: false,
		Reason: "planning-only: artifact verification, inactive-slot write, bootloader transaction and recovery checks are still required",
	}, nil
}

func (s SlotState) MarkPending(targetVersion string) (SlotState, error) {
	plan, err := s.PlanStage(targetVersion)
	if err != nil {
		return SlotState{}, err
	}
	s.PreviousSlot = s.ActiveSlot
	s.PendingSlot = plan.TargetSlot
	s.PendingVersion = targetVersion
	s.BootAttempts = 0
	s.Confirmed = false
	s.RollbackRequired = false
	s.UpdatedAt = time.Now().UTC()
	return s, nil
}

func (s SlotState) RecordBootAttempt() (SlotState, error) {
	if err := s.Validate(); err != nil {
		return SlotState{}, err
	}
	if s.PendingSlot == "" {
		return SlotState{}, errors.New("no pending update")
	}
	if s.Confirmed {
		return SlotState{}, errors.New("pending update is already confirmed")
	}
	if s.BootAttempts >= s.MaxBootAttempts {
		s.RollbackRequired = true
		return s, errors.New("boot-attempt limit already reached")
	}
	s.BootAttempts++
	if s.BootAttempts >= s.MaxBootAttempts {
		s.RollbackRequired = true
	}
	s.UpdatedAt = time.Now().UTC()
	return s, nil
}

func (s SlotState) ConfirmPending() (SlotState, error) {
	if err := s.Validate(); err != nil {
		return SlotState{}, err
	}
	if s.PendingSlot == "" || s.PendingVersion == "" {
		return SlotState{}, errors.New("no pending update to confirm")
	}
	if s.RollbackRequired {
		return SlotState{}, errors.New("cannot confirm an update already marked for rollback")
	}
	s.ActiveSlot = s.PendingSlot
	s.ActiveVersion = s.PendingVersion
	s.PendingSlot = ""
	s.PendingVersion = ""
	s.BootAttempts = 0
	s.Confirmed = true
	s.RollbackRequired = false
	s.UpdatedAt = time.Now().UTC()
	return s, nil
}

func (s SlotState) Rollback() (SlotState, error) {
	if err := s.Validate(); err != nil {
		return SlotState{}, err
	}
	if s.PreviousSlot == "" {
		return SlotState{}, errors.New("no previous slot is known")
	}
	if !validSlot(s.PreviousSlot) {
		return SlotState{}, fmt.Errorf("invalid previous slot %q", s.PreviousSlot)
	}
	// Until bootloader integration exists, rollback state deliberately records
	// intent only. It never claims a disk/firmware transaction occurred.
	s.PendingSlot = ""
	s.PendingVersion = ""
	s.BootAttempts = 0
	s.Confirmed = true
	s.RollbackRequired = false
	s.UpdatedAt = time.Now().UTC()
	return s, nil
}

func validSlot(s Slot) bool { return s == SlotA || s == SlotB }
func otherSlot(s Slot) Slot { if s == SlotA { return SlotB }; return SlotA }
