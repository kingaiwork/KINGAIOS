package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Event struct {
	Time             time.Time `json:"time"`
	Type             string    `json:"type"`
	Agent            string    `json:"agent,omitempty"`
	Capability       string    `json:"capability,omitempty"`
	Allowed          bool      `json:"allowed,omitempty"`
	ApprovalRequired bool      `json:"approval_required,omitempty"`
	Risk             int       `json:"risk,omitempty"`
	TargetHash       string    `json:"target_hash,omitempty"`
}

func HashTarget(target string) string {
	if target == "" { return "" }
	h := sha256.Sum256([]byte(target)); return hex.EncodeToString(h[:])
}

func Append(path string, e Event) error {
	if e.Time.IsZero() { e.Time = time.Now().UTC() }
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { return err }
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); if err != nil { return err }
	defer f.Close()
	b, err := json.Marshal(e); if err != nil { return err }
	if _, err := f.Write(append(b, '\n')); err != nil { return err }
	return f.Sync()
}
