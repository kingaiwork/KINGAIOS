package update

// LoadSlotStateFile exposes the same validated atomic state format used by the
// updater to the offline Recovery environment.
func LoadSlotStateFile(path string) (SlotState, error) { return loadSlotState(path) }

// SaveSlotStateFile atomically persists a validated slot state.
func SaveSlotStateFile(path string, state SlotState) error { return saveSlotState(path, state) }
